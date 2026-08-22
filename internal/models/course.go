package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseStatus enumerates the lifecycle states of a course.
const (
	CourseStatusScheduled = "scheduled"
	CourseStatusOngoing   = "ongoing"
	CourseStatusCompleted = "completed"
	CourseStatusCancelled = "cancelled"
)

// Course mirrors the courses table.
type Course struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	TeacherID   string         `gorm:"type:uuid;not null;index" json:"teacher_id"`
	Teacher     *User          `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Date        string         `gorm:"type:date;not null;index" json:"date"`
	StartTime   string         `gorm:"type:time;not null" json:"start_time"`
	EndTime     string         `gorm:"type:time;not null" json:"end_time"`
	Location    string         `gorm:"size:200" json:"location"`
	Status      string         `gorm:"size:20;not null;default:scheduled" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures new courses receive a UUID before being persisted.
func (c *Course) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}
