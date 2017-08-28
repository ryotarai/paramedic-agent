package paramedic

import (
	"errors"
	"os/exec"
	"syscall"
)

func exitStatusFromError(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	if eErr, ok := err.(*exec.ExitError); ok {
		if s, ok := eErr.Sys().(syscall.WaitStatus); ok {
			return s.ExitStatus(), nil
		}
		return -1, errors.New("an error does not implement syscall.WaitStatus")
	}
	return -1, err
}
