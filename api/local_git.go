package api

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
	"encoding/json"
)

// Functions that need to have their use of exec.Command tested should take
// a function-type argument that satisfies this type declaration. It's a
// "convenience" function that sets up all of the Cmd object's fields after
// instantiating it (fields are harder to mock). Just call this with all of
// the expected arguments, and it'll give you back something with a Run()
// function on it. When your caller passes nil for this function argument,
// use the defaultCommandImpl.
//
type CommandFn func(stdin io.Reader, stdout, stderr io.Writer, cwd string, cmd string, args ...string) CmdIntf
type CmdIntf interface {
	Run() error
}

func defaultCommandImpl(stdin io.Reader, stdout, stderr io.Writer, cwd string, cmd string, args ...string) CmdIntf {
	Cmd := exec.Command(cmd, args...)
	Cmd.Stdin = stdin
	Cmd.Stderr = stderr
	Cmd.Stdout = stdout
	Cmd.Dir = cwd
	return Cmd
}

func findClosestRepoDir(p string) (string, bool) {
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		p = path.Dir(p)
	}

	for len(p) > 0 && p != "/" {
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			break
		}
		if info, err := os.Stat(path.Join(p, ".git")); err == nil && info.IsDir() {
			return p, true
		}
		p = path.Dir(p)
	}

	return p, false
}

func CookedErrorFromGitExec(stdout, stderr *bytes.Buffer, err error) error {
	if err != nil {
		if stderr != nil && stderr.Len() > 0 {
			return errors.New(stderr.String())
		} else {
			if len(err.Error()) == 0 && stdout != nil && stdout.Len() > 0 {
				return errors.New(stdout.String())
			} else {
				return err
			}
		}
	}
	return nil
}

func OpenLocalGitRepo(p string, cmdFn CommandFn) (*LocalGitRepo, error) {
	if cmdFn == nil {
		cmdFn = defaultCommandImpl
	}

	if err := checkForGit(cmdFn); err != nil {
		return nil, err
	}

	gitpath, ok := findClosestRepoDir(p)
	if !ok {
		return nil, fmt.Errorf("Could not find git repository at [%s]", p)
	}

	var stdout, stderr bytes.Buffer
	cmd := cmdFn(nil, &stdout, &stderr, gitpath, "git", "status")

	if err := cmd.Run(); err != nil {
		return nil, CookedErrorFromGitExec(&stdout, &stderr, err)
	}

	return &LocalGitRepo{Path: gitpath}, nil
}

// Use OpenLocalGitRepo to create a LocalGitRepo based on a file system
// directory.
//
type LocalGitRepo struct {
	Path string
}

func (repo *LocalGitRepo) History(path string) (chan Commit, error) {
	cmd := exec.Command("git", "log", "--follow", "--no-color", "--date", "iso-strict", "--", path)
	cmd.Dir = repo.Path

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	out := make(chan Commit)
	done := make(chan struct{})
	scanner := bufio.NewScanner(stdout)

	go func() {
		reCommit := regexp.MustCompile(`^commit ([[:xdigit:]]{40})$`)
		reAuthor := regexp.MustCompile(`^Author:\s+(.+)$`)
		reDate := regexp.MustCompile(`^Date:\s+(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}-\d{2}:\d{2})$`)
		var commit *Commit

		defer close(out)
		defer func() {
			done <- struct{}{}
		}()

		for scanner.Scan() {
			text := scanner.Text()
			for _, line := range strings.Split(text, "\n") {
				match := reCommit.FindStringSubmatch(line)
				if len(match) == 2 {
					commit = new(Commit)
					copy(commit.Hash[:], []byte(match[1]))
				} else {
					match = reAuthor.FindStringSubmatch(line)
					if len(match) == 2 {
						commit.Author = match[1]
					} else {
						match = reDate.FindStringSubmatch(line)
						if len(match) == 2 {
							commit.Date, err = time.Parse(time.RFC3339, match[1])
							if err != nil {
								fmt.Fprintf(os.Stdout, "ERROR: Couldn't parse commit date %s\n", match[1])
							}
							out <- *commit
							commit = nil
						}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR reading git command's stdout: %v\n", err)
		}
	}()

	go func() {
		defer stdout.Close()
		err := cmd.Start()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error starting command", err)
			return
		}

		<-done

		err = cmd.Wait()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error waiting for command completion:", err)
		}
	}()

	return out, nil
}

type Commit struct {
	Hash
	Author string
	Date   time.Time
}

func (c *Commit) forJSON() *commitForJSON {
	return &commitForJSON{
		Hash: c.Hash.String(),
		Author: c.Author,
		Date: c.Date,
	}
}

type commitForJSON struct {
	Hash string		`json:"hash"`
	Author string	`json:"author"`
	Date time.Time	`json:"date"`
}

type CommitList []Commit

func (c *CommitList) ToJSON() ([]byte, error) {
	var facades []commitForJSON
	for _, commit := range *c {
		facades = append(facades, *commit.forJSON())
	}
	return json.Marshal(facades)
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
