package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type CmdIntf interface {
	Run() error
}
type CommandFn func(stdin io.Reader, stdout, stderr io.Writer, cwd string, cmd string, args ...string) CmdIntf

func defaultCommandImpl(stdin io.Reader, stdout, stderr io.Writer, cwd string, cmd string, args ...string) CmdIntf {
	Cmd := exec.Command(cmd, args...)
	Cmd.Stdin = stdin
	Cmd.Stderr = stderr
	Cmd.Stdout = stdout
	Cmd.Dir = cwd
	return Cmd
}

func checkForGit(cmdFn CommandFn) error {
	var stdout, stderr bytes.Buffer
	cmd := cmdFn(nil, &stdout, &stderr, "", "which", "git")
	err := cmd.Run()

	if err != nil && stdout.String() == "" {
		return errors.New("Git not installed")
	}

	return nil
}

type LocalGitRepo struct {
	path string
}

func OpenLocalGitRepo(path string, cmdFn CommandFn) (*LocalGitRepo, error) {
	if cmdFn == nil {
		cmdFn = defaultCommandImpl
	}

	if err := checkForGit(cmdFn); err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("\"%s\" is not a directory", path)
	}

	var stdout, stderr bytes.Buffer
	cmd := cmdFn(nil, &stdout, &stderr, path, "git", "status")
	err = cmd.Run()

	if err != nil {
		if stderr.Len() > 0 {
			return nil, errors.New(stderr.String())
		} else {
			if len(err.Error()) == 0 && stdout.Len() > 0 {
				return nil, errors.New(stdout.String())
			} else {
				return nil, err
			}
		}
		return nil, errors.New(stdout.String())
	}

	return &LocalGitRepo{path: path}, nil
}
