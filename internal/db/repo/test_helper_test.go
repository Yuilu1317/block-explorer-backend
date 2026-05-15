package repo

import (
	"block-explorer-backend/internal/db/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T, migrateModels ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	sqlDB.SetMaxOpenConns(1)

	if len(migrateModels) == 0 {
		migrateModels = []any{&models.Block{}}
	}

	if err := db.AutoMigrate(migrateModels...); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}
