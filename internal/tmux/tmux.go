package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qail/internal/clip"
)

func Attach(sessionName string) {
	cmd := fmt.Sprintf("tmux a -t %s", sessionName)
	clip.Cmd(cmd)
}

func SessionName(path string) string {
	return filepath.Base(path)
}

func isEven(i int) bool {
	return i%2 == 0
}

func Launch(folderPath string) error {
	// Change directory to the given folder
	err := os.Chdir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	sessionName := SessionName(folderPath)
	// create the root session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", folderPath, "-n", "root")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create tmux session: %v", err)
	}

	subFolders, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	if len(subFolders) == 0 {
		return nil
	}

	// Create a new window for SubFolders
	cmd = exec.Command("tmux", "new-window", "-t", sessionName, "-n", "SubFolders")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create new window: %v", err)
	}

	var folderNumber = 0
	var windowIndex = 0

	// Split panes for each subfolder
	for _, subFolder := range subFolders {
		if subFolder.IsDir() && strings.HasPrefix(subFolder.Name(), ".") {
			continue
		}
		if subFolder.IsDir() {
			subfolderPath := filepath.Join(folderPath, subFolder.Name())

			if isEven(folderNumber) {
				windowIndex++
				cmd = exec.Command("tmux", "new-window", "-t", sessionName, "-c", subfolderPath, "-n", "SubFolders"+fmt.Sprintf("%d", windowIndex))
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to create new window %s, %v", cmd, err)
				}
			} else {
				cmd = exec.Command("tmux", "split-window", "-t", fmt.Sprintf("%s:%d", sessionName, windowIndex), "-c", subfolderPath, "-h")

				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to split window: %v, cmd: %s", err, cmd)
				}
			}
			folderNumber++
		}
	}

	return nil
}

// checkTmuxSessionExists checks if a tmux session with the given name exists
func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// tmux returns exit code 1 if the session does not exist
			if exitError.ExitCode() == 1 {
				return false
			}
		}
		fmt.Printf("Error checking tmux session: %v\n", err)
	}
	return true
}

// isInstalled checks if tmux is installed
func IsInstalled() (error, bool) {
	cmd := exec.Command("tmux", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err, false
	}
	fmt.Printf("tmux version: %s\n", output)
	return nil, true
}

func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#S")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running tmux command: %s, stderr: %s", err, stderr.String())
	}

	sessions := strings.Split(strings.TrimSpace(out.String()), "\n")
	return sessions, nil
}

func RemoveSession(session string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", session)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to remove tmux session '%s': %s", session, stderr.String())
	}

	return nil
}
