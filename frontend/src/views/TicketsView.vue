<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowDown, ArrowRight, ChevronLeft, ChevronRight, Plus, Search, TicketCheck } from 'lucide-vue-next'
import { api } from '../api/http'
import { useAuthStore } from '../stores/auth'
import StatusBadge from '../components/StatusBadge.vue'

const auth = useAuthStore()
const router = useRouter()
const canManage = computed(() => ['agent', 'admin'].includes(auth.user?.role))
const items = ref([])
const assignees = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const status = ref('')
const priority = ref('')
const assignee = ref('')
const search = ref('')
const busy = ref(true)
const error = ref('')
const assigneeError = ref('')
const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
let searchTimer

async function load() {
  busy.value = true
  error.value = ''
  try {
    const result = await api.listTickets({
      page: page.value,
      pageSize,
      status: status.value,
      priority: priority.value,
      assignee: assignee.value,
      q: search.value.trim(),
    })
    items.value = result?.items || []
    total.value = Number(result?.total || 0)
  } catch (e) { error.value = e.message }
  finally { busy.value = false }
}

async function loadAssignees() {
  if (!canManage.value) return
  try {
    const result = await api.agents()
    assignees.value = result?.items || []
  } catch (e) {
    assigneeError.value = e.message
  }
}

watch([status, priority, assignee], () => { page.value = 1; load() })
watch(search, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; load() }, 300)
})
watch(page, load)
onMounted(() => {
  load()
  loadAssignees()
})

function formatDate(value) {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(value))
}
</script>

<template>
  <section class="page-content">
    <div class="page-topline"><span>WORKSPACE <span class="crumb-slash">/</span> TICKETS</span><span class="topline-date">{{ new Intl.DateTimeFormat(undefined, { weekday: 'long', month: 'long', day: 'numeric' }).format(new Date()) }}</span></div>
    <div class="page-heading heading-with-action">
      <div><span class="eyebrow">YOUR SUPPORT DESK</span><h1>Tickets <span class="heading-count">{{ total }}</span></h1><p>Keep every conversation moving forward.</p></div>
      <button class="button button-primary" @click="router.push('/tickets/new')"><Plus :size="18" />New ticket</button>
    </div>
    <div class="toolbar">
      <label class="search-box"><Search :size="18" /><input v-model="search" placeholder="Search tickets…" aria-label="Search tickets" /></label>
      <div class="filter-group">
        <label class="select-wrap"><span class="sr-only">Filter by status</span><select v-model="status"><option value="">All statuses</option><option value="open">Open</option><option value="in_progress">In progress</option><option value="resolved">Resolved</option><option value="closed">Closed</option></select><ArrowDown :size="14" /></label>
        <label class="select-wrap"><span class="sr-only">Filter by priority</span><select v-model="priority"><option value="">All priorities</option><option value="urgent">Urgent</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option></select><ArrowDown :size="14" /></label>
        <label v-if="canManage" class="select-wrap"><span class="sr-only">Filter by assignee</span><select v-model="assignee"><option value="">All assignees</option><option value="unassigned">Unassigned</option><option v-for="person in assignees" :key="person.id" :value="person.id">{{ person.name }}</option></select><ArrowDown :size="14" /></label>
      </div>
    </div>
    <div v-if="error" class="alert error-alert page-alert"><span>{{ error }}</span><button class="text-button" @click="load">Try again</button></div>
    <div v-if="assigneeError" class="alert error-alert page-alert"><span>Unable to load assignees: {{ assigneeError }}</span><button class="text-button" @click="loadAssignees">Try again</button></div>
    <div class="ticket-table-wrap">
      <div class="ticket-table table-head"><span>SUBJECT</span><span>STATUS</span><span>PRIORITY</span><span>UPDATED</span><span></span></div>
      <div v-if="busy" class="loading-state"><span class="spinner"></span>Loading your tickets…</div>
      <div v-else-if="!items.length" class="empty-state"><div class="empty-icon"><TicketCheck :size="24" /></div><h3>{{ search || status || priority || assignee ? 'No matching tickets' : 'Nothing in the queue yet' }}</h3><p>{{ search || status || priority || assignee ? 'Try adjusting your filters or search.' : 'Create a ticket and we’ll help get things moving.' }}</p><button class="button button-primary button-small" @click="router.push('/tickets/new')"><Plus :size="16" />Create a ticket</button></div>
      <button v-for="ticket in items" v-else :key="ticket.id" class="ticket-row table-head" @click="router.push(`/tickets/${ticket.id}`)">
        <span class="ticket-subject"><strong>{{ ticket.title }}</strong><small>#{{ ticket.id }} <span>·</span> {{ ticket.category || 'General' }}</small></span>
        <span><StatusBadge :value="ticket.status" /></span>
        <span><StatusBadge :value="ticket.priority" kind="priority" /></span>
        <span class="date-cell">{{ formatDate(ticket.updatedAt) }}</span>
        <span class="row-arrow"><ArrowRight :size="16" /></span>
      </button>
      <div v-if="!busy && items.length" class="table-footer"><span>Showing <strong>{{ (page - 1) * pageSize + 1 }}–{{ Math.min(page * pageSize, total) }}</strong> of <strong>{{ total }}</strong> tickets</span><div class="pagination"><button class="icon-button" :disabled="page <= 1" aria-label="Previous page" @click="page--"><ChevronLeft :size="18" /></button><span>Page {{ page }} of {{ pages }}</span><button class="icon-button" :disabled="page >= pages" aria-label="Next page" @click="page++"><ChevronRight :size="18" /></button></div></div>
    </div>
    <p class="list-note" v-if="auth.user?.role === 'requester'">Your open conversations, all in one place.</p>
  </section>
</template>
