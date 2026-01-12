import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, getCurrentUser, logout } from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
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

  async function doLogin(username, password) {
    const res = await login(username, password)
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', token.value)
    localStorage.setItem('user', JSON.stringify(user.value))

    // 设置默认店铺（仅业务用户需要）
    if (res.data.user.shops && res.data.user.shops.length > 0) {
      setCurrentShop(res.data.user.shops[0].id)
    }

    return res
  }

  async function fetchUser() {
    try {
      const res = await getCurrentUser()
      user.value = res.data
      localStorage.setItem('user', JSON.stringify(user.value))
    } catch (e) {
      doLogout()
    }
  }

  function doLogout() {
    token.value = ''
    user.value = null
    currentShopId.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('currentShopId')
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
    doLogout,
    setCurrentShop,
    getRoleTagType,
    getRoleLabel
  }
})
