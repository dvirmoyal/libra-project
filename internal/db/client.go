package db

import (
	"github.com/pkg/errors"
	gormzerolog "github.com/vitaliy-art/gorm-zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// NewClient creates and returns a new gorm.DB instance using the provided DSN.
func NewClient(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormzerolog.NewGormLogger()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database connection")
	}
	return db, nil
}
