package actions

import (
	"fmt"

	"github.com/ubaniak/qail/internal/config"
)

// ConvertConfig backs up the SQLite DB, then imports the legacy config.json
// at jsonPath into the same store. The .bak preserves the prior state if a
// caller needs to roll back via RestoreConfig.
//
// Takes *config.SQLiteStore (not config.Store) because conversion is
// SQLite-specific — the in-memory store is fresh each run.
func ConvertConfig(s *config.SQLiteStore, jsonPath string) error {
	if err := s.BackUp(); err != nil {
		return fmt.Errorf("back up: %w", err)
	}
	if err := s.ConvertJSON(jsonPath); err != nil {
		return fmt.Errorf("convert json: %w", err)
	}
	return nil
}

// RestoreConfig copies <db>.bak back over the live DB. Pairs with
// ConvertConfig.
func RestoreConfig(s *config.SQLiteStore) error {
	return s.Restore()
}
