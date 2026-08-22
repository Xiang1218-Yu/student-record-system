// Package handler contains the Gin HTTP handlers. Each file owns one
// resource; handlers translate between HTTP and the service layer and hold
// no business logic of their own.
package handler

import (
	"net/http"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// AuthHandler exposes register/login/profile endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	token, user, err := h.auth.Register(in)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.Created(c, gin.H{"token": token, "user": user})
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var in service.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	token, user, err := h.auth.Login(in)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusUnauthorized, err.Error(), err))
		return
	}
	httpx.OK(c, gin.H{"token": token, "user": user})
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := httpx.RequireUserID(c)
	if !ok {
		return
	}
	user, err := h.auth.Profile(userID)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusNotFound, "user not found", err))
		return
	}
	httpx.OK(c, user)
}

// UpdateMe PUT /api/v1/auth/me
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, ok := httpx.RequireUserID(c)
	if !ok {
		return
	}
	var in struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, "invalid request body", err))
		return
	}
	user, err := h.auth.UpdateProfile(userID, in.Name, in.Phone)
	if err != nil {
		httpx.Error(c, httpx.NewAppError(http.StatusBadRequest, err.Error(), err))
		return
	}
	httpx.OK(c, user)
}
