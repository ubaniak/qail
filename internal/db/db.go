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

func Init(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath))
	return db, err
}
