import { defineStore } from 'pinia'
import { ref } from 'vue'

const TOKEN_KEY = 'relay_token'
const USER_KEY = 'relay_user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  const initialized = ref(false)

  function setSession(session) {
    token.value = session.token
    user.value = session.user
    localStorage.setItem(TOKEN_KEY, session.token)
    localStorage.setItem(USER_KEY, JSON.stringify(session.user))
    initialized.value = true
  }

  function setUser(value) {
    user.value = value
    localStorage.setItem(USER_KEY, JSON.stringify(value))
  }

  function logout() {
    token.value = ''
    user.value = null
    initialized.value = true
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  return { token, user, initialized, setSession, setUser, logout }
})
