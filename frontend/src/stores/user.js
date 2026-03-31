import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, getCurrentUser, logout, refreshSession } from '@/api/auth'

const AUTH_UPDATED_EVENT = 'auth:updated'
const AUTH_CLEARED_EVENT = 'auth:cleared'
const REFRESH_EARLY_MS = 2 * 60 * 1000

let authListenersBound = false

function readStoredUser() {
  try {
    return JSON.parse(localStorage.getItem('user') || 'null')
  } catch (error) {
    return null
  }
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const tokenExpiresAt = ref(localStorage.getItem('tokenExpiresAt') || '')
  const user = ref(readStoredUser())
  const currentShopId = ref(parseInt(localStorage.getItem('currentShopId')) || null)

  const isLoggedIn = computed(() => !!token.value)

  // 三层角色判断
  const isSuperAdmin = computed(() => user.value?.role === 'super_admin')
  const isShopAdmin = computed(() => user.value?.role === 'shop_admin')
  const isStaff = computed(() => user.value?.role === 'staff')

  // 是否可以执行业务操作（shop_admin 和 staff）
  const canOperateBusiness = computed(() => isShopAdmin.value || isStaff.value)

  // 是否可以管理店铺和员工（仅 shop_admin）
  const canManageShopAndStaff = computed(() => isShopAdmin.value)

  // 是否可以管理店铺管理员（仅 super_admin）
  const canManageShopAdmins = computed(() => isSuperAdmin.value)

  const userShops = computed(() => user.value?.shops || [])

  function syncFromStorage() {
    token.value = localStorage.getItem('token') || ''
    tokenExpiresAt.value = localStorage.getItem('tokenExpiresAt') || ''
    user.value = readStoredUser()
    currentShopId.value = parseInt(localStorage.getItem('currentShopId')) || null
  }

  if (typeof window !== 'undefined' && !authListenersBound) {
    window.addEventListener(AUTH_UPDATED_EVENT, syncFromStorage)
    window.addEventListener(AUTH_CLEARED_EVENT, syncFromStorage)
    authListenersBound = true
  }

  function ensureCurrentShop(shops) {
    if (!shops || shops.length === 0) {
      currentShopId.value = null
      localStorage.removeItem('currentShopId')
      return
    }

    const hasCurrentShop = shops.some(shop => shop.id === currentShopId.value)
    if (!hasCurrentShop) {
      setCurrentShop(shops[0].id)
    }
  }

  function persistAuth(payload) {
    token.value = payload?.token || ''
    tokenExpiresAt.value = payload?.token_expires_at || ''
    user.value = payload?.user || null

    if (token.value) {
      localStorage.setItem('token', token.value)
    } else {
      localStorage.removeItem('token')
    }

    if (tokenExpiresAt.value) {
      localStorage.setItem('tokenExpiresAt', tokenExpiresAt.value)
    } else {
      localStorage.removeItem('tokenExpiresAt')
    }

    if (user.value) {
      localStorage.setItem('user', JSON.stringify(user.value))
      ensureCurrentShop(user.value.shops)
    } else {
      localStorage.removeItem('user')
      localStorage.removeItem('currentShopId')
      currentShopId.value = null
    }

    window.dispatchEvent(new CustomEvent(AUTH_UPDATED_EVENT))
  }

  async function doLogin(username, password) {
    const res = await login(username, password)
    persistAuth(res.data)
    return res
  }

  async function fetchUser() {
    try {
      const res = await getCurrentUser()
      user.value = res.data
      localStorage.setItem('user', JSON.stringify(user.value))
      ensureCurrentShop(user.value?.shops)
    } catch (e) {
      await doLogout({ remote: false })
    }
  }

  async function refreshAccessToken() {
    const res = await refreshSession()
    persistAuth(res.data)
    return res
  }

  async function initializeAuth() {
    if (!token.value) {
      return
    }

    const expiresAt = Date.parse(tokenExpiresAt.value || '')
    const shouldRefresh = !expiresAt || Number.isNaN(expiresAt) || expiresAt - Date.now() <= REFRESH_EARLY_MS

    try {
      if (shouldRefresh) {
        await refreshAccessToken()
        return
      }

      if (!user.value) {
        await fetchUser()
      }
    } catch (error) {
      await doLogout({ remote: false })
    }
  }

  function clearAuthState() {
    token.value = ''
    tokenExpiresAt.value = ''
    user.value = null
    currentShopId.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('tokenExpiresAt')
    localStorage.removeItem('user')
    localStorage.removeItem('currentShopId')
    window.dispatchEvent(new Event(AUTH_CLEARED_EVENT))
  }

  async function doLogout(options = {}) {
    const { remote = true } = options

    const logoutPromise = remote
      ? logout().catch(error => {
        console.error(error)
      })
      : Promise.resolve()

    clearAuthState()
    await logoutPromise
  }

  function setCurrentShop(shopId) {
    currentShopId.value = shopId
    localStorage.setItem('currentShopId', shopId)
  }

  // 获取角色标签类型
  function getRoleTagType() {
    if (isSuperAdmin.value) return 'danger'
    if (isShopAdmin.value) return 'warning'
    return 'info'
  }

  // 获取角色显示名称
  function getRoleLabel() {
    if (isSuperAdmin.value) return '系统管理员'
    if (isShopAdmin.value) return '店铺管理员'
    return '员工'
  }

  return {
    token,
    tokenExpiresAt,
    user,
    currentShopId,
    isLoggedIn,
    isSuperAdmin,
    isShopAdmin,
    isStaff,
    canOperateBusiness,
    canManageShopAndStaff,
    canManageShopAdmins,
    userShops,
    doLogin,
    fetchUser,
    refreshAccessToken,
    initializeAuth,
    doLogout,
    setCurrentShop,
    getRoleTagType,
    getRoleLabel
  }
})
