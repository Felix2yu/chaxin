import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: () => import('./views/Dashboard.vue') },
    { path: '/repos', name: 'repos', component: () => import('./views/Repos.vue') },
    { path: '/notifications', name: 'notifications', component: () => import('./views/Notifications.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/Settings.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

export default router
