import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, getCurrentUser, logout } from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const currentShopId = ref(parseInt(localStorage.getItem('currentShopId')) || null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const userShops = computed(() => user.value?.shops || [])

  async function doLogin(username, password) {
    const res = await login(username, password)
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', token.value)
    localStorage.setItem('user', JSON.stringify(user.value))

    // 设置默认店铺
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

  return {
    token,
    user,
    currentShopId,
    isLoggedIn,
    isAdmin,
    userShops,
    doLogin,
    fetchUser,
    doLogout,
    setCurrentShop
  }
})
