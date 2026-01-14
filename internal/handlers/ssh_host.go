package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ihxw/webssh/internal/config"
	"github.com/ihxw/webssh/internal/middleware"
	"github.com/ihxw/webssh/internal/models"
	"github.com/ihxw/webssh/internal/utils"
	"gorm.io/gorm"
)

type SSHHostHandler struct {
	db     *gorm.DB
	config *config.Config
}

func NewSSHHostHandler(db *gorm.DB, cfg *config.Config) *SSHHostHandler {
	return &SSHHostHandler{
		db:     db,
		config: cfg,
	}
}

type CreateSSHHostRequest struct {
	Name        string `json:"name" binding:"required"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port"`
	Username    string `json:"username" binding:"required"`
	AuthType    string `json:"auth_type" binding:"required,oneof=password key"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	GroupName   string `json:"group_name"`
	Tags        string `json:"tags"`
	Description string `json:"description"`
}

type UpdateSSHHostRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	AuthType    string `json:"auth_type" binding:"omitempty,oneof=password key"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	GroupName   string `json:"group_name"`
	Tags        string `json:"tags"`
	Description string `json:"description"`
}

// List returns a list of SSH hosts for the current user
func (h *SSHHostHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	group := c.Query("group")
	search := c.Query("search")

	query := h.db.Model(&models.SSHHost{}).Where("user_id = ?", userID)

	// Group filter
	if group != "" {
		query = query.Where("group_name = ?", group)
	}

	// Search filter
	if search != "" {
		query = query.Where("name LIKE ? OR host LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var hosts []models.SSHHost
	if err := query.Find(&hosts).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to fetch hosts")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, hosts)
}

// Get returns a single SSH host
func (h *SSHHostHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var host models.SSHHost
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "host not found")
		return
	}

	// Decrypt credentials
	if host.PasswordEncrypted != "" {
		password, err := utils.DecryptAES(host.PasswordEncrypted, h.config.Security.EncryptionKey)
		if err == nil {
			host.Password = password
		}
	}
	if host.PrivateKeyEncrypted != "" {
		privateKey, err := utils.DecryptAES(host.PrivateKeyEncrypted, h.config.Security.EncryptionKey)
		if err == nil {
			host.PrivateKey = privateKey
		}
	}

	utils.SuccessResponse(c, http.StatusOK, host)
}

// Create creates a new SSH host
func (h *SSHHostHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateSSHHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Validate auth type and credentials
	if req.AuthType == "password" && req.Password == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "password is required for password authentication")
		return
	}
	if req.AuthType == "key" && req.PrivateKey == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "private key is required for key authentication")
		return
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Create host
	host := &models.SSHHost{
		UserID:      userID,
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		AuthType:    req.AuthType,
		GroupName:   req.GroupName,
		Tags:        req.Tags,
		Description: req.Description,
	}

	// Encrypt credentials
	if req.Password != "" {
		encrypted, err := utils.EncryptAES(req.Password, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to encrypt password")
			return
		}
		host.PasswordEncrypted = encrypted
	}
	if req.PrivateKey != "" {
		encrypted, err := utils.EncryptAES(req.PrivateKey, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to encrypt private key")
			return
		}
		host.PrivateKeyEncrypted = encrypted
	}

	if err := h.db.Create(host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to create host")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, host)
}

// Update updates an SSH host
func (h *SSHHostHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var req UpdateSSHHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	var host models.SSHHost
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "host not found")
		return
	}

	// Update fields
	if req.Name != "" {
		host.Name = req.Name
	}
	if req.Host != "" {
		host.Host = req.Host
	}
	if req.Port != 0 {
		host.Port = req.Port
	}
	if req.Username != "" {
		host.Username = req.Username
	}
	if req.AuthType != "" {
		host.AuthType = req.AuthType
	}
	if req.GroupName != "" {
		host.GroupName = req.GroupName
	}
	if req.Tags != "" {
		host.Tags = req.Tags
	}
	if req.Description != "" {
		host.Description = req.Description
	}

	// Update encrypted credentials if provided
	if req.Password != "" {
		encrypted, err := utils.EncryptAES(req.Password, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to encrypt password")
			return
		}
		host.PasswordEncrypted = encrypted
	}
	if req.PrivateKey != "" {
		encrypted, err := utils.EncryptAES(req.PrivateKey, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to encrypt private key")
			return
		}
		host.PrivateKeyEncrypted = encrypted
	}

	if err := h.db.Save(&host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to update host")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, host)
}

// Delete deletes an SSH host
func (h *SSHHostHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	result := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.SSHHost{})
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to delete host")
		return
	}
	if result.RowsAffected == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "host not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"message": "host deleted successfully",
	})
}
