// Command course-attendance is the single entry point that wires config,
// logger, database, services, handlers, and router together. It owns no
// business logic; every package above contributes one capability.
package main

import (
	"fmt"
	"os"

	"course-attendance/config"
	"course-attendance/internal/database"
	"course-attendance/internal/handler"
	"course-attendance/internal/page"
	"course-attendance/internal/repository"
	"course-attendance/internal/router"
	"course-attendance/internal/service"
	"course-attendance/internal/templates"
	"course-attendance/pkg/hashutil"
	"course-attendance/pkg/jwtauth"
	"course-attendance/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	db, err := database.Open(cfg.DBType, cfg.DBDSN)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}

	seedAdminIfNeeded(db, log)

	// Repositories — one per aggregate.
	userRepo := repository.NewUserRepository(db)
	courseRepo := repository.NewCourseRepository(db)
	enrollRepo := repository.NewEnrollmentRepository(db)
	attendanceRepo := repository.NewAttendanceRepository(db)
	qrRepo := repository.NewQRCodeRepository(db)

	// Shared infra.
	tokenMgr := jwtauth.New(cfg.JWTSecret)

	// Services — one per domain.
	authSvc := service.NewAuthService(db, userRepo, tokenMgr, cfg.BaseURL)
	courseSvc := service.NewCourseService(db, courseRepo, userRepo, enrollRepo)
	enrollSvc := service.NewEnrollmentService(db, enrollRepo, userRepo, courseRepo)
	attendanceSvc := service.NewAttendanceService(db, qrRepo, courseRepo, enrollRepo, attendanceRepo, cfg.BaseURL)
	statsSvc := service.NewStatisticsService(db, attendanceRepo, enrollRepo, courseRepo)
	exportSvc := service.NewExportService(statsSvc)
	teacherSvc := service.NewTeacherService(db, userRepo, courseRepo)
	_ = service.NewNotificationService(log) // wired into services when delivery channels land

	// Handlers — one per resource.
	authH := handler.NewAuthHandler(authSvc)
	courseH := handler.NewCourseHandler(courseSvc)
	enrollH := handler.NewEnrollmentHandler(enrollSvc)
	attendanceH := handler.NewAttendanceHandler(attendanceSvc)
	statsH := handler.NewStatisticsHandler(statsSvc)
	exportH := handler.NewExportHandler(exportSvc, courseSvc)
	teacherH := handler.NewTeacherHandler(teacherSvc)
	pageH := page.NewPageHandler(authSvc, courseSvc, enrollSvc, attendanceSvc, statsSvc, teacherSvc, tokenMgr)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	if err := templates.Load(engine, "web/templates"); err != nil {
		log.Fatal("templates", zap.Error(err))
	}
	engine.Static("/static", "web/static")

	router.Register(engine, router.Deps{
		Auth:       authH,
		Course:     courseH,
		Enrollment: enrollH,
		Attendance: attendanceH,
		Statistics: statsH,
		Export:     exportH,
		Teacher:    teacherH,
		Page:       pageH,
		AuthSvc:    authSvc,
		Tokens:     tokenMgr,
	})

	addr := ":" + cfg.ServerPort
	log.Info("starting server", zap.String("addr", addr), zap.String("base_url", cfg.BaseURL))
	if err := engine.Run(addr); err != nil {
		log.Fatal("server", zap.Error(err))
	}
}

// seedAdminIfNeeded creates a default admin account on first run so the app
// is usable immediately. Credentials are printed to stderr once.
func seedAdminIfNeeded(db *gorm.DB, log *zap.Logger) {
	const adminEmail = "admin@example.com"
	const adminPassword = "admin123"

	var count int64
	if err := db.Table("users").Where("email = ?", adminEmail).Count(&count).Error; err != nil {
		log.Warn("seed admin: lookup failed", zap.Error(err))
		return
	}
	if count > 0 {
		return
	}
	hash, err := hashutil.HashPassword(adminPassword)
	if err != nil {
		log.Warn("seed admin: hash failed", zap.Error(err))
		return
	}
	admin := &user{
		Email:        adminEmail,
		PasswordHash: hash,
		Name:         "Default Admin",
		Role:         service.RoleAdmin,
	}
	if err := db.Table("users").Create(admin).Error; err != nil {
		log.Warn("seed admin: create failed", zap.Error(err))
		return
	}
	log.Info("seeded default admin account",
		zap.String("email", adminEmail),
		zap.String("password", adminPassword),
	)
}

// user is a local row struct used only for seeding so main.go does not import
// the models package for a single insert.
type user struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	Email        string `gorm:"size:255"`
	PasswordHash string `gorm:"size:255"`
	Name         string `gorm:"size:100"`
	Role         string `gorm:"size:20"`
}

// BeforeCreate satisfies GORM's hook so the seed row gets a UUID.
func (u *user) BeforeCreate(_ *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
