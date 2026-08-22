package handler

import (
	"errors"
	"net/http"
	"strconv"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// TeacherHandler exposes the teacher CRUD endpoints. It translates between
// HTTP and TeacherService and holds no business logic; role authorization is
// enforced at the router (middleware.RequireRole), so the handler does not
// re-check roles.
type TeacherHandler struct {
	teachers *service.TeacherService
}

// NewTeacherHandler constructs a TeacherHandler.
func NewTeacherHandler(teachers *service.TeacherService) *TeacherHandler {
	return &TeacherHandler{teachers: teachers}
}

// Create POST /api/v1/teachers
func (h *TeacherHandler) Create(c *gin.Context) {
	var in service.TeacherCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	teacher, err := h.teachers.Create(in)
	if err != nil {
		httpx.Error(c, toAppError(err))
		return
	}
	httpx.Created(c, teacher)
}

// List GET /api/v1/teachers
func (h *TeacherHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	teachers, total, err := h.teachers.List(page, pageSize)
	if err != nil {
		httpx.Error(c, toAppError(err))
		return
	}
	httpx.OK(c, gin.H{"items": teachers, "total": total, "page": page, "page_size": pageSize})
}

// Get GET /api/v1/teachers/:id
func (h *TeacherHandler) Get(c *gin.Context) {
	teacher, err := h.teachers.Get(c.Param("id"))
	if err != nil {
		httpx.Error(c, toAppError(err))
		return
	}
	httpx.OK(c, teacher)
}

// Update PUT /api/v1/teachers/:id
func (h *TeacherHandler) Update(c *gin.Context) {
	var in service.TeacherUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	teacher, err := h.teachers.Update(c.Param("id"), in)
	if err != nil {
		httpx.Error(c, toAppError(err))
		return
	}
	httpx.OK(c, teacher)
}

// UpdatePassword PATCH /api/v1/teachers/:id/password
func (h *TeacherHandler) UpdatePassword(c *gin.Context) {
	var in service.TeacherPasswordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	if err := h.teachers.UpdatePassword(c.Param("id"), in); err != nil {
		httpx.Error(c, toAppError(err))
		return
	}
	httpx.OK(c, gin.H{"updated": true})
}

// Delete DELETE /api/v1/teachers/:id
func (h *TeacherHandler) Delete(c *gin.Context) {
	if err := h.teachers.Delete(c.Param("id")); err != nil {
		status := http.StatusNotFound
		appErr := httpx.NewAppError(status, err.Error(), err)
		httpx.Error(c, appErr)
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

// toAppError maps a service error to an *httpx.AppError with the right HTTP
// status by unwrapping the service sentinel errors. Anything unmatched falls
// back to 500 so internal details are not leaked to the client.
func toAppError(err error) *httpx.AppError {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return httpx.NewAppError(http.StatusNotFound, err.Error(), err)
	case errors.Is(err, service.ErrConflict):
		return httpx.NewAppError(http.StatusConflict, err.Error(), err)
	case errors.Is(err, service.ErrInvalidInput):
		return httpx.NewAppError(http.StatusBadRequest, err.Error(), err)
	default:
		return httpx.NewAppError(http.StatusInternalServerError, err.Error(), err)
	}
}
