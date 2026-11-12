<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, CircleHelp } from 'lucide-vue-next'
import { api } from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const name = ref('')
const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  if (name.value.trim().length < 2) { error.value = 'Your name must be at least 2 characters.'; return }
  if (password.value.length < 8) { error.value = 'Use a password with at least 8 characters.'; return }
  busy.value = true
  try {
    auth.setSession(await api.register({ name: name.value.trim(), email: email.value.trim(), password: password.value }))
    router.replace('/tickets')
  } catch (e) { error.value = e.message }
  finally { busy.value = false }
}
</script>

<template>
  <div class="auth-card">
    <RouterLink to="/login" class="brand auth-brand"><span class="brand-mark"><CircleHelp :size="21" /></span><span>relay<span class="brand-dot">.</span></span></RouterLink>
    <div class="auth-heading"><span class="eyebrow">GET STARTED</span><h1>Make support feel simple.</h1><p>Create your workspace account to get started.</p></div>
    <form class="form-stack" @submit.prevent="submit">
      <div v-if="error" class="alert error-alert">{{ error }}</div>
      <label class="field"><span>Your name</span><input v-model="name" autocomplete="name" placeholder="Alex Morgan" required minlength="2" /></label>
      <label class="field"><span>Email address</span><input v-model="email" type="email" autocomplete="email" placeholder="you@company.com" required /></label>
      <label class="field"><span>Password</span><input v-model="password" type="password" autocomplete="new-password" placeholder="At least 8 characters" required minlength="8" /></label>
      <button class="button button-primary button-wide" :disabled="busy">{{ busy ? 'Creating account…' : 'Create account' }}<ArrowRight :size="17" /></button>
    </form>
    <p class="auth-foot">Already have an account? <RouterLink to="/login">Sign in</RouterLink></p>
  </div>
</template>
