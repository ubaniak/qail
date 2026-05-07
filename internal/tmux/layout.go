package tmux

import (
	"fmt"
	"path/filepath"

	"github.com/ubaniak/qail/internal/runner"
)

// Plan returns the sequence of runner.Commands that build a tmux session for
// a workspace rooted at root, with one pane per non-hidden subfolder.
//
// Output shape:
//   - Always starts with: tmux new-session -d -s <sessionName> -c <root> -n root
//   - If subfolders is empty: returns only that command.
//   - Otherwise: tmux new-window -t <sessionName> -n SubFolders
//   - Then alternates per subfolder, starting with split:
//     even index -> split-window -t <sessionName>:<windowIndex> -c <subPath> -h
//     odd  index -> windowIndex++ ; new-window -t <sessionName> -c <subPath>
//     -n SubFolders<windowIndex>
//
// The function is pure: no filesystem, no subprocess, no global state. Filter
// hidden / non-directory entries before calling Plan.
func Plan(sessionName, root string, subfolders []string) []runner.Command {
	cmds := []runner.Command{{
		Name: "tmux",
		Args: []string{"new-session", "-d", "-s", sessionName, "-c", root, "-n", "root"},
	}}
	if len(subfolders) == 0 {
		return cmds
	}

	cmds = append(cmds, runner.Command{
		Name: "tmux",
		Args: []string{"new-window", "-t", sessionName, "-n", "SubFolders"},
	})

	folderNumber := 0
	windowIndex := 0
	for _, name := range subfolders {
		subPath := filepath.Join(root, name)
		if isEven(folderNumber) {
			cmds = append(cmds, runner.Command{
				Name: "tmux",
				Args: []string{"split-window", "-t", fmt.Sprintf("%s:%d", sessionName, windowIndex), "-c", subPath, "-h"},
			})
		} else {
			windowIndex++
			cmds = append(cmds, runner.Command{
				Name: "tmux",
				Args: []string{"new-window", "-t", sessionName, "-c", subPath, "-n", fmt.Sprintf("SubFolders%d", windowIndex)},
			})
		}
		folderNumber++
	}
	return cmds
}

func isEven(i int) bool { return i%2 == 0 }
