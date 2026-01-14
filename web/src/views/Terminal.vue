<template>
  <div style="height: calc(100vh - 80px)">
    <a-card :bordered="false" class="terminal-card">
      <template #title>
        <div style="display: flex; align-items: center; gap: 12px">
          <a-select
            v-model:value="selectedHostId"
            placeholder="Select a host"
            style="width: 300px"
            size="small"
            :loading="loading"
            @change="handleHostSelect"
          >
            <a-select-option
              v-for="host in sshStore.hosts"
              :key="host.id"
              :value="host.id"
            >
              <DatabaseOutlined style="margin-right: 8px" />
              {{ host.name }} ({{ host.host }}:{{ host.port }})
            </a-select-option>
          </a-select>

          <a-button type="primary" size="small" @click="handleAddHost">
            <PlusOutlined />
            New Host
          </a-button>

          <a-button size="small" @click="handleQuickConnect">
            <ThunderboltOutlined />
            Quick Connect
          </a-button>
        </div>
      </template>

      <div class="terminal-container-inner" style="height: 100%; display: flex; flex-direction: column">
        <a-tabs
          v-model:activeKey="activeTerminalKey"
          type="editable-card"
          @edit="onTabEdit"
          class="terminal-tabs"
          style="flex: 1; display: flex; flex-direction: column; overflow: hidden"
        >
          <a-tab-pane
            v-for="terminal in sshStore.terminalList"
            :key="terminal.id"
            :tab="terminal.name"
            :closable="true"
            style="flex: 1; height: 100%"
          >
            <TerminalComponent
              :terminal-id="terminal.id"
              :host-id="terminal.hostId"
              @close="() => closeTerminal(terminal.id)"
            />
          </a-tab-pane>

          <template #addIcon>
            <PlusOutlined />
          </template>
        </a-tabs>

        <div v-if="sshStore.terminalList.length === 0" style="text-align: center; flex: 1; display: flex; align-items: center; justify-content: center">
          <a-empty description="No active terminals">
            <a-button type="primary" size="small" @click="handleQuickConnect">
              <PlusOutlined />
              Connect to SSH Host
            </a-button>
          </a-empty>
        </div>
      </div>
    </a-card>

    <!-- Host Form Modal -->
    <a-modal
      v-model:open="showHostModal"
      :title="editingHost ? 'Edit SSH Host' : 'Add SSH Host'"
      @ok="handleSaveHost"
      :confirmLoading="saving"
    >
      <a-form :model="hostForm" layout="vertical">
        <a-form-item label="Name" required>
          <a-input v-model:value="hostForm.name" placeholder="My Server" />
        </a-form-item>

        <a-form-item label="Host" required>
          <a-input v-model:value="hostForm.host" placeholder="192.168.1.100" />
        </a-form-item>

        <a-form-item label="Port">
          <a-input-number v-model:value="hostForm.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>

        <a-form-item label="Username" required>
          <a-input v-model:value="hostForm.username" placeholder="root" />
        </a-form-item>

        <a-form-item label="Authentication Type" required>
          <a-radio-group v-model:value="hostForm.auth_type">
            <a-radio value="password">Password</a-radio>
            <a-radio value="key">SSH Key</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="hostForm.auth_type === 'password'" label="Password" :required="!editingHost">
          <a-input-password v-model:value="hostForm.password" :placeholder="editingHost ? 'Leave empty to keep current password' : 'Enter password'" />
        </a-form-item>

        <a-form-item v-if="hostForm.auth_type === 'key'" label="Private Key" :required="!editingHost">
          <a-textarea
            v-model:value="hostForm.private_key"
            :placeholder="editingHost ? 'Leave empty to keep current key' : '-----BEGIN RSA PRIVATE KEY-----'"
            :rows="6"
          />
        </a-form-item>

        <a-form-item label="Group">
          <a-input v-model:value="hostForm.group_name" placeholder="Production" />
        </a-form-item>

        <a-form-item label="Description">
          <a-textarea v-model:value="hostForm.description" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  DatabaseOutlined,
  PlusOutlined,
  ThunderboltOutlined
} from '@ant-design/icons-vue'
import { useSSHStore } from '../stores/ssh'
import TerminalComponent from '../components/Terminal.vue'

const sshStore = useSSHStore()

const selectedHostId = ref(null)
const activeTerminalKey = ref(null)
const loading = ref(false)
const showHostModal = ref(false)
const saving = ref(false)
const editingHost = ref(null)

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

onMounted(async () => {
  loading.value = true
  try {
    await sshStore.fetchHosts()
  } catch (error) {
    message.error('Failed to load SSH hosts')
  } finally {
    loading.value = false
  }
})

const handleHostSelect = (hostId) => {
  const host = sshStore.hosts.find(h => h.id === hostId)
  if (host) {
    connectToHost(host)
  }
}

const connectToHost = (host) => {
  const terminalId = sshStore.addTerminal({
    hostId: host.id,
    name: host.name,
    host: host.host,
    port: host.port
  })
  activeTerminalKey.value = terminalId
}

const handleAddHost = () => {
  editingHost.value = null
  showHostModal.value = true
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

const handleQuickConnect = () => {
  handleAddHost()
}

const handleSaveHost = async () => {
  if (!hostForm.value.name || !hostForm.value.host || !hostForm.value.username) {
    message.error('Please fill in all required fields')
    return
  }

  if (!editingHost.value) {
    // Adding new host
    if (hostForm.value.auth_type === 'password' && !hostForm.value.password) {
      message.error('Please enter password')
      return
    }

    if (hostForm.value.auth_type === 'key' && !hostForm.value.private_key) {
      message.error('Please enter private key')
      return
    }
  }

  saving.value = true
  try {
    if (editingHost.value) {
      // Update existing host
      const updateData = { ...hostForm.value }
      // Remove empty password/key fields when editing
      if (!updateData.password) delete updateData.password
      if (!updateData.private_key) delete updateData.private_key
      
      await sshStore.modifyHost(editingHost.value.id, updateData)
      message.success('Host updated successfully')
    } else {
      // Add new host
      const host = await sshStore.addHost(hostForm.value)
      message.success('Host added successfully')
      
      // Connect to the newly added host
      connectToHost(host)
    }
    
    showHostModal.value = false
  } catch (error) {
    message.error(editingHost.value ? 'Failed to update host' : 'Failed to add host')
  } finally {
    saving.value = false
  }
}

const onTabEdit = (targetKey, action) => {
  if (action === 'add') {
    handleQuickConnect()
  } else if (action === 'remove') {
    closeTerminal(targetKey)
  }
}

const closeTerminal = (terminalId) => {
  sshStore.removeTerminal(terminalId)
  
  // Update active key
  const terminals = sshStore.terminalList
  if (terminals.length > 0) {
    activeTerminalKey.value = terminals[terminals.length - 1].id
  } else {
    activeTerminalKey.value = null
  }
}
</script>

<style scoped>
.terminal-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

:deep(.ant-card-body) {
  flex: 1;
  overflow: hidden;
  padding: 0;
  display: flex;
  flex-direction: column;
}

:deep(.terminal-tabs) {
  height: 100%;
}

:deep(.ant-tabs-content) {
  flex: 1;
  height: 100%;
}

:deep(.ant-tabs-tabpane) {
  display: flex;
  flex-direction: column;
}
</style>
