<template>
  <div class="terminal-wrapper" style="height: 100%; display: flex; flex-direction: column; overflow: hidden">
    <div ref="terminalRef" class="terminal-container" style="flex: 1; overflow: hidden; background: #1e1e1e"></div>
    
    <div v-if="connectionStatus" class="terminal-status" style="padding: 4px 8px; background: #1f1f1f; border-top: 1px solid #303030; display: flex; justify-content: space-between; align-items: center; flex-shrink: 0">
      <div>
        <a-tag :color="statusColor" size="small" style="font-size: 10px; line-height: 16px; height: 18px">{{ connectionStatus }}</a-tag>
        <span style="margin-left: 12px; color: #888; font-size: 12px">{{ terminalSize }}</span>
      </div>
      <div>
        <a-space size="small">
          <a-button size="small" @click="reconnect" v-if="connectionStatus === 'Disconnected'">
            <ReloadOutlined />
            Reconnect
          </a-button>
          <a-button size="small" danger @click="disconnect" v-if="connectionStatus === 'Connected'">
            <DisconnectOutlined />
            Disconnect
          </a-button>
        </a-space>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { message } from 'ant-design-vue'
import { ReloadOutlined, DisconnectOutlined } from '@ant-design/icons-vue'
import { getWSTicket } from '../api/auth'
import 'xterm/css/xterm.css'

const props = defineProps({
  terminalId: {
    type: String,
    required: true
  },
  hostId: {
    type: [Number, String],
    required: true
  }
})

const emit = defineEmits(['close'])

const terminalRef = ref(null)
const terminal = ref(null)
const fitAddon = ref(null)
const ws = ref(null)
const connectionStatus = ref('Connecting...')
const terminalSize = ref('')

const statusColor = ref('processing')

watch(connectionStatus, (status) => {
  if (status === 'Connected') statusColor.value = 'success'
  else if (status === 'Disconnected') statusColor.value = 'error'
  else statusColor.value = 'processing'
})

onMounted(async () => {
  initTerminal()
  await connectWebSocket()
})

onUnmounted(() => {
  cleanup()
})

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
    const wsUrl = `${protocol}//${host}/api/ws/ssh/${props.hostId}?ticket=${ticket}`

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
</style>
