// Package middleware holds Gin middlewares: JWT authentication, role
// authorization, and request logging. Each middleware has a single job.
package middleware

import (
	"net/http"
	"strings"

	"course-attendance/pkg/httpx"
	"course-attendance/pkg/jwtauth"

	"github.com/gin-gonic/gin"
)

// Auth returns middleware that validates the JWT and stores the claims in the
// request context. Requests without a valid token are 401'd.
//
// The token is read from the "Authorization: Bearer <token>" header first, and
// falls back to the cas_token cookie. The cookie fallback exists because the
// cookie is HttpOnly (so client JS cannot read it to build the header); the
// browser's same-origin cookie attach is the only way a server-rendered page
// can call a protected API. Programmatic callers (curl, scripts) keep using
// the header.
func Auth(mgr *jwtauth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		} else if cookie, err := c.Cookie(httpx.CookieName); err == nil {
			tokenStr = cookie
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "missing or malformed Authorization header"})
			return
		}
		claims, err := mgr.Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or expired token"})
			return
		}
		c.Set(httpx.ClaimsKey, httpx.UserClaims{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		})
		c.Next()
	}
}

// RequireRole aborts with 403 unless the caller holds one of the allowed
// roles. Must run after Auth.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		uc, ok := httpx.ClaimsFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
			return
		}
		if _, ok := allowed[uc.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: insufficient role"})
			return
		}
		c.Next()
	}
}
