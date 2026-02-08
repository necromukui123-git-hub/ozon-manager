<template>
  <div class="action-list">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="gradient">促销活动管理</h2>
        <span class="action-count">{{ actions.length }} 个活动</span>
      </div>
      <div class="header-actions">
        <el-button
          v-if="!sortMode"
          type="info"
          plain
          @click="enterSortMode"
          :disabled="actions.length < 2"
        >
          <el-icon><Rank /></el-icon>
          调整顺序
        </el-button>
        <template v-else>
          <el-button type="primary" :loading="savingSortOrder" @click="saveSortOrder">
            <el-icon><Check /></el-icon>
            完成排序
          </el-button>
          <el-button @click="cancelSortMode">
            取消
          </el-button>
        </template>
        <el-button v-if="!sortMode" type="primary" :loading="syncing" @click="handleSync">
          <el-icon><Refresh /></el-icon>
          同步活动
        </el-button>
        <el-button v-if="!sortMode" @click="showManualDialog = true">
          <el-icon><Plus /></el-icon>
          手动添加
        </el-button>
      </div>
    </div>

    <!-- 排序模式提示 -->
    <div v-if="sortMode" class="sort-mode-tip">
      <el-icon><InfoFilled /></el-icon>
      <span>拖拽列表项调整顺序，完成后点击"完成排序"保存</span>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row" v-if="!sortMode">
      <StatCard
        :value="actions.length"
        label="活动总数"
        :icon="Ticket"
        variant="primary"
      />
      <StatCard
        :value="activeCount"
        label="进行中"
        :icon="Clock"
        variant="success"
      />
      <StatCard
        :value="upcomingCount"
        label="即将开始"
        :icon="Calendar"
        variant="info"
      />
      <StatCard
        :value="totalProducts"
        label="参与商品"
        :icon="Goods"
        variant="accent"
      />
    </div>

    <!-- 活动卡片网格 -->
    <div class="action-grid" v-loading="loading">
      <!-- 空状态 -->
      <div v-if="actions.length === 0 && !loading" class="empty-state">
        <div class="empty-icon">
          <el-icon :size="48"><Box /></el-icon>
        </div>
        <h3>暂无促销活动</h3>
        <p>点击"同步活动"从 Ozon 获取，或手动添加活动</p>
      </div>

      <!-- 拖拽排序模式 - 列表视图 -->
      <draggable
        v-if="sortMode && actions.length > 0"
        v-model="sortableActions"
        item-key="id"
        class="draggable-list"
        ghost-class="ghost-item"
        drag-class="dragging-item"
        :animation="200"
      >
        <template #item="{ element: action, index }">
          <div class="sort-item">
            <div class="sort-handle">
              <el-icon><Rank /></el-icon>
            </div>
            <div class="sort-type" :class="getTypeClass(action.action_type)">
              {{ formatActionType(action.action_type) }}
            </div>
            <div class="sort-id">
              {{ action.action_id }}
            </div>
            <div class="sort-title">
              {{ action.display_name || action.title || '未命名活动' }}
            </div>
            <div class="sort-date">
              {{ formatDateRange(action) }}
            </div>
            <div class="sort-index">#{{ index + 1 }}</div>
          </div>
        </template>
      </draggable>

      <!-- 普通模式 -->
      <template v-else-if="!sortMode">
        <div
          v-for="action in actions"
          :key="action.id"
          class="action-card"
          :class="{ 'is-active': isActionActive(action) }"
        >
          <!-- 状态指示条 -->
          <div class="status-indicator" :class="getStatusClass(action)"></div>

          <!-- 卡片头部 -->
          <div class="card-top">
            <div class="type-badge" :class="getTypeClass(action.action_type)">
              {{ formatActionType(action.action_type) }}
            </div>
            <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, action)">
              <el-button text circle size="small" class="more-btn">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    设置中文名称
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <el-icon><Delete /></el-icon>
                    删除活动
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <!-- 活动名称 -->
          <div class="card-title">
            <h3>{{ action.display_name || action.title || '未命名活动' }}</h3>
            <el-tooltip
              v-if="action.display_name && action.title"
              :content="action.title"
              placement="top"
              :show-after="300"
            >
              <span class="original-title">{{ truncateText(action.title, 40) }}</span>
            </el-tooltip>
          </div>

          <!-- 活动信息 -->
          <div class="card-meta">
            <div class="meta-row">
              <div class="meta-item">
                <el-icon><Calendar /></el-icon>
                <span>{{ formatDateRange(action) }}</span>
              </div>
            </div>
            <div class="meta-row">
              <div class="meta-item">
                <el-icon><Goods /></el-icon>
                <span class="product-count">{{ action.participating_products_count || 0 }}</span>
                <span>件商品参与</span>
              </div>
            </div>
          </div>

          <!-- 卡片底部 -->
          <div class="card-footer">
            <div class="action-id">
              <span class="label">ID:</span>
              <span class="value">{{ action.action_id }}</span>
            </div>
            <el-tag
              :type="action.is_manual ? 'warning' : 'success'"
              size="small"
              effect="light"
              round
            >
              {{ action.is_manual ? '手动' : 'API' }}
            </el-tag>
          </div>
        </div>
      </template>
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
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getActions, syncActions, createManualAction, deleteAction, updateActionDisplayName, updateActionsSortOrder } from '@/api/promotion'
import { StatCard } from '@/components/bento'
import draggable from 'vuedraggable'
import { Refresh, Plus, Edit, MoreFilled, Delete, Calendar, Goods, Box, Ticket, Clock, Rank, Check, InfoFilled } from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const syncing = ref(false)
const adding = ref(false)
const updating = ref(false)
const showManualDialog = ref(false)
const showEditDialog = ref(false)
const actions = ref([])
const manualFormRef = ref(null)

