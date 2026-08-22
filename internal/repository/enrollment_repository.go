package repository

import (
	"course-attendance/internal/models"

	"gorm.io/gorm"
)

// EnrollmentRepository is the data-access surface for the enrollments table.
type EnrollmentRepository struct{}

// NewEnrollmentRepository returns an EnrollmentRepository bound to the DB.
func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{}
}

// Create persists a new enrollment row.
func (r *EnrollmentRepository) Create(db *gorm.DB, e *models.Enrollment) error {
	return db.Create(e).Error
}

// FindActive loads an active enrollment for a course+student pair.
func (r *EnrollmentRepository) FindActive(db *gorm.DB, courseID, studentID string) (*models.Enrollment, error) {
	var e models.Enrollment
	if err := db.Where("course_id = ? AND student_id = ? AND is_active = ?", courseID, studentID, true).
		First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// ListStudents returns the active enrolled students for a course.
func (r *EnrollmentRepository) ListStudents(db *gorm.DB, courseID string) ([]models.User, error) {
	var students []models.User
	err := db.Table("users").
		Joins("JOIN enrollments ON enrollments.student_id = users.id").
		Where("enrollments.course_id = ? AND enrollments.is_active = ?", courseID, true).
		Order("users.name asc").
		Find(&students).Error
	return students, err
}

// ListByStudent returns the active enrollments (with course) for a student.
func (r *EnrollmentRepository) ListByStudent(db *gorm.DB, studentID string) ([]models.Enrollment, error) {
	var enrollments []models.Enrollment
	err := db.Preload("Course.Teacher").
		Where("student_id = ? AND is_active = ?", studentID, true).
		Order("enrollments.enrolled_at desc").
		Find(&enrollments).Error
	return enrollments, err
}

// Deactivate marks an enrollment inactive (removes the student from a course
// without deleting the history).
func (r *EnrollmentRepository) Deactivate(db *gorm.DB, courseID, studentID string) error {
	return db.Unscoped().Model(&models.Enrollment{}).
		Where("course_id = ? AND student_id = ? AND is_active = ?", courseID, studentID, true).
		Update("is_active", false).Error
}
