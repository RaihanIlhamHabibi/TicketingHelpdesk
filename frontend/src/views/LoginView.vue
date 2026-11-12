<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowRight, CircleHelp } from 'lucide-vue-next'
import { api } from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  if (!email.value.trim() || !password.value) { error.value = 'Enter your email and password to continue.'; return }
  busy.value = true
  try {
    auth.setSession(await api.login({ email: email.value.trim(), password: password.value }))
    router.replace(String(route.query.redirect || '/tickets'))
  } catch (e) { error.value = e.message }
  finally { busy.value = false }
}
</script>

<template>
  <div class="auth-card">
    <RouterLink to="/login" class="brand auth-brand"><span class="brand-mark"><CircleHelp :size="21" /></span><span>relay<span class="brand-dot">.</span></span></RouterLink>
    <div class="auth-heading"><span class="eyebrow">WELCOME BACK</span><h1>Your support, in sync.</h1><p>Sign in to pick up where you left off.</p></div>
    <form class="form-stack" @submit.prevent="submit">
      <div v-if="error" class="alert error-alert">{{ error }}</div>
      <label class="field"><span>Email address</span><input v-model="email" type="email" autocomplete="email" placeholder="you@company.com" required /></label>
      <label class="field"><span>Password</span><input v-model="password" type="password" autocomplete="current-password" placeholder="Enter your password" required /></label>
      <button class="button button-primary button-wide" :disabled="busy">{{ busy ? 'Signing in…' : 'Sign in' }}<ArrowRight :size="17" /></button>
    </form>
    <p class="auth-foot">New to Relay? <RouterLink to="/register">Create an account</RouterLink></p>
  </div>
</template>
