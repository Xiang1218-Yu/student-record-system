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

func TestTeacherDeleteBlocksOngoingCourseAndPreservesOwner(t *testing.T) {
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	teacher := &models.User{ID: "teacher-010", Email: "teacher-010@example.com", PasswordHash: "hash", Role: service.RoleTeacher}
	if err := db.Create(teacher).Error; err != nil {
		t.Fatal(err)
	}
	course := &models.Course{ID: "course-010", Title: "Ongoing course", TeacherID: teacher.ID, Date: time.Now().Format("2006-01-02"), StartTime: "09:00", EndTime: "10:00", Status: models.CourseStatusOngoing}
	if err := db.Create(course).Error; err != nil {
		t.Fatal(err)
	}

	svc := service.NewTeacherService(db, repository.NewUserRepository(db), repository.NewCourseRepository(db))
	h := NewTeacherHandler(svc)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: teacher.ID}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/teachers/"+teacher.ID, nil)
	h.Delete(c)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("ongoing-course delete returned %d, want %d", recorder.Code, http.StatusConflict)
	}
	var retained models.User
	if err := db.First(&retained, "id = ?", teacher.ID).Error; err != nil {
		t.Fatalf("teacher was removed despite owned ongoing course: %v", err)
	}
}
