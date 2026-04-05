package controllers

import (
	"encoding/json"
	"errors"
	"finance/models"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type adminUserResponse struct {
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	RoleID    *uint64 `json:"role_id"`
	Active    bool    `json:"active"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type adminRoleResponse struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Permissions models.Permission `json:"permissions"`
}

type updateUserBody struct {
	Name   *string `json:"name"`
	RoleID *uint64 `json:"role_id"`
	Active *bool   `json:"active"`
}

type upsertRoleBody struct {
	Name        string            `json:"name"`
	Permissions models.Permission `json:"permissions"`
}

func ListUsers(c *fiber.Ctx) error {
	user := models.User{}
	users, err := user.List()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not load users")
	}

	response := make([]adminUserResponse, 0, len(users))
	for _, item := range users {
		response = append(response, toAdminUserResponse(item))
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}

func GetUser(c *fiber.Ctx) error {
	id, err := parseUintParam(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "user id must be a positive integer")
	}

	var user models.User
	if err := user.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "User not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load user")
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   toAdminUserResponse(user),
	})
}

func UpdateUser(c *fiber.Ctx) error {
	id, err := parseUintParam(c.Params("id"))
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "user id must be a positive integer")
	}

	var user models.User
	if err := user.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "User not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load user")
	}

	var body updateUserBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if body.Name == nil && body.RoleID == nil && body.Active == nil {
		return writeError(c, fiber.StatusBadRequest, "At least one field is required for update")
	}

	if body.Name != nil {
		trimmed := strings.TrimSpace(*body.Name)
		if trimmed == "" {
			return writeError(c, fiber.StatusBadRequest, "name must not be empty")
		}
		user.Name = trimmed
	}

	if body.RoleID != nil {
		role := models.Role{}
		if err := role.GetById(uint(*body.RoleID)); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return writeError(c, fiber.StatusBadRequest, "role not found")
			}
			return writeError(c, fiber.StatusInternalServerError, "Could not load role")
		}
		user.RoleID = body.RoleID
	}

	if body.Active != nil {
		user.Active = *body.Active
	}

	if err := user.Update(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not update user")
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   toAdminUserResponse(user),
	})
}

func ListRoles(c *fiber.Ctx) error {
	role := models.Role{}
	roles, err := role.List()
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not load roles")
	}

	response := make([]adminRoleResponse, 0, len(roles))
	for _, item := range roles {
		response = append(response, toAdminRoleResponse(item))
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}

func GetRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Params("id")), 10, 64)
	if err != nil || id == 0 {
		return writeError(c, fiber.StatusBadRequest, "role id must be a positive integer")
	}

	var role models.Role
	if err := role.GetById(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "Role not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load role")
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   toAdminRoleResponse(role),
	})
}

func CreateRole(c *fiber.Ctx) error {
	var body upsertRoleBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	name := strings.ToLower(strings.TrimSpace(body.Name))
	if name == "" {
		return writeError(c, fiber.StatusBadRequest, "name is required")
	}

	role := models.Role{}
	if err := role.GetByName(name); err == nil {
		return writeError(c, fiber.StatusBadRequest, "role with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return writeError(c, fiber.StatusInternalServerError, "Could not validate role name")
	}

	permissionsBytes, err := json.Marshal(body.Permissions)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid permissions payload")
	}

	role = models.Role{
		Name:        name,
		Permissions: string(permissionsBytes),
	}

	if err := role.Create(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not create role")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   toAdminRoleResponse(role),
	})
}

func UpdateRole(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Params("id")), 10, 64)
	if err != nil || id == 0 {
		return writeError(c, fiber.StatusBadRequest, "role id must be a positive integer")
	}

	var role models.Role
	if err := role.GetById(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusNotFound, "Role not found")
		}
		return writeError(c, fiber.StatusInternalServerError, "Could not load role")
	}

	var body upsertRoleBody
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	name := strings.ToLower(strings.TrimSpace(body.Name))
	if name == "" {
		return writeError(c, fiber.StatusBadRequest, "name is required")
	}

	if role.Name != name {
		existing := models.Role{}
		if err := existing.GetByName(name); err == nil {
			return writeError(c, fiber.StatusBadRequest, "role with this name already exists")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return writeError(c, fiber.StatusInternalServerError, "Could not validate role name")
		}
	}

	permissionsBytes, err := json.Marshal(body.Permissions)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid permissions payload")
	}

	role.Name = name
	role.Permissions = string(permissionsBytes)
	if err := role.Update(); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Could not update role")
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   toAdminRoleResponse(role),
	})
}

func toAdminUserResponse(user models.User) adminUserResponse {
	return adminUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		RoleID:    user.RoleID,
		Active:    user.Active,
		CreatedAt: user.CreatedAt.Format(timeLayoutRFC3339),
		UpdatedAt: user.UpdatedAt.Format(timeLayoutRFC3339),
	}
}

func toAdminRoleResponse(role models.Role) adminRoleResponse {
	permissions, _ := role.GetPermissions()
	return adminRoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Permissions: permissions,
	}
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"
