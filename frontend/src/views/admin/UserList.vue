<template>
  <div class="user-list">
    <div class="page-header">
      <h2 class="gradient">用户管理</h2>
      <el-button type="primary" @click="showDialog()">
        <el-icon><Plus /></el-icon>
        添加用户
      </el-button>
    </div>

    <div class="glass-card">
      <div class="card-body">
        <el-table :data="users" v-loading="loading">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="{ row }">
              <span class="code-text">{{ row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="display_name" label="显示名称" width="120" />
          <el-table-column prop="role" label="角色" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" effect="dark" size="small">
                {{ row.role === 'admin' ? '管理员' : '员工' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="dark" size="small">
                {{ row.status === 'active' ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="可访问店铺" min-width="200">
            <template #default="{ row }">
              <template v-if="row.role === 'admin'">
                <el-tag size="small" type="warning" effect="plain">全部店铺</el-tag>
              </template>
              <template v-else-if="row.shops && row.shops.length > 0">
                <div class="shop-tags">
                  <el-tag
                    v-for="shop in row.shops"
                    :key="shop.id"
                    size="small"
                  >
                    {{ shop.name }}
                  </el-tag>
                </div>
              </template>
              <span v-else class="no-data">未分配</span>
            </template>
          </el-table-column>
          <el-table-column prop="last_login_at" label="最后登录" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.last_login_at) || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" align="center">
            <template #default="{ row }">
              <el-button type="primary" size="small" text @click="showShopDialog(row)">
                分配店铺
              </el-button>
              <el-button type="warning" size="small" text @click="showPasswordDialog(row)">
                重置密码
              </el-button>
              <el-button
                :type="row.status === 'active' ? 'danger' : 'success'"
                size="small"
                text
                @click="toggleStatus(row)"
              >
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 创建用户对话框 -->
    <el-dialog v-model="dialogVisible" title="添加用户" width="500px">
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
          <el-input v-model="form.display_name" placeholder="请输入显示名称（员工姓名）" />
        </el-form-item>
        <el-form-item label="分配店铺">
          <el-select v-model="form.shop_ids" multiple placeholder="选择店铺" style="width: 100%">
            <el-option
              v-for="shop in allShops"
              :key="shop.id"
              :label="shop.name"
              :value="shop.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreate">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 分配店铺对话框 -->
    <el-dialog v-model="shopDialogVisible" title="分配店铺" width="500px">
      <el-form label-width="100px">
        <el-form-item label="用户">
          <span class="user-info">{{ editingUser?.display_name }} ({{ editingUser?.username }})</span>
        </el-form-item>
        <el-form-item label="可访问店铺">
          <el-select v-model="selectedShopIds" multiple placeholder="选择店铺" style="width: 100%">
            <el-option
              v-for="shop in allShops"
              :key="shop.id"
              :label="shop.name"
              :value="shop.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="shopDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleUpdateShops">
          保存
        </el-button>
      </template>
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getUsers, createUser, updateUserStatus, updateUserPassword, updateUserShops } from '@/api/user'
import { getShops } from '@/api/shop'

const loading = ref(false)
const saving = ref(false)
const users = ref([])
const allShops = ref([])

const dialogVisible = ref(false)
const formRef = ref(null)
const form = reactive({
  username: '',
  password: '',
  display_name: '',
  shop_ids: []
})
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  display_name: [{ required: true, message: '请输入显示名称', trigger: 'blur' }]
}

const shopDialogVisible = ref(false)
const editingUser = ref(null)
const selectedShopIds = ref([])

const passwordDialogVisible = ref(false)
const passwordFormRef = ref(null)
const passwordForm = reactive({
  new_password: ''
})
const passwordRules = {
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }]
}

onMounted(async () => {
  await Promise.all([fetchUsers(), fetchShops()])
})

async function fetchUsers() {
  loading.value = true
  try {
    const res = await getUsers()
    users.value = res.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function fetchShops() {
  try {
    const res = await getShops()
    allShops.value = res.data || []
  } catch (error) {
    console.error(error)
  }
}

function showDialog() {
  form.username = ''
  form.password = ''
  form.display_name = ''
  form.shop_ids = []
  dialogVisible.value = true
}

async function handleCreate() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    saving.value = true
    try {
      await createUser(form)
      ElMessage.success('创建成功')
      dialogVisible.value = false
      await fetchUsers()
    } catch (error) {
      console.error(error)
    } finally {
      saving.value = false
    }
  })
}

function showShopDialog(user) {
  editingUser.value = user
  selectedShopIds.value = user.shops?.map(s => s.id) || []
  shopDialogVisible.value = true
}

async function handleUpdateShops() {
  saving.value = true
  try {
    await updateUserShops(editingUser.value.id, selectedShopIds.value)
    ElMessage.success('更新成功')
    shopDialogVisible.value = false
    await fetchUsers()
  } catch (error) {
    console.error(error)
  } finally {
    saving.value = false
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
      await updateUserPassword(editingUser.value.id, passwordForm.new_password)
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
      `确定要${action}用户"${user.display_name}"吗？`,
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
    await updateUserStatus(user.id, newStatus)
    ElMessage.success(`${action}成功`)
    await fetchUsers()
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
.user-list {
  min-height: 100%;
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

.no-data {
  color: var(--text-disabled);
}

.shop-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.user-info {
  color: var(--text-primary);
  font-weight: 500;
}
</style>
