package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"course-attendance/internal/database"
	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestCanceledImportDoesNotLeavePartialAccounts(t *testing.T) {
	// ImportStudents and Request.Context() define the cancellation and batch
	// write boundary.
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	users := repository.NewUserRepository(db)
	enrolls := repository.NewEnrollmentRepository(db)
	courses := repository.NewCourseRepository(db)
	svc := service.NewEnrollmentService(db, enrolls, users, courses)
	h := NewEnrollmentHandler(svc)

	ctx, cancel := context.WithCancel(context.Background())
	var canceled atomic.Bool
	db.Callback().Create().After("gorm:create").Register("contract_cancel_import", func(tx *gorm.DB) {
		if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "users" || canceled.Swap(true) {
			return
		}
		cancel()
	})

	body := map[string]any{
		"students": []map[string]string{
			{"name": "First", "email": "first-import@example.com"},
			{"name": "Second", "email": "second-import@example.com"},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students/import", bytes.NewReader(raw)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	h.Import(c)

	if recorder.Code == http.StatusCreated {
		t.Fatalf("canceled import returned success: %s", recorder.Body.String())
	}
	var count int64
	if err := db.Model(&models.User{}).Where("email IN ?", []string{
		"first-import@example.com",
		"second-import@example.com",
	}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("canceled import left %d accounts; batch must be atomic", count)
	}
}
