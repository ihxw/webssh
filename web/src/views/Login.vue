<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-title">
        <CodeOutlined style="font-size: 32px; margin-bottom: 8px" />
        <div>WebSSH</div>
      </div>
      
      <a-form
        :model="formState"
        @finish="handleLogin"
        layout="vertical"
      >
        <a-form-item
          label="Username"
          name="username"
          :rules="[{ required: true, message: 'Please input your username!' }]"
        >
          <a-input
            v-model:value="formState.username"
            placeholder="Enter username or email"
            size="small"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </a-input>
        </a-form-item>

        <a-form-item
          label="Password"
          name="password"
          :rules="[{ required: true, message: 'Please input your password!' }]"
        >
          <a-input-password
            v-model:value="formState.password"
            placeholder="Enter password"
            size="small"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </a-input-password>
        </a-form-item>

        <a-form-item>
          <a-checkbox v-model:checked="formState.remember">
            Remember me
          </a-checkbox>
        </a-form-item>

        <a-form-item>
          <a-button
            type="primary"
            html-type="submit"
            size="small"
            block
            :loading="loading"
          >
            Login
          </a-button>
        </a-form-item>
      </a-form>

      <a-alert
        v-if="error"
        :message="error"
        type="error"
        closable
        @close="error = ''"
        style="margin-top: 16px"
      />
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, CodeOutlined } from '@ant-design/icons-vue'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formState = reactive({
  username: '',
  password: '',
  remember: false
})

const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''

  try {
    await authStore.login(formState.username, formState.password, formState.remember)
    message.success('Login successful!')
    
    // Redirect to the page user was trying to access or dashboard terminal
    if (route.query.redirect) {
      router.push(route.query.redirect)
    } else {
      router.push({ name: 'Terminal' })
    }
  } catch (err) {
    error.value = err.response?.data?.error || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}
</script>
