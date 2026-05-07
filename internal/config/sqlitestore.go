package config

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteStore is the production Store adapter backed by GORM/SQLite. The
// constructor opens the DB and runs AutoMigrate so callers always get a
// store with the current schema; no lazy-init globals.
type SQLiteStore struct {
	db     *gorm.DB
	dbPath string
}

// NewSQLiteStore opens (or creates) a SQLite DB at dbPath and runs schema
// migrations. The caller is responsible for ensuring the parent directory
// exists.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&settingRow{},
		&workspaceRow{},
		&workspaceRepoRow{},
		&repoRow{},
		&scriptRow{},
	); err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db, dbPath: dbPath}, nil
}

// internal GORM models

type settingRow struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (settingRow) TableName() string { return "settings" }

type workspaceRow struct {
	Name     string `gorm:"primaryKey"`
	LastUsed time.Time
}

func (workspaceRow) TableName() string { return "workspaces" }

type workspaceRepoRow struct {
	WorkspaceName string `gorm:"primaryKey"`
	RepoName      string `gorm:"primaryKey"`
}

func (workspaceRepoRow) TableName() string { return "workspace_repos" }

type repoRow struct {
	Name string `gorm:"primaryKey"`
	URL  string
}

func (repoRow) TableName() string { return "repos" }

type scriptRow struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	Type   string // "repo" or "workspace"
	Target string
	Script string
	Idx    int
}

func (scriptRow) TableName() string { return "post_install_scripts" }

// Read loads the full Config snapshot from SQLite.
func (s *SQLiteStore) Read() (Config, error) {
	var cfg Config
	cfg.PostInstallScripts.Repo = make(map[string][]string)
	cfg.PostInstallScripts.Workspace = make(map[string][]string)

	// settings
	var settings []settingRow
	if err := s.db.Find(&settings).Error; err != nil {
		return cfg, err
	}
	for _, set := range settings {
		switch set.Key {
		case "root":
			cfg.Root = set.Value
		case "editor":
			cfg.Editor = set.Value
		}
	}

	// repos
	var repos []repoRow
	if err := s.db.Find(&repos).Error; err != nil {
		return cfg, err
	}
	if len(repos) > 0 {
		cfg.Repos = make(map[string]string, len(repos))
		for _, r := range repos {
			cfg.Repos[r.Name] = r.URL
		}
	}

	// workspaces
	var workspaces []workspaceRow
	if err := s.db.Find(&workspaces).Error; err != nil {
		return cfg, err
	}
	if len(workspaces) > 0 {
		cfg.Workspaces = make(Workspace, len(workspaces))
		for _, w := range workspaces {
			var wsRepos []workspaceRepoRow
			if err := s.db.Where("workspace_name = ?", w.Name).Find(&wsRepos).Error; err != nil {
				return cfg, err
			}
			repoNames := make([]string, len(wsRepos))
			for i, r := range wsRepos {
				repoNames[i] = r.RepoName
			}
			cfg.Workspaces[w.Name] = WorkspaceProfile{
				Repos:    repoNames,
				LastUsed: w.LastUsed,
			}
		}
	}

	// post install scripts
	var scripts []scriptRow
	if err := s.db.Order("idx asc").Find(&scripts).Error; err != nil {
		return cfg, err
	}
	for _, sc := range scripts {
		switch sc.Type {
		case "repo":
			cfg.PostInstallScripts.Repo[sc.Target] = append(cfg.PostInstallScripts.Repo[sc.Target], sc.Script)
		case "workspace":
			cfg.PostInstallScripts.Workspace[sc.Target] = append(cfg.PostInstallScripts.Workspace[sc.Target], sc.Script)
		}
	}

	return cfg, nil
}

// Write replaces the persisted Config inside one transaction.
func (s *SQLiteStore) Write(cfg Config) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// settings (upsert)
		if err := tx.Save(&settingRow{Key: "root", Value: cfg.Root}).Error; err != nil {
			return err
		}
		if err := tx.Save(&settingRow{Key: "editor", Value: cfg.Editor}).Error; err != nil {
			return err
		}

		// repos — replace all
		if err := tx.Exec("DELETE FROM repos").Error; err != nil {
			return err
		}
		for name, url := range cfg.Repos {
			if err := tx.Create(&repoRow{Name: name, URL: url}).Error; err != nil {
				return err
			}
		}

		// workspaces — replace all
		if err := tx.Exec("DELETE FROM workspace_repos").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM workspaces").Error; err != nil {
			return err
		}
		for name, profile := range cfg.Workspaces {
			if err := tx.Create(&workspaceRow{Name: name, LastUsed: profile.LastUsed}).Error; err != nil {
				return err
			}
			for _, repoName := range profile.Repos {
				if err := tx.Create(&workspaceRepoRow{WorkspaceName: name, RepoName: repoName}).Error; err != nil {
					return err
				}
			}
		}

		// post install scripts — replace all
		if err := tx.Exec("DELETE FROM post_install_scripts").Error; err != nil {
			return err
		}
		for target, scripts := range cfg.PostInstallScripts.Repo {
			for i, script := range scripts {
				if err := tx.Create(&scriptRow{Type: "repo", Target: target, Script: script, Idx: i}).Error; err != nil {
					return err
				}
			}
		}
		for target, scripts := range cfg.PostInstallScripts.Workspace {
			for i, script := range scripts {
				if err := tx.Create(&scriptRow{Type: "workspace", Target: target, Script: script, Idx: i}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// DBPath returns the on-disk path of the SQLite file.
func (s *SQLiteStore) DBPath() string { return s.dbPath }
