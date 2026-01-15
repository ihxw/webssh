<template>
  <div class="terminal-wrapper" :style="{ 
    background: themeStore.isDark ? '#1e1e1e' : '#ffffff', 
    color: themeStore.isDark ? '#fff' : '#000',
    height: '100%', 
    display: 'flex', 
    flexDirection: 'column', 
    overflow: 'hidden'
  }">
    <div ref="terminalRef" class="terminal-container" :style="{ 
      background: themeStore.isDark ? '#1e1e1e' : '#ffffff',
      flex: 1,
      overflow: 'hidden'
    }"></div>
    
    <div v-if="connectionStatus" class="terminal-status" :style="{ 
      background: themeStore.isDark ? '#1f1f1f' : '#f0f0f0', 
      borderTop: themeStore.isDark ? '1px solid #303030' : '1px solid #d9d9d9' 
    }">
      <div style="display: flex; align-items: center">
        <a-tag :color="statusColor" size="small" style="font-size: 10px; line-height: 14px; height: 16px; margin-right: 8px">{{ connectionStatus }}</a-tag>
        <span :style="{ color: themeStore.isDark ? '#bbb' : '#666', fontSize: '11px', marginRight: '8px' }">{{ terminalSize }}</span>
        <div v-if="record" :style="{borderLeft: themeStore.isDark ? '1px solid #444' : '1px solid #ccc'}" style="display: flex; align-items: center; gap: 4px; padding-left: 8px; margin-left: 0">
          <span class="recording-dot"></span>
          <span style="color: #ff4d4f; font-size: 11px; font-weight: bold; letter-spacing: 0.5px">RECORDING</span>
        </div>
      </div>
      <div style="display: flex; align-items: center">
        <a-space size="small">
          <a-button class="status-btn" :class="{ 'light-mode': !themeStore.isDark }" size="small" type="text" @click="reconnect" v-if="connectionStatus === 'Disconnected'">
            <template #icon><ReloadOutlined /></template>
            Reconnect
          </a-button>
          <a-button class="status-btn danger" :class="{ 'light-mode': !themeStore.isDark }" size="small" type="text" danger @click="disconnect" v-if="connectionStatus === 'Connected'">
            <template #icon><DisconnectOutlined /></template>
            Disconnect
          </a-button>
        </a-space>
        <a-divider type="vertical" class="status-divider" :style="{ background: themeStore.isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)' }" />
        <a-button class="status-btn" :class="{ 'light-mode': !themeStore.isDark }" size="small" type="text" @click="showSftp = true" :disabled="connectionStatus !== 'Connected'">
          <template #icon><FolderOpenOutlined /></template>
          SFTP
        </a-button>
        <a-divider type="vertical" class="status-divider" :style="{ background: themeStore.isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(0, 0, 0, 0.1)' }" />
        <a-dropdown :disabled="connectionStatus !== 'Connected'" placement="topRight">
          <a-button class="status-btn" :class="{ 'light-mode': !themeStore.isDark }" size="small" type="text">
            <template #icon><ThunderboltOutlined /></template>
            Commands
          </a-button>
          <template #overlay>
            <a-menu @click="handleQuickCommand">
              <a-menu-item v-for="cmd in commandTemplates" :key="cmd.command">
                {{ cmd.name }}
              </a-menu-item>
              <a-menu-divider v-if="commandTemplates.length > 0" />
              <a-menu-item @click="$router.push({ name: 'CommandManagement' })">
                Manage Templates
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!-- SFTP Drawer -->
    <a-drawer
      v-model:open="showSftp"
      title="File Explorer"
      placement="right"
      :width="400"
      :body-style="{ padding: '8px' }"
    >
      <SftpBrowser :host-id="hostId" :visible="showSftp" />
    </a-drawer>
  </div>
</template>

