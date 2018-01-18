package api_test

import (
	"github.com/rbwinslow/morlock/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"os/exec"
	"strings"
	"bytes"
	"os"
)

type cmdMock struct {
	expectedCmd  string
	expectedArgs []string
	stdoutOutput string
	result       error
	wasRun       bool
	stdout       io.Writer
}

func (cm *cmdMock) Run() error {
	cm.wasRun = true
	cm.stdout.Write([]byte(cm.stdoutOutput))
	return cm.result
}

type commandMockFactory struct {
	expectations []*cmdMock
	currentCall  uint
}

func (cmf *commandMockFactory) addExpectation(result error, resultStdout string, cmd string, args ...string) {
	cmf.expectations = append(cmf.expectations, &cmdMock{expectedCmd: cmd, expectedArgs: args, stdoutOutput: resultStdout, result: result})
}

func (cmf *commandMockFactory) assertCommandsWereRun() {
	for _, m := range cmf.expectations {
		Expect(m.wasRun).To(BeTrue())
	}
}

func (cmf *commandMockFactory) factoryFn() api.CommandFn {
	return func(stdin io.Reader, stdout, stderr io.Writer, cwd string, cmd string, args ...string) api.CmdIntf {
		cur := cmf.currentCall
		cmf.currentCall++
		mock := cmf.expectations[cur]
		Expect(cmd).To(Equal(mock.expectedCmd))
		Expect(args).To(Equal(mock.expectedArgs))
		mock.stdout = stdout
		return mock
	}
}

func newCommandMockFactory() *commandMockFactory {
	return &commandMockFactory{expectations: make([]*cmdMock, 0, 0)}
}

var _ = Describe("Local Git repository", func() {

	Describe("Testing prerequisites", func() {
		It("needs git to be installed for tests to work", func() {
			cmd := exec.Command("which", "git")

			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			err := cmd.Run()
			Expect(err).To(BeNil())

			gitpath := strings.TrimRight(stdout.String(), "\n")
			info, err := os.Stat(gitpath)
			Expect(err).To(BeNil())
			Expect(info.Mode() & 0x1).ToNot(BeZero())
		})
	})

	Describe("OpenLocalGitRepo function", func() {
		It("should return 'Git not installed' error if `which git` shows no Git", func() {
			// Given
			mockFact := newCommandMockFactory()
			mockFact.addExpectation(&exec.ExitError{}, "", "which", "git")

			// When
			_, err := api.OpenLocalGitRepo("/does/not/matter", mockFact.factoryFn())

			// Then
			mockFact.assertCommandsWereRun()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Git not installed"))
		})

		It("should return 'No such directory' error when passed a non-existent directory", func() {
			// Given
			noSuchPath := "/there/is/no/such/directory"
			mockFact := newCommandMockFactory()
			mockFact.addExpectation(nil, "/bin/git", "which", "git")

			// When
			_, err := api.OpenLocalGitRepo(noSuchPath, mockFact.factoryFn())

			// Then
			mockFact.assertCommandsWereRun()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("not a directory"))
		})

		It("should return \"Not a git repository\" when passed a path to a non-Git-repo directory", func() {
			// Given
			notARepo := "/etc"

			// When
			_, err := api.OpenLocalGitRepo(notARepo, nil)

			// Then
			Expect(err).ToNot(BeNil())
			Expect(strings.ToLower(err.Error())).To(ContainSubstring("not a git repository"))
		})

		It("should successfully open a real git repository", func() {
			// Given
			withTemporaryGitRepo(func(repo *temporaryGitRepo) {
				// When
				_, err := api.OpenLocalGitRepo(repo.path, nil)

				// Then
				Expect(err).To(BeNil())
			})
		})
	})
})
