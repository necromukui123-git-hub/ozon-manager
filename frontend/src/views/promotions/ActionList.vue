<template>
  <div class="action-list">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">促销活动管理</h2>
      <div class="header-actions">
        <el-button type="primary" :loading="syncing" @click="handleSync">
          <el-icon><Refresh /></el-icon>
          同步活动
        </el-button>
        <el-button @click="showManualDialog = true">
          <el-icon><Plus /></el-icon>
          手动添加
        </el-button>
      </div>
    </div>

    <!-- 活动列表 -->
    <div class="glass-card">
      <div class="card-header">
        <span class="card-title">活动列表</span>
        <span class="card-subtitle">共 {{ actions.length }} 个活动</span>
      </div>
      <div class="card-body">
        <el-table :data="actions" v-loading="loading" empty-text="暂无活动，请先同步或手动添加">
          <el-table-column prop="action_id" label="活动ID" width="120">
            <template #default="{ row }">
              <span class="action-id">{{ row.action_id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="活动名称" min-width="200">
            <template #default="{ row }">
              <div class="action-title-cell">
                <div class="action-title">
                  {{ row.display_name || row.title || '未命名活动' }}
                </div>
                <div v-if="row.display_name && row.title" class="action-original-title">
                  原名: {{ row.title }}
                </div>
                <el-button type="primary" text size="small" @click="openEditDialog(row)" class="edit-btn">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="action_type" label="类型" width="100" />
          <el-table-column label="日期范围" width="200">
            <template #default="{ row }">
              <span class="date-range">{{ formatDateRange(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="participating_products_count" label="参与商品" width="100" align="center">
            <template #default="{ row }">
              <span class="count-badge">{{ row.participating_products_count || 0 }}</span>
            </template>
          </el-table-column>
          <el-table-column label="来源" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_manual ? 'warning' : 'success'" size="small" effect="plain">
                {{ row.is_manual ? '手动' : 'API' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="同步时间" width="160">
            <template #default="{ row }">
              <span class="sync-time">{{ formatTime(row.last_synced_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-popconfirm
                title="确定要删除这个活动吗？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDelete(row)"
              >
                <template #reference>
                  <el-button type="danger" text size="small">
                    删除
                  </el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 手动添加对话框 -->
    <el-dialog
      v-model="showManualDialog"
      title="手动添加活动"
      width="420px"
      :close-on-click-modal="false"
    >
      <el-form :model="manualForm" :rules="manualRules" ref="manualFormRef" label-width="80px">
        <el-form-item label="活动ID" prop="action_id">
          <el-input-number
            v-model="manualForm.action_id"
            :min="1"
            :controls="false"
            placeholder="输入Ozon活动ID"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="活动名称" prop="title">
          <el-input v-model="manualForm.title" placeholder="可选，留空将自动生成" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showManualDialog = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="handleAddManual">
          添加
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑显示名称对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="设置中文显示名称"
      width="420px"
      :close-on-click-modal="false"
    >
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="原始名称">
          <span class="original-name">{{ editForm.originalTitle || '未命名活动' }}</span>
        </el-form-item>
        <el-form-item label="中文显示名称">
          <el-input
            v-model="editForm.displayName"
            placeholder="输入中文显示名称（留空则显示原始名称）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="handleUpdateDisplayName">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getActions, syncActions, createManualAction, deleteAction, updateActionDisplayName } from '@/api/promotion'
import { Refresh, Plus, Edit } from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const syncing = ref(false)
const adding = ref(false)
const updating = ref(false)
const showManualDialog = ref(false)
const showEditDialog = ref(false)
const actions = ref([])
const manualFormRef = ref(null)

const manualForm = reactive({
  action_id: null,
  title: ''
})

const editForm = reactive({
  id: null,
  originalTitle: '',
  displayName: ''
})

const manualRules = {
  action_id: [
    { required: true, message: '请输入活动ID', trigger: 'blur' }
  ]
}

function formatDateRange(row) {
  if (!row.date_start && !row.date_end) return '-'
  const start = row.date_start ? new Date(row.date_start).toLocaleDateString() : ''
  const end = row.date_end ? new Date(row.date_end).toLocaleDateString() : ''
  if (start && end) return `${start} ~ ${end}`
  if (start) return `${start} 起`
  if (end) return `至 ${end}`
  return '-'
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

async function fetchActions() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  loading.value = true
  try {
    const res = await getActions(shopId)
    actions.value = res.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取活动列表失败')
  } finally {
    loading.value = false
  }
}

async function handleSync() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  syncing.value = true
  try {
    const res = await syncActions(shopId)
    actions.value = res.data || []
    ElMessage.success('同步成功')
  } catch (error) {
    console.error(error)
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

async function handleAddManual() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  await manualFormRef.value.validate()

  adding.value = true
  try {
    await createManualAction({
      shop_id: shopId,
      action_id: manualForm.action_id,
      title: manualForm.title
    })
    ElMessage.success('添加成功')
    showManualDialog.value = false
    manualForm.action_id = null
    manualForm.title = ''
    await fetchActions()
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '添加失败')
  } finally {
    adding.value = false
  }
}

async function handleDelete(row) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  try {
    await deleteAction(row.id, shopId)
    ElMessage.success('删除成功')
    await fetchActions()
  } catch (error) {
    console.error(error)
    ElMessage.error('删除失败')
  }
}

function openEditDialog(row) {
  editForm.id = row.id
  editForm.originalTitle = row.title
  editForm.displayName = row.display_name || ''
  showEditDialog.value = true
}

async function handleUpdateDisplayName() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  updating.value = true
  try {
    await updateActionDisplayName(editForm.id, shopId, editForm.displayName)
    ElMessage.success('更新成功')
    showEditDialog.value = false
    await fetchActions()
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '更新失败')
  } finally {
    updating.value = false
  }
}

onMounted(() => {
  fetchActions()
})
</script>

<style scoped>
.action-list {
  min-height: 100%;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-subtitle {
  font-size: 13px;
  color: var(--text-muted);
}

.action-id {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.action-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-range {
  font-size: 13px;
  color: var(--text-secondary);
}

.count-badge {
  font-weight: 600;
  color: var(--primary);
}

.sync-time {
  font-size: 12px;
  color: var(--text-muted);
}

.action-title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-title-cell .action-title {
  flex: 1;
}

.action-original-title {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: 8px;
}

.edit-btn {
  opacity: 0.6;
  transition: opacity 0.2s;
}

.edit-btn:hover {
  opacity: 1;
}

.original-name {
  color: var(--text-muted);
  font-size: 13px;
}
</style>
