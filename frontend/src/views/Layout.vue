<template>
  <div class="layout">
    <!-- 侧边栏 -->
    <aside class="layout-aside">
      <div class="logo">
        <div class="logo-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z"/>
            <path d="M3 6h18"/>
            <path d="M16 10a4 4 0 0 1-8 0"/>
          </svg>
        </div>
        <span class="logo-text">Ozon 管理</span>
      </div>

      <el-menu
        :default-active="currentRoute"
        router
        class="sidebar-menu"
      >
        <el-menu-item index="/">
          <el-icon><DataLine /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>

        <el-menu-item index="/products">
          <el-icon><Goods /></el-icon>
          <span>商品列表</span>
        </el-menu-item>

        <el-sub-menu index="promotions">
          <template #title>
            <el-icon><Promotion /></el-icon>
            <span>促销管理</span>
          </template>
          <el-menu-item index="/promotions/batch-enroll">批量报名</el-menu-item>
          <el-menu-item index="/promotions/loss-process">亏损处理</el-menu-item>
          <el-menu-item index="/promotions/reprice">改价推广</el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="admin">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统管理</span>
          </template>
          <el-menu-item index="/admin/shops">店铺管理</el-menu-item>
          <el-menu-item index="/admin/users">用户管理</el-menu-item>
          <el-menu-item index="/admin/logs">操作日志</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </aside>

    <!-- 主内容区 -->
    <main class="layout-main">
      <header class="layout-header">
        <div class="header-left">
          <el-select
            v-model="currentShopId"
            placeholder="选择店铺"
            style="width: 220px"
            @change="handleShopChange"
          >
            <template #prefix>
              <el-icon><Shop /></el-icon>
            </template>
            <el-option
              v-for="shop in shops"
              :key="shop.id"
              :label="shop.name"
              :value="shop.id"
            />
          </el-select>
        </div>

        <div class="header-right">
          <span class="user-name">{{ userStore.user?.display_name }}</span>
          <el-dropdown @command="handleCommand">
            <div class="user-avatar">
              <el-icon><User /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <div class="layout-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { getShops } from '@/api/shop'
import { DataLine, Goods, Promotion, Setting, User, Shop, SwitchButton } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const shops = ref([])
const currentShopId = ref(userStore.currentShopId)
const currentRoute = computed(() => route.path)

onMounted(async () => {
  await fetchShops()
})

async function fetchShops() {
  try {
    const res = await getShops()
    shops.value = res.data

    if (!currentShopId.value && shops.value.length > 0) {
      currentShopId.value = shops.value[0].id
      userStore.setCurrentShop(currentShopId.value)
    }
  } catch (error) {
    console.error(error)
  }
}

function handleShopChange(shopId) {
  userStore.setCurrentShop(shopId)
}

function handleCommand(command) {
  if (command === 'logout') {
    userStore.doLogout()
    router.push('/login')
  }
}
</script>

<style scoped>
.sidebar-menu {
  background: transparent !important;
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
