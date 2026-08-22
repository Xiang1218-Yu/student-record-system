package handler

import (
	"net/http"
	"time"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// StatisticsHandler exposes personal and per-course attendance stats.
type StatisticsHandler struct {
	stats *service.StatisticsService
}

// NewStatisticsHandler constructs a StatisticsHandler.
func NewStatisticsHandler(stats *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{stats: stats}
}

// MyAttendance GET /api/v1/my-attendance
func (h *StatisticsHandler) MyAttendance(c *gin.Context) {
	userID, ok := httpx.RequireUserID(c)
	if !ok {
		return
	}
	stats, err := h.stats.MyAttendance(userID)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, stats)
}

// MyHistory GET /api/v1/my-attendance/history
func (h *StatisticsHandler) MyHistory(c *gin.Context) {
	userID, ok := httpx.RequireUserID(c)
	if !ok {
		return
	}
	records, err := h.stats.MyHistory(userID)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, records)
}

// CourseAttendance GET /api/v1/statistics/:courseId
func (h *StatisticsHandler) CourseAttendance(c *gin.Context) {
	stats, err := h.stats.CourseAttendance(c.Param("courseId"))
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, stats)
}

// RecentHistory GET /api/v1/attendance/recent?since=2006-01-02
func (h *StatisticsHandler) RecentHistory(c *gin.Context) {
	since := time.Now().AddDate(0, 0, -30)
	if s := c.Query("since"); s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			since = parsed
		}
	}
	records, err := h.stats.RecentHistory(since)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusInternalServerError, err.Error(), err))
		return
	}
	httpx.OK(c, records)
}
