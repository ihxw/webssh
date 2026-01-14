package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ihxw/webssh/internal/config"
	"github.com/ihxw/webssh/internal/middleware"
	"github.com/ihxw/webssh/internal/utils"
	"gorm.io/gorm"
)

type SystemHandler struct {
	db     *gorm.DB
	config *config.Config
}

func NewSystemHandler(db *gorm.DB, cfg *config.Config) *SystemHandler {
	return &SystemHandler{
		db:     db,
		config: cfg,
	}
}

// Backup handles database backup download
func (h *SystemHandler) Backup(c *gin.Context) {
	dbPath := h.config.Database.Path

	// Create a temporary backup file to avoid locking the main DB during download
	tmpBackup := filepath.Join(os.TempDir(), fmt.Sprintf("webssh_backup_%d.db", time.Now().Unix()))

	// Use SQLite's VACUUM INTO for a consistent backup
	err := h.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpBackup)).Error
	if err != nil {
		// Fallback to simple file copy if VACUUM INTO fails (e.g. older SQLite)
		err = copyFile(dbPath, tmpBackup)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to create backup: "+err.Error())
			return
		}
	}
	defer os.Remove(tmpBackup)

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=webssh_backup_%s.db", time.Now().Format("20060102_150405")))
	c.Header("Content-Type", "application/octet-stream")
	c.File(tmpBackup)
}

// Restore handles database restoration from uploaded file
func (h *SystemHandler) Restore(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "no file uploaded")
		return
	}

	// Basic validation: check file extension
	if filepath.Ext(file.Filename) != ".db" {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid file type, must be .db")
		return
	}

	// Save uploaded file to temporary location
	tmpFile := filepath.Join(os.TempDir(), "webssh_restore.db")
	if err := c.SaveUploadedFile(file, tmpFile); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}
	defer os.Remove(tmpFile)

	// Close current DB connections before replacing the file
	sqlDB, err := h.db.DB()
	if err == nil {
		sqlDB.Close()
	}

	// Replace the current database file
	dbPath := h.config.Database.Path
	if err := copyFile(tmpFile, dbPath); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to restore database file: "+err.Error())
		return
	}

	// The server should ideally be restarted here, but for now we'll return success
	// and advise the user that a restart might be needed if they encounter issues,
	// though GORM might re-open connections on the next request.
	// A better way is to signal the main process to re-init the DB.

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"message": "database restored successfully, server should re-connect on next request or may require manual restart",
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// GetSettings returns current editable settings
func (h *SystemHandler) GetSettings(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"ssh_timeout":              h.config.SSH.Timeout,
		"idle_timeout":             h.config.SSH.IdleTimeout,
		"max_connections_per_user": h.config.SSH.MaxConnectionsPerUser,
		"login_rate_limit":         h.config.Security.LoginRateLimit,
	})
}

// UpdateSettingsRequest defines the request body for updating settings
type UpdateSettingsRequest struct {
	SSHTimeout            string `json:"ssh_timeout" binding:"required"`
	IdleTimeout           string `json:"idle_timeout" binding:"required"`
	MaxConnectionsPerUser int    `json:"max_connections_per_user" binding:"required"`
	LoginRateLimit        int    `json:"login_rate_limit" binding:"required"`
}

// Global rate limiter reference for dynamic updates
var LoginRateLimiter *middleware.RateLimiter

// UpdateSettings updates configuration and persists to file
func (h *SystemHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Update in-memory config
	h.config.SSH.Timeout = req.SSHTimeout
	h.config.SSH.IdleTimeout = req.IdleTimeout
	h.config.SSH.MaxConnectionsPerUser = req.MaxConnectionsPerUser
	h.config.Security.LoginRateLimit = req.LoginRateLimit

	// Save to file
	if err := h.config.SaveConfig(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to save configuration: "+err.Error())
		return
	}

	// Hot-reload rate limit if global limiter is set
	if LoginRateLimiter != nil {
		LoginRateLimiter.SetLimit(req.LoginRateLimit)
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"message": "settings updated successfully",
	})
}
