package models

import (
	"finance/database"
	"strings"
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

type RecordListPage struct {
	Records []FinancialRecord
	Total   int64
}

type RecordListFilters struct {
	SearchText string
	RecordType RecordType
	CategoryID *uint64
	From       *time.Time
	To         *time.Time
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

func (r *FinancialRecord) GetByUser(userID uint64, page, perPage int, filters RecordListFilters) (RecordListPage, error) {
	if page < 1 {
		page = 1
	}

	if perPage < 1 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	baseQuery := database.Database.Db.Model(&FinancialRecord{}).Where("user_id = ?", userID)
	baseQuery = applyRecordFilters(baseQuery, filters)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return RecordListPage{}, err
	}

	var records []FinancialRecord
	err := baseQuery.
		Order("date DESC, id DESC").
		Offset(offset).
		Limit(perPage).
		Find(&records).Error
	if err != nil {
		return RecordListPage{}, err
	}

	return RecordListPage{
		Records: records,
		Total:   total,
	}, nil
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

func applyRecordFilters(query *gorm.DB, filters RecordListFilters) *gorm.DB {
	if filters.RecordType != "" {
		query = query.Where("type = ?", filters.RecordType)
	}

	if filters.CategoryID != nil {
		query = query.Where("category_id = ?", *filters.CategoryID)
	}

	if filters.From != nil {
		query = query.Where("date >= ?", *filters.From)
	}

	if filters.To != nil {
		query = query.Where("date <= ?", *filters.To)
	}

	if strings.TrimSpace(filters.SearchText) != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filters.SearchText)) + "%"
		query = query.Where("LOWER(note) LIKE ?", searchTerm)
	}

	return query
}
