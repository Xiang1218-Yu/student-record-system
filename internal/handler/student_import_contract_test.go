package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"course-attendance/internal/database"
	"course-attendance/internal/repository"
	"course-attendance/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestStudentImportRollsBackAndPropagatesStorageFailure(t *testing.T) {
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	studentRepo := repository.NewUserRepository(db)
	enrollSvc := service.NewEnrollmentService(
		db,
		repository.NewEnrollmentRepository(db),
		studentRepo,
		repository.NewCourseRepository(db),
	)
	h := NewEnrollmentHandler(enrollSvc)

	var creates int32
	if err := db.Callback().Create().Before("gorm:create").Register("contract_import_failure", func(tx *gorm.DB) {
		if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "users" {
			return
		}
		if atomic.AddInt32(&creates, 1) == 2 {
			tx.AddError(errors.New("forced storage failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove("contract_import_failure") })

	payload := map[string]any{"students": []map[string]string{
		{"name": "First", "email": "first-006@example.com"},
		{"name": "Second", "email": "second-006@example.com"},
	}}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students/import", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	h.Import(c)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("storage failure was mapped to the wrong status: %d", recorder.Code)
	}
	var count int64
	if err := db.Model(&struct{ ID string }{}).Table("users").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial student import remained after storage failure: %d", count)
	}
}
