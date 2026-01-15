package handlers

import (
	"fmt"
	"net"
	"net/http"
	"time"

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
	// Network Config
	NetInterface string `json:"net_interface"`
	NetResetDay  int    `json:"net_reset_day"`
	// Traffic Limit Config
	NetTrafficLimit          uint64 `json:"net_traffic_limit"`
	NetTrafficUsedAdjustment uint64 `json:"net_traffic_used_adjustment"`
	NetTrafficCounterMode    string `json:"net_traffic_counter_mode"` // total, rx, tx
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
	// Network Config
	if req.NetInterface != "" {
		// If interface selection changed, we MUST reset LastRaw to avoid massive delta spikes
		// (false positive reboot or huge jump)
		if host.NetInterface != req.NetInterface {
			host.NetLastRawRx = 0
			host.NetLastRawTx = 0

			// Optional: Should we also reset Monthly?
			// No, user might just be refining selection.
			// But the accumulated data might be mixed.
			// Ideally we keep monthly but stop the spike.
		}
		host.NetInterface = req.NetInterface
	}
	if req.NetResetDay > 0 && req.NetResetDay <= 31 {
		host.NetResetDay = req.NetResetDay
	}

	// Limit config (0 is valid for Limit/Adjustment, so check presence? JSON unmarshal defaults to 0.
	// Since 0 is meaningful (unlimited or reset adjustment), we might need pointer or just overwrite.
	// For simplicity, we overwrite if present in struct (JSON 0 overwrites).
	// Actually ShouldBindJSON overwrites with 0 if missing? No, 0 is default.
	// But Update logic usually checks for non-zero.
	// To support setting to 0, we can't just check != 0.
	// However, for typical update flow, we send all fields or patch.
	// Our `Update` handler checks `if req.Field != ""`.
	// For numeric fields like NetTrafficLimit, we can't distinguish 0 vs missing.
	// BUT, our current frontend will send valid values.
	// Let's assume if it is part of the request we want to update it.
	// But Go structs don't show "missing".
	// Best practice: Use pointers in struct for optional updates, OR since this is a dedicated config form update,
	// we assume the user intends to set the value.
	// BUT, this handler is general purpose.
	// A simple workaround for this specific "Config Form" usage:
	// If `NetTrafficCounterMode` is provided (non-empty), we assume it's a Limit config update
	// and update the numeric fields too (even if 0).
	// Or we just update them.

	if req.NetTrafficLimit > 0 || req.NetTrafficUsedAdjustment > 0 || req.NetTrafficCounterMode != "" {
		host.NetTrafficLimit = req.NetTrafficLimit
		host.NetTrafficUsedAdjustment = req.NetTrafficUsedAdjustment
		if req.NetTrafficCounterMode != "" {
			host.NetTrafficCounterMode = req.NetTrafficCounterMode
		}
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

// TestConnection tests the connectivity to the SSH host
func (h *SSHHostHandler) TestConnection(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var host models.SSHHost
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "host not found")
		return
	}

	target := fmt.Sprintf("%s:%d", host.Host, host.Port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "offline",
			"latency": 0,
			"error":   err.Error(),
		})
		return
	}
	defer conn.Close()

	c.JSON(http.StatusOK, gin.H{
		"status":  "online",
		"latency": duration.Milliseconds(),
	})
}
