<template>
  <div>
    <a-card title="SSH Hosts" :bordered="false">
      <template #extra>
        <a-space>
          <a-input-search
            v-model:value="searchText"
            placeholder="Search hosts..."
            style="width: 200px"
            size="small"
            @search="handleSearch"
          />
          <a-button type="primary" size="small" @click="handleAdd">
            <PlusOutlined />
            Add Host
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
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="handleConnect(record)">
                <LinkOutlined />
                Connect
              </a-button>
              <a-button size="small" @click="handleEdit(record)">
                <EditOutlined />
                Edit
              </a-button>
              <a-popconfirm
                title="Are you sure you want to delete this host?"
                @confirm="handleDelete(record.id)"
              >
                <a-button size="small" danger>
                  <DeleteOutlined />
                  Delete
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
      :title="editingHost ? 'Edit SSH Host' : 'Add SSH Host'"
      @ok="handleSave"
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
          <a-input-password v-model:value="hostForm.password" :placeholder="editingHost ? 'Leave empty to keep current' : 'Enter password'" />
        </a-form-item>

        <a-form-item v-if="hostForm.auth_type === 'key'" label="Private Key" :required="!editingHost">
          <a-textarea
            v-model:value="hostForm.private_key"
            :placeholder="editingHost ? 'Leave empty to keep current' : '-----BEGIN RSA PRIVATE KEY-----'"
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
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LinkOutlined
} from '@ant-design/icons-vue'
import { useSSHStore } from '../stores/ssh'

const router = useRouter()
const sshStore = useSSHStore()

const loading = ref(false)
const searchText = ref('')
const showModal = ref(false)
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

const columns = [
  { title: 'Name', dataIndex: 'name', key: 'name' },
  { title: 'Host', dataIndex: 'host', key: 'host' },
  { title: 'Port', dataIndex: 'port', key: 'port' },
  { title: 'Username', dataIndex: 'username', key: 'username' },
  { title: 'Group', dataIndex: 'group_name', key: 'group_name' },
  { title: 'Action', key: 'action', width: 250 }
]

onMounted(async () => {
  await loadHosts()
})

const loadHosts = async () => {
  loading.value = true
  try {
    await sshStore.fetchHosts()
  } catch (error) {
    message.error('Failed to load hosts')
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
    message.error('Failed to load host details')
  }
}

const handleSave = async () => {
  if (!hostForm.value.name || !hostForm.value.host || !hostForm.value.username) {
    message.error('Please fill in all required fields')
    return
  }

  if (!editingHost.value) {
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
      const updateData = { ...hostForm.value }
      if (!updateData.password) delete updateData.password
      if (!updateData.private_key) delete updateData.private_key
      
      await sshStore.modifyHost(editingHost.value.id, updateData)
      message.success('Host updated successfully')
    } else {
      await sshStore.addHost(hostForm.value)
      message.success('Host added successfully')
    }
    showModal.value = false
    await loadHosts()
  } catch (error) {
    message.error(editingHost.value ? 'Failed to update host' : 'Failed to add host')
  } finally {
    saving.value = false
  }
}

const handleDelete = async (id) => {
  try {
    await sshStore.removeHost(id)
    message.success('Host deleted successfully')
  } catch (error) {
    message.error('Failed to delete host')
  }
}
</script>
