package page

import (
	"net/http"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"
	"course-attendance/pkg/jwtauth"

	"github.com/gin-gonic/gin"
)

// cookieName is the name of the auth cookie carrying the JWT. It is shared
// with the API Auth middleware (see httpx.CookieName) so the browser's
// same-origin cookie attach authenticates fetch calls without JS ever
// touching the token.
const cookieName = httpx.CookieName

// LoginSubmit handles the login form POST.
func (h *PageHandler) LoginSubmit(c *gin.Context) {
	in := service.LoginInput{
		Email:    c.PostForm("email"),
		Password: c.PostForm("password"),
	}
	token, _, err := h.auth.Login(in)
	if err != nil {
		c.HTML(http.StatusOK, "login.html", viewData{Title: "登录", Data: gin.H{"error": err.Error()}})
		return
	}
	setAuthCookie(c, token)
	c.Redirect(http.StatusFound, "/")
}

// RegisterSubmit handles the register form POST.
func (h *PageHandler) RegisterSubmit(c *gin.Context) {
	in := service.RegisterInput{
		Email:    c.PostForm("email"),
		Password: c.PostForm("password"),
		Name:     c.PostForm("name"),
		Phone:    c.PostForm("phone"),
		Role:     c.PostForm("role"),
	}
	token, _, err := h.auth.Register(in)
	if err != nil {
		c.HTML(http.StatusOK, "register.html", viewData{Title: "注册", Data: gin.H{"error": err.Error()}})
		return
	}
	setAuthCookie(c, token)
	c.Redirect(http.StatusFound, "/")
}

// Logout clears the auth cookie.
func (h *PageHandler) Logout(c *gin.Context) {
	c.SetCookie(cookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// setAuthCookie writes the JWT into a 24h cookie.
func setAuthCookie(c *gin.Context, token string) {
	c.SetCookie(cookieName, token, int(24*3600), "/", "", false, true)
}

// CookieAuth reads the JWT from the cookie (if present) and populates the
// request context with claims. Unlike the API Auth middleware it never
// aborts — pages render for anonymous visitors too.
func CookieAuth(mgr *jwtauth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.Next()
			return
		}
		if claims, err := mgr.Parse(token); err == nil {
			c.Set(httpx.ClaimsKey, httpx.UserClaims{
				UserID: claims.UserID,
				Email:  claims.Email,
				Role:   claims.Role,
			})
		}
		c.Next()
	}
}

// RequireAuth redirects unauthenticated users to /login. Used on pages that
// need an identity (my-schedule, attendance).
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := httpx.ClaimsFromContext(c); !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole redirects users whose role is not in the allow-list to /, and
// unauthenticated users to /login. Used on pages that carry PII and must stay
// admin-only (e.g. teacher management) so access is enforced server-side even
// if a non-admin reaches the URL directly. Must run after CookieAuth.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		uc, ok := httpx.ClaimsFromContext(c)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		if _, ok := allowed[uc.Role]; !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	}
}
