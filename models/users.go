package models

import (
	"finance/database"
	"time"
)

type User struct {
	ID                uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name              string     `gorm:"column:name;type:varchar(100);not null"`
	Email             string     `gorm:"column:email;type:varchar(150);unique;not null"`
	Password          string     `gorm:"column:password;type:varchar(255);not null"`
	RoleID            *uint64    `gorm:"column:role_id"`
	Active            bool       `gorm:"column:active;default:true"`
	PasswordUpdatedAt int64      `gorm:"column:password_updated_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) Create() error {
	return database.Database.Db.Create(u).Error
}

func (u *User) GetByID(id uint64) error {
	return database.Database.Db.First(u, id).Error
}

func (u *User) GetByEmail(email string) error {
	return database.Database.Db.Where("email = ?", email).First(u).Error
}

func (u *User) Update() error {
	return database.Database.Db.Save(u).Error
}

func (u *User) Delete() error {
	return database.Database.Db.Delete(u).Error
}