<script setup>
import { ref, shallowRef, onMounted, onUnmounted, onActivated, nextTick, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { message } from 'ant-design-vue'
import { ReloadOutlined, DisconnectOutlined, FolderOpenOutlined, ThunderboltOutlined } from '@ant-design/icons-vue'
import { getWSTicket } from '../api/auth'
import { listCommandTemplates } from '../api/command'
import SftpBrowser from './SftpBrowser.vue'
import 'xterm/css/xterm.css'

import { useThemeStore } from '../stores/theme'

const props = defineProps({
  hostId: {
    type: [String, Number],
    required: true
  },
  active: {
    type: Boolean,
    default: false
  },
  record: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close'])
const themeStore = useThemeStore()

const terminalRef = ref(null)
const terminal = shallowRef(null)
const fitAddon = shallowRef(null)
const ws = ref(null)
const connectionStatus = ref('Connecting...')
const terminalSize = ref('80x24')
const showSftp = ref(false)
const commandTemplates = ref([])

const statusColor = ref('processing')

watch(() => themeStore.isDark, (isDark) => {
  if (terminal.value) {
    updateTerminalTheme(isDark)
  }
})

// ... watchers for active/status ...

const updateTerminalTheme = (isDark) => {
  if (!terminal.value) return
  
  const theme = isDark ? {
      background: '#1e1e1e',
      foreground: '#ffffff',
      cursor: '#ffffff',
      selection: '#ffffff40',
      black: '#000000',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#d19a66',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf',
      brightBlack: '#5c6370',
      brightRed: '#e06c75',
      brightGreen: '#98c379',
      brightYellow: '#d19a66',
      brightBlue: '#61afef',
      brightMagenta: '#c678dd',
      brightCyan: '#56b6c2',
      brightWhite: '#ffffff'
  } : {
      background: '#ffffff',
      foreground: '#333333',
      cursor: '#333333',
      selection: '#00000040',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#cd3131',
      brightGreen: '#14ce14',
      brightYellow: '#b5ba00',
      brightBlue: '#0451a5',
      brightMagenta: '#bc3fbc',
      brightCyan: '#0598bc',
      brightWhite: '#a5a5a5'
  }
  
  terminal.value.options.theme = theme
}

const handleQuickCommand = ({ key }) => {
  if (key && ws.value && ws.value.readyState === WebSocket.OPEN) {
    ws.value.send(JSON.stringify({ type: 'input', data: key + '\n' }))
  }
}

const loadCommands = async () => {
  try {
    const data = await listCommandTemplates()
    commandTemplates.value = data || []
  } catch (error) {
    console.error('Failed to load command templates:', error)
  }
}

const initTerminal = () => {
  // Create terminal instance
  terminal.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Courier New, monospace',
    theme: {}, // Will be set by updateTerminalTheme
    allowProposedApi: true,
    logLevel: 'info'
  })
  
  updateTerminalTheme(themeStore.isDark)

  // ... rest of init ...

  // Add fit addon
  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)

  // Add web links addon
  const webLinksAddon = new WebLinksAddon()
  terminal.value.loadAddon(webLinksAddon)

  // Open terminal in DOM
  terminal.value.open(terminalRef.value)

  // Fit terminal to container
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon.value && terminal.value) {
      // Ensure container has dimensions
      if (terminalRef.value && (terminalRef.value.clientWidth > 0 || terminalRef.value.clientHeight > 0)) {
         try {
           fitAddon.value.fit()
           updateTerminalSize()
           sendResize()
         } catch (e) {
           console.error('Fit error:', e)
         }
      }
    }
  })
  
  if (terminalRef.value) {
    resizeObserver.observe(terminalRef.value)
  }
  
  // Store observer to cleanup
  terminal.value._resizeObserver = resizeObserver

  // Handle window resize as backup
  window.addEventListener('resize', handleResize)

  // Handle terminal data input
  terminal.value.onData((data) => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type: 'input', data }))
    }
  })
}

