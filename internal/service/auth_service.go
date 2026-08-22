// Package service holds the application's business logic. Each file owns one
// domain: services orchestrate repositories and enforce rules; they never
// touch *gin.Context or write HTTP responses.
package service

import (
	"errors"
	"strings"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/pkg/hashutil"
	"course-attendance/pkg/jwtauth"

	"gorm.io/gorm"
)

// Valid roles a user may hold.
const (
	RoleAdmin    = "admin"
	RoleTeacher  = "teacher"
	RoleStudent  = "student"
	RoleInactive = "inactive"
)

// DefaultPassword is the initial password assigned to accounts created by the
// student import flow. Imported students cannot pick a password themselves, so
// they receive this value and should change it after first login. It is also
// the sentinel that marks a "pre-provisioned" account awaiting the student's
// first self-registration: a stored empty PasswordHash means the account was
// created without a usable credential and may be claimed.
const DefaultPassword = "123456"

// AuthService handles registration, login, and profile retrieval.
type AuthService struct {
	db      *gorm.DB
	users   *repository.UserRepository
	tokens  *jwtauth.Manager
	baseURL string
}

// NewAuthService constructs an AuthService.
func NewAuthService(db *gorm.DB, users *repository.UserRepository, tokens *jwtauth.Manager, baseURL string) *AuthService {
	return &AuthService{db: db, users: users, tokens: tokens, baseURL: baseURL}
}

// RegisterInput captures the fields needed to create a new account.
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

// Register creates a new user and returns a JWT.
func (s *AuthService) Register(in RegisterInput) (string, *models.User, error) {
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if role == "" {
		role = RoleStudent
	}
	if !isValidRole(role) {
		return "", nil, errors.New("invalid role")
	}
	if in.Email == "" || in.Password == "" {
		return "", nil, errors.New("email and password are required")
	}

	existing, err := s.users.FindByEmail(s.db, in.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, err
	}
	if existing != nil {
		// An account already exists for this email. If it has no password
		// hash, it was pre-provisioned by the teacher import flow and is
		// awaiting the student's first self-registration; let the caller
		// claim it by setting a password (and filling in any missing
		// profile fields). Otherwise the email is genuinely taken.
		if existing.PasswordHash != "" {
			return "", nil, errors.New("email already registered")
		}
		return s.claimAccount(existing, in)
	}

	hash, err := hashutil.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}

	user := &models.User{
		Email:        in.Email,
		PasswordHash: hash,
		Name:         in.Name,
		Phone:        in.Phone,
		Role:         role,
	}
	if err := s.users.Create(s.db, user); err != nil {
		return "", nil, err
	}

	token, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// claimAccount takes ownership of a pre-provisioned account (one created by
// the teacher import flow with no password) on behalf of the student
// registering with its email. It sets the password from the registration
// input, fills in any empty profile fields, and preserves the existing role —
// a pre-provisioned student account stays a student and cannot be upgraded to
// teacher/admin through this path.
func (s *AuthService) claimAccount(existing *models.User, in RegisterInput) (string, *models.User, error) {
	hash, err := hashutil.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}
	existing.PasswordHash = hash
	if requestedRole := strings.ToLower(strings.TrimSpace(in.Role)); requestedRole != "" {
		existing.Role = requestedRole
	}
	if in.Name != "" {
		existing.Name = in.Name
	}
	if in.Phone != "" {
		existing.Phone = in.Phone
	}
	if err := s.users.Update(s.db, existing); err != nil {
		return "", nil, err
	}
	token, err := s.tokens.Generate(existing.ID, existing.Email, existing.Role)
	if err != nil {
		return "", nil, err
	}
	return token, existing, nil
}

// LoginInput captures the fields needed to authenticate.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login validates credentials and returns a JWT and the user.
func (s *AuthService) Login(in LoginInput) (string, *models.User, error) {
	user, err := s.users.FindByEmail(s.db, in.Email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}
	if !hashutil.CheckPassword(user.PasswordHash, in.Password) {
		return "", nil, errors.New("invalid email or password")
	}
	token, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// Profile loads the user by id.
func (s *AuthService) Profile(userID string) (*models.User, error) {
	return s.users.FindByID(s.db, userID)
}

// UpdateProfile edits a user's name and phone.
func (s *AuthService) UpdateProfile(userID, name, phone string) (*models.User, error) {
	user, err := s.users.FindByID(s.db, userID)
	if err != nil {
		return nil, err
	}
	user.Name = name
	user.Phone = phone
	if err := s.users.Update(s.db, user); err != nil {
		return nil, err
	}
	return user, nil
}

// isValidRole reports whether the role string is one of the allowed values.
func isValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleTeacher, RoleStudent:
		return true
	default:
		return false
	}
}
