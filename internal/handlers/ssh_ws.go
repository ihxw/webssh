package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ihxw/webssh/internal/config"
	"github.com/ihxw/webssh/internal/models"
	"github.com/ihxw/webssh/internal/ssh"
	"github.com/ihxw/webssh/internal/utils"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type SSHWebSocketHandler struct {
	db     *gorm.DB
	config *config.Config
}

func NewSSHWebSocketHandler(db *gorm.DB, cfg *config.Config) *SSHWebSocketHandler {
	return &SSHWebSocketHandler{
		db:     db,
		config: cfg,
	}
}

type WSMessage struct {
	Type string      `json:"type"` // input, resize
	Data interface{} `json:"data"`
}

type ResizeData struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// HandleWebSocket handles WebSocket connections for SSH
func (h *SSHWebSocketHandler) HandleWebSocket(c *gin.Context) {
	ticketID := c.Query("ticket")
	ticket, ok := utils.ValidateTicket(ticketID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired ticket")
		return
	}

	userID := ticket.UserID
	hostID := c.Param("hostId")

	// Get SSH host from database
	var host models.SSHHost
	if err := h.db.Where("id = ? AND user_id = ?", hostID, userID).First(&host).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "host not found")
		return
	}

	// Decrypt credentials
	var password, privateKey string
	if host.PasswordEncrypted != "" {
		decrypted, err := utils.DecryptAES(host.PasswordEncrypted, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to decrypt password")
			return
		}
		password = decrypted
	}
	if host.PrivateKeyEncrypted != "" {
		decrypted, err := utils.DecryptAES(host.PrivateKeyEncrypted, h.config.Security.EncryptionKey)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to decrypt private key")
			return
		}
		privateKey = decrypted
	}

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer ws.Close()

	// Parse timeout
	timeout, err := time.ParseDuration(h.config.SSH.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create SSH client
	sshClient, err := ssh.NewSSHClient(&ssh.SSHConfig{
		Host:       host.Host,
		Port:       host.Port,
		Username:   host.Username,
		Password:   password,
		PrivateKey: privateKey,
		Timeout:    timeout,
	})
	if err != nil {
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to create SSH client: " + err.Error()})
		return
	}
	defer sshClient.Close()

	// Create connection log
	connLog := &models.ConnectionLog{
		UserID:      userID,
		SSHHostID:   &host.ID,
		Host:        host.Host,
		Port:        host.Port,
		Username:    host.Username,
		Status:      "connecting",
		ConnectedAt: time.Now(),
	}
	h.db.Create(connLog)

	// Connect to SSH server
	if err := sshClient.Connect(); err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to connect: " + err.Error()})
		return
	}

	// Create session
	if err := sshClient.NewSession(); err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to create session: " + err.Error()})
		return
	}

	session := sshClient.GetSession()

	// Request PTY
	if err := sshClient.RequestPTY("xterm-256color", 24, 80); err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to request PTY: " + err.Error()})
		return
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to get stdin pipe: " + err.Error()})
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to get stdout pipe: " + err.Error()})
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to get stderr pipe: " + err.Error()})
		return
	}

	// Start shell
	if err := sshClient.Shell(); err != nil {
		connLog.Status = "failed"
		connLog.ErrorMessage = err.Error()
		h.db.Save(connLog)
		ws.WriteJSON(gin.H{"type": "error", "data": "Failed to start shell: " + err.Error()})
		return
	}

	// Update connection log
	connLog.Status = "success"
	h.db.Save(connLog)

	// Send success message
	ws.WriteJSON(gin.H{"type": "connected", "data": "Connected successfully"})

	// Channel to signal completion
	done := make(chan bool)

	// Handle recording
	record := c.Query("record") == "true"
	var recording *models.TerminalRecording
	var recordFile *os.File
	if record {
		recordingDir := "data/recordings"
		os.MkdirAll(recordingDir, 0755)

		fileName := fmt.Sprintf("%d-%d-%d.cast", userID, host.ID, time.Now().Unix())
		filePath := filepath.Join(recordingDir, fileName)

		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			recordFile = f
			recording = &models.TerminalRecording{
				UserID:    userID,
				SSHHostID: host.ID,
				Host:      host.Host,
				Username:  host.Username,
				FilePath:  filePath,
				StartTime: time.Now(),
			}
			h.db.Create(recording)
		}
	}

	// Read from SSH stdout and send to WebSocket
	go func() {
		buf := make([]byte, 1024)
		start := time.Now()
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from stdout: %v", err)
				}
				done <- true
				return
			}
			if n > 0 {
				data := buf[:n]
				if recordFile != nil {
					// Store as [time_offset, "o", "data"]
					offset := time.Since(start).Seconds()
					entry := fmt.Sprintf("[%f, \"o\", %q]\n", offset, string(data))
					recordFile.WriteString(entry)
				}
				if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("Error writing to WebSocket: %v", err)
					done <- true
					return
				}
			}
		}
	}()

	// Read from SSH stderr and send to WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from stderr: %v", err)
				}
				return
			}
			if n > 0 {
				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					log.Printf("Error writing to WebSocket: %v", err)
					return
				}
			}
		}
	}()

	// Read from WebSocket and send to SSH stdin
	go func() {
		for {
			messageType, message, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Error reading from WebSocket: %v", err)
				done <- true
				return
			}

			if messageType == websocket.TextMessage {
				// Try to parse as JSON message
				var wsMsg WSMessage
				if err := json.Unmarshal(message, &wsMsg); err == nil {
					// Handle structured messages
					switch wsMsg.Type {
					case "resize":
						var resizeData ResizeData
						dataBytes, _ := json.Marshal(wsMsg.Data)
						if err := json.Unmarshal(dataBytes, &resizeData); err == nil {
							sshClient.Resize(resizeData.Rows, resizeData.Cols)
						}
					case "input":
						if data, ok := wsMsg.Data.(string); ok {
							stdin.Write([]byte(data))
						}
					}
				} else {
					// Handle plain text input
					stdin.Write(message)
				}
			}
		}
	}()

	// Wait for completion
	<-done

	// Finalize recording
	if recordFile != nil {
		recordFile.Close()
		if recording != nil {
			now := time.Now()
			recording.EndTime = &now
			recording.Duration = int(now.Sub(recording.StartTime).Seconds())
			h.db.Save(recording)
		}
	}

	// Update connection log
	now := time.Now()
	connLog.DisconnectedAt = &now
	connLog.Duration = int(now.Sub(connLog.ConnectedAt).Seconds())
	connLog.Status = "disconnected"
	h.db.Save(connLog)

	log.Printf("SSH session closed for user %d, host %s", userID, host.Host)
}
