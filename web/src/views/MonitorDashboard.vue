<template>
  <div style="padding: 24px">
    <a-alert
      v-if="!connected"
      message="Disconnected"
      description="Connection lost. Reconnecting..."
      type="error"
      show-icon
      style="margin-bottom: 24px"
    />

    <a-row :gutter="[24, 24]">
      <a-col :xs="24" :sm="12" :md="8" :lg="6" v-for="host in sortedHosts" :key="host.host_id">
        <a-card hoverable class="monitor-card" :class="{ offline: isOffline(host) }">
          <template #title>
            <a-space>
              <component :is="getOsIcon(host.os)" :style="{ fontSize: '20px' }" />
              <span>{{ getHostName(host.host_id) }}</span>
              <span style="color: #8c8c8c; font-size: 12px">({{ host.hostname }})</span>
            </a-space>
          </template>
          
          <div class="card-content">
            <!-- OS & Uptime -->
            <div style="margin-bottom: 16px; font-size: 12px; color: #8c8c8c">
              <div style="display: flex; align-items: center; gap: 8px">
                <OSIcon :os="host.os" />
                <span>{{ host.os || 'Linux' }}</span>
              </div>
              <div style="margin-top: 4px">
                Uptime: {{ formatUptime(host.uptime) }}
              </div>
            </div>

            <!-- CPU -->
            <div style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>CPU</span>
                <span>{{ host.cpu }}%</span>
              </div>
              <a-progress :percent="host.cpu" :status="getStatus(host.cpu)" :show-info="false" stroke-linecap="square" />
            </div>

            <!-- RAM -->
            <div style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>RAM</span>
                <span>{{ formatPct(host.mem_used, host.mem_total) }}%</span>
              </div>
              <a-progress :percent="calcPct(host.mem_used, host.mem_total)" :status="getStatus(calcPct(host.mem_used, host.mem_total))" :show-info="false" stroke-linecap="square" />
              <div style="font-size: 10px; color: #bfbfbf; text-align: right">
                {{ formatBytes(host.mem_used) }} / {{ formatBytes(host.mem_total) }}
              </div>
            </div>

            <!-- Disk -->
             <div style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>Disk (/)</span>
                <span>{{ formatPct(host.disk_used, host.disk_total) }}%</span>
              </div>
              <a-progress :percent="calcPct(host.disk_used, host.disk_total)" :show-info="false" stroke-linecap="square" />
            </div>

            <!-- Network -->
            <div style="margin-top: 16px; display: flex; justify-content: space-between; font-size: 12px">
              <div style="text-align: center">
                <div style="color: #52c41a"><ArrowDownOutlined /> {{ formatSpeed(host.net_rx_rate || 0) }}</div>
                <div style="color: #8c8c8c">Total: {{ formatBytes(host.net_rx) }}</div>
              </div>
              <div style="text-align: center">
                <div style="color: #1890ff"><ArrowUpOutlined /> {{ formatSpeed(host.net_tx_rate || 0) }}</div>
                <div style="color: #8c8c8c">Total: {{ formatBytes(host.net_tx) }}</div>
              </div>
            </div>
          </div>
        </a-card>
      </a-col>

      <a-col :span="24" v-if="hosts.length === 0">
        <a-empty description="No monitored hosts found" />
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, h } from 'vue'
import { useSSHStore } from '../stores/ssh'
import { ArrowDownOutlined, ArrowUpOutlined, AppleOutlined, WindowsOutlined, DesktopOutlined } from '@ant-design/icons-vue'

const sshStore = useSSHStore()

const hosts = ref([])
const connected = ref(true)
const socket = ref(null)

const getHostName = (hostId) => {
  const host = sshStore.hosts.find(h => h.id === hostId)
  return host ? host.name : 'Unknown Host'
}

const getOsIcon = (os) => {
  os = (os || '').toLowerCase()
  if (os.includes('win')) return WindowsOutlined
  if (os.includes('mac') || os.includes('darwin')) return AppleOutlined
  return DesktopOutlined
}

