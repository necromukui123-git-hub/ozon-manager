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
      // 业务操作路由（shop_admin 和 staff）
      {
        path: 'products',
        name: 'Products',
        component: () => import('@/views/products/ProductList.vue'),
        meta: { requiresBusinessRole: true }
      },
      {
        path: 'promotions/batch-enroll',
        name: 'BatchEnroll',
        component: () => import('@/views/promotions/BatchEnroll.vue'),
        meta: { requiresBusinessRole: true }
      },
      {
        path: 'promotions/loss-process',
        name: 'LossProcess',
        component: () => import('@/views/promotions/LossProcess.vue'),
        meta: { requiresBusinessRole: true }
      },
      {
        path: 'promotions/reprice',
        name: 'Reprice',
        component: () => import('@/views/promotions/Reprice.vue'),
        meta: { requiresBusinessRole: true }
      },
      // 店铺管理员专用路由
      {
        path: 'my/shops',
        name: 'MyShops',
        component: () => import('@/views/shop-admin/MyShops.vue'),
        meta: { requiresShopAdmin: true }
      },
      {
        path: 'my/staff',
        name: 'MyStaff',
        component: () => import('@/views/shop-admin/MyStaff.vue'),
        meta: { requiresShopAdmin: true }
      },
      // 系统管理员专用路由
      {
        path: 'admin/shop-admins',
        name: 'ShopAdminList',
        component: () => import('@/views/super-admin/ShopAdminList.vue'),
        meta: { requiresSuperAdmin: true }
      },
      {
        path: 'admin/overview',
        name: 'SystemOverview',
        component: () => import('@/views/super-admin/SystemOverview.vue'),
        meta: { requiresSuperAdmin: true }
      },
      // 旧版管理员路由（保持兼容）
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
  } else if (to.meta.requiresSuperAdmin && !userStore.isSuperAdmin) {
    next('/')
  } else if (to.meta.requiresShopAdmin && !userStore.isShopAdmin) {
    next('/')
  } else if (to.meta.requiresBusinessRole && !userStore.canOperateBusiness) {
    next('/')
  } else if (to.meta.requiresAdmin && !userStore.isAdmin) {
    next('/')
  } else if (to.path === '/login' && userStore.isLoggedIn) {
    next('/')
  } else {
    next()
  }
})

export default router
