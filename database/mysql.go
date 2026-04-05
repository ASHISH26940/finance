package database

import (
	"finance/config"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MysqlDbInstance struct{
	Db *gorm.DB
}

var Database MysqlDbInstance

func MySqlConnectDb(config *config.Configuration){
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	db,err:=gorm.Open(mysql.Open(dsn),&gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err!=nil{
		log.Fatal("Failed to Connect db \n",err.Error())
		os.Exit(2)
	}

	log.Println("connection to Database established")

	if config.DBUsername == "testers"{
		db.Logger=logger.Default.LogMode(logger.Info)
	}else{
		db.Logger=logger.Default.LogMode(logger.Error)
	}

	log.Println("Running Migations")
	Database=MysqlDbInstance{Db:db}

	sqlDB,err:=Database.Db.DB()
	if err!=nil{
		log.Printf("Failed to get underlying sql.DB:%v",err)
		return
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5*time.Minute)
}