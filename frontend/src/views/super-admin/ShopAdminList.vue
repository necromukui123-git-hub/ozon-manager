<template>
  <div class="shop-admin-list">
    <div class="page-header">
      <h2 class="gradient">店铺管理员管理</h2>
      <el-button type="primary" @click="showDialog()">
        <el-icon><Plus /></el-icon>
        添加店铺管理员
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <StatCard
        :value="shopAdmins.length"
        label="管理员总数"
        :icon="UserFilled"
        variant="primary"
      />
      <StatCard
        :value="activeCount"
        label="正常状态"
        :icon="CircleCheckFilled"
        variant="success"
      />
      <StatCard
        :value="disabledCount"
        label="已禁用"
        :icon="WarningFilled"
        variant="warning"
      />
    </div>

    <BentoCard title="管理员列表" :icon="List" size="4x1" no-padding>
      <div class="card-body">
        <el-table :data="shopAdmins" v-loading="loading">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="{ row }">
              <span class="code-text">{{ row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="display_name" label="显示名称" width="120" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="dark" size="small">
                {{ row.status === 'active' ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="店铺数量" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="primary" effect="plain" size="small">
                {{ row.shop_count || 0 }} 个店铺
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="员工数量" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain" size="small">
                {{ row.staff_count || 0 }} 名员工
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="last_login_at" label="最后登录" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.last_login_at) || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center">
            <template #default="{ row }">
              <el-dropdown trigger="click">
                <el-button type="primary" size="small">
                  操作 <el-icon><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="showDetailDialog(row)">查看详情</el-dropdown-item>
                    <el-dropdown-item @click="showPasswordDialog(row)">重置密码</el-dropdown-item>
                    <el-dropdown-item divided @click="toggleStatus(row)">
                      {{ row.status === 'active' ? '禁用账号' : '启用账号' }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </BentoCard>

    <!-- 创建店铺管理员对话框 -->
    <el-dialog v-model="dialogVisible" title="添加店铺管理员" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            placeholder="请输入密码"
          />
        </el-form-item>
        <el-form-item label="显示名称" prop="display_name">
          <el-input v-model="form.display_name" placeholder="请输入显示名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreate">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看详情对话框 -->
    <el-dialog v-model="detailDialogVisible" title="店铺管理员详情" width="600px">
      <div v-loading="detailLoading">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="用户名">{{ detail?.username }}</el-descriptions-item>
          <el-descriptions-item label="显示名称">{{ detail?.display_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="detail?.status === 'active' ? 'success' : 'info'" effect="dark" size="small">
              {{ detail?.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(detail?.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <div class="section-title">店铺列表 ({{ detail?.shops?.length || 0 }})</div>
        <el-table :data="detail?.shops || []" size="small" max-height="200">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="店铺名称" />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                {{ row.is_active ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>

        <div class="section-title">员工列表 ({{ detail?.staff?.length || 0 }})</div>
        <el-table :data="detail?.staff || []" size="small" max-height="200">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="username" label="用户名" />
          <el-table-column prop="display_name" label="显示名称" />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                {{ row.status === 'active' ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="passwordDialogVisible" title="重置密码" width="400px">
      <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="100px">
        <el-form-item label="用户">
          <span class="user-info">{{ editingUser?.display_name }} ({{ editingUser?.username }})</span>
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="passwordForm.new_password"
            type="password"
            show-password
            placeholder="请输入新密码"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleResetPassword">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowDown, UserFilled, CircleCheckFilled, WarningFilled, List } from '@element-plus/icons-vue'
import {
  getShopAdmins,
  getShopAdmin,
  createShopAdmin,
  updateShopAdminStatus,
  resetShopAdminPassword
} from '@/api/admin'
import { hashPassword } from '@/utils/crypto'
import { StatCard, BentoCard } from '@/components/bento'

const loading = ref(false)
const saving = ref(false)
const shopAdmins = ref([])

// 计算统计数据
const activeCount = computed(() => shopAdmins.value.filter(s => s.status === 'active').length)
const disabledCount = computed(() => shopAdmins.value.filter(s => s.status !== 'active').length)

const dialogVisible = ref(false)
const formRef = ref(null)
const form = reactive({
  username: '',
  password: '',
  display_name: ''
})
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  display_name: [{ required: true, message: '请输入显示名称', trigger: 'blur' }]
}

const detailDialogVisible = ref(false)
const detailLoading = ref(false)
const detail = ref(null)

const passwordDialogVisible = ref(false)
const passwordFormRef = ref(null)
const editingUser = ref(null)
const passwordForm = reactive({
  new_password: ''
})
const passwordRules = {
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }]
}

onMounted(async () => {
  await fetchShopAdmins()
})

async function fetchShopAdmins() {
  loading.value = true
  try {
    const res = await getShopAdmins()
    shopAdmins.value = res.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

function showDialog() {
  form.username = ''
  form.password = ''
  form.display_name = ''
  dialogVisible.value = true
}

async function handleCreate() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    saving.value = true
    try {
      await createShopAdmin({
        ...form,
        password: hashPassword(form.password)
      })
      ElMessage.success('创建成功')
      dialogVisible.value = false
      await fetchShopAdmins()
    } catch (error) {
      console.error(error)
    } finally {
      saving.value = false
    }
  })
}

async function showDetailDialog(user) {
  detailDialogVisible.value = true
  detailLoading.value = true
  try {
    const res = await getShopAdmin(user.id)
    detail.value = res.data
  } catch (error) {
    console.error(error)
  } finally {
    detailLoading.value = false
  }
}

function showPasswordDialog(user) {
  editingUser.value = user
  passwordForm.new_password = ''
  passwordDialogVisible.value = true
}

async function handleResetPassword() {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (!valid) return

    saving.value = true
    try {
      await resetShopAdminPassword(editingUser.value.id, hashPassword(passwordForm.new_password))
      ElMessage.success('密码重置成功')
      passwordDialogVisible.value = false
    } catch (error) {
      console.error(error)
    } finally {
      saving.value = false
    }
  })
}

async function toggleStatus(user) {
  const newStatus = user.status === 'active' ? 'disabled' : 'active'
  const action = newStatus === 'disabled' ? '禁用' : '启用'

  try {
    await ElMessageBox.confirm(
      `确定要${action}店铺管理员"${user.display_name}"吗？`,
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  try {
    await updateShopAdminStatus(user.id, newStatus)
    ElMessage.success(`${action}成功`)
    await fetchShopAdmins()
  } catch (error) {
    console.error(error)
  }
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.shop-admin-list {
  min-height: 100%;
}

/* 统计卡片行 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

@media (max-width: 992px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}

.code-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}

.user-info {
  color: var(--text-primary);
  font-weight: 500;
}

.section-title {
  margin: 20px 0 10px;
  font-weight: 600;
  color: var(--text-primary);
}
</style>
