import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue')
      },
      {
        path: 'products',
        name: 'Products',
        component: () => import('@/views/products/ProductList.vue')
      },
      {
        path: 'promotions/batch-enroll',
        name: 'BatchEnroll',
        component: () => import('@/views/promotions/BatchEnroll.vue')
      },
      {
        path: 'promotions/loss-process',
        name: 'LossProcess',
        component: () => import('@/views/promotions/LossProcess.vue')
      },
      {
        path: 'promotions/reprice',
        name: 'Reprice',
        component: () => import('@/views/promotions/Reprice.vue')
      },
      {
        path: 'admin/shops',
        name: 'ShopList',
        component: () => import('@/views/admin/ShopList.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'admin/users',
        name: 'UserList',
        component: () => import('@/views/admin/UserList.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'admin/logs',
        name: 'OperationLogs',
        component: () => import('@/views/admin/OperationLogs.vue'),
        meta: { requiresAdmin: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth !== false && !userStore.isLoggedIn) {
    next('/login')
  } else if (to.meta.requiresAdmin && !userStore.isAdmin) {
    next('/')
  } else if (to.path === '/login' && userStore.isLoggedIn) {
    next('/')
  } else {
    next()
  }
})

export default router