// 排序模式相关
const sortMode = ref(false)
const sortableActions = ref([])
const savingSortOrder = ref(false)

// 计算统计数据
const activeCount = computed(() => {
  return actions.value.filter(a => isActionActive(a)).length
})

const upcomingCount = computed(() => {
  return actions.value.filter(a => {
    if (!a.date_start) return false
    const now = new Date()
    const start = new Date(a.date_start)
    return now < start
  }).length
})

const totalProducts = computed(() => {
  return actions.value.reduce((sum, a) => sum + (a.participating_products_count || 0), 0)
})

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

function handleCommand(command, action) {
  if (command === 'edit') {
    openEditDialog(action)
  } else if (command === 'delete') {
    handleDelete(action)
  }
}

function isActionActive(action) {
  if (!action.date_start || !action.date_end) return false
  const now = new Date()
  const start = new Date(action.date_start)
  const end = new Date(action.date_end)
  return now >= start && now <= end
}

function getStatusClass(action) {
  if (!action.date_start || !action.date_end) return 'status-unknown'
  const now = new Date()
  const start = new Date(action.date_start)
  const end = new Date(action.date_end)
  if (now < start) return 'status-upcoming'
  if (now > end) return 'status-ended'
  return 'status-active'
}

function getTypeClass(type) {
  if (!type) return 'type-default'
  const t = type.toUpperCase()
  if (t.includes('STOCK') || t.includes('DISCOUNT')) return 'type-discount'
  if (t.includes('MARKET') || t.includes('MULTI')) return 'type-market'
  return 'type-default'
}

function formatActionType(type) {
  if (!type) return '未知'
  const typeMap = {
    'STOCK_DISCOUNT': '库存折扣',
    'MARKETPLACE_MULTI_LEVEL_DISCOUNT_ON_AMOUNT': '满减折扣',
    'DISCOUNT': '折扣',
    'FLASH_SALE': '限时特卖',
    'BUNDLE': '捆绑销售'
  }
  return typeMap[type] || type.replace(/_/g, ' ').toLowerCase()
}

function truncateText(text, maxLength) {
  if (!text) return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
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

// 进入排序模式
function enterSortMode() {
  sortableActions.value = [...actions.value]
  sortMode.value = true
}

// 取消排序模式
function cancelSortMode() {
  sortMode.value = false
  sortableActions.value = []
}

// 保存排序
async function saveSortOrder() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  savingSortOrder.value = true
  try {
    const sortOrders = sortableActions.value.map((action, index) => ({
      id: action.id,
      sort_order: index
    }))
    await updateActionsSortOrder(shopId, sortOrders)
    ElMessage.success('排序保存成功')
    sortMode.value = false
    await fetchActions()
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '保存排序失败')
  } finally {
    savingSortOrder.value = false
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

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.action-count {
  font-size: 14px;
  color: var(--text-muted);
  font-weight: 400;
}

.header-actions {
  display: flex;
  gap: 12px;
}

/* 统计卡片行 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}

/* 卡片网格布局 */
.action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  min-height: 200px;
}

/* 空状态 */
.empty-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-state .empty-icon {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
  color: var(--text-muted);
}

.empty-state h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-state p {
  font-size: 14px;
  color: var(--text-muted);
}

