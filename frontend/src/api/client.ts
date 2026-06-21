import axios, { type AxiosInstance } from 'axios'

// 统一响应体。
export interface ApiResponse<T = any> {
  code: string
  message: string
  data: T
}

export interface PageResult<T> {
  items: T[]
  page: number
  page_size: number
  total: number
}

// 业务错误：携带后端错误码，供页面区分处理（如 NO_DUE_WORDS）。
export class ApiError extends Error {
  code: string
  data: any
  constructor(code: string, message: string, data?: any) {
    super(message)
    this.code = code
    this.data = data
  }
}

const TOKEN_KEY = 'lb_token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}
export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

const http: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 70000,
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (resp) => resp,
  (error) => {
    if (error.response?.status === 401) {
      clearToken()
      if (location.hash !== '#/login' && !location.hash.startsWith('#/register')) {
        location.hash = '#/login'
      }
    }
    return Promise.reject(error)
  },
)

// request 解包统一响应，非 OK code 抛 ApiError；对“业务状态码”（NO_DUE_WORDS 等）也走 data 返回。
export async function request<T = any>(
  method: 'get' | 'post' | 'patch' | 'delete',
  url: string,
  body?: any,
  config?: any,
): Promise<ApiResponse<T>> {
  try {
    const resp = await http.request<ApiResponse<T>>({ method, url, data: body, ...config })
    return resp.data
  } catch (e: any) {
    const r = e.response?.data
    if (r && r.code) {
      throw new ApiError(r.code, r.message || '请求失败', r.data)
    }
    throw new ApiError('NETWORK_ERROR', e.message || '网络错误')
  }
}

export const api = {
  get: <T = any>(url: string, config?: any) => request<T>('get', url, undefined, config),
  post: <T = any>(url: string, body?: any, config?: any) => request<T>('post', url, body, config),
  patch: <T = any>(url: string, body?: any) => request<T>('patch', url, body),
  delete: <T = any>(url: string) => request<T>('delete', url),
}
