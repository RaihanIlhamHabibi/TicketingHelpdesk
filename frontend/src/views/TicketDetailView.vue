<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowUpRight, MessageSquare, Plus, Send, ShieldCheck } from 'lucide-vue-next'
import { api } from '../api/http'
import { useAuthStore } from '../stores/auth'
import StatusBadge from '../components/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const creating = computed(() => route.meta.create)
const canManage = computed(() => ['agent', 'admin'].includes(auth.user?.role))
const statusActions = computed(() => {
  const status = ticket.value?.status
  const role = auth.user?.role
  if (role === 'requester') {
    if (status === 'open') return ['closed']
    if (status === 'resolved') return ['closed', 'open']
    return []
  }
  if (role === 'agent') {
    if (status === 'open') return ['in_progress']
    if (status === 'in_progress') return ['resolved']
    if (status === 'resolved') return ['closed']
    return []
  }
  if (role === 'admin') {
    if (status === 'open') return ['in_progress', 'closed']
    if (status === 'in_progress') return ['resolved']
    if (status === 'resolved') return ['closed']
  }
  return []
})
const visibleComments = computed(() => comments.value.filter(comment => canManage.value || !comment.isInternal))
const ticket = ref(null)
const comments = ref([])
const people = ref([])
const busy = ref(true)
const saving = ref(false)
const error = ref('')
const notice = ref('')
const body = ref('')
const internal = ref(false)
const form = ref({ title: '', description: '', category: '', priority: 'medium' })

async function load() {
  if (creating.value) { busy.value = false; return }
  busy.value = true
  error.value = ''
  try {
    const result = await api.ticket(route.params.id)
    ticket.value = result.ticket
    comments.value = result.comments || []
    if (canManage.value) {
      const usersResult = await api.agents()
      people.value = usersResult?.items || []
    }
  } catch (e) { error.value = e.message }
  finally { busy.value = false }
}

watch(() => route.fullPath, load, { immediate: true })

async function create() {
  error.value = ''
  if (form.value.title.trim().length < 4) { error.value = 'Add a title with at least 4 characters.'; return }
  if (form.value.description.trim().length < 10) { error.value = 'Please describe the issue in at least 10 characters.'; return }
  if (!form.value.category.trim()) { error.value = 'Choose a category for this ticket.'; return }
  saving.value = true
  try {
    const result = await api.createTicket({
      ...form.value,
      title: form.value.title.trim(),
      description: form.value.description.trim(),
      category: form.value.category.trim(),
    })
    router.replace(`/tickets/${result.ticket.id}`)
  } catch (e) { error.value = e.message }
  finally { saving.value = false }
}

async function update(changes) {
  notice.value = ''
  error.value = ''
  try {
    const result = await api.updateTicket(ticket.value.id, changes)
    ticket.value = result.ticket
    notice.value = 'Ticket updated.'
    setTimeout(() => { notice.value = '' }, 2200)
  } catch (e) { error.value = e.message; await load() }
}

async function addComment() {
  error.value = ''
  if (body.value.trim().length < 1) { error.value = 'Write a message before sending.'; return }
  saving.value = true
  try {
    const result = await api.addComment(ticket.value.id, { body: body.value.trim(), isInternal: canManage.value && internal.value })
    comments.value.push(result.comment)
    body.value = ''
    internal.value = false
  } catch (e) { error.value = e.message }
  finally { saving.value = false }
}

