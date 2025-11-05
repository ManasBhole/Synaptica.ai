package database

import (
	"fmt"
	"sync"

	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db     *gorm.DB
	dbOnce sync.Once
)

func GetPostgres() (*gorm.DB, error) {
	var err error
	dbOnce.Do(func() {
		cfg := config.Load()
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			cfg.PostgresHost,
			cfg.PostgresUser,
			cfg.PostgresPassword,
			cfg.PostgresDB,
			cfg.PostgresPort,
			cfg.PostgresSSLMode,
		)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.Log.WithError(err).Error("Failed to connect to PostgreSQL")
			return
		}

		logger.Log.Info("Connected to PostgreSQL")
	})

	return db, err
}

func ClosePostgres() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

