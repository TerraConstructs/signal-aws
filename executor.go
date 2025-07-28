package signal

import (
	"log"
	"os"
	"os/exec"
)

type Executor interface {
	Run(cmdLine string) (exitCode int, err error)
}

type DefaultExecutor struct {
	Verbose bool
}

func NewDefaultExecutor(verbose bool) *DefaultExecutor {
	return &DefaultExecutor{
		Verbose: verbose,
	}
}

func (e *DefaultExecutor) Run(cmdLine string) (int, error) {
	if e.Verbose {
		log.Printf("Executing command: %s", cmdLine)
	}

	cmd := exec.Command("sh", "-c", cmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), nil
		}
		return -1, err
	}

	return 0, nil
}
