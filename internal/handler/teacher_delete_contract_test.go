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

func TestTeacherDeleteProtectsScheduledAndOngoingCourses(t *testing.T) {
	// CountActiveByTeacher, Delete, and http.StatusConflict must preserve the
	// status guard and propagate its error through the HTTP boundary.
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	teachers := []*models.User{
		{ID: "teacher-007-scheduled", Email: "scheduled-seven@example.com", PasswordHash: "hash", Role: service.RoleTeacher},
		{ID: "teacher-007-ongoing", Email: "ongoing-seven@example.com", PasswordHash: "hash", Role: service.RoleTeacher},
	}
	for _, teacher := range teachers {
		if err := db.Create(teacher).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, course := range []*models.Course{
		{ID: "course-007-scheduled", Title: "Scheduled", TeacherID: teachers[0].ID, Date: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00", Status: models.CourseStatusScheduled},
		{ID: "course-007-ongoing", Title: "Ongoing", TeacherID: teachers[1].ID, Date: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00", Status: models.CourseStatusOngoing},
	} {
		if err := db.Create(course).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := service.NewTeacherService(db, repository.NewUserRepository(db), repository.NewCourseRepository(db))
	h := NewTeacherHandler(svc)

	for _, teacher := range teachers {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Params = gin.Params{{Key: "id", Value: teacher.ID}}
		h.Delete(c) // DELETE /api/v1/teachers/:id must propagate conflicts.
		if recorder.Code == http.StatusOK {
			t.Fatalf("teacher %s deletion returned success while owning an active course", teacher.ID)
		}
		var stored models.User
		if err := db.Unscoped().First(&stored, "id = ?", teacher.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.DeletedAt.Valid {
			t.Fatalf("teacher %s was soft-deleted despite an active course", teacher.ID)
		}
	}
}
