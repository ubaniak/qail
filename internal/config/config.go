package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var (
	rootDir     string
	configPath  string
	archivePath string
	fileName    = "config.json"
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

	archivePath = filepath.Join(rootDir, "archive")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		os.Create(archivePath)
	}

	configPath = filepath.Join(rootDir, fileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.Create(configPath)
	}
}

type Config struct {
	ArchivePath string
	Root        string              `json:"root"`
	Editor      string              `json:"editor"`
	Workspaces  map[string][]string `json:"workspaces"`
	Repos       map[string]string   `json:"repos"`
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

func ReadFromFile() (Config, error) {
	var config Config

	file, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	json.Unmarshal(file, &config)
	config.ArchivePath = archivePath

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
