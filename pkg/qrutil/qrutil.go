// Package qrutil renders signed check-in tokens into PNG QR codes and into
// the URL a scanned device should open. It performs no database access.
package qrutil

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// EncodeToken returns a PNG byte slice encoding the given content as a QR
// image at the requested size.
func EncodeToken(content string, size int) ([]byte, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("encode qr: %w", err)
	}
	return png, nil
}

// CheckInURL builds the URL a scanned device navigates to for a course's
// check-in confirmation page.
func CheckInURL(baseURL, courseID, token string) string {
	return fmt.Sprintf("%s/checkin/%s?token=%s", baseURL, courseID, token)
}
