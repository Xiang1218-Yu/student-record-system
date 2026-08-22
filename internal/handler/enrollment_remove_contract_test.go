package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"course-attendance/internal/database"
	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/internal/service"

	"github.com/gin-gonic/gin"
)

func TestEnrollmentRemoveReportsMissAndPreservesLifecycle(t *testing.T) {
	// Deactivate must expose RowsAffected while IsActive keeps history and
	// permits a later re-enrollment.
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	teacher := &models.User{ID: "teacher-008", Email: "teacher-eight@example.com", PasswordHash: "hash", Role: service.RoleTeacher}
	student := &models.User{ID: "student-008", Email: "student-eight@example.com", PasswordHash: "hash", Role: service.RoleStudent}
	for _, user := range []*models.User{teacher, student} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	course := &models.Course{
		ID: "course-008", Title: "Enrollment lifecycle", TeacherID: teacher.ID,
		Date: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00",
		Status: models.CourseStatusScheduled,
	}
	if err := db.Create(course).Error; err != nil {
		t.Fatal(err)
	}
	enrollRepo := repository.NewEnrollmentRepository(db)
	svc := service.NewEnrollmentService(
		db,
		enrollRepo,
		repository.NewUserRepository(db),
		repository.NewCourseRepository(db),
	)
	h := NewEnrollmentHandler(svc)

	// DELETE /api/v1/courses/:id/enroll/:studentId must distinguish no update from success.
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: course.ID}, {Key: "studentId", Value: student.ID}}
	h.Remove(c)
	if recorder.Code == http.StatusOK {
		t.Fatal("removing a missing active enrollment returned success")
	}

	if err := svc.Enroll(course.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: course.ID}, {Key: "studentId", Value: student.ID}}
	h.Remove(c)
	if recorder.Code != http.StatusOK {
		t.Fatalf("normal enrollment removal failed with status %d", recorder.Code)
	}
	var inactive models.Enrollment
	if err := db.Unscoped().Where("course_id = ? AND student_id = ?", course.ID, student.ID).First(&inactive).Error; err != nil {
		t.Fatal(err)
	}
	if inactive.IsActive {
		t.Fatal("normal removal did not preserve an inactive history row")
	}

	if err := svc.Enroll(course.ID, student.ID); err != nil {
		t.Fatalf("re-enrollment failed: %v", err)
	}
	var all []models.Enrollment
	if err := db.Unscoped().Where("course_id = ? AND student_id = ?", course.ID, student.ID).Find(&all).Error; err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, row := range all {
		if row.IsActive {
			active++
		}
	}
	if len(all) != 2 || active != 1 {
		t.Fatalf("enrollment lifecycle was not preserved: rows=%d active=%d", len(all), active)
	}
}
