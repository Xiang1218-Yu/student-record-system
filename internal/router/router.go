// Package router wires the Gin engine: it groups routes, attaches the
// right middleware, and binds each path to a handler. It contains no
// business logic — adding an endpoint means one line here.
package router

import (
	"course-attendance/internal/handler"
	"course-attendance/internal/middleware"
	"course-attendance/internal/page"
	"course-attendance/internal/service"
	"course-attendance/pkg/jwtauth"

	"github.com/gin-gonic/gin"
)

// Deps bundles the handler constructors the router needs.
type Deps struct {
	Auth       *handler.AuthHandler
	Course     *handler.CourseHandler
	Enrollment *handler.EnrollmentHandler
	Attendance *handler.AttendanceHandler
	Statistics *handler.StatisticsHandler
	Export     *handler.ExportHandler
	Teacher    *handler.TeacherHandler
	Page       *page.PageHandler

	AuthSvc *service.AuthService
	Tokens  *jwtauth.Manager
}

// Register attaches every API and HTML route to the given engine. The
// engine is created and configured (templates, static, middleware) by the
// caller; this function only owns the routing table.
func Register(r *gin.Engine, d Deps) {
	registerPages(r, d)
	registerAPI(r, d)
}

// registerAPI mounts the /api/v1 JSON endpoints.
func registerAPI(r *gin.Engine, d Deps) {
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", d.Auth.Register)
			auth.POST("/login", d.Auth.Login)
			auth.GET("/me", middleware.Auth(d.Tokens), d.Auth.Me)
			auth.PUT("/me", middleware.Auth(d.Tokens), d.Auth.UpdateMe)
		}

		// Public read endpoints (course list/detail) are open; mutating
		// endpoints require auth.
		courses := api.Group("/courses")
		{
			courses.GET("", d.Course.List)
			courses.GET("/:id", d.Course.Get)
			courses.GET("/:id/students", d.Enrollment.ListStudents)
			courses.GET("/:id/attendance", d.Attendance.ListAttendance)
			courses.GET("/:id/qr", d.Attendance.QR)
			courses.GET("/:id/qr/image", d.Attendance.QRImage)

			authed := courses.Group("", middleware.Auth(d.Tokens))
			{
				authed.POST("", middleware.RequireRole("admin", "teacher"), d.Course.Create)
				authed.POST("/batch", middleware.RequireRole("admin", "teacher"), d.Course.Batch)
				authed.PUT("/:id", d.Course.Update)
				authed.DELETE("/:id", d.Course.Delete)
				authed.POST("/:id/enroll", d.Enrollment.Enroll)
				authed.DELETE("/:id/enroll/:studentId", d.Enrollment.Remove)
				authed.POST("/:id/checkin", d.Attendance.CheckIn)
				authed.POST("/:id/checkin/manual", middleware.RequireRole("admin", "teacher"), d.Attendance.CheckInManual)
			}
		}

		students := api.Group("/students", middleware.Auth(d.Tokens), middleware.RequireRole("admin", "teacher"))
		{
			students.POST("/import", d.Enrollment.Import)
		}

		// Teachers: all endpoints are admin-only. The roster carries PII
		// (email/phone); the course-form dropdown reads teachers server-side
		// via CourseService.ListTeachers, so it does not need this API.
		teachers := api.Group("/teachers", middleware.Auth(d.Tokens), middleware.RequireRole("admin"))
		{
			teachers.GET("", d.Teacher.List)
			teachers.POST("", d.Teacher.Create)
			teachers.GET("/:id", d.Teacher.Get)
			teachers.PUT("/:id", d.Teacher.Update)
			teachers.PATCH("/:id/password", d.Teacher.UpdatePassword)
			teachers.DELETE("/:id", d.Teacher.Delete)
		}

		// Authenticated personal endpoints.
		me := api.Group("", middleware.Auth(d.Tokens))
		{
			me.GET("/my-attendance", d.Statistics.MyAttendance)
			me.GET("/my-attendance/history", d.Statistics.MyHistory)
		}

		api.GET("/statistics/:courseId", d.Statistics.CourseAttendance)
		api.GET("/statistics/export/:courseId", d.Export.ExportCourseAttendance)
		api.GET("/attendance/recent", d.Statistics.RecentHistory)
	}
}

// registerPages mounts the HTML routes from the front-end page table.
func registerPages(r *gin.Engine, d Deps) {
	pages := r.Group("", page.CookieAuth(d.Tokens))
	{
		pages.GET("/login", d.Page.Login)
		pages.POST("/login", d.Page.LoginSubmit)
		pages.GET("/register", d.Page.Register)
		pages.POST("/register", d.Page.RegisterSubmit)
		pages.GET("/logout", d.Page.Logout)

		pages.GET("/", d.Page.Index)
		pages.GET("/courses/new", d.Page.NewCourse)
		pages.GET("/courses/:id", d.Page.CourseDetail)
		pages.GET("/checkin/:courseId", d.Page.CheckInPage)
		pages.GET("/attendance", page.RequireAuth(), d.Page.AttendancePage)
		pages.GET("/my-schedule", page.RequireAuth(), d.Page.MySchedule)
		pages.GET("/students", d.Page.StudentsPage)
		pages.GET("/teachers", page.RequireAuth(), page.RequireRole("admin"), d.Page.TeachersPage)
	}
}
