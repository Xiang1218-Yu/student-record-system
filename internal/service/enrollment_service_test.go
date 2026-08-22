package service

import (
	"testing"
	"time"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB builds an in-memory SQLite database and migrates the full schema.
// Because AutoMigrate reads the model tags — including
// `uniqueIndex:uq_enrollment_course_student` on Enrollment — the resulting
// enrollments table carries the real (course_id, student_id) unique constraint.
// This reproduces the environment in which the old Enroll rejoin path violated
// the constraint, so a regression would fail the test.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Course{}, &models.Enrollment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return db
}

// seed inserts one teacher, one student, and one course and returns their ids.
func seed(t *testing.T, db *gorm.DB) (courseID, studentID, teacherID string) {
	t.Helper()
	teacher := &models.User{Email: "t@example.com", Role: RoleTeacher, Name: "Teacher"}
	if err := db.Create(teacher).Error; err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	student := &models.User{Email: "s@example.com", Role: RoleStudent, Name: "Student"}
	if err := db.Create(student).Error; err != nil {
		t.Fatalf("seed student: %v", err)
	}
	course := &models.Course{
		Title:     "Go 101",
		TeacherID: teacher.ID,
		Date:      "2026-01-01",
		StartTime: "09:00",
		EndTime:   "10:00",
		Status:    models.CourseStatusScheduled,
	}
	if err := db.Create(course).Error; err != nil {
		t.Fatalf("seed course: %v", err)
	}
	return course.ID, student.ID, teacher.ID
}

// countRows returns the number of enrollment rows for the pair (any state).
func countRows(t *testing.T, db *gorm.DB, courseID, studentID string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.Enrollment{}).
		Where("course_id = ? AND student_id = ?", courseID, studentID).
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// activeRows returns the number of *active* enrollment rows for the pair.
// More than one would mean the first-join and rejoin polluted each other.
func activeRows(t *testing.T, db *gorm.DB, courseID, studentID string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.Enrollment{}).
		Where("course_id = ? AND student_id = ? AND is_active = ?", courseID, studentID, true).
		Count(&n).Error; err != nil {
		t.Fatalf("count active: %v", err)
	}
	return n
}

func newEnrollmentService(db *gorm.DB) *EnrollmentService {
	return NewEnrollmentService(
		db,
		repository.NewEnrollmentRepository(db),
		repository.NewUserRepository(db),
		repository.NewCourseRepository(db),
	)
}

// TestEnrollRejoinRestoresSingleRecord drives the full drop/rejoin lifecycle
// and asserts the fix's invariants: rejoining reactivates the original row
// rather than inserting a new one, so exactly one row and one active record
// exist, and the unique constraint is never violated.
func TestEnrollRejoinRestoresSingleRecord(t *testing.T) {
	db := newTestDB(t)
	courseID, studentID, _ := seed(t, db)
	svc := newEnrollmentService(db)

	// 1) First join: creates the relationship.
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	if got := countRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after first enroll: want 1 row, got %d", got)
	}
	if got := activeRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after first enroll: want 1 active, got %d", got)
	}

	// Capture the original row so we can prove rejoin reuses it.
	var first models.Enrollment
	if err := db.First(&first, "course_id = ? AND student_id = ?", courseID, studentID).Error; err != nil {
		t.Fatalf("load first: %v", err)
	}
	firstEnrolledAt := first.EnrolledAt
	firstID := first.ID

	// 2) Drop the course: the row stays but is deactivated (history kept).
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got := countRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after remove: want 1 row (deactivated, not deleted), got %d", got)
	}
	if got := activeRows(t, db, courseID, studentID); got != 0 {
		t.Fatalf("after remove: want 0 active, got %d", got)
	}

	// 3) Rejoin: must reactivate the existing row, not insert a second one.
	//    This is where the old code violated the unique constraint.
	beforeRejoin := time.Now()
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("rejoin enroll: %v", err)
	}
	if got := countRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after rejoin: want 1 row, got %d (rejoin inserted a duplicate)", got)
	}
	if got := activeRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after rejoin: want 1 active, got %d", got)
	}

	// The reactivated row must be the same record (same id) — rejoin restored
	// the original relationship, it did not create a parallel one.
	var after models.Enrollment
	if err := db.First(&after, "course_id = ? AND student_id = ? AND is_active = ?", courseID, studentID, true).Error; err != nil {
		t.Fatalf("load active after rejoin: %v", err)
	}
	if after.ID != firstID {
		t.Fatalf("rejoin created a new row id %q; want reuse of %q (first-join and rejoin polluted)", after.ID, firstID)
	}
	if !after.EnrolledAt.After(firstEnrolledAt) {
		t.Fatalf("rejoin did not refresh enrolled_at; got %v, want later than %v", after.EnrolledAt, firstEnrolledAt)
	}
	if !after.EnrolledAt.After(beforeRejoin.Add(-time.Second)) {
		t.Fatalf("enrolled_at not updated to rejoin time: got %v", after.EnrolledAt)
	}

	// 4) Idempotent rejoin: enrolling again while active is a no-op.
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatalf("idempotent enroll: %v", err)
	}
	if got := countRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after idempotent enroll: want 1 row, got %d", got)
	}
}

// TestEnrollRejoinTwice verifies multiple drop/rejoin cycles keep collapsing
// onto the single original row — repeated rejoin never accumulates rows.
func TestEnrollRejoinTwice(t *testing.T) {
	db := newTestDB(t)
	courseID, studentID, _ := seed(t, db)
	svc := newEnrollmentService(db)

	for i := 0; i < 3; i++ {
		if err := svc.Enroll(courseID, studentID); err != nil {
			t.Fatalf("enroll cycle %d: %v", i, err)
		}
		if got := activeRows(t, db, courseID, studentID); got != 1 {
			t.Fatalf("cycle %d: want 1 active, got %d", i, got)
		}
		if err := svc.Remove(courseID, studentID); err != nil {
			t.Fatalf("remove cycle %d: %v", i, err)
		}
		if got := activeRows(t, db, courseID, studentID); got != 0 {
			t.Fatalf("cycle %d: want 0 active after remove, got %d", i, got)
		}
	}
	if got := countRows(t, db, courseID, studentID); got != 1 {
		t.Fatalf("after 3 drop/rejoin cycles: want 1 row total, got %d", got)
	}
}

// TestEnrollRejoinDoesNotPolluteStats guards the downstream consumers: the
// student's own course count must read exactly 1 even after rejoining, proving
// the first-join and rejoin results do not pollute ListByStudent.
func TestEnrollRejoinDoesNotPolluteStats(t *testing.T) {
	db := newTestDB(t)
	courseID, studentID, _ := seed(t, db)
	svc := newEnrollmentService(db)

	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove(courseID, studentID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Enroll(courseID, studentID); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListByStudent(studentID)
	if err != nil {
		t.Fatalf("list by student: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByStudent after rejoin: want 1 active enrollment, got %d (pollution)", len(list))
	}

	students, err := svc.ListStudents(courseID)
	if err != nil {
		t.Fatalf("list students: %v", err)
	}
	if len(students) != 1 {
		t.Fatalf("course roster after rejoin: want 1 student, got %d (pollution)", len(students))
	}
}
