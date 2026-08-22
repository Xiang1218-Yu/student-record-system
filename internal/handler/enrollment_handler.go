package handler

import (
	"net/http"
	"strings"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// EnrollmentHandler exposes enroll/list/import endpoints.
type EnrollmentHandler struct {
	enrolls *service.EnrollmentService
}

// NewEnrollmentHandler constructs an EnrollmentHandler.
func NewEnrollmentHandler(enrolls *service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{enrolls: enrolls}
}

// Enroll POST /api/v1/courses/:id/enroll
func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	var in struct {
		StudentID string `json:"student_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	if in.StudentID == "" {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "student_id is required", nil))
		return
	}
	if err := h.enrolls.Enroll(c.Param("id"), in.StudentID); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.OK(c, gin.H{"enrolled": true})
}

// Remove DELETE /api/v1/courses/:id/enroll/:studentId
func (h *EnrollmentHandler) Remove(c *gin.Context) {
	if err := h.enrolls.Remove(c.Param("id"), c.Param("studentId")); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.OK(c, gin.H{"removed": true})
}

// ListStudents GET /api/v1/courses/:id/students
func (h *EnrollmentHandler) ListStudents(c *gin.Context) {
	students, err := h.enrolls.ListStudents(c.Param("id"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, students)
}

// Import POST /api/v1/students/import — accepts CSV text under "csv" form
// field or JSON array of {name,email,phone}.
func (h *EnrollmentHandler) Import(c *gin.Context) {
	rows, err := parseImport(c)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	created, err := h.enrolls.ImportStudents(rows)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.Created(c, gin.H{"imported": len(created), "students": created})
}

// parseImport extracts student rows from either a CSV text field or a JSON
// array body.
func parseImport(c *gin.Context) ([]service.StudentImportRow, error) {
	if csv := c.PostForm("csv"); csv != "" {
		return parseCSVRows(csv)
	}
	var in struct {
		Students []service.StudentImportRow `json:"students"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		return nil, err
	}
	return in.Students, nil
}

// parseCSVRows parses simple "name,email,phone" CSV lines.
func parseCSVRows(csv string) ([]service.StudentImportRow, error) {
	rows := []service.StudentImportRow{}
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "name,") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			return nil, httpx.NewAppError(http.StatusBadRequest, "invalid CSV line", nil)
		}
		row := service.StudentImportRow{
			Name:  strings.TrimSpace(fields[0]),
			Email: strings.TrimSpace(fields[1]),
		}
		if len(fields) >= 3 {
			row.Phone = strings.TrimSpace(fields[2])
		}
		rows = append(rows, row)
	}
	return rows, nil
}
