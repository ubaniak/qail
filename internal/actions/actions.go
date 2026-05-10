// Package actions exposes one function per user-facing operation that
// mutates the qail Config. Each action wraps a single atomic
// Read+mutate+Write cycle against a config.Store, so callers (cmd/, tests,
// future REST/TUI front-ends) never juggle the read/write order or the
// associated error paths.
//
// Conventions:
//   - Each action is a free function taking a config.Store as its first
//     argument plus the already-resolved inputs. UI prompting (huh forms,
//     flags) belongs in the caller; actions take concrete values.
//   - Actions return a single error. Cobra handlers wrap them with
//     log.Fatalln; tests inspect the error directly.
//   - Aggregate effects stay inside the action: e.g. RemoveRepos also
//     strips the repo from every workspace's repo list, so callers cannot
//     forget the cascade.
//   - Concurrent callers (HTTP server, Cobra, background goroutines) all
//     funnel writes through readWrite which holds storeMu for the entire
//     read+mutate+write sequence. SQLite's own write lock would serialise
//     the WriteTransaction but cannot prevent the lost-update race when
//     two callers each Read a snapshot, mutate independently, and Write
//     back; the package-level mutex closes that gap. The cost is that all
//     mutations against any Store serialise — fine for qail's workload.
package actions

import (
	"sync"

	"github.com/ubaniak/qail/internal/config"
)

// storeMu serialises every Read-modify-Write cycle so concurrent callers
// (HTTP handler goroutines, Cobra invocations, future background jobs)
// cannot lose each other's mutations. See package doc.
var storeMu sync.Mutex

// readWrite is the load-bearing helper: read snapshot, run mutator, write
// back. Every action is one line: return readWrite(s, func(cfg) {...}).
// The mutex makes the whole cycle atomic with respect to other actions.
func readWrite(s config.Store, mutate func(cfg *config.Config) error) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	if err := mutate(&cfg); err != nil {
		return err
	}
	return s.Write(cfg)
}
