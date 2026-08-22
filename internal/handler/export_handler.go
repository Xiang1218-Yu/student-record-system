package handler

import (
	"net/http"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// ExportHandler renders attendance data as downloadable files.
type ExportHandler struct {
	export  *service.ExportService
	courses *service.CourseService
}

// NewExportHandler constructs an ExportHandler.
func NewExportHandler(export *service.ExportService, courses *service.CourseService) *ExportHandler {
	return &ExportHandler{export: export, courses: courses}
}

// ExportCourseAttendance GET /api/v1/statistics/export/:courseId
func (h *ExportHandler) ExportCourseAttendance(c *gin.Context) {
	courseID := c.Param("courseId")
	course, err := h.courses.Get(courseID)
	title := courseID
	if err == nil && course != nil {
		title = course.Title
	}
	data, filename, err := h.export.ExportCourseAttendance(courseID, title)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
