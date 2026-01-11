<template>
  <div class="layout">
    <aside class="layout-aside">
      <div class="logo">Ozon 管理</div>

      <el-menu
        :default-active="currentRoute"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
        router
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

    <main class="layout-main">
      <header class="layout-header">
        <div class="header-left">
          <el-select
            v-model="currentShopId"
            placeholder="选择店铺"
            style="width: 200px"
            @change="handleShopChange"
          >
            <el-option
              v-for="shop in shops"
              :key="shop.id"
              :label="shop.name"
              :value="shop.id"
            />
          </el-select>
        </div>

        <div class="header-right">
          <span>{{ userStore.user?.display_name }}</span>
          <el-dropdown @command="handleCommand">
            <el-button circle>
              <el-icon><User /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <div class="layout-content">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { getShops } from '@/api/shop'
import { DataLine, Goods, Promotion, Setting, User } from '@element-plus/icons-vue'

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

    // 如果没有选中店铺，默认选第一个
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
.layout {
  display: flex;
  min-height: 100vh;
}

.layout-aside {
  width: 220px;
  background-color: #304156;
  flex-shrink: 0;
}

.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  font-size: 20px;
  font-weight: bold;
  color: #fff;
  background-color: #263445;
}

.el-menu {
  border-right: none;
}

.layout-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.layout-header {
  height: 60px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  flex-shrink: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 15px;
}

.layout-content {
  flex: 1;
  overflow: auto;
  background-color: #f0f2f5;
}
</style>