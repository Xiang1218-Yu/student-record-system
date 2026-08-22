package service

import (
	"path/filepath"
	"testing"

	"course-attendance/internal/database"
	"course-attendance/internal/models"
	"course-attendance/internal/repository"
	"course-attendance/pkg/jwtauth"
)

func TestClaimedStudentCannotEscalateRole(t *testing.T) {
	// claimAccount must preserve the stored role before Generate creates a
	// token that crosses the authorization boundary.
	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	preProvisioned := &models.User{
		ID:           "student-004",
		Email:        "claim@example.com",
		PasswordHash: "",
		Name:         "Imported student",
		Role:         RoleStudent,
	}
	if err := db.Create(preProvisioned).Error; err != nil {
		t.Fatal(err)
	}
	tokens := jwtauth.New("contract-secret")
	auth := NewAuthService(db, repository.NewUserRepository(db), tokens, "http://localhost")

	token, claimed, err := auth.Register(RegisterInput{
		Email:    preProvisioned.Email,
		Password: "password",
		Name:     "Claimed",
		Role:     RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Role != RoleStudent {
		t.Fatalf("claim changed student role to %q", claimed.Role)
	}
	claims, err := tokens.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != RoleStudent || claims.UserID != preProvisioned.ID {
		t.Fatalf("token crossed role or identity boundary: role=%q user_id=%q", claims.Role, claims.UserID)
	}
}
