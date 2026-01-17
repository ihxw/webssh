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
          <template #extra>
             <a-tooltip title="Network Details">
                <a-button type="text" shape="circle" @click="$router.push({ name: 'NetworkDetail', params: { id: host.host_id } })">
                    <template #icon><LineChartOutlined /></template>
                </a-button>
             </a-tooltip>
          </template>
          
          <div class="card-content">
            <!-- OS & Uptime -->
            <div style="margin-bottom: 8px; font-size: 12px; color: #8c8c8c">
              <div style="display: flex; align-items: center; gap: 8px">
                <OSIcon :os="host.os" />
                <span>{{ host.os || 'Linux' }}</span>
              </div>
              <div style="margin-top: 4px">
                {{ t('monitor.uptime') }}: {{ formatUptime(host.uptime) }}
              </div>
            </div>

            <!-- CPU -->
            <div style="margin-bottom: 8px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>{{ t('monitor.cpu') }}</span>
                <span>{{ formatCpu(host.cpu) }}%</span>
              </div>
              <a-progress :percent="host.cpu" :status="getStatus(host.cpu)" :show-info="false" stroke-linecap="square" />
            </div>

            <!-- RAM -->
            <div style="margin-bottom: 8px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>{{ t('monitor.ram') }}</span>
                <span>{{ formatPct(host.mem_used, host.mem_total) }}%</span>
              </div>
              <a-progress :percent="calcPct(host.mem_used, host.mem_total)" :status="getStatus(calcPct(host.mem_used, host.mem_total))" :show-info="false" stroke-linecap="square" />
              <div style="font-size: 10px; color: #bfbfbf; text-align: right">
                {{ formatBytes(host.mem_used) }} / {{ formatBytes(host.mem_total) }}
              </div>
            </div>

            <!-- Disk -->
             <div style="margin-bottom: 8px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>{{ t('monitor.disk') }} (/)</span>
                <span>{{ formatPct(host.disk_used, host.disk_total) }}%</span>
              </div>
              <a-progress :percent="calcPct(host.disk_used, host.disk_total)" :show-info="false" stroke-linecap="square" />
              <div style="font-size: 10px; color: #bfbfbf; text-align: right">
                {{ formatBytes(host.disk_used) }} / {{ formatBytes(host.disk_total) }}
              </div>
            </div>

            <!-- Network -->
            <div style="margin-top: 8px; display: flex; justify-content: space-between; font-size: 12px">
              <div style="text-align: center">
                <div style="color: #52c41a"><ArrowDownOutlined /> {{ formatSpeed(host.net_rx_rate || 0) }}</div>
                <div style="color: #8c8c8c">{{ t('monitor.total') }}: {{ formatBytes(host.net_rx) }}</div>
              </div>
              <div style="text-align: center">
                <div style="color: #1890ff"><ArrowUpOutlined /> {{ formatSpeed(host.net_tx_rate || 0) }}</div>
                <div style="color: #8c8c8c">{{ t('monitor.total') }}: {{ formatBytes(host.net_tx) }}</div>
              </div>
            </div>
            
            <!-- Traffic Usage (If Limit Set) -->
            <div v-if="host.net_traffic_limit > 0" style="margin-top: 8px; border-top: 1px solid #f0f0f0; padding-top: 8px">
               <div style="display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 2px">
                  <span>{{ t('network.usage') }} ({{ getTrafficUsagePct(host) }}%)</span>
                  <span>{{ formatTrafficUsage(host) }}</span>
               </div>
               <a-progress :percent="getTrafficUsagePct(host)" :status="getStatus(getTrafficUsagePct(host))" :show-info="false" stroke-linecap="square" size="small" />
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
import { ArrowDownOutlined, ArrowUpOutlined, AppleOutlined, WindowsOutlined, DesktopOutlined, LineChartOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
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

const syncHostsFromStore = () => {
  const storeHosts = sshStore.hosts.filter(h => h.monitor_enabled)
  
  // 1. Add new hosts or update existing static info
  storeHosts.forEach(sh => {
    const customId = sh.id
    const existing = hosts.value.findIndex(h => h.host_id === customId)
    
    if (existing !== -1) {
      // Update static info
      hosts.value[existing].hostname = sh.host
      // hosts.value[existing].name = sh.name // Using getHostName helper in template anyway
    } else {
      // Add new host with default/empty metrics
      hosts.value.push({
        host_id: sh.id,
        hostname: sh.host,
        os: '',
        uptime: 0,
        cpu: 0,
        mem_used: 0,
        mem_total: 0,
        disk_used: 0,
        disk_total: 0,
        net_rx: 0,
        net_tx: 0,
        last_updated: 0
      })
    }
  })
  
  // 2. Remove hosts that are no longer in store or disabled
  hosts.value = hosts.value.filter(h => {
    return storeHosts.find(sh => sh.id === h.host_id)
  })
}

// Watch for store changes
watch(() => sshStore.hosts, () => {
    syncHostsFromStore()
}, { deep: true })

onMounted(() => {
  sshStore.fetchHosts().then(() => {
      syncHostsFromStore()
  })
  connect()
})

// ... OSIcon ...

const sortedHosts = computed(() => {
    // Sort: Online first, then by ID
    return [...hosts.value].sort((a, b) => {
        const aOffline = isOffline(a)
        const bOffline = isOffline(b)
        if (aOffline === bOffline) return b.host_id - a.host_id
        return aOffline ? 1 : -1
    })
})

// ...

const connect = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
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
        updateHosts(msg.data)
      } else if (msg.type === 'update') {
        updateHosts(msg.data)
      } else if (msg.type === 'remove') {
        // removeHost(msg.data)
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

const enrichHost = (data) => {
  return data
}

const updateHosts = (updates) => {
  if (!updates) return
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
