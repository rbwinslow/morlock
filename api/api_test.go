package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

type temporaryGitRepo struct {
	path string
}

func (tgr *temporaryGitRepo) cleanUp() error {
	return os.RemoveAll(tgr.path)
}

func (tgr *temporaryGitRepo) init() error {
	cmd := exec.Command("git", "init")
	cmd.Dir = tgr.path
	return cmd.Run()
}

func withTemporaryGitRepo(fn func(repo *temporaryGitRepo)) error {
	path, err := ioutil.TempDir("", "morlock-api-test-")
	if err != nil {
		return err
	}
	var tgr = temporaryGitRepo{path: path}
	tgr.init()
	defer tgr.cleanUp()
	fn(&tgr)
	return nil
}

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Morlock API")
}
