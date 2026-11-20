package tmux

import "os/exec"

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Output(name string, args ...string) ([]byte, error)
	Run(name string, args ...string) error
}

type execRunner struct{}

func (execRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (execRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// runner is the package-level command runner, replaceable in tests.
var runner CommandRunner = execRunner{}

// SetRunner replaces the command runner (for testing).
func SetRunner(r CommandRunner) {
	runner = r
}

func output(args ...string) ([]byte, error) {
	return runner.Output("tmux", SocketArgs(args...)...)
}

func run(args ...string) error {
	return runner.Run("tmux", SocketArgs(args...)...)
}
