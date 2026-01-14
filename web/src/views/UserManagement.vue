<template>
  <div>
    <a-card title="User Management" :bordered="false">
      <template #extra>
        <a-button type="primary" size="small" @click="showModal = true">
          <PlusOutlined />
          Add User
        </a-button>
      </template>

      <a-table
        :columns="columns"
        :data-source="users"
        :loading="loading"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'role'">
            <a-tag :color="record.role === 'admin' ? 'red' : 'blue'">
              {{ record.role }}
            </a-tag>
          </template>
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'active' ? 'success' : 'default'">
              {{ record.status }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="handleEdit(record)">
                <EditOutlined />
              </a-button>
              <a-popconfirm
                title="Are you sure you want to delete this user?"
                @confirm="handleDelete(record.id)"
              >
                <a-button size="small" danger>
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { getUsers, deleteUser } from '../api/users'

const loading = ref(false)
const users = ref([])
const showModal = ref(false)

const columns = [
  { title: 'Username', dataIndex: 'username', key: 'username' },
  { title: 'Email', dataIndex: 'email', key: 'email' },
  { title: 'Display Name', dataIndex: 'display_name', key: 'display_name' },
  { title: 'Role', dataIndex: 'role', key: 'role' },
  { title: 'Status', dataIndex: 'status', key: 'status' },
  { title: 'Action', key: 'action', width: 150 }
]

onMounted(() => {
  loadUsers()
})

const loadUsers = async () => {
  loading.value = true
  try {
    const response = await getUsers()
    users.value = response.data || response
  } catch (error) {
    message.error('Failed to load users')
  } finally {
    loading.value = false
  }
}

const handleEdit = (user) => {
  message.info('Edit functionality coming soon')
}

const handleDelete = async (id) => {
  try {
    await deleteUser(id)
    message.success('User deleted successfully')
    loadUsers()
  } catch (error) {
    message.error('Failed to delete user')
  }
}
</script>
