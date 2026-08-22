package service

import (
	"errors"
	"fmt"
	"strings"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/pkg/hashutil"

	"gorm.io/gorm"
)

// TeacherService owns the teacher aggregate lifecycle: creating, updating,
// rotating a teacher's password, deleting, and listing. A "teacher" is a
// users row whose Role is RoleTeacher. The service depends only on the
// UserRepository and CourseRepository plus the shared *gorm.DB; it never
// touches *gin.Context and carries no HTTP concerns.
type TeacherService struct {
	db      *gorm.DB
	users   *repository.UserRepository
	courses *repository.CourseRepository
}

// NewTeacherService constructs a TeacherService.
func NewTeacherService(db *gorm.DB, users *repository.UserRepository, courses *repository.CourseRepository) *TeacherService {
	return &TeacherService{db: db, users: users, courses: courses}
}

// TeacherCreateInput captures the fields needed to create a teacher account.
// Role is intentionally absent: it is always set to RoleTeacher server-side
// so a caller cannot escalate to admin via the teacher endpoint.
type TeacherCreateInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

// TeacherUpdateInput captures the mutable profile fields. Email is optional
// but, when changed, is re-checked for uniqueness.
type TeacherUpdateInput struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// TeacherPasswordInput is the dedicated payload for a password rotation, kept
// separate from TeacherUpdateInput so a profile edit never silently overwrites
// the password.
type TeacherPasswordInput struct {
	NewPassword string `json:"new_password"`
}

// Create persists a new teacher account. Email must be unique and the password
// non-empty.
func (s *TeacherService) Create(in TeacherCreateInput) (*models.User, error) {
	in.Email = strings.TrimSpace(in.Email)
	if in.Email == "" || in.Password == "" {
		return nil, fmt.Errorf("email and password are required: %w", ErrInvalidInput)
	}

	if existing, err := s.users.FindByEmail(s.db, in.Email); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("email already registered: %w", ErrConflict)
	}

	hash, err := hashutil.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        in.Email,
		PasswordHash: hash,
		Name:         in.Name,
		Phone:        in.Phone,
		Role:         RoleTeacher,
	}
	if err := s.users.Create(s.db, user); err != nil {
		return nil, err
	}
	return user, nil
}

// List returns one page of teachers together with the total count.
func (s *TeacherService) List(page, pageSize int) ([]models.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.users.FindTeachersPaged(s.db, page, pageSize)
}

// Get loads a teacher by id. A non-teacher id (admin/student) is reported as
// not found so the teacher endpoint cannot be used to read other accounts.
func (s *TeacherService) Get(id string) (*models.User, error) {
	user, err := s.loadTeacher(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Update edits a teacher's profile fields. Role and password are never touched
// here; password rotation uses UpdatePassword.
func (s *TeacherService) Update(id string, in TeacherUpdateInput) (*models.User, error) {
	user, err := s.loadTeacher(id)
	if err != nil {
		return nil, err
	}

	in.Email = strings.TrimSpace(in.Email)
	if in.Email != "" && in.Email != user.Email {
		if existing, err := s.users.FindByEmail(s.db, in.Email); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else if existing != nil && existing.ID != user.ID {
			return nil, fmt.Errorf("email already registered: %w", ErrConflict)
		}
		user.Email = in.Email
	}
	user.Name = in.Name
	user.Phone = in.Phone

	if err := s.users.Update(s.db, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdatePassword rotates a teacher's password after validating the new value.
func (s *TeacherService) UpdatePassword(id string, in TeacherPasswordInput) error {
	user, err := s.loadTeacher(id)
	if err != nil {
		return err
	}
	if in.NewPassword == "" {
		return fmt.Errorf("new_password is required: %w", ErrInvalidInput)
	}

	hash, err := hashutil.HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	return s.users.Update(s.db, user)
}

// Delete removes a teacher account. It refuses when the teacher still owns any
// scheduled or ongoing course so courses never lose their owner; the caller
// must reassign or complete those courses first. That refusal is a state
// conflict (ErrConflict), not a missing resource — the teacher and their
// courses are all present, and the delete is rejected and left intact. Only a
// teacher with no active courses may be deleted. The delete is a soft delete
// (User carries gorm.DeletedAt) so historical records remain intact.
func (s *TeacherService) Delete(id string) error {
	if _, err := s.loadTeacher(id); err != nil {
		return err
	}
	active, err := s.courses.CountActiveByTeacher(s.db, id)
	if err != nil {
		return err
	}
	if active > 0 {
		message := fmt.Sprintf(
			"cannot delete teacher with %d active course(s); reassign or complete them first",
			active,
		)
		return fmt.Errorf("%s: %w", message, ErrConflict)
	}
	return s.users.Delete(s.db, id)
}

// loadTeacher fetches a user by id and verifies it is a teacher. A missing or
// non-teacher id becomes ErrNotFound so the endpoint does not leak the
// existence of admin/student accounts.
func (s *TeacherService) loadTeacher(id string) (*models.User, error) {
	user, err := s.users.FindByID(s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("teacher not found: %w", ErrNotFound)
		}
		return nil, err
	}
	if user.Role != RoleTeacher {
		return nil, fmt.Errorf("teacher not found: %w", ErrNotFound)
	}
	return user, nil
}
