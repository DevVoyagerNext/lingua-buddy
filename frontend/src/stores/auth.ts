import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, setToken, clearToken, getToken } from '@/api/client'

export interface UserView {
  id: number
  username: string
  email: string | null
  registration_method: string
  english_level: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserView | null>(null)
  const token = ref<string>(getToken())

  function isLoggedIn() {
    return !!token.value
  }

  async function register(username: string, password: string, email?: string) {
    const resp = await api.post<{ token: string; user: UserView }>('/auth/register', {
      username,
      password,
      email: email || undefined,
    })
    token.value = resp.data.token
    setToken(resp.data.token)
    user.value = resp.data.user
  }

  async function login(login: string, password: string) {
    const resp = await api.post<{ token: string; user: UserView }>('/auth/login', { login, password })
    token.value = resp.data.token
    setToken(resp.data.token)
    user.value = resp.data.user
  }

  async function fetchMe() {
    const resp = await api.get<UserView>('/users/me')
    user.value = resp.data
    return resp.data
  }

  async function updateProfile(payload: { email?: string; english_level?: string }) {
    const resp = await api.patch<UserView>('/users/me', payload)
    user.value = resp.data
    return resp.data
  }

  function logout() {
    token.value = ''
    user.value = null
    clearToken()
  }

  return { user, token, isLoggedIn, register, login, fetchMe, updateProfile, logout }
})
