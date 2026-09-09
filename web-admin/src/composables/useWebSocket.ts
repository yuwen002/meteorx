import { ref, onUnmounted, type Ref } from 'vue'
import { useUserStore } from '@/stores/user'

export interface WSMessage {
  type: 'alert' | 'announcement' | 'unread_count' | 'ping' | 'pong'
  payload: any
  time: number
}

export function useWebSocket() {
  const userStore = useUserStore()
  const ws: Ref<WebSocket | null> = ref(null)
  const connected = ref(false)
  const reconnectTimer: Ref<number | null> = ref(null)
  const heartbeatTimer: Ref<number | null> = ref(null)
  let messageHandlers: Array<(msg: WSMessage) => void> = []

  // 获取 WebSocket URL
  function getWebSocketURL(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    return `${protocol}//${host}/api/v1/ws`
  }

  // 连接
  function connect() {
    if (ws.value?.readyState === WebSocket.OPEN) return
    if (!userStore.token) return

    try {
      const url = `${getWebSocketURL()}?token=${userStore.token}`
      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        connected.value = true
        startHeartbeat()
      }

      ws.value.onmessage = (event: MessageEvent) => {
        try {
          const msg: WSMessage = JSON.parse(event.data)
          handleMessage(msg)
        } catch (e) {
          console.error('[WS] Failed to parse message:', e)
        }
      }

      ws.value.onclose = () => {
        connected.value = false
        stopHeartbeat()
        scheduleReconnect()
      }

      ws.value.onerror = () => {
        connected.value = false
        ws.value?.close()
      }
    } catch (e) {
      console.error('[WS] Connection failed:', e)
      scheduleReconnect()
    }
  }

  // 断开连接
  function disconnect() {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
    stopHeartbeat()
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    connected.value = false
  }

  // 发送消息
  function send(msg: WSMessage) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(msg))
    }
  }

  // 注册消息处理器
  function onMessage(handler: (msg: WSMessage) => void) {
    messageHandlers.push(handler)
    return () => {
      messageHandlers = messageHandlers.filter(h => h !== handler)
    }
  }

  // 处理消息
  function handleMessage(msg: WSMessage) {
    messageHandlers.forEach(handler => handler(msg))
  }

  // 心跳
  function startHeartbeat() {
    stopHeartbeat()
    heartbeatTimer.value = window.setInterval(() => {
      send({ type: 'ping', payload: {}, time: Date.now() })
    }, 25000)
  }

  function stopHeartbeat() {
    if (heartbeatTimer.value) {
      clearInterval(heartbeatTimer.value)
      heartbeatTimer.value = null
    }
  }

  // 自动重连
  function scheduleReconnect() {
    if (reconnectTimer.value) return
    reconnectTimer.value = window.setTimeout(() => {
      reconnectTimer.value = null
      connect()
    }, 5000)
  }

  // 清理
  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    connected,
    connect,
    disconnect,
    send,
    onMessage
  }
}