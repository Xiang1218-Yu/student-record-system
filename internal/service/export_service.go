package service

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExportService renders attendance data into an Excel workbook. It depends
// on StatisticsService for the numbers and knows nothing about HTTP.
type ExportService struct {
	stats *StatisticsService
}

// NewExportService constructs an ExportService.
func NewExportService(stats *StatisticsService) *ExportService {
	return &ExportService{stats: stats}
}

// ExportCourseAttendance builds an Excel workbook for a course's attendance,
// returning the bytes and a suggested filename.
func (s *ExportService) ExportCourseAttendance(courseID, courseTitle string) ([]byte, string, error) {
	stats, err := s.stats.CourseAttendance(courseID)
	if err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "出勤报表"
	if _, err := f.NewSheet(sheet); err != nil {
		return nil, "", err
	}

	headers := []string{"学员姓名", "邮箱", "签到时间", "状态", "签到方式"}
	for i, h := range headers {
		if err := setCell(f, sheet, i+1, 1, h); err != nil {
			return nil, "", err
		}
	}

	row := 2
	for _, rec := range stats.Records {
		name, email := "-", "-"
		if rec.Student != nil {
			name = rec.Student.Name
			email = rec.Student.Email
		}
		checkIn := "-"
		if rec.CheckInTime != nil {
			checkIn = rec.CheckInTime.Format("2006-01-02 15:04:05")
		}
		rowVals := []string{name, email, checkIn, rec.Status, rec.CheckInMethod}
		for i, v := range rowVals {
			if err := setCell(f, sheet, i+1, row, v); err != nil {
				return nil, "", err
			}
		}
		row++
	}

	// Summary block.
	summary := []struct {
		label string
		value interface{}
	}{
		{"报名人数", stats.EnrolledCount},
		{"已签到", stats.CheckedIn},
		{"正常", stats.OnTime},
		{"迟到", stats.Late},
		{"缺勤", stats.Absent},
		{"班级出勤率(%)", stats.ClassAttendanceRate},
	}
	for i, s := range summary {
		if err := setCell(f, sheet, 1, row+i+1, s.label); err != nil {
			return nil, "", err
		}
		if err := setCell(f, sheet, 2, row+i+1, s.value); err != nil {
			return nil, "", err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("attendance_%s.xlsx", safeName(courseTitle))
	return buf.Bytes(), filename, nil
}

// setCell writes a single value to a (col,row) cell, swallowing the
// coordinate helper's error for brevity since inputs are constant offsets.
func setCell(f *excelize.File, sheet string, col, row int, value interface{}) error {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return err
	}
	return f.SetCellValue(sheet, cell, value)
}

// safeName makes a course title safe for use in a filename.
func safeName(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '_' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "course"
	}
	return string(out)
}
