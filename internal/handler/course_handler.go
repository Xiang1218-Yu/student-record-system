package handler

import (
	"net/http"
	"strconv"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// CourseHandler exposes course CRUD and batch endpoints.
type CourseHandler struct {
	courses *service.CourseService
}

// NewCourseHandler constructs a CourseHandler.
func NewCourseHandler(courses *service.CourseService) *CourseHandler {
	return &CourseHandler{courses: courses}
}

// Create POST /api/v1/courses
func (h *CourseHandler) Create(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		httpx.Error(c, httpx.NewAppError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	if uc.Role != service.RoleAdmin && uc.Role != service.RoleTeacher {
		httpx.Error(c, httpx.NewAppError(http.StatusForbidden, "only admins and teachers can create courses", nil))
		return
	}
	var in service.CreateCourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	if in.TeacherID == "" {
		in.TeacherID = uc.UserID
	}
	course, err := h.courses.Create(in)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.Created(c, course)
}

// Batch POST /api/v1/courses/batch
func (h *CourseHandler) Batch(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		httpx.Error(c, httpx.NewAppError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	if uc.Role != service.RoleAdmin && uc.Role != service.RoleTeacher {
		httpx.Error(c, httpx.NewAppError(http.StatusForbidden, "only admins and teachers can create courses", nil))
		return
	}
	var in service.BatchCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	if in.TeacherID == "" {
		in.TeacherID = uc.UserID
	}
	courses, err := h.courses.BatchCreate(in)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.Created(c, courses)
}

// List GET /api/v1/courses
func (h *CourseHandler) List(c *gin.Context) {
	date := c.Query("date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	courses, total, err := h.courses.List(date, page, pageSize)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, gin.H{"items": courses, "total": total, "page": page, "page_size": pageSize})
}

// Get GET /api/v1/courses/:id
func (h *CourseHandler) Get(c *gin.Context) {
	course, err := h.courses.Get(c.Param("id"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusNotFound, "course not found", err))
		return
	}
	httpx.OK(c, course)
}

// Update PUT /api/v1/courses/:id
func (h *CourseHandler) Update(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		httpx.Error(c, httpx.NewAppError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	var in service.CreateCourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	course, err := h.courses.Update(c.Param("id"), uc.UserID, uc.Role, in)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusForbidden, err.Error(), err))
		return
	}
	httpx.OK(c, course)
}

// Delete DELETE /api/v1/courses/:id
func (h *CourseHandler) Delete(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		httpx.Error(c, httpx.NewAppError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	if err := h.courses.Delete(c.Param("id"), uc.UserID, uc.Role); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusForbidden, err.Error(), err))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}