/* 活动卡片 */
.action-card {
  background: var(--bg-secondary);
  border: var(--neo-border-width) solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  padding: 20px;
  position: relative;
  overflow: hidden;
  transition: all var(--transition-normal);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.action-card:hover {
  border-color: var(--neo-border-color);
  box-shadow: 3px 3px 0 var(--neo-border-color);
  transform: translate(-1px, -1px);
}

.action-card.is-active {
  border-color: var(--success);
}

/* 状态指示条 */
.status-indicator {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}

.status-indicator.status-active {
  background: var(--success);
}

.status-indicator.status-upcoming {
  background: var(--info);
}

.status-indicator.status-ended {
  background: var(--text-muted);
}

.status-indicator.status-unknown {
  background: var(--border-color);
}

/* 卡片头部 */
.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.type-badge {
  padding: 4px 10px;
  border-radius: var(--neo-radius);
  border: 2px solid var(--neo-border-color);
  font-size: 12px;
  font-weight: 700;
}

.type-badge.type-discount {
  background: #dbeafe;
  color: #1e3a8a;
}

.type-badge.type-market {
  background: #e0f2fe;
  color: #155e75;
}

.type-badge.type-default {
  background: #fef3c7;
  color: #78350f;
}

.more-btn {
  opacity: 0.6;
  transition: opacity var(--transition-fast);
}

.more-btn:hover {
  opacity: 1;
}

/* 卡片标题 */
.card-title h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.original-title {
  font-size: 12px;
  color: var(--text-muted);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
}

/* 卡片信息 */
.card-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.meta-row {
  display: flex;
  align-items: center;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.meta-item .el-icon {
  color: var(--text-muted);
  font-size: 14px;
}

.meta-item .product-count {
  font-weight: 600;
  color: var(--primary);
}

/* 卡片底部 */
.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--border-color-light);
  margin-top: auto;
}

.action-id {
  font-size: 12px;
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.action-id .label {
  color: var(--text-muted);
  margin-right: 4px;
}

.action-id .value {
  color: var(--text-secondary);
}

/* 对话框中的原始名称 */
.original-name {
  color: var(--text-muted);
  font-size: 13px;
}

/* 响应式 */
@media (max-width: 768px) {
  .action-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .header-left {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

/* 排序模式提示 */
.sort-mode-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #dbeafe;
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: 3px 3px 0 var(--neo-border-color);
  margin-bottom: 20px;
  color: #1e3a8a;
  font-size: 14px;
  font-weight: 600;
}

.sort-mode-tip .el-icon {
  font-size: 16px;
}

/* 拖拽列表 */
.draggable-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  grid-column: 1 / -1;
}

/* 排序列表项 */
.sort-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 48px;
  padding: 0 16px;
  background: var(--bg-secondary);
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: 2px 2px 0 var(--neo-border-color);
  cursor: grab;
  user-select: none;
  transition: all var(--transition-fast);
  width: 100%;
  box-sizing: border-box;
}

.sort-item:hover {
  border-color: var(--neo-border-color);
  background: var(--bg-tertiary);
  box-shadow: 3px 3px 0 var(--neo-border-color);
}

.sort-item:active {
  cursor: grabbing;
}

/* 拖拽手柄 */
.sort-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  color: var(--text-muted);
  flex-shrink: 0;
  transition: color var(--transition-fast);
}

.sort-item:hover .sort-handle {
  color: var(--primary);
}

/* 类型标签 */
.sort-type {
  padding: 4px 10px;
  border-radius: var(--neo-radius);
  border: 2px solid var(--neo-border-color);
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
  min-width: 70px;
  text-align: center;
}

.sort-type.type-discount {
  background: #dbeafe;
  color: #1e3a8a;
}

.sort-type.type-market {
  background: #e0f2fe;
  color: #155e75;
}

.sort-type.type-default {
  background: #fef3c7;
  color: #78350f;
}

/* 活动ID */
.sort-id {
  font-size: 13px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: var(--text-primary);
  font-weight: 600;
  flex-shrink: 0;
  width: 80px;
}

/* 活动名称 */
.sort-title {
  flex: 1 1 200px;
  min-width: 120px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 日期范围 */
.sort-date {
  font-size: 13px;
  color: var(--text-secondary);
  flex-shrink: 0;
  width: 160px;
  text-align: right;
}

/* 序号 */
.sort-index {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  flex-shrink: 0;
  width: 32px;
  text-align: right;
}

/* 拖拽时的幽灵项 */
.ghost-item {
  opacity: 0.5;
  background: #dbeafe;
  border: 2px dashed var(--primary);
}

/* 正在拖拽的项 */
.dragging-item {
  opacity: 0.95;
  box-shadow: 4px 4px 0 var(--neo-border-color);
  background: var(--bg-secondary);
  border-color: var(--primary);
}

@media (max-width: 768px) {
  .sort-item {
    height: auto;
    min-height: 48px;
    padding: 12px 16px;
    flex-wrap: wrap;
  }

  .sort-date {
    min-width: auto;
    width: 100%;
    text-align: left;
    margin-top: 4px;
    padding-left: 36px;
  }

  .sort-index {
    position: absolute;
    right: 16px;
    top: 50%;
    transform: translateY(-50%);
  }
}
</style>
