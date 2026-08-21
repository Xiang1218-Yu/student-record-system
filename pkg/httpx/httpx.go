// Package httpx contains HTTP-layer helpers shared by every handler:
// JSON responses, error envelopes, and claims retrieval from the request
// context.
package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClaimsKey is the context key under which auth middleware stores the JWT
// claims for the current request.
const ClaimsKey = "auth.claims"

// CookieName is the name of the auth cookie carrying the JWT. It is shared by
// the cookie-setting page handlers and the API Auth middleware's cookie
// fallback, so the two never drift apart.
const CookieName = "cas_token"

// UserClaims is the identity made available to handlers after auth succeeds.
type UserClaims struct {
	UserID string
	Email  string
	Role   string
}

// AppError carries an HTTP status alongside a user-facing message so that
// services can return rich errors without knowing about gin.
type AppError struct {
	Status  int
	Message string
	Cause   error
}

// Error implements error.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Message
}

// NewAppError builds an AppError.
func NewAppError(status int, message string, cause error) *AppError {
	return &AppError{Status: status, Message: message, Cause: cause}
}

// OK writes a 200 JSON success envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// Created writes a 201 JSON success envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}

// Error writes the appropriate status and a JSON error envelope. Non-AppError
// values default to 500.
func Error(c *gin.Context, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		c.JSON(ae.Status, gin.H{"success": false, "error": ae.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
}

// ClaimsFromContext returns the authenticated user claims, or false if none
// are present.
func ClaimsFromContext(c *gin.Context) (UserClaims, bool) {
	v, ok := c.Get(ClaimsKey)
	if !ok {
		return UserClaims{}, false
	}
	uc, ok := v.(UserClaims)
	return uc, ok
}

// RequireUserID extracts the authenticated user id or aborts with 401.
func RequireUserID(c *gin.Context) (string, bool) {
	uc, ok := ClaimsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return "", false
	}
	return uc.UserID, true
}
