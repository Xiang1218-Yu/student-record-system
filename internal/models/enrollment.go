package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Enrollment mirrors the enrollments table linking students to courses.
type Enrollment struct {
	ID         string         `gorm:"type:uuid;primaryKey" json:"id"`
	CourseID   string         `gorm:"type:uuid;not null;index" json:"course_id"`
	Course     *Course        `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	StudentID  string         `gorm:"type:uuid;not null;index" json:"student_id"`
	Student    *User          `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	EnrolledAt time.Time      `json:"enrolled_at"`
	IsActive   bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures new enrollments receive a UUID before being persisted.
func (e *Enrollment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
