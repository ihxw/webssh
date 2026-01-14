<template>
  <a-config-provider :theme="{ algorithm: themeStore.themeAlgorithm, token: themeStore.themeToken }">
    <a-layout class="compact-layout" style="min-height: 100vh">
      <a-layout-header :style="{ background: themeStore.isDark ? '#1f1f1f' : '#fff', padding: '0 24px', borderBottom: themeStore.isDark ? '1px solid #303030' : '1px solid #f0f0f0', lineHeight: '48px', height: '48px' }">
        <div style="display: flex; align-items: center; justify-content: space-between">
          <div style="display: flex; align-items: center; gap: 24px">
            <div :style="{ color: themeStore.isDark ? '#fff' : '#001529', fontSize: '18px', fontWeight: '600', display: 'flex', alignItems: 'center' }">
              <CodeOutlined style="margin-right: 8px" />
              WebSSH
            </div>
            
            <a-menu
              v-model:selectedKeys="selectedKeys"
              mode="horizontal"
              :theme="themeStore.isDark ? 'dark' : 'light'"
              :style="{ flex: 1, minWidth: 0, background: 'transparent', border: 'none', lineHeight: '48px' }"
              @select="handleMenuSelect"
            >
              <a-menu-item key="terminal">
                <CodeOutlined />
                Terminal
              </a-menu-item>
              <a-menu-item key="hosts">
                <DatabaseOutlined />
                Hosts
              </a-menu-item>
              <a-menu-item key="history">
                <HistoryOutlined />
                History
              </a-menu-item>
              <a-menu-item key="users" v-if="authStore.isAdmin">
                <TeamOutlined />
                Users
              </a-menu-item>
            </a-menu>
          </div>

          <div style="display: flex; align-items: center; gap: 16px">
            <a-button size="small" @click="themeStore.toggleTheme" :icon="themeStore.isDark ? h(BulbOutlined) : h(BulbFilled)">
              {{ themeStore.isDark ? '浅色' : '深色' }}
            </a-button>

            <a-dropdown>
              <a class="ant-dropdown-link" @click.prevent :style="{ color: themeStore.isDark ? '#fff' : '#001529' }">
                <UserOutlined style="margin-right: 8px" />
                {{ authStore.user?.username }}
                <DownOutlined style="margin-left: 8px" />
              </a>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="profile" @click="router.push('/dashboard/profile')">
                    <UserOutlined />
                    Profile
                  </a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="logout" @click="handleLogout">
                    <LogoutOutlined />
                    Logout
                  </a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </div>
        </div>
      </a-layout-header>

      <a-layout-content :style="{ background: themeStore.isDark ? '#141414' : '#f0f2f5' }">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-config-provider>
</template>

<script setup>
import { ref, watch, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  CodeOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  TeamOutlined,
  UserOutlined,
  DownOutlined,
  LogoutOutlined,
  BulbOutlined,
  BulbFilled
} from '@ant-design/icons-vue'
import { useAuthStore } from '../stores/auth'
import { useThemeStore } from '../stores/theme'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const selectedKeys = ref(['terminal'])

// Initialize theme on mount
onMounted(() => {
  themeStore.initTheme()
})

// Update selected menu based on route
watch(() => route.path, (path) => {
  if (path.includes('/terminal')) selectedKeys.value = ['terminal']
  else if (path.includes('/hosts')) selectedKeys.value = ['hosts']
  else if (path.includes('/history')) selectedKeys.value = ['history']
  else if (path.includes('/users')) selectedKeys.value = ['users']
  else if (path.includes('/profile')) selectedKeys.value = ['profile']
}, { immediate: true })

const handleMenuSelect = ({ key }) => {
  router.push(`/dashboard/${key}`)
}

const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}
</script>
