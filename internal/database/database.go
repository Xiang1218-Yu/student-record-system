// Package database owns the single GORM *gorm.DB connection and the schema
// migration list. Other packages depend on the *gorm.DB it returns; they do
// not open connections themselves.
package database

import (
	"fmt"

	"course-attendance/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates the GORM connection for the configured database type and runs
// AutoMigrate so the schema matches the model structs.
func Open(dbType, dsn string) (*gorm.DB, error) {
	gdb, err := openDriver(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s db: %w", dbType, err)
	}

	if err := migrate(gdb); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return gdb, nil
}

// openDriver selects the GORM dialect for the configured database type.
// SQLite is the default; Postgres can be enabled by adding its driver and a
// case here.
func openDriver(dbType, dsn string) (*gorm.DB, error) {
	switch dbType {
	case "sqlite", "":
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	default:
		return nil, fmt.Errorf("unsupported DB_TYPE %q", dbType)
	}
}

// migrate runs AutoMigrate over every domain aggregate. Adding a new table
// means adding its model to this slice.
func migrate(gdb *gorm.DB) error {
	return gdb.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Enrollment{},
		&models.Attendance{},
		&models.QRCode{},
	)
}
