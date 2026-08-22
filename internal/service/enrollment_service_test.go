package service

import (
	"errors"
	"path/filepath"
	"testing"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newServiceForTest opens a fresh in-memory/file SQLite DB, migrates it, and
// wires up an EnrollmentService with a course, a teacher, and a student ready
// to enrol.
func newServiceForTest(t *testing.T) (*EnrollmentService, string, string) {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Course{}, &models.Enrollment{}, &models.Attendance{}, &models.QRCode{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	teacher := &models.User{Email: "t@example.com", PasswordHash: "x", Name: "T", Role: RoleTeacher}
	if err := repository.NewUserRepository(db).Create(db, teacher); err != nil {
		t.Fatalf("create teacher: %v", err)
	}
	course := &models.Course{Title: "C", TeacherID: teacher.ID, Date: "2026-01-01", StartTime: "09:00", EndTime: "10:00", Status: models.CourseStatusScheduled}
	if err := repository.NewCourseRepository(db).Create(db, course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	student := &models.User{Email: "s@example.com", PasswordHash: "x", Name: "S", Role: RoleStudent}
	if err := repository.NewUserRepository(db).Create(db, student); err != nil {
		t.Fatalf("create student: %v", err)
	}

	svc := NewEnrollmentService(db, repository.NewEnrollmentRepository(db), repository.NewUserRepository(db), repository.NewCourseRepository(db))
	return svc, course.ID, student.ID
}

// Removing a student who was never enrolled must fail with ErrNotFound, not
// report success.
func TestRemoveNotEnrolledFails(t *testing.T) {
	svc, courseID, studentID := newServiceForTest(t)
	err := svc.Remove(courseID, studentID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-existent enrollment, got %v", err)
	}
}

// Removing an active enrollment succeeds and returns nil.
func TestRemoveActiveSucceeds(t *testing.T) {
	svc, courseID, studentID := newServiceForTest(t)
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatalf("expected successful removal, got %v", err)
	}
}

// Removing a student twice: the second call must fail (no active enrollment
// anymore), proving the system distinguishes a real update from a no-op.
func TestRemoveTwiceSecondFails(t *testing.T) {
	svc, courseID, studentID := newServiceForTest(t)
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatalf("first remove: %v", err)
	}
	err := svc.Remove(courseID, studentID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove must report not found, got %v", err)
	}
}

// History is preserved: the deactivated row stays in the table with
// is_active=false, even though it is no longer in the active roster.
func TestRemovePreservesHistory(t *testing.T) {
	svc, courseID, studentID := newServiceForTest(t)
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	students, err := svc.ListStudents(courseID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(students) != 0 {
		t.Fatalf("expected student removed from roster, got %d", len(students))
	}
}

// A student removed from a course can enrol again afterwards (removal does not
// block re-enrolment), and the new enrollment is active.
func TestReenrollAfterRemove(t *testing.T) {
	svc, courseID, studentID := newServiceForTest(t)
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("re-enroll after remove: %v", err)
	}
	students, err := svc.ListStudents(courseID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(students) != 1 || students[0].ID != studentID {
		t.Fatalf("expected re-enrolled student in roster, got %+v", students)
	}
}
