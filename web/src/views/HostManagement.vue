<template>
  <div>
    <a-card :title="t('nav.hosts')" :bordered="false">
      <template #extra>
        <a-space>
          <a-input-search
            v-model:value="searchText"
            :placeholder="t('host.searchPlaceholder')"
            style="width: 200px"
            size="small"
            @search="handleSearch"
          />
          <a-button type="primary" size="small" @click="handleAdd">
            <PlusOutlined />
            {{ t('host.addHost') }}
          </a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="sshStore.hosts"
        :loading="loading"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <div style="display: flex; align-items: center">
              <a-tooltip :title="hostStatuses[record.id]?.status === 'online' ? 'Online' : (hostStatuses[record.id]?.error || 'Checking...')">
                <a-tag v-if="hostStatuses[record.id]?.status === 'online'" color="success">
                  {{ hostStatuses[record.id]?.latency }}ms
                </a-tag>
                <a-tag v-else-if="hostStatuses[record.id]?.status === 'offline'" color="error">
                  Offline
                </a-tag>
                <a-tag v-else color="processing">
                  <template #icon><LoadingOutlined /></template>
                  Checking
                </a-tag>
              </a-tooltip>
            </div>
          </template>
          <template v-if="column.key === 'monitor'">
             <div style="display: flex; align-items: center">
                <a-tag v-if="record.monitor_enabled" color="processing">
                  <template #icon><DashboardOutlined /></template>
                  Enabled
                </a-tag>
                <a-tag v-else color="default">
                  Disabled
                </a-tag>
             </div>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button 
                v-if="!record.monitor_enabled"
                size="small" 
                :loading="monitorLoading[record.id]"
                @click="openDeployModal(record)"
              >
                <DashboardOutlined />
                Monitor
              </a-button>
              <a-popconfirm
                v-else
                title="Disable monitoring?"
                @confirm="handleStopMonitor(record)"
              >
                 <a-button size="small" danger :loading="monitorLoading[record.id]">
                   <StopOutlined />
                   Stop
                 </a-button>
              </a-popconfirm>
              <a-button size="small" @click="handleConnect(record)">
                <LinkOutlined />
                {{ t('terminal.connect') }}
              </a-button>
              <a-button size="small" @click="handleEdit(record)">
                <EditOutlined />
                {{ t('common.edit') }}
              </a-button>
              <a-popconfirm
                :title="t('host.deleteConfirm')"
                @confirm="handleDelete(record.id)"
              >
                <a-button size="small" danger>
                  <DeleteOutlined />
                  {{ t('common.delete') }}
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Edit/Add Host Modal -->
    <a-modal
      v-model:open="showModal"
      :title="editingHost ? t('host.editHost') : t('host.addHost')"
      @ok="handleSave"
      :confirmLoading="saving"
    >
      <a-form :model="hostForm" layout="vertical">
        <a-form-item :label="t('host.name')" required>
          <a-input v-model:value="hostForm.name" :placeholder="t('host.placeholderName')" />
        </a-form-item>

        <a-form-item :label="t('host.host')" required>
          <a-input v-model:value="hostForm.host" :placeholder="t('host.placeholderHost')" />
        </a-form-item>

        <a-form-item :label="t('host.port')">
          <a-input-number v-model:value="hostForm.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>

        <a-form-item :label="t('host.username')" required>
          <a-input v-model:value="hostForm.username" :placeholder="t('host.placeholderUsername')" />
        </a-form-item>

        <a-form-item :label="t('host.authMethod')" required>
          <a-radio-group v-model:value="hostForm.auth_type">
            <a-radio value="password">{{ t('host.authPassword') }}</a-radio>
            <a-radio value="key">{{ t('host.authKey') }}</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="hostForm.auth_type === 'password'" :label="t('host.password')" :required="!editingHost">
          <a-input-password v-model:value="hostForm.password" :placeholder="editingHost ? t('host.placeholderKeepPassword') : t('host.placeholderPassword')" />
        </a-form-item>

        <a-form-item v-if="hostForm.auth_type === 'key'" :label="t('host.privateKey')" :required="!editingHost">
          <a-textarea
            v-model:value="hostForm.private_key"
            :placeholder="editingHost ? t('host.placeholderKeepKey') : t('host.placeholderPrivateKey')"
            :rows="6"
          />
        </a-form-item>

        <a-form-item :label="t('host.group')">
          <a-input v-model:value="hostForm.group_name" :placeholder="t('host.placeholderGroup')" />
        </a-form-item>

        <a-form-item :label="t('host.description')">
          <a-textarea v-model:value="hostForm.description" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Deploy Monitor Modal -->
    <a-modal
      v-model:open="deployVisible"
      title="Deploy Monitor Agent"
      @ok="handleDeploy"
      :confirmLoading="deploying"
    >
      <p>Are you sure you want to deploy the monitor agent to <b>{{ deployHost?.name }}</b>?</p>
      <a-checkbox v-model:checked="deployInsecure">
        Skip SSL Certificate Verification (Insecure)
      </a-checkbox>
      <p style="margin-top: 8px; font-size: 12px; color: #faad14;" v-if="deployInsecure">
        Warning: Skipping SSL verification may expose the connection to MITM attacks. Use only for trusted networks or self-signed certificates.
      </p>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LinkOutlined,
  LoadingOutlined,
  DashboardOutlined,
  StopOutlined
} from '@ant-design/icons-vue'
import { useSSHStore } from '../stores/ssh'
import { useI18n } from 'vue-i18n'
import { deployMonitor, stopMonitor } from '../api/ssh'

