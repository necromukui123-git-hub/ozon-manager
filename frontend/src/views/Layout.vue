<template>
  <div class="layout" :class="{ 'layout--mobile-open': mobileMenuOpen }">
    <!-- 移动端遮罩 -->
    <div v-if="mobileMenuOpen" class="layout-overlay" @click="closeMobileMenu"></div>

    <!-- 侧边栏 -->
    <aside class="layout-aside" :class="{ 'open': mobileMenuOpen }">
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
        @select="handleMenuSelect"
      >
        <!-- 仪表盘 - 所有用户可见 -->
        <el-menu-item index="/">
          <el-icon><DataLine /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>

        <!-- 系统管理员专用菜单 -->
        <template v-if="userStore.isSuperAdmin">
          <el-menu-item index="/admin/shop-admins">
            <el-icon><UserFilled /></el-icon>
            <span>管理员管理</span>
          </el-menu-item>
          <el-menu-item index="/admin/overview">
            <el-icon><DataAnalysis /></el-icon>
            <span>系统概览</span>
          </el-menu-item>
        </template>

        <!-- 店铺管理员专用菜单 -->
        <template v-if="userStore.isShopAdmin">
          <el-sub-menu index="my-management">
            <template #title>
              <el-icon><Management /></el-icon>
              <span>我的管理</span>
            </template>
            <el-menu-item index="/my/shops">我的店铺</el-menu-item>
            <el-menu-item index="/my/staff">我的员工</el-menu-item>
          </el-sub-menu>
        </template>

        <!-- 业务操作菜单 - shop_admin 和 staff 可见 -->
        <template v-if="userStore.canOperateBusiness">
          <el-menu-item index="/products">
            <el-icon><Goods /></el-icon>
            <span>商品列表</span>
          </el-menu-item>

          <el-sub-menu index="promotions">
            <template #title>
              <el-icon><Promotion /></el-icon>
              <span>促销管理</span>
            </template>
            <el-menu-item index="/promotions/actions">活动列表</el-menu-item>
            <el-menu-item index="/promotions/batch-enroll">批量报名</el-menu-item>
            <el-menu-item index="/promotions/loss-process">亏损处理</el-menu-item>
            <el-menu-item index="/promotions/reprice">改价推广</el-menu-item>
          </el-sub-menu>
        </template>

        <!-- 操作日志 - 仅 shop_admin 可见 -->
        <el-menu-item v-if="userStore.isShopAdmin" index="/admin/logs">
          <el-icon><Document /></el-icon>
          <span>操作日志</span>
        </el-menu-item>
      </el-menu>
    </aside>

    <!-- 主内容区 -->
    <main class="layout-main">
      <header class="layout-header">
        <div class="header-left">
          <!-- 移动端菜单按钮 -->
          <button class="mobile-menu-btn" @click="toggleMobileMenu">
            <el-icon :size="22">
              <Fold v-if="mobileMenuOpen" />
              <Expand v-else />
            </el-icon>
          </button>

          <!-- 店铺选择器 - 仅业务用户可见 -->
          <el-select
            v-if="userStore.canOperateBusiness"
            v-model="currentShopId"
            placeholder="选择店铺"
            class="shop-selector"
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
          <!-- 系统管理员提示 -->
          <div v-else class="admin-hint">
            <el-icon><InfoFilled /></el-icon>
            <span>系统管理员模式</span>
          </div>
        </div>

        <div class="header-right">
          <el-select
            v-model="selectedTheme"
            size="small"
            class="theme-switcher"
            aria-label="主题切换"
            @change="handleThemeChange"
          >
            <el-option
              v-for="item in themeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
          <el-tag :type="userStore.getRoleTagType()" effect="dark" size="small" class="role-tag">
            {{ userStore.getRoleLabel() }}
          </el-tag>
          <span class="user-name">{{ userStore.user?.display_name }}</span>
          <el-dropdown @command="handleCommand">
            <div class="user-avatar">
              <el-icon><User /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="changePassword">
                  <el-icon><Lock /></el-icon>
                  修改密码
                </el-dropdown-item>
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

    <!-- 修改密码对话框 -->
    <el-dialog v-model="passwordDialogVisible" title="修改密码" width="400px">
      <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="80px">
        <el-form-item label="原密码" prop="old_password">
          <el-input
            v-model="passwordForm.old_password"
            type="password"
            show-password
            placeholder="请输入原密码"
          />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="passwordForm.new_password"
            type="password"
            show-password
            placeholder="请输入新密码（至少6位）"
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input
            v-model="passwordForm.confirm_password"
            type="password"
            show-password
            placeholder="请再次输入新密码"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="passwordLoading" @click="handleChangePassword">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { getShops } from '@/api/shop'
