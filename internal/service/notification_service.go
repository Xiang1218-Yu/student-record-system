package service

import (
	"go.uber.org/zap"
)

// NotificationService is the single place where user-facing notifications are
// emitted. The full system spec describes email/微信/IM channels; here it
// logs structured events so the rest of the app can call Notify* without
// knowing the delivery mechanism. Swapping in a real sender means changing
// only this file.
type NotificationService struct {
	logger *zap.Logger
}

// NewNotificationService constructs a NotificationService.
func NewNotificationService(logger *zap.Logger) *NotificationService {
	return &NotificationService{logger: logger}
}

// CourseReminder records a "course starts soon" reminder for a user.
func (n *NotificationService) CourseReminder(userID, courseTitle, when string) {
	n.logger.Info("notification: course reminder",
		zap.String("user_id", userID),
		zap.String("course", courseTitle),
		zap.String("when", when),
	)
}

// CheckInResult records a check-in success/failure acknowledgement.
func (n *NotificationService) CheckInResult(userID, courseID, status, detail string) {
	n.logger.Info("notification: check-in result",
		zap.String("user_id", userID),
		zap.String("course_id", courseID),
		zap.String("status", status),
		zap.String("detail", detail),
	)
}

// CourseChanged records a course-change broadcast to enrolled students.
func (n *NotificationService) CourseChanged(courseID, change string, studentIDs []string) {
	n.logger.Info("notification: course changed",
		zap.String("course_id", courseID),
		zap.String("change", change),
		zap.Int("recipients", len(studentIDs)),
	)
}
