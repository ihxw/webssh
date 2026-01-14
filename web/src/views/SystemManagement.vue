<template>
  <div class="system-management">
    <a-card :title="t('nav.system')" :bordered="false">
      <a-divider orientation="left">{{ t('system.backupTitle') }}</a-divider>
      <div class="management-section">
        <p>{{ t('system.backupDesc') }}</p>
        <a-button type="primary" :loading="backupLoading" @click="handleBackup">
          <template #icon><DownloadOutlined /></template>
          {{ t('system.startBackup') }}
        </a-button>
      </div>

      <a-divider orientation="left">{{ t('system.restoreTitle') }}</a-divider>
      <div class="management-section">
        <a-alert
          :message="t('system.restoreWarningTitle')"
          :description="t('system.restoreWarningDesc')"
          type="warning"
          show-icon
          style="margin-bottom: 24px"
        />
        <p>{{ t('system.restoreDesc') }}</p>
        <a-upload
          name="file"
          :multiple="false"
          :show-upload-list="false"
          :before-upload="beforeRestoreUpload"
          @change="handleRestoreChange"
        >
          <a-button :loading="restoreLoading">
            <template #icon><UploadOutlined /></template>
            {{ t('system.startRestore') }}
          </a-button>
        </a-upload>
      </div>

      <a-divider orientation="left">{{ t('system.settingsTitle') }}</a-divider>
      <div class="management-section">
        <a-form :model="settingsForm" layout="vertical" @finish="handleSaveSettings">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item :label="t('system.sshTimeout')" name="ssh_timeout">
                <a-input v-model:value="settingsForm.ssh_timeout" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('system.idleTimeout')" name="idle_timeout">
                <a-input v-model:value="settingsForm.idle_timeout" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('system.maxConnectionsPerUser')" name="max_connections_per_user">
                <a-input-number v-model:value="settingsForm.max_connections_per_user" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('system.loginRateLimit')" name="login_rate_limit">
                <a-input-number v-model:value="settingsForm.login_rate_limit" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item>
            <a-button type="primary" :loading="settingsLoading" html-type="submit">
              {{ t('common.save') }}
            </a-button>
          </a-form-item>
        </a-form>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { message, Modal } from 'ant-design-vue'
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons-vue'
import { useThemeStore } from '../stores/theme'
import api from '../api'

const { t } = useI18n()
const themeStore = useThemeStore()
const backupLoading = ref(false)
const restoreLoading = ref(false)
const settingsLoading = ref(false)

const settingsForm = reactive({
  ssh_timeout: '30s',
  idle_timeout: '30m',
  max_connections_per_user: 10,
  login_rate_limit: 20
})

const fetchSettings = async () => {
  try {
    const response = await api.get('/system/settings')
    Object.assign(settingsForm, response.data) // Assuming response.data contains the settings object
  } catch (err) {
    message.error(t('system.fetchSettingsFailed'))
  }
}

onMounted(() => {
  fetchSettings()
})

const handleSaveSettings = async () => {
  settingsLoading.value = true
  try {
    await api.put('/system/settings', settingsForm)
    message.success(t('system.saveSettingsSuccess'))
  } catch (err) {
    message.error(err.response?.data?.error || t('system.saveSettingsFailed'))
  } finally {
    settingsLoading.value = false
  }
}

const handleBackup = async () => {
  backupLoading.value = true
  try {
    // We use a direct window.open or a hidden anchor for downloading binary files via GET
    const token = localStorage.getItem('token')
    const downloadUrl = `/api/system/backup?token=${token}`
    
    const link = document.createElement('a')
    link.href = downloadUrl
    link.setAttribute('download', 'webssh_backup.db')
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    
    message.success(t('system.backupSuccess'))
  } catch (err) {
    message.error(t('system.backupFailed'))
  } finally {
    backupLoading.value = false
  }
}

const beforeRestoreUpload = (file) => {
  const isDb = file.name.endsWith('.db')
  if (!isDb) {
    message.error(t('system.invalidFileType'))
  }
  return isDb
}

const handleRestoreChange = (info) => {
  if (info.file.status === 'uploading') {
    return
  }
  
  Modal.confirm({
    title: t('system.restoreConfirmTitle'),
    content: t('system.restoreConfirmContent'),
    okText: t('common.confirm'),
    cancelText: t('common.cancel'),
    onOk: () => performRestore(info.file.originFileObj),
  })
}

const performRestore = async (file) => {
  restoreLoading.value = true
  const formData = new FormData()
  formData.append('file', file)

  try {
    await api.post('/system/restore', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    message.success(t('system.restoreSuccess'))
    // Advise restart if needed, or simply reload to check
    setTimeout(() => {
      window.location.reload()
    }, 2000)
  } catch (err) {
    message.error(err.response?.data?.error || t('system.restoreFailed'))
  } finally {
    restoreLoading.value = false
  }
}
</script>

<style scoped>
.system-management {
  padding: 24px;
}
.management-section {
  padding: 16px;
  background: v-bind('themeStore.isDark ? "#1f1f1f" : "#fafafa"');
  border-radius: 4px;
}
.management-section p {
  margin-bottom: 24px;
  color: #8c8c8c;
}
</style>
