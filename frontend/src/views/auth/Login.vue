<template>
  <div class="login-container">
    <div class="login-card">
      <!-- Logo -->
      <div class="login-logo">
        <div class="logo-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z"/>
            <path d="M3 6h18"/>
            <path d="M16 10a4 4 0 0 1-8 0"/>
          </svg>
        </div>
      </div>

      <h2 class="login-title">Ozon 店铺管理</h2>
      <p class="login-subtitle">智能电商运营管理平台</p>

      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="请输入用户名"
            size="large"
          >
            <template #prefix>
              <el-icon><User /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            size="large"
            show-password
            @keyup.enter="handleLogin"
          >
            <template #prefix>
              <el-icon><Lock /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-button
          type="primary"
          size="large"
          class="login-btn"
          :loading="loading"
          @click="handleLogin"
        >
          <span v-if="!loading">登 录</span>
          <span v-else>登录中...</span>
        </el-button>
      </el-form>

      <div class="login-footer">
        <span class="version">v1.0.0</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { hashPassword } from '@/utils/crypto'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref(null)
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
}

async function handleLogin() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      // 在发送前使用 SHA-256 哈希密码
      const hashedPassword = hashPassword(form.password)

      // 立即清空明文密码
      const originalPassword = form.password
      form.password = ''

      await userStore.doLogin(form.username, hashedPassword)
      ElMessage.success('登录成功')
      router.push('/')
    } catch (error) {
      console.error(error)
      // 登录失败时不恢复密码,让用户重新输入
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.login-footer {
  margin-top: 32px;
  text-align: center;
}

.version {
  font-size: 12px;
  color: var(--text-disabled);
}
</style>