const router = useRouter()
const sshStore = useSSHStore()
const { t } = useI18n()

const loading = ref(false)
const searchText = ref('')
const showModal = ref(false)
const saving = ref(false)
const editingHost = ref(null)

const deployVisible = ref(false)
const deployInsecure = ref(false)
const deployHost = ref(null)
const deploying = ref(false)

const hostForm = ref({
  name: '',
  host: '',
  port: 22,
  username: '',
  auth_type: 'password',
  password: '',
  private_key: '',
  group_name: '',
  description: ''
})

const columns = computed(() => [
  { title: t('host.name'), dataIndex: 'name', key: 'name' },
  { title: t('host.host'), dataIndex: 'host', key: 'host' },
  { title: 'Status', key: 'status', width: 100 },
  { title: 'Monitor', key: 'monitor', width: 100 },
  { title: t('host.port'), dataIndex: 'port', key: 'port' },
  { title: t('host.username'), dataIndex: 'username', key: 'username' },
  { title: t('host.group'), dataIndex: 'group_name', key: 'group_name' },
  { title: t('common.edit'), key: 'action', width: 320 }
])

const monitorLoading = ref({})

const openDeployModal = (host) => {
    deployHost.value = host
    deployInsecure.value = false
    deployVisible.value = true
}

const handleDeploy = async () => {
    if (!deployHost.value) return
    deploying.value = true
    monitorLoading.value[deployHost.value.id] = true
    
    try {
        await deployMonitor(deployHost.value.id, deployInsecure.value)
        message.success('Monitor agent deployed successfully')
        deployHost.value.monitor_enabled = true
        deployVisible.value = false
    } catch (error) {
        message.error('Failed to deploy monitor: ' + (error.response?.data?.error || error.message))
    } finally {
        deploying.value = false
        monitorLoading.value[deployHost.value.id] = false
    }
}

const handleStopMonitor = async (host) => {
  monitorLoading.value[host.id] = true
  try {
    await stopMonitor(host.id)
    message.success('Monitoring disabled')
    host.monitor_enabled = false
  } catch (error) {
    message.error('Failed to stop monitor')
  } finally {
    monitorLoading.value[host.id] = false
  }
}

const hostStatuses = ref({})
const checkingStatus = ref(false)

onMounted(async () => {
  await loadHosts()
  checkAllStatuses()
})

const checkAllStatuses = async () => {
  if (checkingStatus.value || sshStore.hosts.length === 0) return
  
  checkingStatus.value = true
  // Check in batches or parallel? Parallel is fine for small numbers.
  const checks = sshStore.hosts.map(async (host) => {
    hostStatuses.value[host.id] = { status: 'loading' }
    try {
      const result = await sshStore.testHostConnection(host.id)
      hostStatuses.value[host.id] = result
    } catch (e) {
      hostStatuses.value[host.id] = { status: 'offline', error: 'Failed to check' }
    }
  })
  
  await Promise.allSettled(checks)
  checkingStatus.value = false
}

const loadHosts = async () => {
  loading.value = true
  try {
    await sshStore.fetchHosts()
    checkAllStatuses()
  } catch (error) {
    message.error(t('host.failLoad'))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  loadHosts()
}

const handleAdd = () => {
  editingHost.value = null
  showModal.value = true
  hostForm.value = {
    name: '',
    host: '',
    port: 22,
    username: '',
    auth_type: 'password',
    password: '',
    private_key: '',
    group_name: '',
    description: ''
  }
}

const handleConnect = (host) => {
  sshStore.addTerminal({
    hostId: host.id,
    name: host.name,
    host: host.host,
    port: host.port
  })
  router.push('/dashboard/terminal')
}

const handleEdit = async (host) => {
  editingHost.value = host
  showModal.value = true
  
  // Load full host details
  try {
    const fullHost = await sshStore.fetchHost(host.id)
    hostForm.value = {
      name: fullHost.name,
      host: fullHost.host,
      port: fullHost.port,
      username: fullHost.username,
      auth_type: fullHost.auth_type,
      password: '',
      private_key: '',
      group_name: fullHost.group_name || '',
      description: fullHost.description || ''
    }
  } catch (error) {
    message.error(t('host.failLoad'))
  }
}

const handleSave = async () => {
  if (!hostForm.value.name || !hostForm.value.host || !hostForm.value.username) {
    message.error(t('host.validationRequired'))
    return
  }

  if (!editingHost.value) {
    if (hostForm.value.auth_type === 'password' && !hostForm.value.password) {
      message.error(t('host.validationPassword'))
      return
    }
    if (hostForm.value.auth_type === 'key' && !hostForm.value.private_key) {
      message.error(t('host.validationKey'))
      return
    }
  }

  saving.value = true
  try {
    if (editingHost.value) {
      const updateData = { ...hostForm.value }
      if (!updateData.password) delete updateData.password
      if (!updateData.private_key) delete updateData.private_key
      
      await sshStore.modifyHost(editingHost.value.id, updateData)
      message.success(t('host.successUpdate'))
    } else {
      await sshStore.addHost(hostForm.value)
      message.success(t('host.successAdd'))
    }
    showModal.value = false
    await loadHosts()
  } catch (error) {
    message.error(editingHost.value ? t('host.failUpdate') : t('host.failAdd'))
  } finally {
    saving.value = false
  }
}

const handleDelete = async (id) => {
  try {
    await sshStore.removeHost(id)
    message.success(t('host.hostDeleted'))
  } catch (error) {
    message.error(t('common.error'))
  }
}
</script>
