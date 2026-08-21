package handler

import (
	"net/http"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// AttendanceHandler exposes QR, check-in, and attendance-list endpoints.
type AttendanceHandler struct {
	attendance *service.AttendanceService
}

// NewAttendanceHandler constructs an AttendanceHandler.
func NewAttendanceHandler(attendance *service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendance: attendance}
}

// QR GET /api/v1/courses/:id/qr — returns the active QR token metadata.
func (h *AttendanceHandler) QR(c *gin.Context) {
	qr, err := h.attendance.GetOrCreateQRCode(c.Param("id"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusNotFound, err.Error(), err))
		return
	}
	httpx.OK(c, qr)
}

// QRImage GET /api/v1/courses/:id/qr/image — returns the QR PNG bytes.
func (h *AttendanceHandler) QRImage(c *gin.Context) {
	png, _, err := h.attendance.QRImage(c.Param("id"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusNotFound, err.Error(), err))
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

// CheckIn POST /api/v1/courses/:id/checkin — scan-based check-in.
func (h *AttendanceHandler) CheckIn(c *gin.Context) {
	userID, ok := httpx.RequireUserID(c)
	if !ok {
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	rec, err := h.attendance.CheckInByToken(c.Param("id"), in.Token, userID)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.OK(c, rec)
}

// CheckInManual POST /api/v1/courses/:id/checkin/manual — teacher/admin补签.
func (h *AttendanceHandler) CheckInManual(c *gin.Context) {
	var in struct {
		StudentID string `json:"student_id"`
		Method    string `json:"method"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	if in.StudentID == "" {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "student_id is required", nil))
		return
	}
	rec, err := h.attendance.CheckInManual(c.Param("id"), in.StudentID, in.Method)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.OK(c, rec)
}

// ListAttendance GET /api/v1/courses/:id/attendance
func (h *AttendanceHandler) ListAttendance(c *gin.Context) {
	records, err := h.attendance.ListByCourse(c.Param("id"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, records)
}
