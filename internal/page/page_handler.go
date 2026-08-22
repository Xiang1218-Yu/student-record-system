// Package page renders server-side HTML pages for the front-end. It reads
// the JWT from a cookie (set on login) and passes a minimal context to the
// templates; it calls the API service layer directly, never the DB.
package page

import (
	"net/http"
	"strconv"

	"course-attendance/internal/service"
	"course-attendance/pkg/httpx"
	"course-attendance/pkg/jwtauth"

	"github.com/gin-gonic/gin"
)

// PageHandler owns the HTML routes listed in the front-end page table.
type PageHandler struct {
	auth       *service.AuthService
	courses    *service.CourseService
	enrolls    *service.EnrollmentService
	attendance *service.AttendanceService
	stats      *service.StatisticsService
	teachers   *service.TeacherService
	tokens     *jwtauth.Manager
}

// NewPageHandler constructs a PageHandler.
func NewPageHandler(auth *service.AuthService, courses *service.CourseService, enrolls *service.EnrollmentService, attendance *service.AttendanceService, stats *service.StatisticsService, teachers *service.TeacherService, tokens *jwtauth.Manager) *PageHandler {
	return &PageHandler{auth: auth, courses: courses, enrolls: enrolls, attendance: attendance, stats: stats, teachers: teachers, tokens: tokens}
}

// viewData is the context passed to every template.
type viewData struct {
	Title   string
	User    interface{}
	Data    interface{}
	BaseURL string
}

// render writes a template with the standard chrome.
func (h *PageHandler) render(c *gin.Context, name, title string, data interface{}) {
	vd := viewData{Title: title, Data: data}
	if uc, ok := httpx.ClaimsFromContext(c); ok {
		if user, err := h.auth.Profile(uc.UserID); err == nil {
			vd.User = user
		}
	}
	c.HTML(http.StatusOK, name, vd)
}

// Login GET /login
func (h *PageHandler) Login(c *gin.Context) {
	h.render(c, "login.html", "登录", nil)
}

// Register GET /register
func (h *PageHandler) Register(c *gin.Context) {
	h.render(c, "register.html", "注册", nil)
}

// Index GET / — course list home page.
func (h *PageHandler) Index(c *gin.Context) {
	date := c.Query("date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	courses, total, _ := h.courses.List(date, page, 20)
	h.render(c, "index.html", "课程列表", gin.H{"courses": courses, "total": total, "date": date})
}

// CourseDetail GET /courses/:id
func (h *PageHandler) CourseDetail(c *gin.Context) {
	course, err := h.courses.Get(c.Param("id"))
	if err != nil {
		c.String(http.StatusNotFound, "course not found")
		return
	}
	students, _ := h.enrolls.ListStudents(course.ID)
	attendance, _ := h.attendance.ListByCourse(course.ID)
	h.render(c, "course_detail.html", course.Title, gin.H{
		"course":     course,
		"students":   students,
		"attendance": attendance,
	})
}

// NewCourse GET /courses/new
func (h *PageHandler) NewCourse(c *gin.Context) {
	teachers, _ := h.courses.ListTeachers()
	h.render(c, "course_form.html", "创建课程", gin.H{"teachers": teachers})
}

// MySchedule GET /my-schedule
func (h *PageHandler) MySchedule(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	enrollments, _ := h.enrolls.ListByStudent(uc.UserID)
	h.render(c, "my_schedule.html", "我的课表", gin.H{"enrollments": enrollments})
}

// CheckInPage GET /checkin/:courseId — the page a scanned device lands on.
func (h *PageHandler) CheckInPage(c *gin.Context) {
	courseID := c.Param("courseId")
	token := c.Query("token")
	course, err := h.courses.Get(courseID)
	if err != nil {
		c.String(http.StatusNotFound, "course not found")
		return
	}
	h.render(c, "checkin.html", "签到", gin.H{"course": course, "token": token})
}

// AttendancePage GET /attendance — personal attendance records.
func (h *PageHandler) AttendancePage(c *gin.Context) {
	uc, ok := httpx.ClaimsFromContext(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	stats, _ := h.stats.MyAttendance(uc.UserID)
	records, _ := h.stats.MyHistory(uc.UserID)
	h.render(c, "attendance.html", "出勤统计", gin.H{"stats": stats, "records": records})
}

// StudentsPage GET /students — admin/teacher student management.
func (h *PageHandler) StudentsPage(c *gin.Context) {
	students, _ := h.enrolls.ListAllStudents()
	h.render(c, "students.html", "学员管理", gin.H{"students": students})
}

// TeachersPage GET /teachers — admin-only teacher management.
func (h *PageHandler) TeachersPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	teachers, total, _ := h.teachers.List(page, 100)
	h.render(c, "teachers.html", "讲师管理", gin.H{"teachers": teachers, "total": total})
}
