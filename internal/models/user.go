// Package models defines the GORM entity structs that map to database
// tables. Each file in this package owns the schema for exactly one domain
// aggregate, so changes to one table never touch another.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User mirrors the users table.
type User struct {
	ID           string         `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Name         string         `gorm:"size:100" json:"name"`
	Phone        string         `gorm:"size:20" json:"phone"`
	Role         string         `gorm:"size:20;not null" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures new users receive a UUID before being persisted.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
