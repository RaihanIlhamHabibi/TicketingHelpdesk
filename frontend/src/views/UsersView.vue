<script setup>
import { computed, onMounted, ref } from 'vue'
import { ArrowDown, Search, UsersRound } from 'lucide-vue-next'
import { api } from '../api/http'

const users = ref([])
const search = ref('')
const busy = ref(true)
const error = ref('')
const savingId = ref(null)
const notice = ref('')
const filtered = computed(() => users.value.filter(user => `${user.name} ${user.email}`.toLowerCase().includes(search.value.toLowerCase())))

async function load() {
  busy.value = true
  error.value = ''
  try { users.value = (await api.users())?.items || [] }
  catch (e) { error.value = e.message }
  finally { busy.value = false }
}

async function changeRole(user, role) {
  if (user.role === role) return
  savingId.value = user.id
  error.value = ''
  notice.value = ''
  try {
    const result = await api.updateUserRole(user.id, role)
    const index = users.value.findIndex(item => item.id === user.id)
    if (index >= 0) users.value[index] = result.user
    notice.value = `Role updated for ${result.user.name}.`
    setTimeout(() => { notice.value = '' }, 2500)
  } catch (e) { error.value = e.message; await load() }
  finally { savingId.value = null }
}

onMounted(load)

function initial(name) { return name?.trim()?.[0]?.toUpperCase() || '?' }
function roleClass(role) { return `role-${role}` }
</script>

<template>
  <section class="page-content">
    <div class="page-topline"><span>WORKSPACE <span class="crumb-slash">/</span> PEOPLE</span></div>
    <div class="page-heading"><span class="eyebrow">TEAM & ACCESS</span><h1>People</h1><p>Manage your workspace members and their roles.</p></div>
    <div class="user-summary"><div class="summary-icon"><UsersRound :size="19" /></div><div><strong>{{ users.length }} {{ users.length === 1 ? 'member' : 'members' }}</strong><span>in your workspace</span></div></div>
    <div class="toolbar user-toolbar"><label class="search-box"><Search :size="18" /><input v-model="search" placeholder="Search people…" aria-label="Search people" /></label><span class="results-label">Workspace members</span></div>
    <div v-if="error" class="alert error-alert page-alert"><span>{{ error }}</span><button class="text-button" @click="load">Try again</button></div>
    <div v-if="notice" class="alert success-alert page-alert">{{ notice }}</div>
    <div class="ticket-table-wrap people-table">
      <div class="people-row people-head"><span>NAME</span><span>ROLE</span><span>JOINED</span><span></span></div>
      <div v-if="busy" class="loading-state"><span class="spinner"></span>Loading people…</div>
      <div v-else-if="!filtered.length" class="empty-state compact-empty"><div class="empty-icon"><UsersRound :size="23" /></div><h3>{{ search ? 'No matching people' : 'No team members yet' }}</h3><p>Try searching by name or email.</p></div>
      <div v-for="person in filtered" v-else :key="person.id" class="people-row">
        <div class="user-identity"><span class="avatar" :class="roleClass(person.role)">{{ initial(person.name) }}</span><div><strong>{{ person.name }}</strong><small>{{ person.email }}</small></div></div>
        <div><span class="role-badge" :class="roleClass(person.role)">{{ person.role }}</span></div>
        <div class="date-cell">{{ person.createdAt ? new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(person.createdAt)) : '—' }}</div>
        <label class="select-wrap role-select"><span class="sr-only">Role for {{ person.name }}</span><select :value="person.role" :disabled="savingId === person.id" @change="changeRole(person, $event.target.value)"><option value="requester">Requester</option><option value="agent">Agent</option><option value="admin">Admin</option></select><ArrowDown :size="14" /></label>
      </div>
    </div>
    <p class="list-note">Role changes take effect immediately.</p>
  </section>
</template>