function formatDate(value, withTime = false) {
  if (!value) return ''
  return new Intl.DateTimeFormat(undefined, withTime ? { month: 'long', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' } : { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(value))
}

function assignedName(id) {
  return people.value.find(person => String(person.id) === String(id))?.name || 'Unassigned'
}

function requesterName() {
  return ticket.value.requesterName ||
    people.value.find(person => String(person.id) === String(ticket.value.requesterId))?.name ||
    (String(ticket.value.requesterId) === String(auth.user?.id) ? auth.user?.name : 'Requester')
}
</script>

<template>
  <section class="page-content detail-page">
    <div class="page-topline"><span><RouterLink to="/tickets">TICKETS</RouterLink> <span class="crumb-slash">/</span> {{ creating ? 'NEW TICKET' : `#${route.params.id}` }}</span></div>
    <button class="back-link" @click="router.push('/tickets')"><ArrowLeft :size="16" />Back to tickets</button>
    <template v-if="creating">
      <div class="page-heading detail-heading"><span class="eyebrow">WE’RE HERE TO HELP</span><h1>Create a ticket</h1><p>Tell us what’s going on and we’ll take it from here.</p></div>
      <form class="panel create-panel form-stack" @submit.prevent="create">
        <div v-if="error" class="alert error-alert">{{ error }}</div>
        <label class="field"><span>Subject</span><input v-model="form.title" placeholder="A short summary of the issue" required minlength="4" maxlength="160" /></label>
        <label class="field"><span>Description</span><textarea v-model="form.description" rows="6" placeholder="Share the details that will help us understand…" required minlength="10"></textarea><small>Include any steps to reproduce or relevant details.</small></label>
        <div class="form-row">
          <label class="field"><span>Category</span><input v-model="form.category" placeholder="e.g. Billing, Account access" required maxlength="100" /></label>
          <label class="field"><span>Priority</span><select v-model="form.priority"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option><option value="urgent">Urgent</option></select></label>
        </div>
        <div class="form-actions"><button type="button" class="button button-quiet" @click="router.push('/tickets')">Cancel</button><button class="button button-primary" :disabled="saving"><Plus :size="17" />{{ saving ? 'Creating…' : 'Create ticket' }}</button></div>
      </form>
    </template>
    <div v-else-if="busy" class="loading-state standalone"><span class="spinner"></span>Loading ticket…</div>
    <div v-else-if="error && !ticket" class="panel detail-error"><div class="alert error-alert">{{ error }}</div><button class="button button-secondary" @click="load">Try again</button></div>
    <template v-else-if="ticket">
      <div class="detail-title-row"><div><div class="ticket-id">TICKET #{{ ticket.id }}</div><h1>{{ ticket.title }}</h1><p class="detail-meta">Created {{ formatDate(ticket.createdAt) }} <span>·</span> {{ ticket.category || 'General' }}</p></div><StatusBadge :value="ticket.status" /></div>
      <div v-if="error" class="alert error-alert page-alert">{{ error }}</div>
      <div v-if="notice" class="alert success-alert page-alert">{{ notice }}</div>
      <div class="detail-layout">
        <div class="conversation-column">
          <article class="panel description-panel"><div class="section-title"><span class="avatar avatar-small">{{ requesterName()?.slice(0, 1)?.toUpperCase() }}</span><div><strong>{{ requesterName() }}</strong><small>Request details</small></div><time>{{ formatDate(ticket.createdAt) }}</time></div><p class="description-copy">{{ ticket.description }}</p></article>
          <div class="conversation-heading"><div><h2>Conversation</h2><span>{{ visibleComments.length }} {{ visibleComments.length === 1 ? 'reply' : 'replies' }}</span></div><MessageSquare :size="19" /></div>
          <div v-if="!visibleComments.length" class="panel no-comments">No replies yet. Add a message to keep the conversation going.</div>
          <article v-for="comment in visibleComments" :key="comment.id" class="panel comment-card" :class="{ 'internal-comment': comment.isInternal }">
            <div class="section-title"><span class="avatar avatar-small">{{ comment.authorName?.slice(0, 1)?.toUpperCase() || 'T' }}</span><div><strong>{{ comment.authorName || 'Team member' }} <span v-if="comment.isInternal" class="internal-label"><ShieldCheck :size="13" />Internal note</span></strong><small>{{ comment.isInternal ? 'Visible to the support team' : 'Reply' }}</small></div><time>{{ formatDate(comment.createdAt, true) }}</time></div>
            <p class="description-copy">{{ comment.body }}</p>
          </article>
          <form class="panel reply-panel" @submit.prevent="addComment">
            <label class="field"><span>{{ internal ? 'Internal note' : 'Your reply' }}</span><textarea v-model="body" rows="4" :placeholder="internal ? 'Add a note for your team…' : 'Write a message…'"></textarea></label>
            <div class="reply-actions"><label v-if="canManage" class="check-control"><input v-model="internal" type="checkbox" /><ShieldCheck :size="15" />Internal note</label><span v-else class="reply-hint">Your reply will be visible to the support team.</span><button class="button button-primary button-small" :disabled="saving"><Send :size="15" />{{ saving ? 'Sending…' : 'Send reply' }}</button></div>
          </form>
        </div>
        <aside class="panel properties-panel">
          <div class="panel-heading"><h2>Ticket details</h2><ArrowUpRight :size="17" /></div>
          <div class="property"><span>Status</span><template v-if="statusActions.length"><select :value="ticket.status" @change="update({ status: $event.target.value })"><option :value="ticket.status">{{ ticket.status.replace('_', ' ') }}</option><option v-for="status in statusActions" :key="status" :value="status">{{ status.replace('_', ' ') }}</option></select></template><StatusBadge v-else :value="ticket.status" /></div>
          <div class="property"><span>Priority</span><template v-if="canManage"><select :value="ticket.priority" @change="update({ priority: $event.target.value })"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option><option value="urgent">Urgent</option></select></template><StatusBadge v-else :value="ticket.priority" kind="priority" /></div>
          <div class="property"><span>Assignee</span><template v-if="canManage"><select :value="ticket.assigneeId || ''" @change="update({ assigneeId: $event.target.value || null })"><option value="">Unassigned</option><option v-for="person in people.filter(p => ['agent', 'admin'].includes(p.role))" :key="person.id" :value="person.id">{{ person.name }}</option></select></template><strong v-else>{{ assignedName(ticket.assigneeId) }}</strong></div>
          <div class="property"><span>Category</span><strong>{{ ticket.category || 'General' }}</strong></div>
          <div class="property"><span>Requester</span><strong>{{ requesterName() }}</strong></div>
          <div class="property last-property"><span>Last updated</span><strong>{{ formatDate(ticket.updatedAt) }}</strong></div>
          <div class="property-note"><ShieldCheck :size="16" /><p>Replies and ticket updates are shared with the requester.</p></div>
        </aside>
      </div>
    </template>
  </section>
</template>
