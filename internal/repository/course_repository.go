package repository

import (
	"course-attendance/internal/models"

	"gorm.io/gorm"
)

// CourseRepository is the data-access surface for the courses table.
type CourseRepository struct{}

// NewCourseRepository returns a CourseRepository bound to the given DB.
func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{}
}

// Create persists a single course.
func (r *CourseRepository) Create(db *gorm.DB, course *models.Course) error {
	return db.Create(course).Error
}

// CreateInBatch persists multiple courses in one transaction.
func (r *CourseRepository) CreateInBatch(db *gorm.DB, courses []*models.Course) error {
	return db.CreateInBatches(courses, len(courses)).Error
}

// FindByID loads a course and its teacher.
func (r *CourseRepository) FindByID(db *gorm.DB, id string) (*models.Course, error) {
	var c models.Course
	if err := db.Preload("Teacher").First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns courses ordered by date, optionally filtered by date string
// and paged. dateFilter="" returns all rows.
func (r *CourseRepository) List(db *gorm.DB, dateFilter string, page, pageSize int) ([]models.Course, int64, error) {
	var courses []models.Course
	var total int64
	q := db.Model(&models.Course{}).Preload("Teacher")
	if dateFilter != "" {
		q = q.Where("date = ?", dateFilter)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("date asc, start_time asc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// FindByTeacher returns courses owned by a teacher.
func (r *CourseRepository) FindByTeacher(db *gorm.DB, teacherID string) ([]models.Course, error) {
	var courses []models.Course
	if err := db.Preload("Teacher").Where("teacher_id = ?", teacherID).
		Order("date asc").Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// CountActiveByTeacher returns the number of scheduled or ongoing courses
// owned by a teacher. Used by the teacher-delete referential-integrity check
// so a teacher cannot be removed while still responsible for live courses.
func (r *CourseRepository) CountActiveByTeacher(db *gorm.DB, teacherID string) (int64, error) {
	var n int64
	err := db.Model(&models.Course{}).
		Where("teacher_id = ? AND status IN ?", teacherID,
			[]string{models.CourseStatusScheduled, models.CourseStatusOngoing}).
		Count(&n).Error
	return n, err
}

// Update writes the course back.
func (r *CourseRepository) Update(db *gorm.DB, course *models.Course) error {
	return db.Save(course).Error
}

// Delete soft-deletes a course.
func (r *CourseRepository) Delete(db *gorm.DB, id string) error {
	return db.Delete(&models.Course{}, "id = ?", id).Error
}
