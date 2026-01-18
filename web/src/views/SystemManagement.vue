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
            <a-col :span="6">
              <a-form-item :label="t('system.sshTimeout')" name="ssh_timeout">
                <a-input v-model:value="settingsForm.ssh_timeout" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.idleTimeout')" name="idle_timeout">
                <a-input v-model:value="settingsForm.idle_timeout" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.maxConnectionsPerUser')" name="max_connections_per_user">
                <a-input-number v-model:value="settingsForm.max_connections_per_user" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.loginRateLimit')" name="login_rate_limit">
                <a-input-number v-model:value="settingsForm.login_rate_limit" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.accessExpiration')" name="access_expiration">
                <a-input v-model:value="settingsForm.access_expiration" placeholder="60m" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.refreshExpiration')" name="refresh_expiration">
                <a-input v-model:value="settingsForm.refresh_expiration" placeholder="168h" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-divider orientation="left">{{ t('system.notificationTitle') }}</a-divider>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item :label="t('system.smtpServer')" name="smtp_server">
                <a-input v-model:value="settingsForm.smtp_server" placeholder="smtp.example.com" />
              </a-form-item>
            </a-col>
            <a-col :span="4">
              <a-form-item :label="t('system.smtpPort')" name="smtp_port">
                <a-input v-model:value="settingsForm.smtp_port" placeholder="587" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.smtpUser')" name="smtp_user">
                <a-input v-model:value="settingsForm.smtp_user" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('system.smtpPassword')" name="smtp_password">
                <a-input-password v-model:value="settingsForm.smtp_password" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('system.smtpFrom')" name="smtp_from">
                <a-input v-model:value="settingsForm.smtp_from" placeholder="noreply@example.com" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('system.smtpTo')" name="smtp_to">
                <a-input v-model:value="settingsForm.smtp_to" placeholder="admin@example.com" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
             <a-col :span="12">
              <a-form-item :label="t('system.telegramToken')" name="telegram_bot_token">
                <a-input v-model:value="settingsForm.telegram_bot_token" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item :label="t('system.telegramChatId')" name="telegram_chat_id">
                <a-input v-model:value="settingsForm.telegram_chat_id" />
              </a-form-item>
            </a-col>

          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item :label="t('system.notificationTemplate')" name="notification_template">
                <a-textarea v-model:value="settingsForm.notification_template" :rows="6" />
                <div style="margin-top: 8px">
                    <a-button @click="resetNotificationTemplate" size="small">{{ t('system.resetTemplate') }}</a-button>
                    <span style="margin-left: 8px; font-size: 12px; color: #888">
                        {{ t('system.templateHelp') }}: <span v-pre>{{emoji}}, {{event}}, {{client}}, {{message}}, {{time}}</span>
                    </span>
                </div>
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

    <!-- Backup Password Modal -->
    <a-modal
      v-model:visible="backupPasswordModalVisible"
      :title="t('system.backupPasswordTitle')"
      @ok="executeBackup"
      @cancel="backupPasswordModalVisible = false"
    >
      <p>{{ t('system.backupPasswordDesc') }}</p>
      <a-input-password
        v-model:value="backupPassword"
        :placeholder="t('system.passwordPlaceholder')"
      />
    </a-modal>

    <!-- Restore Password Modal -->
    <a-modal
      v-model:visible="restorePasswordModalVisible"
      :title="t('system.restorePasswordTitle')"
      @ok="executeRestore"
      @cancel="closeRestoreModal"
    >
      <p>{{ t('system.restorePasswordDesc') }}</p>
      <a-input-password
        v-model:value="restorePassword"
        :placeholder="t('system.passwordPlaceholder')"
      />
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { message, Modal } from 'ant-design-vue'
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons-vue'
import { useThemeStore } from '../stores/theme'
import api from '../api'
import { getWSTicket } from '../api/auth'

const { t } = useI18n()
const themeStore = useThemeStore()
const backupLoading = ref(false)
const restoreLoading = ref(false)
const settingsLoading = ref(false)

