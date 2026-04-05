package controllers

import (
	"errors"
	"finance/config"
	"finance/models"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type signupBody struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserResponse struct {
	ID                uint64  `json:"id"`
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	RoleID            *uint64 `json:"role_id"`
	Active            bool    `json:"active"`
	PasswordUpdatedAt int64   `json:"password_updated_at"`
}

func Signup(c *fiber.Ctx) error {
	var body signupBody
	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
		return nil
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))

	if body.Name == "" || body.Email == "" || body.Password == "" {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Name, email and password are required",
		})
		return nil
	}

	if len(body.Password) < 8 {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Password must be at least 8 characters",
		})
		return nil
	}

	var existing models.User
	if err := existing.GetByEmail(body.Email); err == nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Unable to create account/account with this email already exists",
		})
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Unable to create account",
		})
		return nil
	}

	roleID, err := resolveSignupRoleID()
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not hash password",
		})
		return nil
	}

	now := time.Now().Unix()
	user := models.User{
		Name:              body.Name,
		Email:             body.Email,
		Password:          string(hash),
		RoleID:            roleID,
		Active:            true,
		PasswordUpdatedAt: now,
	}

	if err := user.Create(); err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not create user",
		})
		return nil
	}

	tokenString, err := generateAuthToken(user)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
		return nil
	}

	setAuthCookie(c, tokenString)
	c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Signup successful",
		"user":    toAuthUserResponse(user),
	})
	return nil
}

func Login(c *fiber.Ctx) error {
	var body loginBody
	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
		return nil
	}

	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	if body.Email == "" || body.Password == "" {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Email and password are required",
		})
		return nil
	}

	var user models.User
	if err := user.GetByEmail(body.Email); err != nil {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid credentials",
		})
		return nil
	}

	if !user.Active {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid credentials",
		})
		return nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid credentials",
		})
		return nil
	}

	tokenString, err := generateAuthToken(user)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
		return nil
	}

	setAuthCookie(c, tokenString)

	c.JSON(fiber.Map{
		"status":  "success",
		"message": "Login successful",
		"user":    toAuthUserResponse(user),
	})
	return nil
}

func generateAuthToken(user models.User) (string, error) {
	if config.GlobalConfig == nil || strings.TrimSpace(config.GlobalConfig.Secret) == "" {
		return "", errors.New("server auth secret is not configured")
	}

	var roleID float64
	if user.RoleID != nil {
		roleID = float64(*user.RoleID)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":             float64(user.ID),
		"role_id":             roleID,
		"password_updated_at": float64(user.PasswordUpdatedAt),
		"iat":                 now.Unix(),
		"exp":                 now.Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GlobalConfig.Secret))
}

func resolveSignupRoleID() (*uint64, error) {
	var role models.Role
	if err := role.GetByName(string(models.RoleViewer)); err != nil {
		return nil, errors.New("default role not found, seed roles first")
	}

	id := uint64(role.ID)
	return &id, nil
}

func setAuthCookie(c *fiber.Ctx, token string) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("FN_ENV")), "PROD")
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		HTTPOnly: true,
		Secure:   isProd,
		SameSite: "Lax",
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func toAuthUserResponse(user models.User) authUserResponse {
	return authUserResponse{
		ID:                user.ID,
		Name:              user.Name,
		Email:             user.Email,
		RoleID:            user.RoleID,
		Active:            user.Active,
		PasswordUpdatedAt: user.PasswordUpdatedAt,
	}
}