const connectWebSocket = async () => {
  try {
    // 1. Get one-time ticket
    const response = await getWSTicket()
    const ticket = response.ticket

    // 2. Connect via WebSocket with ticket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/api/ws/ssh/${props.hostId}?ticket=${ticket}${props.record ? '&record=true' : ''}`
    
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      connectionStatus.value = 'Connected'
      message.success('SSH connection established')
      // Send initial resize
      sendResize()
      // Auto focus
      terminal.value.focus()
    }

    ws.value.onmessage = (event) => {
      // console.log('WS Message:', event.data)
      if (!terminal.value) return
      try {
        const msg = JSON.parse(event.data)
        // Only treat as structured message if it's an object with a 'type' field
        if (msg && typeof msg === 'object' && msg.type) {
          if (msg.type === 'error') {
            terminal.value.writeln(`\r\n\x1b[31mError: ${msg.data}\x1b[0m\r\n`)
            connectionStatus.value = 'Error'
          } else if (msg.type === 'connected') {
            terminal.value.writeln(`\r\n\x1b[32m${msg.data}\x1b[0m\r\n`)
          }
        } else {
          // If it's valid JSON but not our structured message (e.g. a single number '1')
          // write it as raw data
          terminal.value.write(event.data)
        }
      } catch (e) {
        // Not valid JSON, must be raw terminal output
        terminal.value.write(event.data)
      }
    }

    ws.value.onerror = (error) => {
      console.error('WebSocket error:', error)
      connectionStatus.value = 'Error'
      message.error('Connection error')
    }

    ws.value.onclose = () => {
      connectionStatus.value = 'Disconnected'
      if (terminal.value) {
        terminal.value.writeln('\r\n\x1b[33mConnection closed\x1b[0m\r\n')
      }
    }
  } catch (error) {
    console.error('Failed to get WS ticket:', error)
    connectionStatus.value = 'Error'
    message.error('Failed to authenticate WebSocket')
  }
}

const handleResize = () => {
  if (fitAddon.value && terminal.value) {
    try {
      fitAddon.value.fit()
      updateTerminalSize()
      sendResize()
    } catch (e) {
      console.error('Fit error:', e)
    }
  }
}

const updateTerminalSize = () => {
  if (terminal.value) {
    terminalSize.value = `${terminal.value.cols}x${terminal.value.rows}`
  }
}

const sendResize = () => {
  if (ws.value && ws.value.readyState === WebSocket.OPEN && terminal.value) {
    ws.value.send(JSON.stringify({
      type: 'resize',
      data: {
        cols: terminal.value.cols,
        rows: terminal.value.rows
      }
    }))
  }
}

const reconnect = async () => {
  cleanup()
  initTerminal()
  await connectWebSocket()
}

onMounted(async () => {
  initTerminal()
  await connectWebSocket()
  loadCommands()
})

const disconnect = () => {
  if (ws.value) {
    ws.value.close()
  }
}

const cleanup = () => {
  window.removeEventListener('resize', handleResize)
  
  if (terminal.value && terminal.value._resizeObserver) {
    terminal.value._resizeObserver.disconnect()
  }
  
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  
  if (terminal.value) {
    terminal.value.dispose()
    terminal.value = null
  }
}
</script>

<style scoped>
.terminal-wrapper {
  /* background managed by inline style */
}

.terminal-container {
  padding: 0;
  margin: 0;
}

:deep(.xterm) {
  padding: 4px;
}

.status-btn {
  padding: 0 7px !important;
  height: 24px !important;
  font-size: 14px !important;
  color: rgba(255, 255, 255, 0.85) !important;
  display: flex !important;
  align-items: center !important;
}

.status-btn:hover {
  color: #fff !important;
  background: rgba(255, 255, 255, 0.08) !important;
}

.status-btn.danger {
  color: #ff4d4f !important;
}

.status-btn.danger:hover {
  color: #ff7875 !important;
  background: rgba(255, 77, 79, 0.1) !important;
}

:deep(.status-btn .anticon) {
  font-size: 12px !important;
}

.status-divider {
  margin: 0 4px !important;
}

.status-btn.light-mode {
  color: rgba(0, 0, 0, 0.65) !important;
}

.status-btn.light-mode:hover {
  color: #000 !important;
  background: rgba(0, 0, 0, 0.05) !important;
}

.status-btn.danger.light-mode:hover {
  color: #ff4d4f !important;
  background: rgba(255, 77, 79, 0.1) !important;
}
</style>
