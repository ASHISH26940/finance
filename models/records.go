package models

import (
	"finance/database"
	"time"

	"gorm.io/gorm"
)

type RecordType string

const (
	Income  RecordType = "income"
	Expense RecordType = "expense"
)

type Category struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement"`
	Name      string     `gorm:"size:100;not null"`
	Type      RecordType `gorm:"type:enum('income','expense');not null"`
	UserID    *uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Category) TableName() string {
	return "categories"
}

func (c *Category) Create() error {
	return database.Database.Db.Create(c).Error
}

func (c *Category) GetByID(id uint64) error {
	return database.Database.Db.First(c, id).Error
}

func (c *Category) GetAllByUser(userID uint64) ([]Category, error) {
	var categories []Category
	err := database.Database.Db.Where("user_id = ?", userID).Find(&categories).Error
	return categories, err
}

func (c *Category) Update() error {
	return database.Database.Db.Save(c).Error
}

func (c *Category) Delete() error {
	return database.Database.Db.Delete(c).Error
}

type FinancialRecord struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	UserID     uint64     `gorm:"not null"`
	Amount     float64    `gorm:"type:decimal(15,2);not null"`
	Type       RecordType `gorm:"type:enum('income','expense');not null"`
	CategoryID *uint64
	Date       time.Time `gorm:"not null"`
	Note       string    `gorm:"type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (FinancialRecord) TableName() string {
	return "financial_records"
}

func (r *FinancialRecord) Create() error {
	return database.Database.Db.Create(r).Error
}

func (r *FinancialRecord) GetByID(id uint64) error {
	return database.Database.Db.First(r, id).Error
}

func (r *FinancialRecord) GetByUser(userID uint64) ([]FinancialRecord, error) {
	var records []FinancialRecord
	err := database.Database.Db.Where("user_id = ?", userID).Find(&records).Error
	return records, err
}

func (r *FinancialRecord) Update() error {
	return database.Database.Db.Save(r).Error
}

func (r *FinancialRecord) Delete() error {
	return database.Database.Db.Delete(r).Error
}

func (r *FinancialRecord) Filter(userID uint64, recordType RecordType, categoryID *uint64, from, to *time.Time) ([]FinancialRecord, error) {
	query := database.Database.Db.Where("user_id=?", userID)

	if recordType != "" {
		query = query.Where("type=?", recordType)
	}

	if categoryID != nil {
		query = query.Where("category_id=?", *categoryID)
	}

	if from != nil && to != nil {
		query = query.Where("date BEWTWEEN ? AND ?", *from, *to)
	}

	var records []FinancialRecord
	err := query.Find(&records).Error
	return records, err
}