import { changePassword } from '@/api/user'
import { hashPassword } from '@/utils/crypto'
import { THEMES, getTheme, applyTheme } from '@/utils/theme'
import { ElMessage } from 'element-plus'
import {
  DataLine, Goods, Promotion, Document, User, Shop, SwitchButton, Lock,
  UserFilled, DataAnalysis, Management, InfoFilled, Fold, Expand
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const shops = ref([])
const currentShopId = ref(userStore.currentShopId)
const currentRoute = computed(() => route.path)
const themeOptions = THEMES
const selectedTheme = getTheme()

// 移动端菜单状态
const mobileMenuOpen = ref(false)

// 修改密码相关
const passwordDialogVisible = ref(false)
const passwordLoading = ref(false)
const passwordFormRef = ref(null)
const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.new_password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为6位', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 响应式处理
function handleResize() {
  if (window.innerWidth > 768) {
    mobileMenuOpen.value = false
  }
}

onMounted(async () => {
  await fetchShops()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
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
  } else if (command === 'changePassword') {
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm_password = ''
    passwordDialogVisible.value = true
  }
}

async function handleChangePassword() {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (!valid) return

    passwordLoading.value = true
    try {
      await changePassword(
        hashPassword(passwordForm.old_password),
        hashPassword(passwordForm.new_password)
      )
      ElMessage.success('密码修改成功')
      passwordDialogVisible.value = false
    } catch (error) {
      console.error(error)
    } finally {
      passwordLoading.value = false
    }
  })
}

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

function handleMenuSelect() {
  // 移动端选择菜单后自动关闭
  if (window.innerWidth <= 768) {
    mobileMenuOpen.value = false
  }
}
function handleThemeChange(theme) {
  applyTheme(theme)
}
</script>

<style scoped>
.sidebar-menu {
  background: transparent !important;
}

.admin-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--bg-tertiary);
  border-radius: 8px;
  color: var(--text-muted);
  font-size: 14px;
}

.role-tag {
  margin-right: 12px;
}

.theme-switcher {
  width: 110px;
  margin-right: 12px;
}

/* 移动端菜单按钮 */
.mobile-menu-btn {
  display: none;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--bg-tertiary);
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: 2px 2px 0 var(--neo-border-color);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-normal);
}

.mobile-menu-btn:hover {
  background: var(--surface-bg-hover);
  color: var(--primary);
  box-shadow: 3px 3px 0 var(--neo-border-color);
}

/* 移动端遮罩 */
.layout-overlay {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 99;
}

/* 店铺选择器 */
.shop-selector {
  width: 220px;
}

/* 响应式 */
@media (max-width: 768px) {
  .mobile-menu-btn {
    display: flex;
  }

  .layout-overlay {
    display: block;
  }

  .shop-selector {
    width: 160px;
  }

  .user-name {
    display: none;
  }

  .role-tag {
    display: none;
  }

  .theme-switcher {
    width: 90px;
    margin-right: 8px;
  }

  .admin-hint span {
    display: none;
  }
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
