package service

import (
	"time"

	"course-attendance/internal/models"
	"course-attendance/internal/repository"

	"gorm.io/gorm"
)

// StatisticsService computes attendance rates for students and courses.
type StatisticsService struct {
	db         *gorm.DB
	attendance *repository.AttendanceRepository
	enrolls    *repository.EnrollmentRepository
	courses    *repository.CourseRepository
}

// NewStatisticsService constructs a StatisticsService.
func NewStatisticsService(db *gorm.DB, attendance *repository.AttendanceRepository, enrolls *repository.EnrollmentRepository, courses *repository.CourseRepository) *StatisticsService {
	return &StatisticsService{db: db, attendance: attendance, enrolls: enrolls, courses: courses}
}

// StudentStats summarizes one student's attendance across their enrolled
// courses.
type StudentStats struct {
	TotalCourses   int64   `json:"total_courses"`
	CheckedIn      int64   `json:"checked_in"`
	OnTime         int64   `json:"on_time"`
	Late           int64   `json:"late"`
	AttendanceRate float64 `json:"attendance_rate"`
	LateRate       float64 `json:"late_rate"`
}

// MyAttendance returns the stats for the given student.
func (s *StatisticsService) MyAttendance(studentID string) (*StudentStats, error) {
	enrollments, err := s.enrolls.ListByStudent(s.db, studentID)
	if err != nil {
		return nil, err
	}
	total := int64(len(enrollments))
	onTime, err := s.attendance.CountByStudentAndStatus(s.db, studentID, []string{models.AttendanceOnTime})
	if err != nil {
		return nil, err
	}
	late, err := s.attendance.CountByStudentAndStatus(s.db, studentID, []string{models.AttendanceLate})
	if err != nil {
		return nil, err
	}
	checkedIn := onTime + late
	stats := &StudentStats{
		TotalCourses: total,
		CheckedIn:    checkedIn,
		OnTime:       onTime,
		Late:         late,
	}
	if total > 0 {
		stats.AttendanceRate = round2(float64(checkedIn) / float64(total) * 100)
		stats.LateRate = round2(float64(late) / float64(total) * 100)
	}
	return stats, nil
}

// MyHistory returns the raw attendance records for the student.
func (s *StatisticsService) MyHistory(studentID string) ([]models.Attendance, error) {
	return s.attendance.ListByStudent(s.db, studentID)
}

// CourseStats summarizes attendance for one course.
type CourseStats struct {
	CourseID       string                   `json:"course_id"`
	EnrolledCount  int                      `json:"enrolled_count"`
	CheckedIn       int                      `json:"checked_in"`
	OnTime          int                      `json:"on_time"`
	Late            int                      `json:"late"`
	Absent          int                      `json:"absent"`
	ClassAttendanceRate float64             `json:"class_attendance_rate"`
	Records         []models.Attendance     `json:"records"`
}

// CourseAttendance returns the stats for one course.
func (s *StatisticsService) CourseAttendance(courseID string) (*CourseStats, error) {
	students, err := s.enrolls.ListStudents(s.db, courseID)
	if err != nil {
		return nil, err
	}
	records, err := s.attendance.ListByCourse(s.db, courseID)
	if err != nil {
		return nil, err
	}

	stats := &CourseStats{
		CourseID:      courseID,
		EnrolledCount: len(students),
		Records:       records,
	}
	checkedIn, onTime, late := 0, 0, 0
	for _, r := range records {
		switch r.Status {
		case models.AttendanceOnTime:
			onTime++
		case models.AttendanceLate:
			late++
		}
		checkedIn++
	}
	stats.CheckedIn = checkedIn
	stats.OnTime = onTime
	stats.Late = late
	stats.Absent = len(students) - checkedIn
	if len(students) > 0 {
		stats.ClassAttendanceRate = round2(float64(checkedIn) / float64(len(students)) * 100)
	}
	return stats, nil
}

// RecentHistory returns attendance records since the given time.
func (s *StatisticsService) RecentHistory(since time.Time) ([]models.Attendance, error) {
	return s.attendance.ListSince(s.db, since)
}

// round2 rounds to two decimal places.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
