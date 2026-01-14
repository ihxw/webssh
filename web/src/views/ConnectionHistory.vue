<template>
  <div>
    <a-card title="Connection History" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="logs"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ record.status }}
            </a-tag>
          </template>
          <template v-if="column.key === 'connected_at'">
            {{ formatDate(record.connected_at) }}
          </template>
          <template v-if="column.key === 'duration'">
            {{ formatDuration(record.duration) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getConnectionLogs } from '../api/logs'

const loading = ref(false)
const logs = ref([])
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0
})

const columns = [
  { title: 'Host', dataIndex: 'host', key: 'host' },
  { title: 'Port', dataIndex: 'port', key: 'port' },
  { title: 'Username', dataIndex: 'username', key: 'username' },
  { title: 'Status', dataIndex: 'status', key: 'status' },
  { title: 'Connected At', dataIndex: 'connected_at', key: 'connected_at' },
  { title: 'Duration', dataIndex: 'duration', key: 'duration' }
]

onMounted(() => {
  loadLogs()
})

const loadLogs = async () => {
  loading.value = true
  try {
    const response = await getConnectionLogs({
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    })
    logs.value = response.data || response
    if (response.pagination) {
      pagination.value.total = response.pagination.total
    }
  } catch (error) {
    message.error('Failed to load connection logs')
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pag) => {
  pagination.value.current = pag.current
  pagination.value.pageSize = pag.pageSize
  loadLogs()
}

const getStatusColor = (status) => {
  const colors = {
    success: 'success',
    failed: 'error',
    disconnected: 'default',
    connecting: 'processing'
  }
  return colors[status] || 'default'
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleString()
}

const formatDuration = (seconds) => {
  if (!seconds) return '-'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  return `${hours}h ${minutes}m ${secs}s`
}
</script>
