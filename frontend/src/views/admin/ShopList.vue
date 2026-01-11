<template>
  <div class="shop-list">
    <div class="page-header">
      <h2 class="gradient">店铺管理</h2>
      <el-button type="primary" @click="showDialog()">
        <el-icon><Plus /></el-icon>
        添加店铺
      </el-button>
    </div>

    <div class="glass-card">
      <div class="card-body">
        <el-table :data="shops" v-loading="loading">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="店铺名称" />
          <el-table-column prop="client_id" label="Client ID">
            <template #default="{ row }">
              <span class="code-text">{{ row.client_id }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" effect="dark" size="small">
                {{ row.is_active ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" align="center">
            <template #default="{ row }">
              <el-button type="primary" size="small" text @click="showDialog(row)">
                编辑
              </el-button>
              <el-button type="danger" size="small" text @click="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑店铺' : '添加店铺'"
      width="500px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="店铺名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入店铺名称" />
        </el-form-item>
        <el-form-item label="Client ID" prop="client_id">
          <el-input v-model="form.client_id" placeholder="请输入 Ozon Client ID" />
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input
            v-model="form.api_key"
            type="password"
            show-password
            :placeholder="isEdit ? '留空则不修改' : '请输入 Ozon API Key'"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getShops, createShop, updateShop, deleteShop } from '@/api/shop'

const loading = ref(false)
const saving = ref(false)
const shops = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const editId = ref(null)

const form = reactive({
  name: '',
  client_id: '',
  api_key: '',
  is_active: true
})

const rules = {
  name: [{ required: true, message: '请输入店铺名称', trigger: 'blur' }],
  client_id: [{ required: true, message: '请输入 Client ID', trigger: 'blur' }]
}

onMounted(() => {
  fetchShops()
})

async function fetchShops() {
  loading.value = true
  try {
    const res = await getShops()
    shops.value = res.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

function showDialog(row = null) {
  isEdit.value = !!row
  editId.value = row?.id || null

  if (row) {
    form.name = row.name
    form.client_id = row.client_id
    form.api_key = ''
    form.is_active = row.is_active
  } else {
    form.name = ''
    form.client_id = ''
    form.api_key = ''
    form.is_active = true
  }

  dialogVisible.value = true
}

async function handleSave() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    if (!isEdit.value && !form.api_key) {
      ElMessage.warning('请输入 API Key')
      return
    }

    saving.value = true
    try {
      const data = {
        name: form.name,
        client_id: form.client_id,
        is_active: form.is_active
      }

      if (form.api_key) {
        data.api_key = form.api_key
      }

      if (isEdit.value) {
        await updateShop(editId.value, data)
        ElMessage.success('更新成功')
      } else {
        await createShop(data)
        ElMessage.success('创建成功')
      }

      dialogVisible.value = false
      await fetchShops()
    } catch (error) {
      console.error(error)
    } finally {
      saving.value = false
    }
  })
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确定要删除店铺"${row.name}"吗？删除后无法恢复。`,
      '确认删除',
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
    await deleteShop(row.id)
    ElMessage.success('删除成功')
    await fetchShops()
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
.shop-list {
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
</style>
