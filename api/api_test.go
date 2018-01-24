package api_test

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rbwinslow/morlock/api"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"fmt"
	"strings"
	"regexp"
)

type temporaryGitRepo struct {
	path      string
	userName  string
	userEmail string
}

func (tgr *temporaryGitRepo) addFile(path string, contents string) error {
	absolutePath := filepath.Join(tgr.path, path)

	file, err := os.Create(absolutePath)
	if err != nil {
		return err
	}

	file.WriteString(contents)
	file.Close()

	err, stdout, stderr := tgr.runGitCommand("add", path)
	if err != nil {
		return api.CookedErrorFromGitExec(stdout, stderr, err)
	}

	return nil
}

func (tgr *temporaryGitRepo) cleanUp() error {
	return os.RemoveAll(tgr.path)
}

func (tgr *temporaryGitRepo) config(key, value string) error {
	err, stdout, stderr := tgr.runGitCommand("config", key, value)
	if err != nil {
		return api.CookedErrorFromGitExec(stdout, stderr, err)
	}
	return nil
}

func (tgr *temporaryGitRepo) commit(message string) (api.ShortHash, error) {
	nohash := api.ShortHash{}
	err, stdout, stderr := tgr.runGitCommand("commit", "-m", message)
	if err != nil {
		return nohash, api.CookedErrorFromGitExec(stdout, stderr, err)
	}
	output := stdout.String()
	firstLine := output[:strings.Index(output, "\n")]
	re := regexp.MustCompile(`\[\w[\w\d\-/]*(?:[^\]]+) ([[:xdigit:]]{7})\]`)
	matches := re.FindStringSubmatch(firstLine)
	if len(matches) == 2 && len(matches[1]) == len(api.ShortHash{}) {
		var hash api.ShortHash
		copy(hash[:], matches[1])
		return hash, nil
	} else {
		return nohash, fmt.Errorf("Couldn't find commit hash in first line of output from git commit:\n%v", firstLine)
	}
}

func (tgr *temporaryGitRepo) init() error {
	err, stdout, stderr := tgr.runGitCommand("init")
	if err != nil {
		return api.CookedErrorFromGitExec(stdout, stderr, err)
	}
	return nil
}

func (tgr *temporaryGitRepo) runGitCommand(args ...string) (e error, stdout, stderr *bytes.Buffer) {
	cmd := exec.Command("git", args...)
	cmd.Dir = tgr.path

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	e = cmd.Run()
	return
}

func withTemporaryGitRepo(fn func(repo *temporaryGitRepo)) error {
	path, err := ioutil.TempDir("", "morlock-api-test-")
	if err != nil {
		return err
	}
	var tgr = temporaryGitRepo{path: path, userName: "Morlock Test", userEmail: "morlock@test.com"}
	tgr.init()
	tgr.config("user.name", tgr.userName)
	tgr.config("user.email", tgr.userEmail)
	defer tgr.cleanUp()
	fn(&tgr)
	return nil
}

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Morlock API")
}
