package service

import (
	"errors"
	"fmt"
	"time"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"

	"gorm.io/gorm"
)

// CourseService implements course CRUD and batch creation.
type CourseService struct {
	db       *gorm.DB
	courses  *repository.CourseRepository
	users    *repository.UserRepository
	enrolls  *repository.EnrollmentRepository
}

// NewCourseService constructs a CourseService.
func NewCourseService(db *gorm.DB, courses *repository.CourseRepository, users *repository.UserRepository, enrolls *repository.EnrollmentRepository) *CourseService {
	return &CourseService{db: db, courses: courses, users: users, enrolls: enrolls}
}

// CreateCourseInput captures the fields needed to create a course.
type CreateCourseInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TeacherID   string `json:"teacher_id"`
	Date        string `json:"date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
}

// Create persists a new course, validating that the teacher exists.
func (s *CourseService) Create(in CreateCourseInput) (*models.Course, error) {
	if err := validateCourseInput(in); err != nil {
		return nil, err
	}
	if _, err := s.users.FindByID(s.db, in.TeacherID); err != nil {
		return nil, errors.New("teacher not found")
	}
	course := &models.Course{
		Title:       in.Title,
		Description: in.Description,
		TeacherID:   in.TeacherID,
		Date:        in.Date,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		Location:    in.Location,
		Status:      models.CourseStatusScheduled,
	}
	if err := s.courses.Create(s.db, course); err != nil {
		return nil, err
	}
	return course, nil
}

// BatchCreateInput drives the weekly/repeated course creation flow.
type BatchCreateInput struct {
	CreateCourseInput
	RepeatWeekly bool   `json:"repeat_weekly"`
	Count        int    `json:"count"`
	EndDate      string `json:"end_date"`
}

// BatchCreate creates one or more course rows. When RepeatWeekly is true it
// repeats on the same weekday up to Count times (or up to EndDate).
func (s *CourseService) BatchCreate(in BatchCreateInput) ([]*models.Course, error) {
	if err := validateCourseInput(in.CreateCourseInput); err != nil {
		return nil, err
	}
	if _, err := s.users.FindByID(s.db, in.TeacherID); err != nil {
		return nil, errors.New("teacher not found")
	}

	dates, err := computeRepeatDates(in)
	if err != nil {
		return nil, err
	}

	courses := make([]*models.Course, 0, len(dates))
	for _, d := range dates {
		courses = append(courses, &models.Course{
			Title:       in.Title,
			Description: in.Description,
			TeacherID:   in.TeacherID,
			Date:        d,
			StartTime:   in.StartTime,
			EndTime:     in.EndTime,
			Location:    in.Location,
			Status:      models.CourseStatusScheduled,
		})
	}
	if err := s.courses.CreateInBatch(s.db, courses); err != nil {
		return nil, err
	}
	return courses, nil
}

// computeRepeatDates expands a batch request into a list of date strings.
func computeRepeatDates(in BatchCreateInput) ([]string, error) {
	base, err := time.Parse("2006-01-02", in.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}
	if !in.RepeatWeekly {
		if in.Count <= 0 {
			in.Count = 1
		}
		dates := make([]string, 0, in.Count)
		for i := 0; i < in.Count; i++ {
			dates = append(dates, base.AddDate(0, 0, i*7).Format("2006-01-02"))
		}
		return dates, nil
	}

	// Weekly repeat bounded by Count or EndDate (whichever is set).
	limit := in.Count
	if limit <= 0 {
		limit = 52 // one year ceiling
	}
	var end time.Time
	if in.EndDate != "" {
		end, err = time.Parse("2006-01-02", in.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
	}
	dates := make([]string, 0, limit)
	cur := base
	for i := 0; i < limit; i++ {
		if !end.IsZero() && cur.After(end) {
			break
		}
		dates = append(dates, cur.Format("2006-01-02"))
		cur = cur.AddDate(0, 0, 7)
	}
	if len(dates) == 0 {
		return nil, errors.New("batch produced no dates")
	}
	return dates, nil
}

// List returns a page of courses optionally filtered by date.
func (s *CourseService) List(dateFilter string, page, pageSize int) ([]models.Course, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.courses.List(s.db, dateFilter, page, pageSize)
}

// Get loads a course by id with its teacher.
func (s *CourseService) Get(id string) (*models.Course, error) {
	return s.courses.FindByID(s.db, id)
}

// Update modifies a course owned by the given actor. Only the course's
// teacher or an admin may update it.
func (s *CourseService) Update(id, actorID, actorRole string, in CreateCourseInput) (*models.Course, error) {
	course, err := s.courses.FindByID(s.db, id)
	if err != nil {
		return nil, err
	}
	if !canModifyCourse(course, actorID, actorRole) {
		return nil, errors.New("forbidden: only the teacher or an admin can update this course")
	}
	course.Title = in.Title
	course.Description = in.Description
	course.TeacherID = in.TeacherID
	course.Date = in.Date
	course.StartTime = in.StartTime
	course.EndTime = in.EndTime
	course.Location = in.Location
	if err := s.courses.Update(s.db, course); err != nil {
		return nil, err
	}
	return course, nil
}

// Delete removes a course owned by the given actor.
func (s *CourseService) Delete(id, actorID, actorRole string) error {
	course, err := s.courses.FindByID(s.db, id)
	if err != nil {
		return err
	}
	if !canModifyCourse(course, actorID, actorRole) {
		return errors.New("forbidden: only the teacher or an admin can delete this course")
	}
	return s.courses.Delete(s.db, id)
}

// ListTeachers returns all teachers for the course form dropdown.
func (s *CourseService) ListTeachers() ([]models.User, error) {
	return s.users.FindTeachers(s.db)
}

// canModifyCourse enforces the ownership rule for edit/delete.
func canModifyCourse(course *models.Course, actorID, actorRole string) bool {
	if actorRole == RoleAdmin {
		return true
	}
	return course.TeacherID == actorID
}

// validateCourseInput checks required fields.
func validateCourseInput(in CreateCourseInput) error {
	if in.Title == "" {
		return errors.New("title is required")
	}
	if in.TeacherID == "" {
		return errors.New("teacher_id is required")
	}
	if in.Date == "" || in.StartTime == "" || in.EndTime == "" {
		return errors.New("date, start_time and end_time are required")
	}
	return nil
}
