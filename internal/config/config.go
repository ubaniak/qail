package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	rootDir string
	dbPath  string
	db      *gorm.DB
	dbOnce  sync.Once
	dbErr   error
)

// internal GORM models

type settingRow struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (settingRow) TableName() string { return "settings" }

type workspaceRow struct {
	Name     string    `gorm:"primaryKey"`
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

func initDB() error {
	dbOnce.Do(func() {
		h, err := os.UserHomeDir()
		if err != nil {
			dbErr = err
			return
		}

		rootDir = filepath.Join(h, ".qail")
		if _, err := os.Stat(rootDir); os.IsNotExist(err) {
			if err := os.Mkdir(rootDir, 0755); err != nil {
				dbErr = err
				return
			}
		}

		dbPath = filepath.Join(rootDir, "qail.db")
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			dbErr = err
			return
		}

		if err = db.AutoMigrate(
			&settingRow{},
			&workspaceRow{},
			&workspaceRepoRow{},
			&repoRow{},
			&scriptRow{},
		); err != nil {
			dbErr = err
		}
	})
	return dbErr
}

// Public types

type Workspace map[string]WorkspaceProfile

type WorkspaceProfile struct {
	Repos    []string
	LastUsed time.Time
}

func NewWorkspaceProfile(repos []string, lastUsed time.Time) WorkspaceProfile {
	return WorkspaceProfile{
		Repos:    repos,
		LastUsed: lastUsed,
	}
}

type PostInstallScripts struct {
	Repo      map[string][]string
	Workspace map[string][]string
}

type Config struct {
	Root               string
	Editor             string
	Workspaces         Workspace
	Repos              map[string]string
	PostInstallScripts PostInstallScripts
}

func ValidateConfig() error {
	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}
	if cfg.Root == "" {
		return errors.New("root folder is not set")
	}
	return nil
}

func WithConfig(fn func(cfg *Config) error) error {
	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}
	err = fn(&cfg)
	if err != nil {
		return err
	}
	return WriteToFile(cfg)
}

func ReadFromFile() (Config, error) {
	if err := initDB(); err != nil {
		return Config{}, err
	}
	var cfg Config
	cfg.PostInstallScripts.Repo = make(map[string][]string)
	cfg.PostInstallScripts.Workspace = make(map[string][]string)

	// settings
	var settings []settingRow
	if err := db.Find(&settings).Error; err != nil {
		return cfg, err
	}
	for _, s := range settings {
		switch s.Key {
		case "root":
			cfg.Root = s.Value
		case "editor":
			cfg.Editor = s.Value
		}
	}

	// repos
	var repos []repoRow
	if err := db.Find(&repos).Error; err != nil {
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
	if err := db.Find(&workspaces).Error; err != nil {
		return cfg, err
	}
	if len(workspaces) > 0 {
		cfg.Workspaces = make(Workspace, len(workspaces))
		for _, w := range workspaces {
			var wsRepos []workspaceRepoRow
			if err := db.Where("workspace_name = ?", w.Name).Find(&wsRepos).Error; err != nil {
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
	if err := db.Order("idx asc").Find(&scripts).Error; err != nil {
		return cfg, err
	}
	for _, s := range scripts {
		switch s.Type {
		case "repo":
			cfg.PostInstallScripts.Repo[s.Target] = append(cfg.PostInstallScripts.Repo[s.Target], s.Script)
		case "workspace":
			cfg.PostInstallScripts.Workspace[s.Target] = append(cfg.PostInstallScripts.Workspace[s.Target], s.Script)
		}
	}

	return cfg, nil
}

func WriteToFile(cfg Config) error {
	if err := initDB(); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
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
