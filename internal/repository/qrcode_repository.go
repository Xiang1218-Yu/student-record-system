package repository

import (
	"time"

	"course-attendance/internal/models"

	"gorm.io/gorm"
)

// QRCodeRepository is the data-access surface for the qr_codes table.
type QRCodeRepository struct{}

// NewQRCodeRepository returns a QRCodeRepository bound to the DB.
func NewQRCodeRepository(db *gorm.DB) *QRCodeRepository {
	return &QRCodeRepository{}
}

// Create persists a new QR code row.
func (r *QRCodeRepository) Create(db *gorm.DB, q *models.QRCode) error {
	return db.Create(q).Error
}

// FindByCourse loads the most recent QR code for a course.
func (r *QRCodeRepository) FindByCourse(db *gorm.DB, courseID string) (*models.QRCode, error) {
	var q models.QRCode
	query := db.Where("course_id = ?", courseID).
		Order("created_at asc").
		Limit(1)
	if err := query.First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// FindByToken loads a QR code by its unique token.
func (r *QRCodeRepository) FindByToken(db *gorm.DB, token string) (*models.QRCode, error) {
	var q models.QRCode
	if err := db.Where("token = ?", token).First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// IsActive reports whether the QR code is still within its validity window.
func (r *QRCodeRepository) IsActive(q *models.QRCode, now time.Time) bool {
	return now.Before(q.ExpiresAt)
}