// Backup & Restore State
const backupPasswordModalVisible = ref(false)
const backupPassword = ref('')
const restorePasswordModalVisible = ref(false)
const restorePassword = ref('')
const restoreFile = ref(null)

const settingsForm = reactive({
  ssh_timeout: '30s',
  idle_timeout: '30m',
  max_connections_per_user: 10,
  login_rate_limit: 20,
  access_expiration: '60m',
  access_expiration: '60m',
  refresh_expiration: '168h',
  smtp_server: '',
  smtp_port: '',
  smtp_user: '',
  smtp_password: '',
  smtp_from: '',
  smtp_to: '',
  smtp_to: '',
  telegram_bot_token: '',
  telegram_chat_id: '',
  notification_template: ''
})

const DefaultNotificationTemplate = `{{emoji}}{{emoji}}{{emoji}}
Event: {{event}}
Clients: {{client}}
Message: {{message}}
Time: {{time}}`

const resetNotificationTemplate = () => {
    settingsForm.notification_template = DefaultNotificationTemplate
}

const fetchSettings = async () => {
  try {
    const response = await api.get('/system/settings')
    Object.assign(settingsForm, response)
    // Auto-fill default template if empty
    if (!settingsForm.notification_template) {
        settingsForm.notification_template = DefaultNotificationTemplate
    }
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

const handleBackup = () => {
  backupPassword.value = ''
  backupPasswordModalVisible.value = true
}

const executeBackup = async () => {
  backupPasswordModalVisible.value = false
  backupLoading.value = true
  try {
    const res = await getWSTicket()
    const ticket = res.ticket
    let downloadUrl = `/api/system/backup?token=${ticket}`
    if (backupPassword.value) {
      downloadUrl += `&password=${encodeURIComponent(backupPassword.value)}`
    }
    
    // Check if browser supports direct download via anchor
    // If we want to check for errors first, we might need fetch/blob approach, 
    // but for large files streaming via direct link is better.
    // If backend errors, it returns JSON which browser might try to download.
    // A better approach for error handling is doing a HEAD or simple check first,
    // but here we stick to simple anchor click.
    
    const link = document.createElement('a')
    link.href = downloadUrl
    // Don't set a static filename here if we want the server-provided one (from Content-Disposition)
    // But 'download' attribute is useful. We can try to guess or leave it empty to respect header.
    // However, if we set 'download', it forces download.
    // If we want to support dynamic naming from server, we should omit the filename in 'download' attribute 
    // or set it after checking headers (which requires fetch).
    // For now, let's just let it download.
    // link.setAttribute('download', '') 
    
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
  
  // Store file and show password modal
  restoreFile.value = info.file.originFileObj
  restorePassword.value = ''
  restorePasswordModalVisible.value = true
}

const closeRestoreModal = () => {
  restorePasswordModalVisible.value = false
  restoreFile.value = null
}

const executeRestore = () => {
  restorePasswordModalVisible.value = false
  if (restoreFile.value) {
    Modal.confirm({
      title: t('system.restoreConfirmTitle'),
      content: t('system.restoreConfirmContent'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => performRestore(restoreFile.value, restorePassword.value),
      onCancel: () => {
        restoreFile.value = null
      }
    })
  }
}

const performRestore = async (file, password) => {
  restoreLoading.value = true
  const formData = new FormData()
  formData.append('file', file)
  if (password) {
    formData.append('password', password)
  }

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
    restoreFile.value = null // Clear on success
  } catch (err) {
    // Check for incorrect password (403 Forbidden or specific message)
    if (err.response?.status === 403 || err.response?.data?.error === 'incorrect password') {
        message.error(t('system.incorrectPassword'))
        // Re-open modal for retry
        restorePasswordModalVisible.value = true
        // Do NOT clear restoreFile.value so we can retry with same file
    } else {
        message.error(err.response?.data?.error || t('system.restoreFailed'))
        restoreFile.value = null // Clear on other errors
    }
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
