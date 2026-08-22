package service

import (
	"path/filepath"
	"testing"
	"time"

	"course-attendance/internal/database"
	"course-attendance/internal/models"
	"course-attendance/internal/repository"
)

func TestReenrollReactivatesOneRetainedEnrollment(t *testing.T) {
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	teacher := &models.User{ID: "teacher-005", Email: "teacher-005@example.com", PasswordHash: "hash", Role: RoleTeacher}
	student := &models.User{ID: "student-005", Email: "student-005@example.com", PasswordHash: "hash", Role: RoleStudent}
	if err := db.Create(teacher).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(student).Error; err != nil {
		t.Fatal(err)
	}
	course := &models.Course{
		ID: "course-005", Title: "Re-entry", TeacherID: teacher.ID,
		Date: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00",
		Status: models.CourseStatusScheduled,
	}
	if err := db.Create(course).Error; err != nil {
		t.Fatal(err)
	}

	enrollRepo := repository.NewEnrollmentRepository(db)
	svc := NewEnrollmentService(
		db,
		enrollRepo,
		repository.NewUserRepository(db),
		repository.NewCourseRepository(db),
	)
	if err := svc.Enroll(course.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	first, err := enrollRepo.FindActive(db, course.ID, student.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove(course.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Enroll(course.ID, student.ID); err != nil {
		t.Fatalf("re-enrollment failed: %v", err)
	}

	var rows []models.Enrollment
	if err := db.Unscoped().Where("course_id = ? AND student_id = ?", course.ID, student.ID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != first.ID || !rows[0].IsActive {
		t.Fatalf("re-enrollment did not retain and reactivate one row: %#v", rows)
	}
	if err := db.Where("course_id = ? AND student_id = ? AND is_active = ?", course.ID, student.ID, true).First(&models.Enrollment{}).Error; err != nil {
		t.Fatalf("active enrollment missing after re-entry: %v", err)
	}
}
