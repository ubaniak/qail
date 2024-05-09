package git

import (
	"os/exec"
	"qail/internal/forms"
)

func runCmd(args []string) (string, error) {

	cmd := exec.Command("git", args...)

	out, err := cmd.Output()

	return string(out), err
}

func Clone(repo, path string) (string, error) {
	args := []string{"clone", repo, path}
	return runCmd(args)
}

func ConeWithProgress(repo, path, message string) {
	clone := func() {
		Clone(repo, path)
	}
	forms.Spinner(clone, message)
}
