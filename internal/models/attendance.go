package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Attendance status values recorded when a student checks in (or fails to).
const (
	AttendanceOnTime = "on_time"
	AttendanceLate   = "late"
	AttendanceAbsent = "absent"
)

// Check-in method values describing how an attendance record was produced.
const (
	MethodScan   = "scan"
	MethodManual = "manual"
)

// Attendance mirrors the attendances table.
type Attendance struct {
	ID            string         `gorm:"type:uuid;primaryKey" json:"id"`
	CourseID      string         `gorm:"type:uuid;not null;index:idx_attendance_identity,unique" json:"course_id"`
	StudentID     string         `gorm:"type:uuid;not null;index:idx_attendance_identity,unique" json:"student_id"`
	Student       *User          `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	CheckInTime   *time.Time     `json:"check_in_time"`
	Status        string         `gorm:"size:20;not null" json:"status"`
	CheckInMethod string         `gorm:"size:20;index:idx_attendance_identity,unique" json:"check_in_method"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures new attendance records receive a UUID before being
// persisted.
func (a *Attendance) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
