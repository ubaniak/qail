package db

import (
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
}

func Init(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath))
	return db, err
}
