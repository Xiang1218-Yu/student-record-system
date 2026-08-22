// Package repository encapsulates all GORM data access. Each file in this
// package owns the queries for exactly one aggregate, so the data-access
// surface for a table lives in one place.
package repository

import (
	"course-attendance/internal/models"

	"gorm.io/gorm"
)

// UserRepository is the data-access surface for the users table.
type UserRepository struct{}

// NewUserRepository returns a UserRepository bound to the given DB.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{}
}

// Create persists a new user.
func (r *UserRepository) Create(db *gorm.DB, user *models.User) error {
	return db.Create(user).Error
}

// CreateBatch persists imported users atomically: every user is written, or
// none are. A failure partway through the batch rolls the whole group back so
// the batch never leaves partial accounts behind — the import caller can then
// surface a server error with the original cause rather than masking a
// half-finished import as success on the next attempt. The underlying create
// error is returned unwrapped so callers can preserve the original reason.
func (r *UserRepository) CreateBatch(db *gorm.DB, users []*models.User) error {
	if len(users) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			if err := tx.Create(user).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindByEmail loads a user by email.
func (r *UserRepository) FindByEmail(db *gorm.DB, email string) (*models.User, error) {
	var u models.User
	if err := db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID loads a user by primary key.
func (r *UserRepository) FindByID(db *gorm.DB, id string) (*models.User, error) {
	var u models.User
	if err := db.First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Update writes the non-zero fields of the user struct back to the row.
func (r *UserRepository) Update(db *gorm.DB, user *models.User) error {
	return db.Save(user).Error
}

// FindTeachers returns all users whose role is teacher.
func (r *UserRepository) FindTeachers(db *gorm.DB) ([]models.User, error) {
	var teachers []models.User
	if err := db.Where("role = ?", "teacher").Order("name asc").Find(&teachers).Error; err != nil {
		return nil, err
	}
	return teachers, nil
}

// FindTeachersPaged returns one page of teachers (role = teacher) together
// with the total count, ordered by name. Mirrors CourseRepository.List's
// paging shape so callers can return the same envelope.
func (r *UserRepository) FindTeachersPaged(db *gorm.DB, page, pageSize int) ([]models.User, int64, error) {
	var teachers []models.User
	var total int64
	q := db.Model(&models.User{}).Where("role = ?", "teacher")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("name asc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&teachers).Error; err != nil {
		return nil, 0, err
	}
	return teachers, total, nil
}

// Delete soft-deletes a user by id. GORM applies the DeletedAt filter because
// User carries gorm.DeletedAt; subsequent FindByID / list queries skip the row.
func (r *UserRepository) Delete(db *gorm.DB, id string) error {
	return db.Delete(&models.User{}, "id = ?", id).Error
}

// FindStudents returns all users whose role is student.
func (r *UserRepository) FindStudents(db *gorm.DB) ([]models.User, error) {
	var students []models.User
	if err := db.Where("role = ?", "student").Order("name asc").Find(&students).Error; err != nil {
		return nil, err
	}
	return students, nil
}
