package service

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"course-attendance/internal/database"
	"course-attendance/internal/models"
	"course-attendance/internal/repository"
)

func TestConcurrentCheckInCreatesSingleAttendance(t *testing.T) {
	// FindByCourseStudentAndMethod followed by Create must enforce one
	// cross-method attendance identity under a real concurrent schedule.
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	_ = db.Exec("PRAGMA journal_mode=WAL")

	teacher := &models.User{ID: "teacher-001", Email: "teacher@example.com", PasswordHash: "hash", Role: RoleTeacher}
	student := &models.User{ID: "student-001", Email: "student@example.com", PasswordHash: "hash", Role: RoleStudent}
	if err := db.Create(teacher).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(student).Error; err != nil {
		t.Fatal(err)
	}
	course := &models.Course{
		ID:        "course-001",
		Title:     "Concurrency",
		TeacherID: teacher.ID,
		Date:      time.Now().Format("2006-01-02"),
		StartTime: "09:00",
		EndTime:   "10:00",
		Status:    models.CourseStatusOngoing,
	}
	if err := db.Create(course).Error; err != nil {
		t.Fatal(err)
	}
	enrollment := &models.Enrollment{
		ID:         "enrollment-001",
		CourseID:   course.ID,
		StudentID:  student.ID,
		EnrolledAt: time.Now(),
		IsActive:   true,
	}
	if err := db.Create(enrollment).Error; err != nil {
		t.Fatal(err)
	}
	qr := &models.QRCode{
		ID:        "qr-001",
		CourseID:  course.ID,
		Token:     "token-001",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(qr).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewAttendanceService(
		db,
		repository.NewQRCodeRepository(db),
		repository.NewCourseRepository(db),
		repository.NewEnrollmentRepository(db),
		repository.NewAttendanceRepository(db),
		"http://localhost",
	)

	// CheckInByToken and CheckInManual must share one atomic identity rule.
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, callErr := svc.CheckInByToken(course.ID, qr.Token, student.ID)
		errs <- callErr
	}()
	go func() {
		defer wg.Done()
		<-start
		_, callErr := svc.CheckInManual(course.ID, student.ID, models.MethodManual)
		errs <- callErr
	}()
	close(start)
	wg.Wait()
	close(errs)

	for callErr := range errs {
		if callErr != nil {
			t.Fatalf("concurrent check-in failed: %v", callErr)
		}
	}
	var records []models.Attendance
	if err := db.Where("course_id = ? AND student_id = ?", course.ID, student.ID).Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one attendance record after concurrent check-in, got %d", len(records))
	}
}
