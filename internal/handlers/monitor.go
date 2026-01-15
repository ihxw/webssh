package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ihxw/webssh/internal/config"
	"github.com/ihxw/webssh/internal/models"
	"github.com/ihxw/webssh/internal/monitor"
	"github.com/ihxw/webssh/internal/utils"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type MonitorHandler struct {
	DB         *gorm.DB
	Config     *config.Config
	lastDbSave map[uint]time.Time
	saveMu     sync.Mutex
}

func NewMonitorHandler(db *gorm.DB, cfg *config.Config) *MonitorHandler {
	// Start the hub
	go monitor.GlobalHub.Run()
	return &MonitorHandler{
		DB:         db,
		Config:     cfg,
		lastDbSave: make(map[uint]time.Time),
	}
}

// Agent Script Template
const agentScriptTmpl = `#!/bin/bash
SERVER_URL="{{.ServerURL}}"
SECRET="{{.Secret}}"
HOST_ID="{{.HostID}}"

while true; do
  # Collect Metrics
  
  # Uptime (seconds)
  uptime=$(cat /proc/uptime | awk '{print $1}' | cut -d. -f1)
  
  # Load
  load=$(cat /proc/loadavg | awk '{print $1}')
  
  # CPU Usage (grep 'cpu ' /proc/stat) - Simplified calculation
  # Previous
  cpu1=$(grep 'cpu ' /proc/stat)
  prev_idle=$(echo "$cpu1" | awk '{print $5}')
  prev_total=$(echo "$cpu1" | awk '{print $2+$3+$4+$5+$6+$7+$8}')
  sleep 1
  # Current
  cpu2=$(grep 'cpu ' /proc/stat)
  idle=$(echo "$cpu2" | awk '{print $5}')
  total=$(echo "$cpu2" | awk '{print $2+$3+$4+$5+$6+$7+$8}')
  
  diff_idle=$((idle - prev_idle))
  diff_total=$((total - prev_total))
  if [ "$diff_total" -eq 0 ]; then diff_total=1; fi
  cpu_usage=$(( (100 * (diff_total - diff_idle)) / diff_total ))
  if [ "$cpu_usage" -lt 0 ]; then cpu_usage=0; fi

  # Memory (Bytes)
  mem_total=$(grep MemTotal /proc/meminfo | awk '{print $2 * 1024}')
  mem_free=$(grep MemFree /proc/meminfo | awk '{print $2 * 1024}')
  mem_buffers=$(grep Buffers /proc/meminfo | awk '{print $2 * 1024}')
  mem_cached=$(grep ^Cached /proc/meminfo | awk '{print $2 * 1024}')
  # Used = Total - Free - Buffers - Cached
  # Calculate derived values
  if [ -z "$mem_buffers" ]; then mem_buffers=0; fi
  if [ -z "$mem_cached" ]; then mem_cached=0; fi
  if [ -z "$mem_total" ]; then mem_total=0; fi
  if [ -z "$mem_free" ]; then mem_free=0; fi
  mem_used=$((mem_total - mem_free - mem_buffers - mem_cached))

  # Disk (Bytes) - Root partition
  disk_total=$(df -B1 / 2>/dev/null | tail -1 | awk '{print $2}')
  disk_used=$(df -B1 / 2>/dev/null | tail -1 | awk '{print $3}')
  if [ -z "$disk_total" ]; then disk_total=0; fi
  if [ -z "$disk_used" ]; then disk_used=0; fi

  # Network (Bytes)
  net_rx=$(cat /proc/net/dev 2>/dev/null | grep -v lo | awk '{sum+=$2} END {printf "%.0f", sum}')
  net_tx=$(cat /proc/net/dev 2>/dev/null | grep -v lo | awk '{sum+=$10} END {printf "%.0f", sum}')
  if [ -z "$net_rx" ]; then net_rx=0; fi
  if [ -z "$net_tx" ]; then net_tx=0; fi
  
  if [ -z "$uptime" ]; then uptime=0; fi
  if [ -z "$cpu_usage" ]; then cpu_usage=0; fi

  # OS Info
  if [ -f /etc/os-release ]; then
    os=$(grep PRETTY_NAME /etc/os-release | cut -d= -f2 | tr -d '"')
  else
    os=$(uname -s)
  fi
  hostname=$(hostname)

  # Check if curl exists
  if ! command -v curl &> /dev/null; then
     # Try wget? No complex logic for now
     sleep 10
     continue
  fi

  # Send Data
  JSON_DATA=$(cat <<EOF
{
  "host_id": $HOST_ID,
  "uptime": $uptime,
  "cpu": $cpu_usage,
  "mem_used": $mem_used,
  "mem_total": $mem_total,
  "disk_used": $disk_used,
  "disk_total": $disk_total,
  "net_rx": $net_rx,
  "net_tx": $net_tx,
  "os": "$os",
  "hostname": "$hostname"
}
EOF
)

  curl -s -X POST "$SERVER_URL/api/monitor/pulse" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $SECRET" \
    -d "$JSON_DATA"

  sleep 2
done
`

