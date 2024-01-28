package model

import (
	"fmt"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	// enable WAL mode by default to improve performance
	err := db.Exec("PRAGMA journal_mode=WAL").Error
	if err != nil {
		return fmt.Errorf("set WAL mode: %w", err)
	}
	return db.AutoMigrate(&Repo{}, &RepoMeta{})
}
