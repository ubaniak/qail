package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var (
	rootDir    string
	configPath string
	fileName   = "config.json"
)

func init() {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	rootDir = filepath.Join(h, ".qail")
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		os.Mkdir(rootDir, 0755)
	}

	configPath = filepath.Join(rootDir, fileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.Create(configPath)
	}
}

type Workspace map[string]WorkspaceProfile

type WorkspaceProfile struct {
	Repos    []string  `json:"repos"`
	LastUsed time.Time `json:"last_used"`
}

func NewWorkspaceProfile(repos []string, lastUsed time.Time) WorkspaceProfile {
	return WorkspaceProfile{
		Repos:    repos,
		LastUsed: lastUsed,
	}
}

type Config struct {
	Root       string            `json:"root"`
	Editor     string            `json:"editor"`
	Workspaces Workspace         `json:"workspaces"`
	Repos      map[string]string `json:"repos"`
}

func ValidateConfig() error {
	configPath = filepath.Join(rootDir, fileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return errors.New("config file does not exist")
	}

	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}

	if cfg.Root == "" {
		return errors.New("root folder is not set")
	}

	return nil
}

func GetConfig(fn func(cfg Config)) error {

	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}

	fn(cfg)

	return WriteToFile(cfg)
}

func ReadFromFile() (Config, error) {
	var config Config

	file, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	json.Unmarshal(file, &config)

	return config, nil
}

func WriteToFile(config Config) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	os.WriteFile(configPath, data, 0600)

	return nil
}
