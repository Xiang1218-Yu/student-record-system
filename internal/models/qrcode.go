package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QRCode mirrors the qr_codes table and ties a single check-in token to a
// course and an expiry window.
type QRCode struct {
	ID        string         `gorm:"type:uuid;primaryKey" json:"id"`
	CourseID  string         `gorm:"type:uuid;not null;index" json:"course_id"`
	Course    *Course        `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Token     string         `gorm:"size:64;uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures new QR codes receive a UUID before being persisted.
func (q *QRCode) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	return nil
}
