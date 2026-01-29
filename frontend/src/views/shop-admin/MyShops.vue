<template>
  <div class="my-shops">
    <div class="page-header">
      <h2 class="gradient">我的店铺</h2>
      <el-button type="primary" @click="showDialog()">
        <el-icon><Plus /></el-icon>
        添加店铺
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <StatCard
        :value="shops.length"
        label="店铺总数"
        :icon="Shop"
        variant="primary"
      />
      <StatCard
        :value="activeCount"
        label="正常运营"
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

    <BentoCard title="店铺列表" :icon="List" size="4x1" no-padding>
      <div class="card-body">
        <el-table :data="shops" v-loading="loading">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="店铺名称" width="150" />
          <el-table-column prop="client_id" label="Client ID" width="200">
            <template #default="{ row }">
              <span class="code-text">{{ row.client_id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="api_key" label="API Key" min-width="200">
            <template #default="{ row }">
              <div class="api-key-cell">
                <span class="code-text">{{ maskApiKey(row.api_key) }}</span>
                <el-button type="primary" size="small" text @click="copyApiKey(row.api_key)">
                  复制
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" effect="dark" size="small">
                {{ row.is_active ? '正常' : '禁用' }}
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
              <el-button type="primary" size="small" text @click="showEditDialog(row)">
                编辑
              </el-button>
              <el-button type="danger" size="small" text @click="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </BentoCard>

    <!-- 创建/编辑店铺对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑店铺' : '添加店铺'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="店铺名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入店铺名称" />
        </el-form-item>
        <el-form-item label="Client ID" prop="client_id">
          <el-input v-model="form.client_id" placeholder="请输入Ozon Client ID" />
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input
            v-model="form.api_key"
            type="password"
            show-password
            placeholder="请输入Ozon API Key"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">
          {{ isEdit ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Shop, CircleCheckFilled, WarningFilled, List } from '@element-plus/icons-vue'
import { getMyShops, createMyShop, updateMyShop, deleteMyShop } from '@/api/shopAdmin'
import { StatCard, BentoCard } from '@/components/bento'

const loading = ref(false)
const saving = ref(false)
const shops = ref([])

// 计算统计数据
const activeCount = computed(() => shops.value.filter(s => s.is_active).length)
const disabledCount = computed(() => shops.value.filter(s => !s.is_active).length)

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref(null)
const formRef = ref(null)
const form = reactive({
  name: '',
  client_id: '',
  api_key: ''
})
const rules = {
  name: [{ required: true, message: '请输入店铺名称', trigger: 'blur' }],
  client_id: [{ required: true, message: '请输入Client ID', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入API Key', trigger: 'blur' }]
}

onMounted(async () => {
  await fetchShops()
})

async function fetchShops() {
  loading.value = true
  try {
    const res = await getMyShops()
    shops.value = res.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

function showDialog() {
  isEdit.value = false
  editingId.value = null
  form.name = ''
  form.client_id = ''
  form.api_key = ''
  dialogVisible.value = true
}

function showEditDialog(shop) {
  isEdit.value = true
  editingId.value = shop.id
  form.name = shop.name
  form.client_id = shop.client_id
  form.api_key = shop.api_key
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    saving.value = true
    try {
      if (isEdit.value) {
        await updateMyShop(editingId.value, form)
        ElMessage.success('更新成功')
      } else {
        await createMyShop(form)
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

async function handleDelete(shop) {
  try {
    await ElMessageBox.confirm(
      `确定要删除店铺"${shop.name}"吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  try {
    await deleteMyShop(shop.id)
    ElMessage.success('删除成功')
    await fetchShops()
  } catch (error) {
    console.error(error)
  }
}

function maskApiKey(key) {
  if (!key || key.length < 10) return key
  return key.substring(0, 6) + '****' + key.substring(key.length - 4)
}

function copyApiKey(key) {
  navigator.clipboard.writeText(key)
  ElMessage.success('已复制到剪贴板')
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.my-shops {
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

.api-key-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
