package git

import (
	"os/exec"
	"strings"
)

func Run(cwd string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	return cmd.Run()
}

func Output(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func IsRepo(cwd string) bool {
	_, err := Output(cwd, "rev-parse", "--is-inside-work-tree")
	return err == nil
}
