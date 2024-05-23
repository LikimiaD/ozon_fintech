package database

import (
	"errors"
	"fmt"
	"github.com/likimiad/ozon_fintech/internal/config"
	"github.com/likimiad/ozon_fintech/internal/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log/slog"
	"time"
)

type Database struct {
	*gorm.DB
}

var (
	ErrDatabaseConnect   = errors.New("failed to connect to the database")
	ErrDatabaseMigration = errors.New("error during database auto-migration")
)

// makeConnection establishes a connection to the PostgreSQL database.
func makeConnection(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s port=%s",
		cfg.Host, cfg.User, cfg.Name, cfg.Password, cfg.Port)
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("error when connecting to the database", "error", err)
		return nil, ErrDatabaseConnect
	}
	return &Database{gormDB}, nil
}

// GetDB initializes the database and Redis clients, and performs migrations.
func GetDB(cfg config.Config) (*PostService, error) {
	defer func(start time.Time) {
		slog.Info("database connection is established", "duration", time.Since(start))
	}(time.Now())

	db, err := makeConnection(cfg.DatabaseConfig)
	if err != nil {
		return nil, ErrDatabaseConnect
	}

	rc, err := NewRedisClient(cfg.RedisConfig)
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}); err != nil {
		return nil, ErrDatabaseMigration
	}

	postService := NewPostService(db, rc)

	return postService, nil
}
