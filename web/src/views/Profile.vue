<template>
  <div>
    <a-card title="User Profile" :bordered="false">
      <a-descriptions bordered :column="1" size="small">
        <a-descriptions-item label="Username">
          {{ authStore.user?.username }}
        </a-descriptions-item>
        <a-descriptions-item label="Email">
          {{ authStore.user?.email }}
        </a-descriptions-item>
        <a-descriptions-item label="Display Name">
          {{ authStore.user?.display_name || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Role">
          <a-tag :color="authStore.user?.role === 'admin' ? 'red' : 'blue'">
            {{ authStore.user?.role }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="Status">
          <a-tag :color="authStore.user?.status === 'active' ? 'success' : 'default'">
            {{ authStore.user?.status }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="Last Login">
          {{ formatDate(authStore.user?.last_login_at) }}
        </a-descriptions-item>
      </a-descriptions>

      <a-divider />

      <a-button type="primary" size="small" @click="showPasswordModal = true">
        Change Password
      </a-button>
    </a-card>

    <a-modal
      v-model:open="showPasswordModal"
      title="Change Password"
      :confirmLoading="loading"
      @ok="handleChangePassword"
    >
      <a-form layout="vertical">
        <a-form-item label="Current Password">
          <a-input-password v-model:value="passwordForm.current" />
        </a-form-item>
        <a-form-item label="New Password">
          <a-input-password v-model:value="passwordForm.new" />
        </a-form-item>
        <a-form-item label="Confirm Password">
          <a-input-password v-model:value="passwordForm.confirm" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { useAuthStore } from '../stores/auth'
import { changePassword } from '../api/auth'

const authStore = useAuthStore()

const showPasswordModal = ref(false)
const loading = ref(false)
const passwordForm = ref({
  current: '',
  new: '',
  confirm: ''
})

const formatDate = (dateString) => {
  if (!dateString) return '-'
  return new Date(dateString).toLocaleString()
}

const handleChangePassword = async () => {
  if (!passwordForm.value.current || !passwordForm.value.new || !passwordForm.value.confirm) {
    message.error('Please fill in all fields')
    return
  }

  if (passwordForm.value.new !== passwordForm.value.confirm) {
    message.error('New passwords do not match')
    return
  }

  if (passwordForm.value.new.length < 6) {
    message.error('New password must be at least 6 characters')
    return
  }

  loading.value = true
  try {
    await changePassword(passwordForm.value.current, passwordForm.value.new)
    message.success('Password changed successfully')
    showPasswordModal.value = false
    // Clear form
    passwordForm.value = {
      current: '',
      new: '',
      confirm: ''
    }
  } catch (error) {
    // Error is handled by global interceptor
    console.error('Password change failed:', error)
  } finally {
    loading.value = false
  }
}
</script>
