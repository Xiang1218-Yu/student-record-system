package repository

import (
	"time"

	"course-attendance/internal/models"

	"gorm.io/gorm"
)

// AttendanceRepository is the data-access surface for the attendances table.
type AttendanceRepository struct{}

// NewAttendanceRepository returns an AttendanceRepository bound to the DB.
func NewAttendanceRepository(db *gorm.DB) *AttendanceRepository {
	return &AttendanceRepository{}
}

// Create persists a new attendance record.
func (r *AttendanceRepository) Create(db *gorm.DB, a *models.Attendance) error {
	return db.Create(a).Error
}

// FindByCourseStudent loads an attendance row for a course+student pair.
func (r *AttendanceRepository) FindByCourseStudent(db *gorm.DB, courseID, studentID string) (*models.Attendance, error) {
	var a models.Attendance
	if err := db.Where("course_id = ? AND student_id = ?", courseID, studentID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// FindByCourseStudentAndMethod loads a record using the way the student
// checked in as part of the identity.
func (r *AttendanceRepository) FindByCourseStudentAndMethod(db *gorm.DB, courseID, studentID, method string) (*models.Attendance, error) {
	var a models.Attendance
	if err := db.Where("course_id = ? AND student_id = ? AND check_in_method = ?", courseID, studentID, method).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByCourse returns all attendance rows for a course, with student loaded.
func (r *AttendanceRepository) ListByCourse(db *gorm.DB, courseID string) ([]models.Attendance, error) {
	var records []models.Attendance
	err := db.Preload("Student").
		Where("course_id = ?", courseID).
		Order("check_in_time asc").
		Find(&records).Error
	return records, err
}

// ListByStudent returns all attendance records for a student.
func (r *AttendanceRepository) ListByStudent(db *gorm.DB, studentID string) ([]models.Attendance, error) {
	var records []models.Attendance
	err := db.Where("student_id = ?", studentID).
		Order("check_in_time desc").
		Find(&records).Error
	return records, err
}

// CountByStudentAndStatus returns the number of records for a student where
// status matches one of the given values.
func (r *AttendanceRepository) CountByStudentAndStatus(db *gorm.DB, studentID string, statuses []string) (int64, error) {
	var count int64
	q := db.Model(&models.Attendance{}).Where("student_id = ?", studentID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	err := q.Count(&count).Error
	return count, err
}

// ListSince returns attendance records created at or after the given time,
// ordered newest first.
func (r *AttendanceRepository) ListSince(db *gorm.DB, since time.Time) ([]models.Attendance, error) {
	var records []models.Attendance
	err := db.Preload("Student").
		Where("created_at >= ?", since).
		Order("created_at desc").
		Find(&records).Error
	return records, err
}
