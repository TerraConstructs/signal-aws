package signal

import (
	"os"
	"os/exec"

	"go.uber.org/zap"
)

type Executor interface {
	Run(cmdLine string) (exitCode int, err error)
}

type DefaultExecutor struct {
	Logger Logger
}

func NewDefaultExecutor(logger Logger) *DefaultExecutor {
	return &DefaultExecutor{
		Logger: logger,
	}
}

func (e *DefaultExecutor) Run(cmdLine string) (int, error) {
	e.Logger.Debug("Executing command", zap.String("command", cmdLine))

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
