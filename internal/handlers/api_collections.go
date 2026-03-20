package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

type ApiCollectionHandler struct {
	db *gorm.DB
}

func NewApiCollectionHandler(db *gorm.DB) *ApiCollectionHandler {
	return &ApiCollectionHandler{db: db}
}

// List handles GET /api/api-collections
func (h *ApiCollectionHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var collections []models.ApiCollection
	h.db.Where("user_id = ?", user.ID).Order("updated_at desc").Find(&collections)

	return c.JSON(fiber.Map{"collections": collections})
}

// Create handles POST /api/api-collections
func (h *ApiCollectionHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		BaseURL     string `json:"base_url"`
		AuthType    string `json:"auth_type"`
		Visibility  string `json:"visibility"`
		TeamID      string `json:"team_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	slug := slugify(body.Name)
	var existing models.ApiCollection
	base := slug
	counter := 1
	for {
		if err := h.db.Where("user_id = ? AND slug = ?", user.ID, slug).First(&existing).Error; err != nil {
			break
		}
		counter++
		slug = base + "-" + strconv.Itoa(counter)
	}

	col := models.ApiCollection{
		UserID:      user.ID,
		Name:        body.Name,
		Slug:        slug,
		Description: body.Description,
		BaseURL:     body.BaseURL,
		AuthType:    body.AuthType,
		Visibility:  body.Visibility,
	}
	if col.AuthType == "" {
		col.AuthType = "none"
	}
	if col.Visibility == "" {
		col.Visibility = "private"
	}
	if body.TeamID != "" {
		col.TeamID = &body.TeamID
	}

	if err := h.db.Create(&col).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create collection"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"collection": col})
}

// Show handles GET /api/api-collections/:id
func (h *ApiCollectionHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var col models.ApiCollection
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	var endpoints []models.ApiEndpoint
	h.db.Where("collection_id = ?", col.ID).Order("sort_order asc, created_at asc").Find(&endpoints)

	var environments []models.ApiEnvironment
	h.db.Where("collection_id = ?", col.ID).Order("name asc").Find(&environments)

	return c.JSON(fiber.Map{
		"collection":   col,
		"endpoints":    endpoints,
		"environments": environments,
	})
}

// Update handles PUT /api/api-collections/:id
func (h *ApiCollectionHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var col models.ApiCollection
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		BaseURL     *string `json:"base_url"`
		AuthType    *string `json:"auth_type"`
		Visibility  *string `json:"visibility"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name != nil {
		col.Name = *body.Name
		col.Slug = slugify(*body.Name)
	}
	if body.Description != nil {
		col.Description = *body.Description
	}
	if body.BaseURL != nil {
		col.BaseURL = *body.BaseURL
	}
	if body.AuthType != nil {
		col.AuthType = *body.AuthType
	}
	if body.Visibility != nil {
		col.Visibility = *body.Visibility
	}

	if err := h.db.Save(&col).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update collection"})
	}

	return c.JSON(fiber.Map{"collection": col})
}

// Delete handles DELETE /api/api-collections/:id
func (h *ApiCollectionHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var col models.ApiCollection
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	// Cascade: delete endpoints and environments.
	h.db.Where("collection_id = ?", col.ID).Delete(&models.ApiEndpoint{})
	h.db.Where("collection_id = ?", col.ID).Delete(&models.ApiEnvironment{})
	h.db.Delete(&col)

	return c.JSON(fiber.Map{"deleted": true})
}

// ── Endpoints CRUD ──────────────────────────────────────────────────────────

