package test_util

import (
	"strings"
	"regexp"
	"fmt"
	"bytes"
	"path/filepath"
	"os"
	"github.com/rbwinslow/morlock/api"
	"os/exec"
	"io/ioutil"
)

type TemporaryGitRepo struct {
	Path      string
	UserName  string
	UserEmail string
}

func (tgr *TemporaryGitRepo) CleanUp() error {
	return os.RemoveAll(tgr.Path)
}

func (tgr *TemporaryGitRepo) MustAddFile(path string, contents string) {
	panicFormat := "MustAddFile couldn't: %s"
	absolutePath := filepath.Join(tgr.Path, path)

	file, err := os.Create(absolutePath)
	if err != nil {
		panic(fmt.Sprintf(panicFormat, err.Error()))
	}

	file.WriteString(contents)
	file.Close()

	err, stdout, stderr := tgr.runGitCommand("add", path)
	if err != nil {
		panic(fmt.Sprintf(panicFormat, api.CookedErrorFromGitExec(stdout, stderr, err).Error()))
	}
}

func (tgr *TemporaryGitRepo) MustCommit(message string) api.ShortHash {
	hashLineRE := regexp.MustCompile(`\[\w[\w\d\-/]*(?:[^\]]+) ([[:xdigit:]]{7})\]`)

	err, stdout, stderr := tgr.runGitCommand("commit", "-m", message)
	if err != nil {
		panic(fmt.Sprintf("MustCommit couldn't: %s", api.CookedErrorFromGitExec(stdout, stderr, err).Error()))
	}

	output := stdout.String()
	firstLine := output[:strings.Index(output, "\n")]
	matches := hashLineRE.FindStringSubmatch(firstLine)
	if len(matches) == 2 && len(matches[1]) == len(api.ShortHash{}) {
		var hash api.ShortHash
		copy(hash[:], matches[1])
		return hash
	} else {
		panic(fmt.Sprintf("MustCommit couldn't find hash in git commit output:\n%s\n", output))
	}
}

func (tgr *TemporaryGitRepo) MustConfig(key, value string) {
	if err, stdout, stderr := tgr.runGitCommand("config", key, value) ; err != nil {
		msg := api.CookedErrorFromGitExec(stdout, stderr, err).Error()
		panic(fmt.Sprintf("MustConfig (%s=%s) couldn't: %s", key, value, msg))
	}
}

func (tgr *TemporaryGitRepo) MustInit() {
	if err, stdout, stderr := tgr.runGitCommand("init") ; err != nil {
		panic(fmt.Sprintf("MustInit couldn't: %s", api.CookedErrorFromGitExec(stdout, stderr, err).Error()))
	}
}

func (tgr *TemporaryGitRepo) runGitCommand(args ...string) (e error, stdout, stderr *bytes.Buffer) {
	cmd := exec.Command("git", args...)
	cmd.Dir = tgr.Path

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	e = cmd.Run()
	return
}

func WithTemporaryGitRepo(fn func(repo *TemporaryGitRepo)) error {
	path, err := ioutil.TempDir("", "morlock-api-test-")
	if err != nil {
		return err
	}
	var tgr = TemporaryGitRepo{Path: path, UserName: "Morlock Test", UserEmail: "morlock@test.com"}
	defer tgr.CleanUp()
	tgr.MustInit()
	tgr.MustConfig("user.name", tgr.UserName)
	tgr.MustConfig("user.email", tgr.UserEmail)
	fn(&tgr)
	return nil
}

