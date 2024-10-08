package db

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	gorm.Model
	Editor    string
	Workspace string
}

type Repo struct {
	gorm.Model
	Name string
	Url  string
}

type Workspace struct {
	gorm.Model
	Name        string
	Repo        []Repo
	LastUpdated time.Time
}

func CreateSqliteDb(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath))
	return db, err
}

type dbConfig struct {
	db *gorm.DB
}

func New(db *gorm.DB) *dbConfig {
	return &dbConfig{db}
}

func (c *dbConfig) SetupDB() {
	c.db.AutoMigrate(&Repo{})
	c.db.AutoMigrate(&Config{})
	c.db.AutoMigrate(&Workspace{})
}

func (c *dbConfig) NewConfig(config *Config) {
	c.db.Create(config)
}