// CreateEndpoint handles POST /api/api-collections/:id/endpoints
func (h *ApiCollectionHandler) CreateEndpoint(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	colID := c.Params("id")
	var col models.ApiCollection
	if err := h.db.Where("id = ? AND user_id = ?", colID, user.ID).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	var body struct {
		Name        string `json:"name"`
		Method      string `json:"method"`
		Path        string `json:"path"`
		Body        string `json:"body"`
		BodyType    string `json:"body_type"`
		Description string `json:"description"`
		FolderPath  string `json:"folder_path"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	ep := models.ApiEndpoint{
		CollectionID: col.ID,
		UserID:       user.ID,
		Name:         body.Name,
		Method:       body.Method,
		Path:         body.Path,
		Body:         body.Body,
		BodyType:     body.BodyType,
		Description:  body.Description,
		FolderPath:   body.FolderPath,
	}
	if ep.Method == "" {
		ep.Method = "GET"
	}
	if ep.BodyType == "" {
		ep.BodyType = "none"
	}

	if err := h.db.Create(&ep).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create endpoint"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"endpoint": ep})
}

// UpdateEndpoint handles PUT /api/api-collections/:id/endpoints/:epId
func (h *ApiCollectionHandler) UpdateEndpoint(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	epID := c.Params("epId")
	var ep models.ApiEndpoint
	if err := h.db.Where("id = ? AND user_id = ?", epID, user.ID).First(&ep).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "endpoint not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if v, ok := body["name"].(string); ok {
		ep.Name = v
	}
	if v, ok := body["method"].(string); ok {
		ep.Method = v
	}
	if v, ok := body["path"].(string); ok {
		ep.Path = v
	}
	if v, ok := body["body"].(string); ok {
		ep.Body = v
	}
	if v, ok := body["body_type"].(string); ok {
		ep.BodyType = v
	}
	if v, ok := body["description"].(string); ok {
		ep.Description = v
	}
	if v, ok := body["folder_path"].(string); ok {
		ep.FolderPath = v
	}
	if v, ok := body["sort_order"].(float64); ok {
		ep.SortOrder = int(v)
	}

	if err := h.db.Save(&ep).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update endpoint"})
	}

	return c.JSON(fiber.Map{"endpoint": ep})
}

// DeleteEndpoint handles DELETE /api/api-collections/:id/endpoints/:epId
func (h *ApiCollectionHandler) DeleteEndpoint(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	epID := c.Params("epId")
	var ep models.ApiEndpoint
	if err := h.db.Where("id = ? AND user_id = ?", epID, user.ID).First(&ep).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "endpoint not found"})
	}

	h.db.Delete(&ep)
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Environments CRUD ───────────────────────────────────────────────────────

// CreateEnvironment handles POST /api/api-collections/:id/environments
func (h *ApiCollectionHandler) CreateEnvironment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	colID := c.Params("id")
	var col models.ApiCollection
	if err := h.db.Where("id = ? AND user_id = ?", colID, user.ID).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	env := models.ApiEnvironment{
		CollectionID: col.ID,
		UserID:       user.ID,
		Name:         body.Name,
	}

	if err := h.db.Create(&env).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create environment"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"environment": env})
}

// UpdateEnvironment handles PUT /api/api-collections/:id/environments/:envId
func (h *ApiCollectionHandler) UpdateEnvironment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	envID := c.Params("envId")
	var env models.ApiEnvironment
	if err := h.db.Where("id = ? AND user_id = ?", envID, user.ID).First(&env).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "environment not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if v, ok := body["name"].(string); ok {
		env.Name = v
	}
	if v, ok := body["is_active"].(bool); ok {
		env.IsActive = v
	}
	if v, ok := body["variables"]; ok {
		vBytes, _ := json.Marshal(v)
		env.Variables = vBytes
	}

	if err := h.db.Save(&env).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update environment"})
	}

	return c.JSON(fiber.Map{"environment": env})
}

// DeleteEnvironment handles DELETE /api/api-collections/:id/environments/:envId
func (h *ApiCollectionHandler) DeleteEnvironment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	envID := c.Params("envId")
	var env models.ApiEnvironment
	if err := h.db.Where("id = ? AND user_id = ?", envID, user.ID).First(&env).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "environment not found"})
	}

	h.db.Delete(&env)
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Public endpoints ────────────────────────────────────────────────────────

// PublicList handles GET /api/public/api-collections/:handle
func (h *ApiCollectionHandler) PublicList(c fiber.Ctx) error {
	handle := c.Params("handle")

	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var collections []models.ApiCollection
	h.db.Where("user_id = ? AND visibility = 'public'", user.ID).
		Order("updated_at desc").Find(&collections)

	return c.JSON(fiber.Map{
		"collections": collections,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}

// PublicShow handles GET /api/public/api-collections/:handle/:slug
func (h *ApiCollectionHandler) PublicShow(c fiber.Ctx) error {
	handle := c.Params("handle")
	slug := c.Params("slug")

	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var col models.ApiCollection
	if err := h.db.Where("user_id = ? AND slug = ? AND visibility = 'public'", user.ID, slug).First(&col).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "collection not found"})
	}

	var endpoints []models.ApiEndpoint
	h.db.Where("collection_id = ?", col.ID).Order("sort_order asc, created_at asc").Find(&endpoints)

	return c.JSON(fiber.Map{
		"collection": col,
		"endpoints":  endpoints,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}
