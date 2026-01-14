<template>
  <div class="terminal-wrapper" style="height: 100%; display: flex; flex-direction: column; overflow: hidden">
    <div ref="terminalRef" class="terminal-container" style="flex: 1; overflow: hidden; background: #1e1e1e"></div>
    
    <div v-if="connectionStatus" class="terminal-status" style="padding: 2px 8px; background: #1f1f1f; border-top: 1px solid #303030; display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; min-height: 28px">
      <div>
        <a-tag :color="statusColor" size="small" style="font-size: 10px; line-height: 14px; height: 16px; margin-right: 8px">{{ connectionStatus }}</a-tag>
        <span style="color: #bbb; font-size: 11px; margin-right: 8px">{{ terminalSize }}</span>
        <a-switch v-model:checked="isRecordingEnabled" size="small" :disabled="connectionStatus === 'Connected'" />
        <span style="color: #bbb; font-size: 10px; margin-left: 4px">Record</span>
      </div>
      <div style="display: flex; align-items: center">
        <a-space size="small">
          <a-button class="status-btn" size="small" type="text" @click="reconnect" v-if="connectionStatus === 'Disconnected'">
            <template #icon><ReloadOutlined /></template>
            Reconnect
          </a-button>
          <a-button class="status-btn danger" size="small" type="text" danger @click="disconnect" v-if="connectionStatus === 'Connected'">
            <template #icon><DisconnectOutlined /></template>
            Disconnect
          </a-button>
        </a-space>
        <a-divider type="vertical" class="status-divider" />
        <a-button class="status-btn" size="small" type="text" @click="showSftp = true" :disabled="connectionStatus !== 'Connected'">
          <template #icon><FolderOpenOutlined /></template>
          SFTP
        </a-button>
        <a-divider type="vertical" class="status-divider" />
        <a-dropdown :disabled="connectionStatus !== 'Connected'" placement="topRight">
          <a-button class="status-btn" size="small" type="text">
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
import { ref, onMounted, onUnmounted, onActivated, nextTick, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { message } from 'ant-design-vue'
import { ReloadOutlined, DisconnectOutlined, FolderOpenOutlined, ThunderboltOutlined } from '@ant-design/icons-vue'
import { getWSTicket } from '../api/auth'
import { listCommandTemplates } from '../api/command'
import SftpBrowser from './SftpBrowser.vue'
import 'xterm/css/xterm.css'

const props = defineProps({
  hostId: {
    type: [String, Number],
    required: true
  }
})

const emit = defineEmits(['close'])

const terminalRef = ref(null)
const terminal = ref(null)
const fitAddon = ref(null)
const ws = ref(null)
const connectionStatus = ref('Connecting...')
const terminalSize = ref('80x24')
const showSftp = ref(false)
const commandTemplates = ref([])
const isRecordingEnabled = ref(false)

const statusColor = ref('processing')

watch(connectionStatus, (status) => {
  if (status === 'Connected') statusColor.value = 'success'
  else if (status === 'Disconnected') statusColor.value = 'error'
  else statusColor.value = 'processing'
})

onMounted(async () => {
  initTerminal()
  await connectWebSocket()
  loadCommands()
})

onUnmounted(() => {
  cleanup()
})

onActivated(() => {
  // Refit terminal on activation (e.g. from keep-alive)
  nextTick(() => {
    if (fitAddon.value) {
      fitAddon.value.fit()
      updateTerminalSize()
    }
  })
})

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
    theme: {
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
    },
    allowProposedApi: true
  })

  // Add fit addon
  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)

  // Add web links addon
  const webLinksAddon = new WebLinksAddon()
  terminal.value.loadAddon(webLinksAddon)

  // Open terminal in DOM
  terminal.value.open(terminalRef.value)

  // Fit terminal to container
  nextTick(() => {
    if (fitAddon.value) {
      fitAddon.value.fit()
      updateTerminalSize()
    }
  })

  // Handle window resize
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
    const wsUrl = `${protocol}//${host}/api/ws/ssh/${props.hostId}?ticket=${ticket}${isRecordingEnabled.value ? '&record=true' : ''}`
    
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      connectionStatus.value = 'Connected'
      message.success('SSH connection established')
      // Send initial resize
      sendResize()
    }

    ws.value.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'error') {
          terminal.value.writeln(`\r\n\x1b[31mError: ${msg.data}\x1b[0m\r\n`)
          connectionStatus.value = 'Error'
        } else if (msg.type === 'connected') {
          terminal.value.writeln(`\r\n\x1b[32m${msg.data}\x1b[0m\r\n`)
        }
      } catch (e) {
        // Plain text output
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
      terminal.value.writeln('\r\n\x1b[33mConnection closed\x1b[0m\r\n')
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

const disconnect = () => {
  if (ws.value) {
    ws.value.close()
  }
}

const cleanup = () => {
  window.removeEventListener('resize', handleResize)
  
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
  background: #1e1e1e;
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
  background: rgba(255, 255, 255, 0.2) !important;
  margin: 0 4px !important;
}
</style>
