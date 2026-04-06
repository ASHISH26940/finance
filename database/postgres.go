package database

import (
	"finance/config"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func PostgresConnectDb(config *config.Configuration) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
			config.DBHost,
			config.DBUsername,
			config.DBPassword,
			config.DBName,
			config.DBPort,
		)
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

	Database = MysqlDbInstance{Db: db}

	sqlDB, err := Database.Db.DB()
	if err != nil {
		log.Printf("Failed to get underlying sql.DB:%v", err)
		return
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
}
