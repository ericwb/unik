package osv

import (
	"io"
	"github.com/emc-advanced-dev/unik/pkg/types"
	"os"
	"github.com/Sirupsen/logrus"
	"io/ioutil"

	unikos "github.com/emc-advanced-dev/unik/pkg/os"
	unikutil "github.com/emc-advanced-dev/unik/pkg/util"
	"os/exec"
	"github.com/emc-advanced-dev/pkg/errors"
	"path/filepath"
)

type OsvCompiler struct {
	ExtraConfig types.ExtraConfig
}

func (osvCompiler *OsvCompiler) CompileRawImage(sourceTar io.ReadCloser, args string, mntPoints []string) (_ *types.RawImage, err error) {
	localFolder, err := ioutil.TempDir(unikutil.UnikTmpDir(), "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(localFolder)
	logrus.Debugf("extracting uploaded files to "+localFolder)
	if err := unikos.ExtractTar(sourceTar, localFolder); err != nil {
		return nil, err
	}
	cmd := exec.Command("docker", "run", "--rm", "--privileged",
		"-v", "/dev/:/dev/",
		"-v", localFolder+"/:/project_directory/",
		"projectunik/compilers-osv-java",
	)
	logrus.WithFields(logrus.Fields{
		"command": cmd.Args,
	}).Debugf("running compilers-osv-java container")
	unikutil.LogCommand(cmd, true)
	err = cmd.Run()
	if err != nil {
		return nil, errors.New("failed running compilers-osv-java on "+localFolder, err)
	}

	resultFile, err := ioutil.TempFile(unikutil.UnikTmpDir(), "osv-vmdk")
	if err != nil {
		return nil, errors.New("failed to create tmpfile for result", err)
	}
	defer func(){
		if err != nil {
			os.Remove(resultFile.Name())
		}
	}()

	if err := os.Rename(filepath.Join(localFolder, "boot.qcow2"), resultFile.Name()); err != nil {
		return nil, errors.New("failed to rename result file", err)
	}

	return &types.RawImage{
		LocalImagePath: resultFile.Name(),
		ExtraConfig: 	osvCompiler.ExtraConfig,
		DeviceMappings: []types.DeviceMapping{}, //TODO: not supported yet
	}, nil
}