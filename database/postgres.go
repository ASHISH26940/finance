package database

import (
	"finance/config"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresDbInstance struct {
	Db *gorm.DB
}

var Database PostgresDbInstance

func PostgresConnectDb(config *config.Configuration) {
	sslMode := strings.TrimSpace(config.DBSSLMode)
	if sslMode == "" {
		sslMode = "require"
	}

	dsn := ""
	if strings.TrimSpace(config.DBHost) != "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Kolkata",
			config.DBHost,
			config.DBUsername,
			config.DBPassword,
			config.DBName,
			config.DBPort,
			sslMode,
		)
	} else if strings.TrimSpace(config.DatabaseURL) != "" {
		dsn = config.DatabaseURL
	} else {
		log.Fatal("database configuration missing: set DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME or DATABASE_URL")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	log.Println("connection to Database established")

	if config.DBUsername == "testers" {
		db.Logger = logger.Default.LogMode(logger.Info)
	} else {
		db.Logger = logger.Default.LogMode(logger.Error)
	}

	Database = PostgresDbInstance{Db: db}

	sqlDB, err := Database.Db.DB()
	if err != nil {
		log.Printf("Failed to get underlying sql.DB:%v", err)
		return
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
}
