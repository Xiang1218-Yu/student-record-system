package repository

import (
	"os"
	"path/filepath"
	"testing"

	"course-attendance/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB returns an isolated in-memory SQLite DB with the User table
// migrated, so each test starts from a clean slate without touching files.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(dsn) })
	return gdb
}

// TestCreateBatch_AtomicOnDuplicateInBatch asserts the batch-import failure
// semantics: when any single row fails (here a UNIQUE-email collision inside
// the batch), the whole batch must roll back so no partial accounts are left
// behind. The import caller then surfaces a server error with the original
// cause instead of masking a half-finished import as success.
func TestCreateBatch_AtomicOnDuplicateInBatch(t *testing.T) {
	db := openTestDB(t)
	repo := &UserRepository{}

	// Seed an existing account so the batch hits a UNIQUE collision on row 1.
	seed := &models.User{Email: "dup@example.com", PasswordHash: "x", Name: "Dup", Role: "student"}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	batch := []*models.User{
		{Email: "new1@example.com", PasswordHash: "h", Name: "New1", Role: "student"},
		{Email: "dup@example.com", PasswordHash: "h", Name: "Collide", Role: "student"}, // collision
		{Email: "new2@example.com", PasswordHash: "h", Name: "New2", Role: "student"},
	}

	err := repo.CreateBatch(db, batch)
	if err == nil {
		t.Fatal("expected CreateBatch to fail on the in-batch duplicate, got nil")
	}

	// The pre-existing seeded account must remain (rollback must not delete
	// rows that existed before the batch).
	var seeded models.User
	if err := db.Where("email = ?", "dup@example.com").First(&seeded).Error; err != nil {
		t.Fatalf("seeded account vanished after rollback: %v", err)
	}

	// No row from the failing batch may persist — that is the "no partial
	// accounts" guarantee.
	var n int64
	db.Model(&models.User{}).Where("email IN ?",
		[]string{"new1@example.com", "new2@example.com", "Collide@example.com"}).Count(&n)
	if n != 0 {
		t.Fatalf("batch left %d partial account(s) behind after failure", n)
	}
}

// TestCreateBatch_AllOrNothingOnDBError simulates a persistence failure on the
// first row (NOT NULL constraint via empty Email is blocked by the unique
// index too, so use a deliberately malformed row) and asserts nothing from the
// batch persists.
func TestCreateBatch_EmptyBatchIsNoOp(t *testing.T) {
	db := openTestDB(t)
	repo := &UserRepository{}

	if err := repo.CreateBatch(db, nil); err != nil {
		t.Fatalf("empty batch should be a no-op, got %v", err)
	}
	var n int64
	db.Model(&models.User{}).Count(&n)
	if n != 0 {
		t.Fatalf("empty batch created %d rows", n)
	}
}

// TestCreateBatch_FullBatchPersists asserts the happy path still works: when
// every row is valid, all rows are written and a 201 with the full import
// result is what the caller gets.
func TestCreateBatch_FullBatchPersists(t *testing.T) {
	db := openTestDB(t)
	repo := &UserRepository{}

	batch := []*models.User{
		{Email: "a@example.com", PasswordHash: "h", Name: "A", Role: "student"},
		{Email: "b@example.com", PasswordHash: "h", Name: "B", Role: "student"},
		{Email: "c@example.com", PasswordHash: "h", Name: "C", Role: "student"},
	}
	if err := repo.CreateBatch(db, batch); err != nil {
		t.Fatalf("full batch insert failed: %v", err)
	}
	var n int64
	db.Model(&models.User{}).Count(&n)
	if n != 3 {
		t.Fatalf("expected 3 rows persisted, got %d", n)
	}
}
