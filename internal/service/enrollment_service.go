package service

import (
	"errors"
	"strings"
	"time"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/pkg/hashutil"

	"gorm.io/gorm"
)

// EnrollmentService handles enrolling/removing students and importing them.
type EnrollmentService struct {
	db      *gorm.DB
	enrolls *repository.EnrollmentRepository
	users   *repository.UserRepository
	courses *repository.CourseRepository
}

// NewEnrollmentService constructs an EnrollmentService.
func NewEnrollmentService(db *gorm.DB, enrolls *repository.EnrollmentRepository, users *repository.UserRepository, courses *repository.CourseRepository) *EnrollmentService {
	return &EnrollmentService{db: db, enrolls: enrolls, users: users, courses: courses}
}

// Enroll adds a student to a course. It is idempotent for active enrollments.
func (s *EnrollmentService) Enroll(courseID, studentID string) error {
	if _, err := s.courses.FindByID(s.db, courseID); err != nil {
		return errors.New("course not found")
	}
	if _, err := s.users.FindByID(s.db, studentID); err != nil {
		return errors.New("student not found")
	}

	if existing, err := s.enrolls.FindActive(s.db, courseID, studentID); err == nil && existing != nil {
		return nil // already enrolled, nothing to do
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	enrollment := &models.Enrollment{
		CourseID:   courseID,
		StudentID:  studentID,
		EnrolledAt: time.Now(),
		IsActive:   true,
	}
	return s.enrolls.Create(s.db, enrollment)
}

// Remove deactivates a student's enrollment in a course.
func (s *EnrollmentService) Remove(courseID, studentID string) error {
	return s.enrolls.Deactivate(s.db, courseID, studentID)
}

// ListStudents returns the active enrolled students for a course.
func (s *EnrollmentService) ListStudents(courseID string) ([]models.User, error) {
	return s.enrolls.ListStudents(s.db, courseID)
}

// ListAllStudents returns every student account (for the management page).
func (s *EnrollmentService) ListAllStudents() ([]models.User, error) {
	return s.users.FindStudents(s.db)
}

// ListByStudent returns the active enrollments (with course) for a student.
func (s *EnrollmentService) ListByStudent(studentID string) ([]models.Enrollment, error) {
	return s.enrolls.ListByStudent(s.db, studentID)
}

// StudentImportRow is one line of a CSV import.
type StudentImportRow struct {
	Name  string
	Email string
	Phone string
}

// ImportStudents creates accounts for the given rows (skipping existing
// emails) and returns the created users.
func (s *EnrollmentService) ImportStudents(rows []StudentImportRow) ([]models.User, error) {
	created := make([]models.User, 0, len(rows))
	for _, row := range rows {
		row.Email = strings.TrimSpace(row.Email)
		row.Name = strings.TrimSpace(row.Name)
		if row.Email == "" {
			continue
		}
		existing, err := s.users.FindByEmail(s.db, row.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil {
			continue
		}
		// Imported students cannot choose a password themselves, so assign
		// the shared default and let them change it after first login.
		// Without this the account would be unauthenticated and unclaimable
		// until re-registered — see AuthService.claimAccount.
		hash, err := hashutil.HashPassword(DefaultPassword)
		if err != nil {
			return nil, err
		}
		user := &models.User{
			Email:        row.Email,
			PasswordHash: hash,
			Name:         row.Name,
			Phone:        row.Phone,
			Role:         RoleStudent,
		}
		if err := s.users.Create(s.db, user); err != nil {
			return nil, err
		}
		created = append(created, *user)
	}
	return created, nil
}
