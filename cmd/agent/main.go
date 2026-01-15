package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// InterfaceData holds per-interface metrics
type InterfaceData struct {
	Name string `json:"name"`
	Rx   uint64 `json:"rx"`
	Tx   uint64 `json:"tx"`
}

// MetricData matches the webssh backend struct
type MetricData struct {
	HostID     uint64          `json:"host_id"`
	Uptime     uint64          `json:"uptime"`
	CPU        float64         `json:"cpu"`
	MemUsed    uint64          `json:"mem_used"`
	MemTotal   uint64          `json:"mem_total"`
	DiskUsed   uint64          `json:"disk_used"`
	DiskTotal  uint64          `json:"disk_total"`
	NetRx      uint64          `json:"net_rx"` // Sum of all interfaces
	NetTx      uint64          `json:"net_tx"` // Sum of all interfaces
	Interfaces []InterfaceData `json:"interfaces"`
	OS         string          `json:"os"`
	Hostname   string          `json:"hostname"`
}

var (
	serverURL string
	secret    string
	hostID    uint64

	// Cache static info
	cachedOS       string
	cachedHostname string

	// Previous CPU stats for calculation
	prevIdle  uint64
	prevTotal uint64
)

func main() {
	flag.StringVar(&serverURL, "server", "", "Server URL (e.g. http://localhost:8080)")
	flag.StringVar(&secret, "secret", "", "Monitor Secret")
	flag.Uint64Var(&hostID, "id", 0, "Host ID")
	flag.Parse()

	if serverURL == "" || secret == "" || hostID == 0 {
		log.Fatal("Usage: agent -server <url> -secret <secret> -id <host_id>")
	}

	// Initialize basic info
	initSystemInfo()

	log.Printf("Agent started for Host %d. Target: %s", hostID, serverURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for {
		metrics := collectMetrics()
		if err := sendMetrics(client, metrics); err != nil {
			log.Printf("Failed to report metrics: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func initSystemInfo() {
	cachedHostname, _ = os.Hostname()

	// Read OS release
	if content, err := ioutil.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				cachedOS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}
	if cachedOS == "" {
		cachedOS = runtime.GOOS
	}
}

func collectMetrics() MetricData {
	data := MetricData{
		HostID:   hostID,
		OS:       cachedOS,
		Hostname: cachedHostname,
	}

	// Uptime
	if uptimeBytes, err := ioutil.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(uptimeBytes))
		if len(parts) > 0 {
			if val, err := strconv.ParseFloat(parts[0], 64); err == nil {
				data.Uptime = uint64(val)
			}
		}
	}

	// Memory
	getMemory(&data)

	// CPU
	getCPU(&data)

	// Disk
	getDisk(&data)

	// Network
	getNetwork(&data)

	return data
}

func getMemory(data *MetricData) {
	content, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}

	var memFree, memBuffers, memCached uint64
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		val *= 1024 // KB to Bytes

		switch key {
		case "MemTotal":
			data.MemTotal = val
		case "MemFree":
			memFree = val
		case "Buffers":
			memBuffers = val
		case "Cached":
			memCached = val
		}
	}

	// Used = Total - Free - Buffers - Cached
	if data.MemTotal > 0 {
		data.MemUsed = data.MemTotal - memFree - memBuffers - memCached
	}
}

func getCPU(data *MetricData) {
	content, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			parts := strings.Fields(line)
			if len(parts) < 5 {
				continue
			}

			// user nice system idle iowait irq softirq steal
			var total uint64
			var idle uint64

			for i, v := range parts[1:] {
				val, _ := strconv.ParseUint(v, 10, 64)
				total += val
				if i == 3 { // idle is the 4th field (index 3)
					idle = val
				}
			}

			if prevTotal > 0 {
				diffTotal := total - prevTotal
				diffIdle := idle - prevIdle

				if diffTotal > 0 {
					usage := float64(diffTotal-diffIdle) / float64(diffTotal) * 100
					data.CPU = float64(int(usage*10)) / 10 // Round to 1 decimal
				}
			}

			prevTotal = total
			prevIdle = idle
			break
		}
	}
}

func getDisk(data *MetricData) {
	// Using df command is widely compatible
	cmd := exec.Command("df", "-B1", "/")
	out, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return
	}

	// Filesystem 1K-blocks Used Available Use% Mounted on
	fields := strings.Fields(lines[1])
	if len(fields) >= 3 {
		data.DiskTotal, _ = strconv.ParseUint(fields[1], 10, 64)
		data.DiskUsed, _ = strconv.ParseUint(fields[2], 10, 64)
	}
}

func getNetwork(data *MetricData) {
	content, err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		return
	}

	data.NetRx = 0 // Reset total Rx/Tx before recalculating
	data.NetTx = 0
	data.Interfaces = []InterfaceData{}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			// Clean up line: Replace first colon with space to handle "eth0:123" vs "eth0: 123"
			cleanLine := strings.Replace(line, ":", " ", 1)
			fields := strings.Fields(cleanLine)

			// fields[0] is name, fields[1] is rx_bytes, fields[9] is tx_bytes
			if len(fields) >= 10 {
				name := fields[0]
				rx, _ := strconv.ParseUint(fields[1], 10, 64)
				tx, _ := strconv.ParseUint(fields[9], 10, 64)

				if name != "lo" {
					data.NetRx += rx
					data.NetTx += tx
				}

				data.Interfaces = append(data.Interfaces, InterfaceData{
					Name: name,
					Rx:   rx,
					Tx:   tx,
				})
			}
		}
	}
}

func sendMetrics(client *http.Client, data MetricData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", serverURL+"/api/monitor/pulse", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
