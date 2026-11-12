import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { api } from './api/http'
import LoginView from './views/LoginView.vue'
import RegisterView from './views/RegisterView.vue'
import TicketsView from './views/TicketsView.vue'
import TicketDetailView from './views/TicketDetailView.vue'
import UsersView from './views/UsersView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { guest: true, plain: true } },
    { path: '/register', name: 'register', component: RegisterView, meta: { guest: true, plain: true } },
    { path: '/', redirect: '/tickets' },
    { path: '/tickets', name: 'tickets', component: TicketsView, meta: { auth: true } },
    { path: '/tickets/new', name: 'ticket-new', component: TicketDetailView, meta: { auth: true, create: true } },
    { path: '/tickets/:id', name: 'ticket-detail', component: TicketDetailView, meta: { auth: true } },
    { path: '/users', name: 'users', component: UsersView, meta: { auth: true, roles: ['admin'] } },
    { path: '/:pathMatch(.*)*', redirect: '/tickets' },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.token && !auth.initialized) {
    try {
      auth.setUser(await api.me())
    } catch {
      auth.logout()
    }
    auth.initialized = true
  }
  if (to.meta.auth && !auth.token) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta.guest && auth.token) return { name: 'tickets' }
  if (to.meta.roles && !to.meta.roles.includes(auth.user?.role)) return { name: 'tickets' }
})

export default router
