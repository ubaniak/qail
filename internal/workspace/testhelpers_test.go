package workspace

import (
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// buildTrackedWorkspace returns a config.Workspace pre-populated with the
// given names. Used by clean tests to seed "tracked" entries the cleaner
// should preserve.
func buildTrackedWorkspace(names ...string) config.Workspace {
	ws := make(config.Workspace, len(names))
	for _, n := range names {
		ws[n] = config.WorkspaceProfile{LastUsed: time.Now()}
	}
	return ws
}
