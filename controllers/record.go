package controllers

import (
	"finance/logic"
	"finance/models"
	"strconv"
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

	record := models.FinancialRecord{}
	records, err := record.GetByUser(userID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
		return nil
	}

	c.JSON(records)
	return nil
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