onMounted(() => {
  sshStore.loadHosts()
  connect()
})
// Mock for OSIcon component
const OSIcon = (props) => {
  const os = (props.os || '').toLowerCase()
  if (os.includes('ubuntu') || os.includes('debian') || os.includes('centos') || os.includes('linux')) return h(DesktopOutlined)
  if (os.includes('darwin') || os.includes('mac')) return h(AppleOutlined)
  if (os.includes('win')) return h(WindowsOutlined)
  return h(DesktopOutlined)
}

const sortedHosts = computed(() => {
  return [...hosts.value].sort((a, b) => b.host_id - a.host_id)
})

const isOffline = (host) => {
  const now = Date.now() / 1000
  return (now - host.last_updated) > 15
}

const getStatus = (pct) => {
  if (pct >= 90) return 'exception'
  if (pct >= 80) return 'active' // Orange-ish in some themes, or use custom color
  return 'normal' // Blue/Green
}

const calcPct = (used, total) => {
  if (!total) return 0
  return Math.round((used / total) * 100)
}

const formatPct = (used, total) => calcPct(used, total)

const formatUptime = (seconds) => {
  const dys = Math.floor(seconds / 86400)
  const hrs = Math.floor((seconds % 86400) / 3600)
  const min = Math.floor((seconds % 3600) / 60)
  if (dys > 0) return `${dys}d ${hrs}h`
  if (hrs > 0) return `${hrs}h ${min}m`
  return `${min}m`
}

const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const formatSpeed = (bytesPerSec) => {
  return formatBytes(bytesPerSec) + '/s'
}

const connect = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  // Use current auth token if needed, but the backend Stream handler didn't check generic JWT in the WS upgrade 
  // (It used router group middleware?). Yes, router group has AuthMiddleware. 
  // Standard Browser WebSocket API doesn't support custom headers easily.
  // We usually pass token in query param or cookie. 
  // Note: The backend route /api/monitor/stream requires JWT. 
  // We need to pass ?token=... or use cookie. 
  // Let's try to grab token from storage.
  const token = localStorage.getItem('token')
  const wsUrl = `${protocol}//${window.location.host}/api/monitor/stream?token=${token}`
  
  socket.value = new WebSocket(wsUrl)

  socket.value.onopen = () => {
    connected.value = true
  }

  socket.value.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'init') {
        // Calculate rates? Init doesn't have rates usually unless backend sends them
        hosts.value = msg.data.map(enrichHost)
      } else if (msg.type === 'update') {
        updateHosts(msg.data)
      } else if (msg.type === 'remove') {
        removeHost(msg.data)
      }
    } catch (e) {
      console.error(e)
    }
  }

  socket.value.onclose = () => {
    connected.value = false
    setTimeout(connect, 3000)
  }
}

// Store previous state to calculate rates if backend doesn't (Backend partially does, but let's be safe)
// Actually Backend sends 'net_rx_rate' in struct but logic was "Calculate rates if previous data exists".
// Let's trust backend sends rates.

const enrichHost = (data) => {
  // Add derived fields or temporary state if needed
  return data
}

const updateHosts = (updates) => {
  updates.forEach(update => {
    const index = hosts.value.findIndex(h => h.host_id === update.host_id)
    if (index !== -1) {
      hosts.value[index] = { ...hosts.value[index], ...update }
    } else {
      hosts.value.push(enrichHost(update))
    }
  })
}

const removeHost = (hostId) => {
  hosts.value = hosts.value.filter(h => h.host_id !== hostId)
}

onMounted(() => {
  connect()
})

onUnmounted(() => {
  if (socket.value) socket.value.close()
})
</script>

<style scoped>
.monitor-card {
  transition: all 0.3s;
  height: 100%;
}
.monitor-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}
.card-content {
  display: flex;
  flex-direction: column;
}
.offline {
  filter: grayscale(100%);
  opacity: 0.7;
}
</style>
