package tmux

import (
	"reflect"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

func cmd(args ...string) runner.Command {
	return runner.Command{Name: "tmux", Args: args}
}

func TestPlan(t *testing.T) {
	const sess = "work"
	const root = "/ws/work"

	cases := []struct {
		name       string
		subfolders []string
		want       []runner.Command
	}{
		{
			name:       "no subfolders: only the root session",
			subfolders: nil,
			want: []runner.Command{
				cmd("new-session", "-d", "-s", sess, "-c", root, "-n", "root"),
			},
		},
		{
			name:       "one subfolder splits at window 0",
			subfolders: []string{"a"},
			want: []runner.Command{
				cmd("new-session", "-d", "-s", sess, "-c", root, "-n", "root"),
				cmd("new-window", "-t", sess, "-n", "SubFolders"),
				cmd("split-window", "-t", "work:0", "-c", "/ws/work/a", "-h"),
			},
		},
		{
			name:       "two subfolders: split then new-window SubFolders1",
			subfolders: []string{"a", "b"},
			want: []runner.Command{
				cmd("new-session", "-d", "-s", sess, "-c", root, "-n", "root"),
				cmd("new-window", "-t", sess, "-n", "SubFolders"),
				cmd("split-window", "-t", "work:0", "-c", "/ws/work/a", "-h"),
				cmd("new-window", "-t", sess, "-c", "/ws/work/b", "-n", "SubFolders1"),
			},
		},
		{
			name:       "three subfolders: split, new-window, split window 1",
			subfolders: []string{"a", "b", "c"},
			want: []runner.Command{
				cmd("new-session", "-d", "-s", sess, "-c", root, "-n", "root"),
				cmd("new-window", "-t", sess, "-n", "SubFolders"),
				cmd("split-window", "-t", "work:0", "-c", "/ws/work/a", "-h"),
				cmd("new-window", "-t", sess, "-c", "/ws/work/b", "-n", "SubFolders1"),
				cmd("split-window", "-t", "work:1", "-c", "/ws/work/c", "-h"),
			},
		},
		{
			name:       "four subfolders: even/odd alternation continues",
			subfolders: []string{"a", "b", "c", "d"},
			want: []runner.Command{
				cmd("new-session", "-d", "-s", sess, "-c", root, "-n", "root"),
				cmd("new-window", "-t", sess, "-n", "SubFolders"),
				cmd("split-window", "-t", "work:0", "-c", "/ws/work/a", "-h"),
				cmd("new-window", "-t", sess, "-c", "/ws/work/b", "-n", "SubFolders1"),
				cmd("split-window", "-t", "work:1", "-c", "/ws/work/c", "-h"),
				cmd("new-window", "-t", sess, "-c", "/ws/work/d", "-n", "SubFolders2"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Plan(sess, root, tc.subfolders)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Plan() mismatch\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}
