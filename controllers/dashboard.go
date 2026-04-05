package controllers

import (
	"finance/database"
	"finance/logic"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type dashboardSummaryResponse struct {
	TotalIncome    float64          `json:"total_income"`
	TotalExpenses  float64          `json:"total_expenses"`
	NetBalance     float64          `json:"net_balance"`
	CategoryTotals []categoryTotal  `json:"category_totals"`
	RecentActivity []recentActivity `json:"recent_activity"`
	MonthlyTrends  []trendPoint     `json:"monthly_trends"`
	GeneratedAt    time.Time        `json:"generated_at"`
}

type categoryTotal struct {
	CategoryID   *uint64 `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Type         string  `json:"type"`
	Total        float64 `json:"total"`
}

type recentActivity struct {
	ID           uint64    `json:"id"`
	Amount       float64   `json:"amount"`
	Type         string    `json:"type"`
	CategoryID   *uint64   `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Date         time.Time `json:"date"`
	Note         string    `json:"note"`
}

type trendPoint struct {
	Period  string  `json:"period"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
}

type trendResponse struct {
	Period      string       `json:"period"`
	Points      int          `json:"points"`
	Trends      []trendPoint `json:"trends"`
	GeneratedAt time.Time    `json:"generated_at"`
}

func GetDashboardSummary(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token claims",
		})
		return nil
	}

	totalIncome, totalExpense, err := getIncomeExpenseTotals(userID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not load dashboard totals",
		})
		return nil
	}

	categoryTotals, err := getCategoryTotals(userID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not load category totals",
		})
		return nil
	}

	recent, err := getRecentActivity(userID, 10)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not load recent activity",
		})
		return nil
	}

	monthlyTrends, err := getMonthlyTrends(userID, 6)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not load monthly trends",
		})
		return nil
	}

	c.JSON(fiber.Map{
		"status": "success",
		"data": dashboardSummaryResponse{
			TotalIncome:    totalIncome,
			TotalExpenses:  totalExpense,
			NetBalance:     totalIncome - totalExpense,
			CategoryTotals: categoryTotals,
			RecentActivity: recent,
			MonthlyTrends:  monthlyTrends,
			GeneratedAt:    time.Now(),
		},
	})
	return nil
}

func GetDashboardTrends(c *fiber.Ctx) error {
	userID, ok := logic.ExtractUserID(c)
	if !ok {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid token claims",
		})
		return nil
	}

	period := strings.ToLower(strings.TrimSpace(c.Query("period", "monthly")))
	points := parsePositiveInt(c.Query("points"), 6)

	var (
		trends []trendPoint
		err    error
	)

	switch period {
	case "weekly":
		trends, err = getWeeklyTrends(userID, points)
	default:
		period = "monthly"
		trends, err = getMonthlyTrends(userID, points)
	}

	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not load trends",
		})
		return nil
	}

	c.JSON(fiber.Map{
		"status": "success",
		"data": trendResponse{
			Period:      period,
			Points:      points,
			Trends:      trends,
			GeneratedAt: time.Now(),
		},
	})
	return nil
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	if parsed > 24 {
		return 24
	}

	return parsed
}

func getIncomeExpenseTotals(userID uint64) (float64, float64, error) {
	type totalsRow struct {
		Income  float64 `gorm:"column:income"`
		Expense float64 `gorm:"column:expense"`
	}

	var row totalsRow
	err := database.Database.Db.
		Table("financial_records").
		Select(`
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense
		`).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Scan(&row).Error
	if err != nil {
		return 0, 0, err
	}

	return row.Income, row.Expense, nil
}

func getCategoryTotals(userID uint64) ([]categoryTotal, error) {
	var rows []categoryTotal
	err := database.Database.Db.
		Table("financial_records").
		Select(`
			category_id,
			type,
			COALESCE(SUM(amount), 0) AS total
		`).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Group("category_id, type").
		Order("total DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	categoryNames, err := getCategoryNamesByIDs(extractCategoryIDsFromTotals(rows))
	if err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].CategoryName = resolveCategoryName(categoryNames, rows[i].CategoryID)
	}

	return rows, nil
}

func getRecentActivity(userID uint64, limit int) ([]recentActivity, error) {
	var rows []recentActivity
	err := database.Database.Db.
		Table("financial_records").
		Select(`
			id,
			amount,
			type,
			category_id,
			date,
			note
		`).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("date DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	categoryNames, err := getCategoryNamesByIDs(extractCategoryIDsFromRecent(rows))
	if err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].CategoryName = resolveCategoryName(categoryNames, rows[i].CategoryID)
	}

	return rows, nil
}

func getMonthlyTrends(userID uint64, points int) ([]trendPoint, error) {
	var rows []trendPoint
	err := database.Database.Db.
		Table("financial_records").
		Select(`
			DATE_FORMAT(date, '%Y-%m') AS period,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS net
		`).
		Where("user_id = ? AND deleted_at IS NULL AND date >= DATE_SUB(CURDATE(), INTERVAL ? MONTH)", userID, points).
		Group("DATE_FORMAT(date, '%Y-%m')").
		Order("period ASC").
		Scan(&rows).Error
	return rows, err
}

func getWeeklyTrends(userID uint64, points int) ([]trendPoint, error) {
	var rows []trendPoint
	err := database.Database.Db.
		Table("financial_records").
		Select(`
			CONCAT(YEAR(date), '-W', LPAD(WEEK(date, 3), 2, '0')) AS period,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS net
		`).
		Where("user_id = ? AND deleted_at IS NULL AND date >= DATE_SUB(CURDATE(), INTERVAL ? WEEK)", userID, points).
		Group("YEAR(date), WEEK(date, 3)").
		Order("YEAR(date) ASC, WEEK(date, 3) ASC").
		Scan(&rows).Error
	return rows, err
}

func getCategoryNamesByIDs(categoryIDs []uint64) (map[uint64]string, error) {
	if len(categoryIDs) == 0 {
		return map[uint64]string{}, nil
	}

	type categoryRow struct {
		ID   uint64 `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}

	var rows []categoryRow
	err := database.Database.Db.
		Table("categories").
		Select("id, name").
		Where("id IN ?", categoryIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	categoryNames := make(map[uint64]string, len(rows))
	for _, row := range rows {
		categoryNames[row.ID] = row.Name
	}

	return categoryNames, nil
}

func extractCategoryIDsFromTotals(rows []categoryTotal) []uint64 {
	seen := make(map[uint64]struct{})
	categoryIDs := make([]uint64, 0, len(rows))

	for _, row := range rows {
		if row.CategoryID == nil {
			continue
		}

		if _, exists := seen[*row.CategoryID]; exists {
			continue
		}

		seen[*row.CategoryID] = struct{}{}
		categoryIDs = append(categoryIDs, *row.CategoryID)
	}

	return categoryIDs
}

func extractCategoryIDsFromRecent(rows []recentActivity) []uint64 {
	seen := make(map[uint64]struct{})
	categoryIDs := make([]uint64, 0, len(rows))

	for _, row := range rows {
		if row.CategoryID == nil {
			continue
		}

		if _, exists := seen[*row.CategoryID]; exists {
			continue
		}

		seen[*row.CategoryID] = struct{}{}
		categoryIDs = append(categoryIDs, *row.CategoryID)
	}

	return categoryIDs
}

func resolveCategoryName(categoryNames map[uint64]string, categoryID *uint64) string {
	if categoryID == nil {
		return "Uncategorized"
	}

	name, ok := categoryNames[*categoryID]
	if !ok || strings.TrimSpace(name) == "" {
		return "Uncategorized"
	}

	return name
}
