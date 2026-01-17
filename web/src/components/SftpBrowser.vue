<template>
  <div class="sftp-browser">
    <div class="browser-header">
      <a-breadcrumb size="small">
        <a-breadcrumb-item v-for="(part, index) in pathParts" :key="index">
          <a @click="navigateTo(index)">{{ part || '/' }}</a>
        </a-breadcrumb-item>
      </a-breadcrumb>
      <div class="header-actions">
        <a-button size="small" @click="refresh">
          <template #icon><ReloadOutlined /></template>
        </a-button>
        <a-upload
          :custom-request="handleUpload"
          :show-upload-list="false"
          accept="*"
        >
          <a-button size="small" type="primary">
            <template #icon><UploadOutlined /></template>
            Upload
          </a-button>
        </a-upload>
      </div>
    </div>

    <div class="browser-content" v-loading="loading">
      <a-table
        :columns="columns"
        :data-source="files"
        :pagination="false"
        size="small"
        :scroll="{ y: 'calc(100vh - 150px)' }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a v-if="record.is_dir" @click="enterDir(record.name)">
              <FolderFilled style="color: #faad14; margin-right: 8px" />
              {{ record.name }}
            </a>
            <span v-else>
              <FileOutlined style="color: #8c8c8c; margin-right: 8px" />
              {{ record.name }}
            </span>
          </template>
          <template v-else-if="column.key === 'size'">
            {{ record.is_dir ? '-' : formatSize(record.size) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space size="small">
              <a-button size="small" type="text" v-if="!record.is_dir" @click="download(record.name)">
                <template #icon><DownloadOutlined /></template>
              </a-button>
              <a-popconfirm
                title="Are you sure to delete this file?"
                @confirm="remove(record.name)"
              >
                <a-button size="small" type="text" danger>
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, h } from 'vue'
import { message, notification, Progress } from 'ant-design-vue'
import { 
  FolderFilled, 
  FileOutlined, 
  ReloadOutlined, 
  UploadOutlined, 
  DownloadOutlined, 
  DeleteOutlined 
} from '@ant-design/icons-vue'
import { listFiles, uploadFile, downloadFile, deleteFile } from '../api/sftp'

const props = defineProps({
  hostId: {
    type: [String, Number],
    required: true
  },
  visible: {
    type: Boolean,
    default: false
  }
})

const currentPath = ref('.')
const files = ref([])
const loading = ref(false)

const pathParts = computed(() => {
  if (currentPath.value === '.') return ['']
  const parts = currentPath.value.split('/').filter(p => p !== '')
  return ['', ...parts]
})

const columns = [
  { title: 'Name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
  { title: 'Size', key: 'size', width: 100, sorter: (a, b) => a.size - b.size },
  { title: 'Action', key: 'action', width: 80 }
]

const loadFiles = async () => {
  if (!props.hostId) return
  loading.value = true
  try {
    const data = await listFiles(props.hostId, currentPath.value)
    files.value = data || []
  } catch (error) {
    console.error('Failed to list files:', error)
  } finally {
    loading.value = false
  }
}

const refresh = () => loadFiles()

const enterDir = (name) => {
  if (currentPath.value === '.') {
    currentPath.value = name
  } else {
    currentPath.value = currentPath.value.endsWith('/') 
      ? currentPath.value + name 
      : currentPath.value + '/' + name
  }
  loadFiles()
}

const navigateTo = (index) => {
  if (index === 0) {
    currentPath.value = '.'
  } else {
    const parts = pathParts.value.slice(1, index + 1)
    currentPath.value = parts.join('/')
  }
  loadFiles()
}

const handleUpload = async ({ file, onSuccess, onError }) => {
  const key = `upload-${Date.now()}`
  try {
    notification.open({
        key,
        message: 'Uploading...',
        description: h('div', [
            h(Progress, { percent: 0, status: 'active', size: 'small' }),
            h('div', { style: 'margin-top: 8px' }, file.name)
        ]),
        duration: 0,
        placement: 'bottomRight'
    })

    await uploadFile(props.hostId, currentPath.value, file, (percent) => {
        notification.open({
            key,
            message: 'Uploading...',
            description: h('div', [
                h(Progress, { percent: percent, status: 'active', size: 'small' }),
                h('div', { style: 'margin-top: 8px' }, file.name)
            ]),
            duration: 0,
            placement: 'bottomRight'
        })
    })
    
    notification.success({
        key,
        message: 'Upload Complete',
        description: `${file.name} uploaded successfully`,
        duration: 3,
        placement: 'bottomRight'
    })
    
    loadFiles()
    onSuccess()
  } catch (error) {
    notification.error({
        key,
        message: 'Upload Failed',
        description: error.message || 'Failed to upload file',
        duration: 4.5,
        placement: 'bottomRight'
    })
    onError(error)
  }
}

const download = async (name) => {
  const fullPath = currentPath.value === '.' ? name : `${currentPath.value}/${name}`
  const key = `download-${Date.now()}`
  
  try {
     notification.open({
        key,
        message: 'Downloading...',
        description: h('div', [
            h(Progress, { percent: 0, status: 'active', size: 'small' }),
            h('div', { style: 'margin-top: 8px' }, name)
        ]),
        duration: 0,
        placement: 'bottomRight'
    })

    const response = await downloadFile(props.hostId, fullPath, (percent) => {
         notification.open({
            key,
            message: 'Downloading...',
            description: h('div', [
                h(Progress, { percent: percent, status: 'active', size: 'small' }),
                h('div', { style: 'margin-top: 8px' }, name)
            ]),
            duration: 0,
            placement: 'bottomRight'
        })
    })

    // Create blobs and trigger downloads
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)

    notification.success({
        key,
        message: 'Download Complete',
        description: `${name} downloaded successfully`,
        duration: 3,
        placement: 'bottomRight'
    })
  } catch (error) {
      console.error(error)
      notification.error({
        key,
        message: 'Download Failed',
        description: 'Failed to download file',
        duration: 4.5,
        placement: 'bottomRight'
    })
  }
}

const remove = async (name) => {
  const fullPath = currentPath.value === '.' ? name : `${currentPath.value}/${name}`
  try {
    await deleteFile(props.hostId, fullPath)
    message.success('Deleted successfully')
    loadFiles()
  } catch (error) {
    console.error('Failed to delete:', error)
  }
}

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

watch(() => props.visible, (newVal) => {
  if (newVal && files.value.length === 0) {
    loadFiles()
  }
})

onMounted(() => {
  if (props.visible) {
    loadFiles()
  }
})
</script>

<style scoped>
.sftp-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.browser-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 4px 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.browser-content {
  flex: 1;
  overflow: auto;
}

:deep(.ant-table-cell) {
  padding: 4px 8px !important;
}
</style>
