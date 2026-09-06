import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus/es/components/message/index'
import router from '@/router'
import { useUserStore } from '@/stores/user'

const service: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000
})

// 请求拦截器：自动注入 JWT token
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：统一处理返回格式和错误
service.interceptors.response.use(
  (response: AxiosResponse) => {
    const res = response.data
    // 你后端返回的是 { code, message, data } 或类似结构
    // 如果有业务错误，这里提示
    if (res && typeof res === 'object' && 'code' in res && res.code !== 0 && res.code !== 200) {
      if (res.code === 401 || res.code === 40001) {
        handleLogoutSync()
      } else {
        ElMessage.error(res.message || '请求失败')
      }
      return Promise.reject(new Error(res.message || 'Error'))
    }
    // 兼容分页响应 { code, data, pagination }：合并保留，由 toPageResult 统一归一化
    if (res && typeof res === 'object' && 'pagination' in res && res.pagination) {
      return { data: res.data, pagination: res.pagination }
    }
    // 统一返回 data 字段内容，避免每个页面都写 res.data.xxx
    return res.data ?? res
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      if (status === 401) {
        handleLogoutSync()
      } else {
        const msg = error.response.data?.message || error.response.statusText || `HTTP ${status}`
        ElMessage.error(msg)
      }
    } else {
      ElMessage.error(error.message || '网络请求失败')
    }
    return Promise.reject(error)
  }
)

// 同步处理登出（用于拦截器中，不等待 API 响应）
function handleLogoutSync() {
  const userStore = useUserStore()
  // 不等待异步 logout API 调用，直接清理本地状态
  if (userStore.logoutSync) {
    userStore.logoutSync()
  } else {
    userStore.logout()
  }
  ElMessage.warning('登录状态已过期，请重新登录')
  router.push('/login')
}

// 通用类型封装：后端返回 { code, message, data }
export interface ApiResult<T = any> {
  code: number
  message: string
  data: T
}

export function request<T = any>(config: AxiosRequestConfig): Promise<T> {
  return service.request<any, T>(config)
}

export function get<T = any>(url: string, params?: any, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ url, method: 'GET', params, ...config })
}

export function post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ url, method: 'POST', data, ...config })
}

export function put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ url, method: 'PUT', data, ...config })
}

export function del<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ url, method: 'DELETE', ...config })
}

export default service