// Pulse receives metrics from agents
func (h *MonitorHandler) Pulse(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	secret := authHeader[7:]

	// Validate secret against generic verification (optimize: cache secrets?)
	// For now, assume if ID matches header, we verify DB.
	// But JSON body isn't parsed yet.
	var data monitor.MetricData
	if err := c.ShouldBindJSON(&data); err != nil {
		log.Printf("Monitor Pulse: Bind JSON failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify Host and Secret
	var host models.SSHHost
	if err := h.DB.Select("monitor_secret, monitor_enabled").First(&host, data.HostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Host not found"})
		return
	}

	if !host.MonitorEnabled || host.MonitorSecret != secret {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid secret or monitoring disabled"})
		return
	}

	monitor.GlobalHub.Update(data)

	// Save to DB periodically (e.g. every minute)
	h.saveMu.Lock()
	lastSave, exists := h.lastDbSave[data.HostID]
	shouldSave := !exists || time.Since(lastSave) > 1*time.Minute
	if shouldSave {
		h.lastDbSave[data.HostID] = time.Now()
	}
	h.saveMu.Unlock()

	if shouldSave {
		go func(d monitor.MetricData) {
			record := models.MonitorRecord{
				HostID:    d.HostID,
				CPU:       d.CPU,
				MemUsed:   d.MemUsed,
				MemTotal:  d.MemTotal,
				DiskUsed:  d.DiskUsed,
				DiskTotal: d.DiskTotal,
				NetRx:     d.NetRx,
				NetTx:     d.NetTx,
			}
			h.DB.Create(&record)
		}(data)
	}

	c.Status(http.StatusOK)
}

// Stream WebSocket for Dashboard
func (h *MonitorHandler) Stream(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	monitor.GlobalHub.Register(conn)

	// Keep alive loop
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			monitor.GlobalHub.Unregister(conn)
			break
		}
	}
}

