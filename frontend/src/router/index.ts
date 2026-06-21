import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { getToken } from '@/api/client'

const routes: RouteRecordRaw[] = [
  { path: '/login', component: () => import('@/pages/Login.vue'), meta: { public: true } },
  { path: '/register', component: () => import('@/pages/Register.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', component: () => import('@/pages/Dashboard.vue') },
      { path: 'onboarding', component: () => import('@/pages/Onboarding.vue') },
      { path: 'dictionary', component: () => import('@/pages/Dictionary.vue') },
      { path: 'vocabulary', component: () => import('@/pages/Vocabulary.vue') },
      { path: 'word-plans', component: () => import('@/pages/WordPlans.vue') },
      { path: 'word-learning', component: () => import('@/pages/WordLearning.vue') },
      { path: 'review', component: () => import('@/pages/WordLearning.vue') },
      { path: 'sentences', component: () => import('@/pages/Sentences.vue') },
      { path: 'translate', component: () => import('@/pages/Translate.vue') },
      { path: 'speech', component: () => import('@/pages/Speech.vue') },
      { path: 'grammar', component: () => import('@/pages/Grammar.vue') },
      { path: 'history', component: () => import('@/pages/History.vue') },
      { path: 'settings', component: () => import('@/pages/Settings.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  if (!to.meta.public && !getToken()) {
    return '/login'
  }
  return true
})

export default router
