package controllers

import (
	"finance/logic"
	"finance/models"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type createRecordBody struct {
	Amount     float64 `json:"amount"`
	Type       string  `json:"type"`
	CategoryID *uint64 `json:"category_id"`
	Date       string  `json:"date"`
	Note       string  `json:"note"`
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
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error"})
		return nil
	}

	userID, ok := logic.ExtractUserID(c)
	if !ok {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token claims",
		})
		return nil
	}

	parsedDate, err := time.Parse(time.RFC3339, body.Date)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error"})
		return nil
	}

	record := models.FinancialRecord{
		UserID:     userID,
		Amount:     body.Amount,
		Type:       models.RecordType(body.Type),
		CategoryID: body.CategoryID,
		Date:       parsedDate,
		Note:       body.Note,
	}

	if err := record.Create(); err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
		return nil
	}

	c.JSON(record)
	return nil
}

func GetRecords(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token claims",
		})
		return nil
	}

	page := parsePageQuery(c.Query("page"), 1)
	perPage := parsePerPageQuery(c.Query("per_page"), 20)
	recordType, err := parseRecordTypeQuery(c.Query("type"))
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
		return nil
	}

	categoryID, err := parseOptionalUintQuery(c.Query("category_id"))
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "category_id must be a positive integer",
		})
		return nil
	}

	from, err := parseDateQuery(c.Query("from"), false)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "from must be RFC3339 or YYYY-MM-DD",
		})
		return nil
	}

	to, err := parseDateQuery(c.Query("to"), true)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "to must be RFC3339 or YYYY-MM-DD",
		})
		return nil
	}

	if from != nil && to != nil && from.After(*to) {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "from must be earlier than or equal to to",
		})
		return nil
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
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
		return nil
	}

	c.JSON(fiber.Map{
		"status": "success",
		"data":   recordPage.Records,
		"pagination": paginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      recordPage.Total,
			TotalPages: calculateTotalPages(recordPage.Total, perPage),
		},
	})
	return nil
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

func UpdateRecord(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error"})
		return nil
	}

	record := models.FinancialRecord{}
	if err := record.GetByID(id); err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error"})
		return nil
	}

	if err := c.BodyParser(&record); err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error"})
		return nil
	}

	if err := record.Update(); err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
		return nil
	}

	c.JSON(record)
	return nil
}

func DeleteRecord(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error"})
		return nil
	}

	record := models.FinancialRecord{}
	if err := record.GetByID(id); err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error"})
		return nil
	}

	if err := record.Delete(); err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
		return nil
	}

	c.JSON(fiber.Map{"status": "deleted"})
	return nil
}
