package controllers

import (
	"errors"
	"finance/logic"
	"finance/models"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type createRecordBody struct {
	Amount     float64 `json:"amount"`
	Type       string  `json:"type"`
	CategoryID *uint64 `json:"category_id"`
	Date       string  `json:"date"`
	Note       string  `json:"note"`
}

type updateRecordBody struct {
	Amount     *float64 `json:"amount"`
	Type       *string  `json:"type"`
	CategoryID *uint64  `json:"category_id"`
	Date       *string  `json:"date"`
	Note       *string  `json:"note"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func CreateRecord(c *fiber.Ctx) error {
	var body createRecordBody

	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	userID, ok := logic.ExtractUserID(c)
	if !ok {
		return writeError(c, fiber.StatusUnauthorized, "Invalid token claims")
	}

	recordType, err := parseRecordTypeQuery(body.Type)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}
	if recordType == "" {
		return writeError(c, fiber.StatusBadRequest, "type is required")
	}

	if body.Amount <= 0 {
		return writeError(c, fiber.StatusBadRequest, "amount must be greater than 0")
	}

	if strings.TrimSpace(body.Date) == "" {
		return writeError(c, fiber.StatusBadRequest, "date is required")
	}

	parsedDate, err := time.Parse(time.RFC3339, body.Date)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "date must be RFC3339 format")
	}

	if err := validateCategoryForRecord(userID, body.CategoryID, recordType); err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	record := models.FinancialRecord{
		UserID:     userID,
		Amount:     body.Amount,
		Type:       recordType,
		CategoryID: body.CategoryID,
		Date:       parsedDate,
		Note:       strings.TrimSpace(body.Note),
	}

	if err := record.Create(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not create record")
	}

	return c.JSON(record)
}

func GetRecords(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		return writeError(c, fiber.StatusUnauthorized, "Invalid token claims")
	}

	page := parsePageQuery(c.Query("page"), 1)
	perPage := parsePerPageQuery(c.Query("per_page"), 20)
	recordType, err := parseRecordTypeQuery(c.Query("type"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	categoryID, err := parseOptionalUintQuery(c.Query("category_id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "category_id must be a positive integer")
	}

	from, err := parseDateQuery(c.Query("from"), false)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "from must be RFC3339 or YYYY-MM-DD")
	}

	to, err := parseDateQuery(c.Query("to"), true)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "to must be RFC3339 or YYYY-MM-DD")
	}

	if from != nil && to != nil && from.After(*to) {
		return writeError(c, fiber.StatusBadRequest, "from must be earlier than or equal to to")
	}

	record := models.FinancialRecord{}
	recordPage, err := record.GetByUser(userID, page, perPage, models.RecordListFilters{
		SearchText: c.Query("search"),
		RecordType: recordType,
		CategoryID: categoryID,
		From:       from,
		To:         to,
	})
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not load records")
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   recordPage.Records,
		"pagination": paginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      recordPage.Total,
			TotalPages: calculateTotalPages(recordPage.Total, perPage),
		},
	})
}

func UpdateRecord(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		return writeError(c, fiber.StatusUnauthorized, "Invalid token claims")
	}

	id, err := parseUintParam(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "record id must be a positive integer")
	}

	record := models.FinancialRecord{}
	if err := record.GetByIDForUser(id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "Record not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load record")
	}

	var body updateRecordBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if body.Amount == nil && body.Type == nil && body.CategoryID == nil && body.Date == nil && body.Note == nil {
		return writeError(c, fiber.StatusBadRequest, "At least one field is required for update")
	}

	nextType := record.Type
	if body.Type != nil {
		parsedType, err := parseRecordTypeQuery(*body.Type)
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, err.Error())
		}
		if parsedType == "" {
			return writeError(c, fiber.StatusBadRequest, "type must not be empty")
		}
		nextType = parsedType
		record.Type = parsedType
	}

	if body.Amount != nil {
		if *body.Amount <= 0 {
			return writeError(c, fiber.StatusBadRequest, "amount must be greater than 0")
		}
		record.Amount = *body.Amount
	}

	if body.Date != nil {
		if strings.TrimSpace(*body.Date) == "" {
			return writeError(c, fiber.StatusBadRequest, "date must not be empty")
		}
		parsedDate, err := time.Parse(time.RFC3339, *body.Date)
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, "date must be RFC3339 format")
		}
		record.Date = parsedDate
	}

	if body.Note != nil {
		record.Note = strings.TrimSpace(*body.Note)
	}

	if body.CategoryID != nil {
		if err := validateCategoryForRecord(userID, body.CategoryID, nextType); err != nil {
			return writeError(c, fiber.StatusBadRequest, err.Error())
		}
		record.CategoryID = body.CategoryID
	} else if body.Type != nil && record.CategoryID != nil {
		if err := validateCategoryForRecord(userID, record.CategoryID, nextType); err != nil {
			return writeError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	if err := record.Update(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not update record")
	}

	return c.JSON(record)
}

func DeleteRecord(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		return writeError(c, fiber.StatusUnauthorized, "Invalid token claims")
	}

	id, err := parseUintParam(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "record id must be a positive integer")
	}

	record := models.FinancialRecord{}
	if err := record.GetByIDForUser(id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "Record not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load record")
	}

	if err := record.Delete(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not delete record")
	}

	return c.JSON(fiber.Map{"status": "deleted"})
}

func parsePageQuery(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}

	return parsed
}

func parsePerPageQuery(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}

	if parsed > 100 {
		return 100
	}

	return parsed
}

func calculateTotalPages(total int64, perPage int) int {
	if total == 0 || perPage <= 0 {
		return 0
	}

	return int((total + int64(perPage) - 1) / int64(perPage))
}

func parseRecordTypeQuery(value string) (models.RecordType, error) {
	recordType := models.RecordType(strings.ToLower(strings.TrimSpace(value)))
	if recordType == "" {
		return "", nil
	}

	switch recordType {
	case models.Income, models.Expense:
		return recordType, nil
	default:
		return "", fiber.NewError(fiber.StatusBadRequest, "type must be either income or expense")
	}
}

func parseOptionalUintQuery(value string) (*uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid unsigned integer")
	}

	return &parsed, nil
}

func parseDateQuery(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}

	parsedDate, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}

	if endOfDay {
		parsedDate = parsedDate.Add(24*time.Hour - time.Nanosecond)
	}

	return &parsedDate, nil
}

func parseUintParam(value string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed == 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid unsigned integer")
	}

	return parsed, nil
}

func validateCategoryForRecord(userID uint64, categoryID *uint64, recordType models.RecordType) error {
	if categoryID == nil {
		return nil
	}

	category := models.Category{}
	if err := category.GetByIDForUser(*categoryID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "category not found")
		}
		return err
	}

	if recordType != "" && category.Type != recordType {
		return fiber.NewError(fiber.StatusBadRequest, "category type must match record type")
	}

	return nil
}

func writeError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}
