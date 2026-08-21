package service

import (
	"errors"
	"fmt"
	"time"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/pkg/qrutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QR validity window: a code is active from 30 minutes before the course
// starts until 30 minutes after it ends.
const (
	qrLeadMinutes    = 30
	qrTrailMinutes   = 30
	lateGraceMinutes = 30
)

// AttendanceService handles QR generation, scan/manual check-in, and the
// per-course attendance list.
type AttendanceService struct {
	db         *gorm.DB
	qrs        *repository.QRCodeRepository
	courses    *repository.CourseRepository
	enrolls    *repository.EnrollmentRepository
	attendance *repository.AttendanceRepository
	baseURL    string
}

// NewAttendanceService constructs an AttendanceService.
func NewAttendanceService(db *gorm.DB, qrs *repository.QRCodeRepository, courses *repository.CourseRepository, enrolls *repository.EnrollmentRepository, attendance *repository.AttendanceRepository, baseURL string) *AttendanceService {
	return &AttendanceService{db: db, qrs: qrs, courses: courses, enrolls: enrolls, attendance: attendance, baseURL: baseURL}
}

// GetOrCreateQRCode returns an active QR token for a course. If none exists
// or the existing one is expired, it issues a fresh one within the course's
// validity window.
func (s *AttendanceService) GetOrCreateQRCode(courseID string) (*models.QRCode, error) {
	course, err := s.courses.FindByID(s.db, courseID)
	if err != nil {
		return nil, errors.New("course not found")
	}
	now := time.Now()

	if existing, err := s.qrs.FindByCourse(s.db, courseID); err == nil && s.qrs.IsActive(existing, now) {
		return existing, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	qr := &models.QRCode{
		CourseID:  courseID,
		Token:     uuid.NewString(),
		ExpiresAt: expiryFor(course, now),
	}
	if err := s.qrs.Create(s.db, qr); err != nil {
		return nil, err
	}
	return qr, nil
}

// QRImage renders the active QR code for a course as a PNG, returning the
// bytes and the check-in URL it encodes.
func (s *AttendanceService) QRImage(courseID string) ([]byte, string, error) {
	qr, err := s.GetOrCreateQRCode(courseID)
	if err != nil {
		return nil, "", err
	}
	url := qrutil.CheckInURL(s.baseURL, courseID, qr.Token)
	png, err := qrutil.EncodeToken(url, 256)
	if err != nil {
		return nil, "", err
	}
	return png, url, nil
}

// CheckInByToken records a scan-based check-in. It validates the token,
// enrollment, and computes on-time/late status from the server clock.
func (s *AttendanceService) CheckInByToken(courseID, token, studentID string) (*models.Attendance, error) {
	qr, err := s.qrs.FindByToken(s.db, token)
	if err != nil {
		return nil, errors.New("二维码无效或已过期")
	}
	if qr.CourseID != courseID {
		return nil, errors.New("二维码与课程不匹配")
	}
	if !s.qrs.IsActive(qr, time.Now()) {
		return nil, errors.New("二维码已过期")
	}
	return s.recordCheckIn(courseID, studentID, models.MethodScan)
}

// CheckInManual lets a teacher/admin add a record for a student.
func (s *AttendanceService) CheckInManual(courseID, studentID string) (*models.Attendance, error) {
	return s.recordCheckIn(courseID, studentID, models.MethodManual)
}

// recordCheckIn is the shared path for scan and manual check-in: it verifies
// enrollment, prevents duplicates, and writes the record with computed status.
func (s *AttendanceService) recordCheckIn(courseID, studentID string, method string) (*models.Attendance, error) {
	if _, err := s.enrolls.FindActive(s.db, courseID, studentID); err != nil {
		return nil, errors.New("您未报名该课程")
	}
	if existing, err := s.attendance.FindByCourseStudent(s.db, courseID, studentID, method); err == nil && existing != nil {
		return nil, fmt.Errorf("已签到，无需重复签到")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	course, err := s.courses.FindByID(s.db, courseID)
	if err != nil {
		return nil, errors.New("course not found")
	}
	now := time.Now()
	status := computeStatus(course, now)

	rec := &models.Attendance{
		CourseID:      courseID,
		StudentID:     studentID,
		CheckInTime:   &now,
		Status:        status,
		CheckInMethod: method,
	}
	if err := s.attendance.Create(s.db, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// ListByCourse returns all attendance records for a course.
func (s *AttendanceService) ListByCourse(courseID string) ([]models.Attendance, error) {
	return s.attendance.ListByCourse(s.db, courseID)
}

// computeStatus decides on_time vs late from the course's start time.
func computeStatus(course *models.Course, now time.Time) string {
	courseStart, ok := courseStart(course)
	if !ok {
		return models.AttendanceOnTime
	}
	if now.After(courseStart.Add(lateGraceMinutes * time.Minute)) {
		return models.AttendanceLate
	}
	return models.AttendanceOnTime
}

// expiryFor computes the QR expiry: course end + trail. Returns a sane
// fallback when the course's date/time cannot be parsed.
func expiryFor(course *models.Course, now time.Time) time.Time {
	courseEnd, ok := courseEnd(course)
	if !ok {
		return now.Add(24 * time.Hour)
	}
	return courseEnd.Add(qrTrailMinutes * time.Minute)
}

// courseStart parses the course's date + start_time into a wall-clock time.
// It tolerates the "2006-01-02" and the GORM-returned "2006-01-02T00:00:00Z"
// date forms, and "15:04" / "15:04:05" time forms.
func courseStart(course *models.Course) (time.Time, bool) {
	date, ok := parseDateLoose(course.Date)
	if !ok {
		return time.Time{}, false
	}
	start, ok := parseTimeLoose(course.StartTime)
	if !ok {
		return time.Time{}, false
	}
	return assemble(date, start), true
}

// courseEnd parses the course's date + end_time into a wall-clock time.
func courseEnd(course *models.Course) (time.Time, bool) {
	date, ok := parseDateLoose(course.Date)
	if !ok {
		return time.Time{}, false
	}
	end, ok := parseTimeLoose(course.EndTime)
	if !ok {
		return time.Time{}, false
	}
	return assemble(date, end), true
}

// parseDateLoose accepts both plain dates and full date-times.
func parseDateLoose(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05Z07:00", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseTimeLoose accepts "15:04" and "15:04:05".
func parseTimeLoose(s string) (time.Time, bool) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// assemble combines a date and a time-of-day into a single local time.
func assemble(date, tod time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, time.Local)
}
