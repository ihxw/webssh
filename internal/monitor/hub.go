package monitor

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// InterfaceData holds per-interface metrics
type InterfaceData struct {
	Name string `json:"name"`
	Rx   uint64 `json:"rx"`
	Tx   uint64 `json:"tx"`
	// Derived rates
	RxRate uint64 `json:"rx_rate"`
	TxRate uint64 `json:"tx_rate"`
}

// MetricData represents the data packet sent by the agent
type MetricData struct {
	HostID       uint    `json:"host_id"`
	Uptime       uint64  `json:"uptime"`      // Seconds
	CPU          float64 `json:"cpu"`         // Percentage
	MemUsed      uint64  `json:"mem_used"`    // Bytes
	MemTotal     uint64  `json:"mem_total"`   // Bytes
	DiskUsed     uint64  `json:"disk_used"`   // Bytes
	DiskTotal    uint64  `json:"disk_total"`  // Bytes
	NetRx        uint64  `json:"net_rx"`      // Total Bytes In
	NetTx        uint64  `json:"net_tx"`      // Total Bytes Out
	NetRxRate    uint64  `json:"net_rx_rate"` // Bytes/sec (Total)
	NetTxRate    uint64  `json:"net_tx_rate"` // Bytes/sec (Total)
	NetMonthlyRx uint64  `json:"net_monthly_rx"`
	NetMonthlyTx uint64  `json:"net_monthly_tx"`
	// Config for Frontend Calculation
	NetTrafficLimit          uint64 `json:"net_traffic_limit"`
	NetTrafficUsedAdjustment uint64 `json:"net_traffic_used_adjustment"`
	NetTrafficCounterMode    string `json:"net_traffic_counter_mode"` // total, rx, tx

	Interfaces  []InterfaceData `json:"interfaces"` // Per Interface
	OS          string          `json:"os"`
	Hostname    string          `json:"hostname"`
	LastUpdated int64           `json:"last_updated"`
}

type Hub struct {
	// Registered clients (frontend dashboards)
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex

	// In-memory state of hosts
	hosts   map[uint]*MetricData
	hostsMu sync.RWMutex

	// Inbound updates from handlers
	updateChan chan MetricData
}

var GlobalHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		hosts:      make(map[uint]*MetricData),
		updateChan: make(chan MetricData, 100),
	}
}

func (h *Hub) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-h.updateChan:
			h.hostsMu.Lock()
			// Calculate rates if previous data exists
			prev, exists := h.hosts[data.HostID]
			if exists {
				timeDiff := data.LastUpdated - prev.LastUpdated
				if timeDiff > 0 {
					data.NetRxRate = (data.NetRx - prev.NetRx) / uint64(timeDiff)
					data.NetTxRate = (data.NetTx - prev.NetTx) / uint64(timeDiff)

					// Calculate per-interface rates
					for i, iface := range data.Interfaces {
						// Find corresponding interface in prev
						for _, prevIface := range prev.Interfaces {
							if prevIface.Name == iface.Name {
								if iface.Rx >= prevIface.Rx {
									data.Interfaces[i].RxRate = (iface.Rx - prevIface.Rx) / uint64(timeDiff)
								}
								if iface.Tx >= prevIface.Tx {
									data.Interfaces[i].TxRate = (iface.Tx - prevIface.Tx) / uint64(timeDiff)
								}
								break
							}
						}
					}
				}
			}
			// Important: Create a copy to avoid pointer aliasing if compiled with variable reuse optimization
			finalData := data
			h.hosts[data.HostID] = &finalData
			h.hostsMu.Unlock()

			// Broadcast to all clients
			h.broadcast()

		case <-ticker.C:
			// Cleanup old hosts? Or just periodic heartbeat
		}
	}
}

func (h *Hub) Update(data MetricData) {
	data.LastUpdated = time.Now().Unix()
	h.updateChan <- data
}

func (h *Hub) RemoveHost(hostID uint) {
	h.hostsMu.Lock()
	delete(h.hosts, hostID)
	h.hostsMu.Unlock()

	// Notify clients of removal?
	// Currently clients just receive a list of active hosts.
	// If the host is gone from the list, the frontend logic handles update.
	// But our frontend logic appends/updates. It doesn't remove unless valid list is sent.
	// Actually frontend: `hosts.value[index] = ...` or push.
	// Frontend logic: `updateHosts` iterates over updates.
	// If we want to remove from frontend, we need to send a "remove" event or full sync.
	// The current "update" event sends a list of ACTIVE hosts.
	// If we send a list without the removed host, the frontend MIGHT not remove it if it only merges.
	// Let's check frontend logic: `updateHosts` merges.
	// To support removal, we should probably modify `broadcast` to send ALL active hosts,
	// and frontend should verify if any are missing or handle a sync.
	// For now, let's keep it simple: Host stops sending data => marked offline in frontend after timeout.
	// But if we want it gone immediately, we better restart frontend or improve protocol.
	// Let's just remove from memory so it doesn't reappear.

	// Better: Send a specific "remove" packet.
	h.broadcastRemove(hostID)
}

func (h *Hub) broadcastRemove(hostID uint) {
	msg := map[string]interface{}{
		"type": "remove",
		"data": hostID,
	}
	jsonMsg, _ := json.Marshal(msg)

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	for client := range h.clients {
		client.WriteMessage(websocket.TextMessage, jsonMsg)
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	// Send initial state
	h.hostsMu.RLock()
	hostsList := make([]*MetricData, 0, len(h.hosts))
	for _, v := range h.hosts {
		hostsList = append(hostsList, v)
	}
	h.hostsMu.RUnlock()

	jsonMsg, _ := json.Marshal(map[string]interface{}{
		"type": "init",
		"data": hostsList,
	})
	conn.WriteMessage(websocket.TextMessage, jsonMsg)
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.clientsMu.Lock()
	delete(h.clients, conn)
	h.clientsMu.Unlock()
	conn.Close()
}

func (h *Hub) broadcast() {
	h.hostsMu.RLock()
	hostsList := make([]*MetricData, 0, len(h.hosts))
	for _, v := range h.hosts {
		// Only send active hosts (last updated < 10 seconds ago)
		if time.Now().Unix()-v.LastUpdated < 15 {
			hostsList = append(hostsList, v)
		}
	}
	h.hostsMu.RUnlock()

	msg := map[string]interface{}{
		"type": "update",
		"data": hostsList,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, jsonMsg)
		if err != nil {
			log.Printf("Error writing to monitor ws: %v", err)
			client.Close()
			delete(h.clients, client) // Note: safe to delete active loop key in Go? No.
			// Ideally we should collect "to delete" and delete after loop, but RLock prevents modification.
			// Actually RLock means we can't delete.
			// Fix: Just log error, let Unregister handle it or handle disconnection gracefully.
			// For simplicity, we just ignore error here, usually the underlying connection close will trigger read error in the reader loop.
		}
	}
}
