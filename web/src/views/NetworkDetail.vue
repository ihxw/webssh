<template>
  <div style="padding: 24px">
    <a-page-header @back="$router.back()" title="Network Details" :sub-title="host?.name || 'Unknown Host'">
      <template #extra>
         <a-tag color="blue" v-if="connected">Connected</a-tag>
         <a-tag color="red" v-else>Disconnected</a-tag>
      </template>
    </a-page-header>
    
    <div v-if="!host" style="text-align: center; margin-top: 48px">
        <a-spin /> Loading...
    </div>

    <a-row :gutter="12" style="margin-top: 12px" v-else>
      <!-- Config -->
      <a-col :xs="24" :lg="8" style="margin-bottom: 12px">
        <a-card title="Configuration" :bordered="false" size="small">
            <a-form layout="vertical" style="margin-bottom: 0">
                <a-form-item label="Primary Interface" help="Interface used for main dashboard statistics" style="margin-bottom: 12px">
                    <a-select v-model:value="config.net_interface_list" mode="multiple" placeholder="Select interfaces" size="small">
                        <a-select-option value="auto">Auto (Total)</a-select-option>
                        <a-select-option v-for="iface in interfaces" :key="iface.name" :value="iface.name">{{ iface.name }}</a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="Traffic Reset Day" help="Day of month to reset cycle" style="margin-bottom: 12px">
                    <a-select v-model:value="config.net_reset_day" size="small">
                        <a-select-option v-for="n in 31" :key="n" :value="n">{{ n }}</a-select-option>
                    </a-select>
                </a-form-item>

                <a-divider style="margin: 12px 0">Traffic Limit</a-divider>

                <a-form-item label="Monthly Limit (GB)" help="0 for unlimited" style="margin-bottom: 12px">
                    <a-input-number v-model:value="config.limit_gb" :min="0" style="width: 100%" size="small" />
                </a-form-item>
                 <a-form-item label="Already Used (GB)" help="Correction for current month" style="margin-bottom: 12px">
                    <a-input-number v-model:value="config.adjustment_gb" :min="0" style="width: 100%" size="small" />
                </a-form-item>
                 <a-form-item label="Counter Mode" help="Which traffic counts towards limit" style="margin-bottom: 12px">
                    <a-select v-model:value="config.net_traffic_counter_mode" size="small">
                        <a-select-option value="total">Total (Upload + Download)</a-select-option>
                        <a-select-option value="tx">Upload Only (Tx)</a-select-option>
                        <a-select-option value="rx">Download Only (Rx)</a-select-option>
                    </a-select>
                </a-form-item>

                <a-button type="primary" @click="saveConfig" :loading="saving" block size="small">Save Configuration</a-button>
            </a-form>
        </a-card>
        
        <a-card title="Monthly Traffic" :bordered="false" size="small" style="margin-top: 12px">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
                <a-statistic title="Inbound (Rx)" :value="formatBytes(monthlyRx)" :valueStyle="{ color: '#3f8600', fontSize: '16px' }">
                    <template #prefix><ArrowDownOutlined /></template>
                </a-statistic>
                <a-statistic title="Outbound (Tx)" :value="formatBytes(monthlyTx)" :valueStyle="{ color: '#cf1322', fontSize: '16px' }">
                     <template #prefix><ArrowUpOutlined /></template>
                </a-statistic>
            </div>
            
            <div v-if="config.limit_gb > 0" style="margin-top: 12px">
                <div style="display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 12px">
                    <span>Usage ({{ usagePercentage }}%)</span>
                    <span>{{ formatBytes(totalUsedBytes) }} / {{ config.limit_gb }} GB</span>
                </div>
                <a-progress :percent="usagePercentage" :status="usageStatus" size="small" />
                <div style="margin-top: 4px; font-size: 12px; color: #8c8c8c">
                    Remaining: {{ formatBytes(remainingBytes) }}
                </div>
            </div>

             <a-alert message="Calculated based on Primary Interface" type="info" show-icon style="font-size: 12px; margin-top: 12px" />
        </a-card>
      </a-col>

      <!-- Interface List -->
      <a-col :xs="24" :lg="16">
        <a-card title="Interfaces" :bordered="false" size="small">
           <a-table :dataSource="interfaces" :columns="columns" :pagination="false" rowKey="name" size="small">
                <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'name'">
                        <span style="font-weight: bold">{{ record.name }}</span>
                         <a-tag v-if="config.net_interface_list.includes(record.name) || (config.net_interface_list.includes('auto') && record.name)" color="blue" style="margin-left: 8px">Primary</a-tag>
                    </template>
                    <template v-if="column.key === 'speed'">
                        <div style="white-space: nowrap">
                            <span style="color: #52c41a"><ArrowDownOutlined/> {{formatSpeed(record.rx_rate || 0)}}</span>
                            <a-divider type="vertical" />
                            <span style="color: #1890ff"><ArrowUpOutlined/> {{formatSpeed(record.tx_rate || 0)}}</span>
                        </div>
                    </template>
                    <template v-else-if="column.key === 'total'">
                         <div style="white-space: nowrap">
                            <div>Rx: {{ formatBytes(record.rx) }}</div>
                            <div>Tx: {{ formatBytes(record.tx) }}</div>
                         </div>
                    </template>
                </template>
           </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useSSHStore } from '../stores/ssh'