// Deploy installs the agent
func (h *MonitorHandler) Deploy(c *gin.Context) {
	id := c.Param("id")
	var host models.SSHHost
	if err := h.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Host not found"})
		return
	}

	// Generate Secret
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	secret := hex.EncodeToString(randomBytes)

	// Save Secret to DB (but not enabled yet, or keep enabled false)
	// We need to save the secret so the agent can authenticate if it starts immediately
	host.MonitorSecret = secret
	// host.MonitorEnabled = false // Keep false until success
	h.DB.Save(&host)

	// Prepare Script
	// Infer Server URL from request Host if possible, or Config
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	// If using dev proxy (port 5173 -> 8080), we might need correction.
	// But usually this request comes to backend port directly or via Nginx.
	// Let's rely on c.Request.Host which is likely what the browser sees.

	// Better: Use a config setting if behind proxy. For now, best effort.

	tmpl, err := template.New("agent").Parse(agentScriptTmpl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Template error"})
		return
	}

	var scriptBuf bytes.Buffer
	if err := tmpl.Execute(&scriptBuf, map[string]interface{}{
		"ServerURL": serverURL,
		"Secret":    secret,
		"HostID":    host.ID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Template execute error"})
		return
	}

	// Connect SSH (Need decrypted keys or password)
	password, _ := utils.DecryptAES(host.PasswordEncrypted, h.Config.Security.EncryptionKey)
	privateKey, _ := utils.DecryptAES(host.PrivateKeyEncrypted, h.Config.Security.EncryptionKey)

	authMethods := []ssh.AuthMethod{}
	if host.AuthType == "key" && privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use TOFU
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Host, host.Port), sshConfig)
	if err != nil {
		log.Printf("Monitor Deploy: SSH Dial failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SSH Connection failed: %v", err)})
		return
	}
	defer client.Close()

	// 1. Upload Script
	session, _ := client.NewSession()
	defer session.Close()

	// We use cat to write file. Simple and effective for text files.
	// Ensure directory exists
	setupCmd := "mkdir -p /opt/webssh/agent"
	session.Run(setupCmd)
	session.Close()

	session, _ = client.NewSession()
	go func() {
		w, _ := session.StdinPipe()
		w.Write(scriptBuf.Bytes())
		w.Close()
	}()
	if err := session.Run("cat > /opt/webssh/agent/monitor.sh && chmod +x /opt/webssh/agent/monitor.sh"); err != nil {
		log.Printf("Monitor Deploy: Failed to upload script: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload script"})
		return
	}
	session.Close()

	// 2. Create Systemd Service
	serviceContent := `[Unit]
Description=WebSSH Monitor Agent
After=network.target

[Service]
ExecStart=/opt/webssh/agent/monitor.sh
Restart=always
User=root
WorkingDirectory=/opt/webssh/agent

[Install]
WantedBy=multi-user.target
`
	session, _ = client.NewSession()
	go func() {
		w, _ := session.StdinPipe()
		w.Write([]byte(serviceContent))
		w.Close()
	}()
	// Note: Writing to /etc/systemd requires root.
	// If user is not root, this will fail unless we use sudo.
	// For MVP, assume root or user with passwordless sudo?
	// The user provided "root" in examples.
	targetPath := "/etc/systemd/system/webssh-agent.service"
	if host.Username != "root" {
		// Try to write to /tmp and sudo mv
		targetPath = "/tmp/webssh-agent.service"
	}

	if err := session.Run(fmt.Sprintf("cat > %s", targetPath)); err != nil {
		// If failed, maybe try sudo?
		// Complex handling omitted for brevity of MVP.
		log.Printf("Failed to write service file: %v", err)
	}
	session.Close()

	if host.Username != "root" && targetPath == "/tmp/webssh-agent.service" {
		// Move with sudo
		session, _ := client.NewSession()
		session.Run("echo " + password + " | sudo -S mv /tmp/webssh-agent.service /etc/systemd/system/webssh-agent.service")
		session.Close()
	}

	// 3. Enable and Start
	session, _ = client.NewSession()
	cmd := "systemctl daemon-reload && systemctl enable --now webssh-agent"
	if host.Username != "root" {
		cmd = "echo " + password + " | sudo -S sh -c '" + cmd + "'"
	}
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Printf("Monitor Deploy: Failed to start service: %v, Output: %s", err, string(output))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start service: %v", err)})
		return
	}

	// 4. Update DB to indicate Monitoring is Enabled
	h.DB.Model(&host).Update("monitor_enabled", true)

	c.JSON(http.StatusOK, gin.H{"message": "Agent deployed successfully"})
}

func (h *MonitorHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	var host models.SSHHost
	if err := h.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Host not found"})
		return
	}

	// 1. Connect SSH to stop service
	password, _ := utils.DecryptAES(host.PasswordEncrypted, h.Config.Security.EncryptionKey)
	privateKey, _ := utils.DecryptAES(host.PrivateKeyEncrypted, h.Config.Security.EncryptionKey)

	authMethods := []ssh.AuthMethod{}
	if host.AuthType == "key" && privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Host, host.Port), sshConfig)
	if err != nil {
		// Even if connection fails, we still disable in DB?
		// Maybe warn user.
		log.Printf("Monitor Stop: SSH Dial failed: %v", err)
		// We proceed to update DB so user can at least toggle it off if host is dead
	} else {
		defer client.Close()
		session, _ := client.NewSession()
		defer session.Close()

		// Commands to stop and remove
		// systemctl disable --now webssh-agent
		// rm /etc/systemd/system/webssh-agent.service
		// systemctl daemon-reload
		// rm -rf /opt/webssh/agent

		cmd := "systemctl disable --now webssh-agent && rm -f /etc/systemd/system/webssh-agent.service && systemctl daemon-reload && rm -rf /opt/webssh/agent"
		if host.Username != "root" {
			// Sudo handling
			cmd = "echo " + password + " | sudo -S sh -c '" + cmd + "'"
		}

		if err := session.Run(cmd); err != nil {
			log.Printf("Monitor Stop: Failed to run cleanup commands: %v", err)
		}
	}

}
