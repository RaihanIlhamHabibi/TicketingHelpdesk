<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CircleHelp, LayoutDashboard, LogOut, Menu, UsersRound } from 'lucide-vue-next'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const plain = computed(() => route.meta.plain)
const active = computed(() => route.path.startsWith('/users') ? 'users' : 'tickets')

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div v-if="plain" class="auth-shell"><RouterView /></div>
  <div v-else class="app-shell">
    <aside class="sidebar">
      <RouterLink to="/tickets" class="brand"><span class="brand-mark"><CircleHelp :size="21" /></span><span>relay<span class="brand-dot">.</span></span></RouterLink>
      <div class="workspace-label">WORKSPACE</div>
      <nav class="side-nav">
        <RouterLink to="/tickets" class="nav-link" :class="{ selected: active === 'tickets' }"><LayoutDashboard :size="18" />Tickets</RouterLink>
        <RouterLink v-if="auth.user?.role === 'admin'" to="/users" class="nav-link" :class="{ selected: active === 'users' }"><UsersRound :size="18" />People</RouterLink>
      </nav>
      <div class="sidebar-bottom">
        <div class="profile">
          <div class="avatar">{{ auth.user?.name?.slice(0, 1)?.toUpperCase() || 'U' }}</div>
          <div class="profile-copy"><strong>{{ auth.user?.name }}</strong><span>{{ auth.user?.role }}</span></div>
          <button class="icon-button logout-button" title="Sign out" @click="logout"><LogOut :size="17" /></button>
        </div>
      </div>
    </aside>
    <main class="main-area">
      <header class="mobile-header">
        <RouterLink to="/tickets" class="brand"><span class="brand-mark"><CircleHelp :size="19" /></span><span>relay<span class="brand-dot">.</span></span></RouterLink>
        <button class="icon-button" title="Sign out" @click="logout"><LogOut :size="18" /></button>
      </header>
      <RouterView />
    </main>
  </div>
</template>