import { ArrowDownOutlined, ArrowUpOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

const route = useRoute()
const sshStore = useSSHStore()
const hostId = parseInt(route.params.id)

const host = computed(() => sshStore.hosts.find(h => h.id === hostId))
const connected = ref(false)
const socket = ref(null)
const interfaces = ref([])
const monthlyRx = ref(0)
const monthlyTx = ref(0)

const saving = ref(false)

const config = ref({
    net_interface: 'auto',
    net_interface_list: ['auto'],
    net_reset_day: 1,
    limit_gb: 0,
    adjustment_gb: 0,
    net_traffic_counter_mode: 'total'
})

// Initialize config from host when loaded
const initConfig = () => {
    if (host.value) {
        config.value.net_interface = host.value.net_interface || 'auto'
        if (config.value.net_interface.includes(',')) {
            config.value.net_interface_list = config.value.net_interface.split(',')
        } else {
            config.value.net_interface_list = [config.value.net_interface]
        }
        
        config.value.net_reset_day = host.value.net_reset_day || 1
        
        // Convert bytes to GB for display
        config.value.limit_gb = parseFloat(( (host.value.net_traffic_limit || 0) / (1024*1024*1024) ).toFixed(2))
        config.value.adjustment_gb = parseFloat(( (host.value.net_traffic_used_adjustment || 0) / (1024*1024*1024) ).toFixed(2))
        config.value.net_traffic_counter_mode = host.value.net_traffic_counter_mode || 'total'
    }
}

onMounted(async () => {
    if (sshStore.hosts.length === 0) {
        await sshStore.fetchHosts()
    }
    initConfig()
    connect()
})

// Computed Usage logic
const totalUsedBytes = computed(() => {
    let measured = 0
    if (config.value.net_traffic_counter_mode === 'total') {
        measured = monthlyRx.value + monthlyTx.value
    } else if (config.value.net_traffic_counter_mode === 'rx') {
        measured = monthlyRx.value
    } else if (config.value.net_traffic_counter_mode === 'tx') {
        measured = monthlyTx.value
    }
    
    // Add adjustment (GB -> Bytes)
    const adjustmentBytes = (config.value.adjustment_gb || 0) * 1024 * 1024 * 1024
    return measured + adjustmentBytes
})

const limitBytes = computed(() => {
    return (config.value.limit_gb || 0) * 1024 * 1024 * 1024
})

const remainingBytes = computed(() => {
    const rem = limitBytes.value - totalUsedBytes.value
    return rem > 0 ? rem : 0
})

const usagePercentage = computed(() => {
    if (limitBytes.value === 0) return 0
    const pct = Math.round((totalUsedBytes.value / limitBytes.value) * 100)
    return pct > 100 ? 100 : pct
})

const usageStatus = computed(() => {
    if (usagePercentage.value >= 90) return 'exception'
    if (usagePercentage.value >= 80) return 'active'
    return 'normal'
})


const columns = [
    { title: 'Interface', key: 'name', dataIndex: 'name' },
    { title: 'Real-time Speed', key: 'speed' },
    { title: 'Total Traffic (Since Boot)', key: 'total' },
]

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
  const token = localStorage.getItem('token')
  const wsUrl = `${protocol}//${window.location.host}/api/monitor/stream?token=${token}`
  
  socket.value = new WebSocket(wsUrl)

  socket.value.onopen = () => {
    connected.value = true
  }

  socket.value.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      const dataList = msg.type === 'init' || msg.type === 'update' ? msg.data : []
      if (!Array.isArray(dataList)) return

      const myData = dataList.find(h => h.host_id === hostId)
      if (myData) {
          interfaces.value = myData.interfaces || []
          monthlyRx.value = myData.net_monthly_rx || 0
          monthlyTx.value = myData.net_monthly_tx || 0
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

const saveConfig = async () => {
    saving.value = true
    try {
        // Convert GB back to Bytes
        const trafficLimit = Math.floor(config.value.limit_gb * 1024 * 1024 * 1024)
        const trafficAdj = Math.floor(config.value.adjustment_gb * 1024 * 1024 * 1024)
        
        // Join list to string
        const interfaceStr = config.value.net_interface_list.join(',')
        
        await sshStore.modifyHost(hostId, {
            net_interface: interfaceStr,
            net_reset_day: config.value.net_reset_day,
            net_traffic_limit: trafficLimit,
            net_traffic_used_adjustment: trafficAdj,
            net_traffic_counter_mode: config.value.net_traffic_counter_mode
        })
        message.success('Configuration saved')
    } catch (e) {
        message.error('Failed to save configuration')
        console.error(e)
    } finally {
        saving.value = false
    }
}

onUnmounted(() => {
  if (socket.value) socket.value.close()
})
</script>
