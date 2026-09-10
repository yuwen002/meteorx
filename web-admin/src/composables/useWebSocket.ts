import { ref, onUnmounted, type Ref } from 'vue'
import { useUserStore } from '@/stores/user'

export interface WSMessage {
  type: 'alert' | 'announcement' | 'unread_count' | 'ping' | 'pong'
  payload: any
  time: number
}

// 重连配置
const RECONNECT_BASE_DELAY = 1000 // 基础延迟 1s
const RECONNECT_MAX_DELAY = 30000 // 最大延迟 30s
const RECONNECT_MAX_ATTEMPTS = 10 // 最大重连次数
const HEARTBEAT_INTERVAL = 25000 // 心跳间隔 25s
const HEARTBEAT_TIMEOUT = 10000 // 心跳超时 10s
const OFFLINE_QUEUE_MAX = 100 // 离线消息队列最大长度
const STORAGE_KEY = 'ws_offline_messages' // localStorage 存储键

export function useWebSocket() {
  const userStore = useUserStore()
  const ws: Ref<WebSocket | null> = ref(null)
  const connected = ref(false)
  const connecting = ref(false)
  const reconnectAttempts = ref(0)
  const lastConnectedTime = ref(0)
  const reconnectTimer: Ref<number | null> = ref(null)
  const heartbeatTimer: Ref<number | null> = ref(null)
  const messageHandlers: Array<(msg: WSMessage) => void> = []
  let offlineQueue: WSMessage[] = loadOfflineQueue()

  // 获取 WebSocket URL
  function getWebSocketURL(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    return `${protocol}//${host}/api/v1/ws`
  }

  // 连接
  function connect() {
    if (ws.value?.readyState === WebSocket.OPEN) return
    if (connecting.value) return
    if (!userStore.token) return

    connecting.value = true

    try {
      const url = `${getWebSocketURL()}?token=${userStore.token}`
      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        connected.value = true
        connecting.value = false
        reconnectAttempts.value = 0
        lastConnectedTime.value = Date.now()
        startHeartbeat()
        // 连接成功后发送离线消息
        flushOfflineQueue()
      }

      ws.value.onmessage = (event: MessageEvent) => {
        try {
          const msg: WSMessage = JSON.parse(event.data)
          // 处理心跳响应
          if (msg.type === 'pong') return
          handleMessage(msg)
        } catch (e) {
          console.error('[WS] Failed to parse message:', e)
        }
      }

      ws.value.onclose = (event: CloseEvent) => {
        connected.value = false
        connecting.value = false
        stopHeartbeat()
        // 非正常关闭时重连
        if (event.code !== 1000 && event.code !== 1001) {
          scheduleReconnect()
        }
      }

      ws.value.onerror = () => {
        connected.value = false
        connecting.value = false
        ws.value?.close()
      }
    } catch (e) {
      console.error('[WS] Connection failed:', e)
      connecting.value = false
      scheduleReconnect()
    }
  }

  // 断开连接
  function disconnect() {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
    reconnectAttempts.value = 0
    stopHeartbeat()
    if (ws.value) {
      ws.value.close(1000, 'Client disconnect')
      ws.value = null
    }
    connected.value = false
    connecting.value = false
  }

  // 发送消息
  function send(msg: WSMessage) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(msg))
    } else {
      // 离线时入队，等待重连后发送
      enqueueOffline(msg)
    }
  }

  // 注册消息处理器
  function onMessage(handler: (msg: WSMessage) => void) {
    messageHandlers.push(handler)
    return () => {
      const idx = messageHandlers.indexOf(handler)
      if (idx >= 0) messageHandlers.splice(idx, 1)
    }
  }

  // 处理消息
  function handleMessage(msg: WSMessage) {
    messageHandlers.forEach(handler => {
      try {
        handler(msg)
      } catch (e) {
        console.error('[WS] Handler error:', e)
      }
    })
  }

  // 心跳
  let lastPongTime = 0

  function startHeartbeat() {
    stopHeartbeat()
    lastPongTime = Date.now()
    heartbeatTimer.value = window.setInterval(() => {
      // 检测心跳超时
      if (Date.now() - lastPongTime > HEARTBEAT_INTERVAL + HEARTBEAT_TIMEOUT) {
        console.warn('[WS] Heartbeat timeout, reconnecting...')
        ws.value?.close()
        return
      }
      send({ type: 'ping', payload: {}, time: Date.now() })
    }, HEARTBEAT_INTERVAL)
  }

  function stopHeartbeat() {
    if (heartbeatTimer.value) {
      clearInterval(heartbeatTimer.value)
      heartbeatTimer.value = null
    }
  }

  // 指数退避重连
  function scheduleReconnect() {
    if (reconnectTimer.value) return
    if (reconnectAttempts.value >= RECONNECT_MAX_ATTEMPTS) {
      console.warn('[WS] Max reconnection attempts reached')
      return
    }

    const delay = Math.min(
      RECONNECT_BASE_DELAY * Math.pow(2, reconnectAttempts.value),
      RECONNECT_MAX_DELAY
    )
    // 添加随机抖动，避免大量客户端同时重连
    const jitter = Math.random() * 1000
    const totalDelay = delay + jitter

    reconnectAttempts.value++
    console.log(`[WS] Reconnecting in ${Math.round(totalDelay / 1000)}s (attempt ${reconnectAttempts.value}/${RECONNECT_MAX_ATTEMPTS})`)

    reconnectTimer.value = window.setTimeout(() => {
      reconnectTimer.value = null
      connect()
    }, totalDelay)
  }

  // 离线消息队列
  function enqueueOffline(msg: WSMessage) {
    if (offlineQueue.length >= OFFLINE_QUEUE_MAX) {
      offlineQueue.shift() // 移除最旧的消息
    }
    offlineQueue.push(msg)
    saveOfflineQueue()
  }

  function flushOfflineQueue() {
    if (offlineQueue.length === 0) return
    const queue = [...offlineQueue]
    offlineQueue = []
    saveOfflineQueue()
    // 依次发送离线消息
    queue.forEach(msg => {
      if (ws.value?.readyState === WebSocket.OPEN) {
        ws.value.send(JSON.stringify(msg))
      }
    })
  }

  function loadOfflineQueue(): WSMessage[] {
    try {
      const data = localStorage.getItem(STORAGE_KEY)
      if (data) return JSON.parse(data)
    } catch {
      // ignore
    }
    return []
  }

  function saveOfflineQueue() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(offlineQueue))
    } catch {
      // ignore
    }
  }

  // 清理
  onUnmounted(() => {
    disconnect()
    messageHandlers.length = 0
  })

  return {
    ws,
    connected,
    connecting,
    reconnectAttempts,
    lastConnectedTime,
    connect,
    disconnect,
    send,
    onMessage
  }
}