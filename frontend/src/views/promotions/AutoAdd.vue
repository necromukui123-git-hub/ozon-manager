<template>
  <div class="auto-promotion-add">
    <div class="page-header">
      <h2 class="gradient">自动加促销</h2>
      <div class="page-actions">
        <el-button :loading="saving" @click="handleSaveConfig">保存配置</el-button>
        <el-button type="primary" :loading="running" @click="handleRunNow">手动执行</el-button>
      </div>
    </div>

    <div class="bento-grid--2col">
      <BentoCard title="自动执行配置" :icon="Clock" size="1x1">
        <el-form label-width="110px" class="config-form">
          <el-form-item label="启用自动执行">
            <el-switch v-model="form.enabled" />
          </el-form-item>
          <el-form-item label="执行时间">
            <el-time-picker
              v-model="form.schedule_time"
              value-format="HH:mm"
              format="HH:mm"
              placeholder="09:05"
            />
          </el-form-item>
          <el-form-item label="目标日期">
            <el-date-picker
              v-model="form.target_date"
              type="date"
              value-format="YYYY-MM-DD"
              format="YYYY-MM-DD"
              placeholder="选择日期"
            />
          </el-form-item>
          <el-form-item>
            <div class="form-tip">
              定时任务会每天按已保存的绝对日期执行。若要切换到其他日期，需要重新保存配置。
            </div>
          </el-form-item>
        </el-form>
      </BentoCard>

      <BentoCard title="执行说明" :icon="InfoFilled" size="1x1">
        <div class="hint-list">
          <div>1. 执行前会强制刷新 Ozon 商品目录，并按该目录的上架日期过滤商品。</div>
          <div>2. 官方活动会先执行，只有成功的商品才会继续进入店铺活动。</div>
          <div>3. 历史记录会保留逐商品失败原因，不会在商品列表主表打额外标签。</div>
        </div>
      </BentoCard>
    </div>

    <div class="bento-grid--2col actions-grid">
      <BentoCard title="官方活动" :icon="Flag" size="1x1">
        <template #actions>
          <el-button text size="small" :loading="actionsLoading" @click="loadActions">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </template>

        <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
        <el-checkbox-group v-else v-model="form.official_action_ids" class="action-group">
          <el-checkbox
            v-for="action in officialActions"
            :key="action.id"
            :value="action.id"
            class="action-checkbox"
          >
            <div class="action-item">
              <div class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</div>
              <div class="action-meta">官方 ID: {{ action.action_id }}</div>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </BentoCard>

      <BentoCard title="店铺活动" :icon="Discount" size="1x1">
        <el-empty v-if="!actionsLoading && shopActions.length === 0" description="暂无店铺活动" />
        <el-checkbox-group v-else v-model="form.shop_action_ids" class="action-group">
          <el-checkbox
            v-for="action in shopActions"
            :key="action.id"
            :value="action.id"
            class="action-checkbox"
          >
            <div class="action-item">
              <div class="action-title">{{ action.display_name || action.title || `活动 #${action.source_action_id}` }}</div>
              <div class="action-meta">店铺 ID: {{ action.source_action_id }}</div>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </BentoCard>
    </div>

    <BentoCard title="执行历史" :icon="List" size="4x1" class="history-card" no-padding>
      <el-table :data="runs" v-loading="runsLoading">
        <el-table-column prop="id" label="任务ID" width="90" />
        <el-table-column label="触发方式" width="110">
          <template #default="{ row }">
            <el-tag :type="row.trigger_mode === 'scheduled' ? 'warning' : 'primary'">
              {{ row.trigger_mode === 'scheduled' ? '定时' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_date" label="目标日期" width="120" />
        <el-table-column label="状态" width="140">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="统计" min-width="220">
          <template #default="{ row }">
            <div class="count-line">候选 {{ row.total_candidates }} / 选中 {{ row.total_selected }} / 处理 {{ row.total_processed }}</div>
            <div class="count-line success">成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="错误摘要" min-width="240">
          <template #default="{ row }">
            <span class="error-text">{{ row.error_message || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="openRunDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </BentoCard>

    <el-dialog v-model="detailVisible" title="执行详情" width="1100px">
      <div v-if="detail" class="detail-summary">
        <el-tag :type="statusTagType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
        <span>目标日期：{{ detail.target_date }}</span>
        <span>成功 {{ detail.success_items }} / 失败 {{ detail.failed_items }} / 跳过 {{ detail.skipped_items }}</span>
      </div>

      <el-table v-if="detail" :data="detail.items" v-loading="detailLoading" max-height="520">
        <el-table-column prop="source_sku" label="SKU" width="160" />
        <el-table-column prop="product_name" label="商品" min-width="220" />
        <el-table-column prop="listing_date" label="上架日期" width="110" />
        <el-table-column label="总状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="官方结果" min-width="230">
          <template #default="{ row }">
            <div class="result-lines">
              <div v-for="item in row.official_results" :key="`official-${row.id}-${item.promotion_action_id}`">
                {{ item.title }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.official_results || row.official_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="店铺结果" min-width="260">
          <template #default="{ row }">
            <div class="result-lines">
              <div v-for="item in row.shop_results" :key="`shop-${row.id}-${item.promotion_action_id}`">
                {{ item.title }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.shop_results || row.shop_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import {
  getActions,
  getAutoPromotionConfig,
  updateAutoPromotionConfig,
  startAutoPromotionRun,
  listAutoPromotionRuns,
  getAutoPromotionRunDetail
} from '@/api/promotion'
import { BentoCard } from '@/components/bento'
import { Clock, Discount, Flag, InfoFilled, List, Refresh } from '@element-plus/icons-vue'

const userStore = useUserStore()

const saving = ref(false)
const running = ref(false)
const actionsLoading = ref(false)
const runsLoading = ref(false)
const detailLoading = ref(false)
const actions = ref([])
const runs = ref([])
const detail = ref(null)
const detailVisible = ref(false)
let pollTimer = null

const form = reactive({
  enabled: false,
  schedule_time: '09:05',
  target_date: '',
  official_action_ids: [],
  shop_action_ids: []
})

const officialActions = computed(() => actions.value.filter(action => action.source === 'official'))
const shopActions = computed(() => actions.value.filter(action => action.source === 'shop'))

watch(
  () => userStore.currentShopId,
  () => {
    resetForm()
    loadPageData()
  }
)

onMounted(() => {
  loadPageData()
})

onUnmounted(() => {
  stopPolling()
})

function resetForm() {
  form.enabled = false
  form.schedule_time = '09:05'
  form.target_date = ''
  form.official_action_ids = []
  form.shop_action_ids = []
}

async function loadPageData() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  try {
    await Promise.all([loadActions(), loadConfig(), loadRuns()])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '加载自动加促销页面失败')
  }
}

async function loadActions() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  actionsLoading.value = true
  try {
    const res = await getActions(shopId)
    actions.value = res.data || []
  } finally {
    actionsLoading.value = false
  }
}

async function loadConfig() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  const res = await getAutoPromotionConfig(shopId)
  const data = res.data || {}
  form.enabled = !!data.enabled
  form.schedule_time = data.schedule_time || '09:05'
  form.target_date = data.target_date || ''
  form.official_action_ids = Array.isArray(data.official_action_ids) ? data.official_action_ids : []
  form.shop_action_ids = Array.isArray(data.shop_action_ids) ? data.shop_action_ids : []
}

async function loadRuns(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) runsLoading.value = true
  try {
    const res = await listAutoPromotionRuns({ shop_id: shopId, page: 1, page_size: 20 })
    runs.value = res.data?.items || []
    updatePollingState()
    return runs.value
  } finally {
    if (!silent) runsLoading.value = false
  }
}

async function handleSaveConfig() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  saving.value = true
  try {
    await updateAutoPromotionConfig({
      shop_id: shopId,
      enabled: form.enabled,
      schedule_time: form.schedule_time,
      target_date: form.target_date,
      official_action_ids: form.official_action_ids,
      shop_action_ids: form.shop_action_ids
    })
    ElMessage.success('配置已保存')
    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存配置失败')
  } finally {
    saving.value = false
  }
}

async function handleRunNow() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  try {
    await ElMessageBox.confirm(
      '手动执行会先刷新 Ozon 商品目录，再按当前选中的活动顺序执行加促销。是否继续？',
      '确认执行',
      { type: 'warning' }
    )
  } catch {
    return
  }

  running.value = true
  try {
    await startAutoPromotionRun({
      shop_id: shopId,
      target_date: form.target_date,
      official_action_ids: form.official_action_ids,
      shop_action_ids: form.shop_action_ids
    })
    ElMessage.success('已创建自动加促销任务')
    await loadRuns()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '触发任务失败')
  } finally {
    running.value = false
  }
}

async function openRunDetail(row) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  detailVisible.value = true
  detailLoading.value = true
  try {
    const res = await getAutoPromotionRunDetail(row.id, shopId)
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

function updatePollingState() {
  const hasActiveRun = runs.value.some(item => ['pending', 'running'].includes(item.status))
  if (!hasActiveRun) {
    stopPolling()
    return
  }
  if (pollTimer) return

  pollTimer = setInterval(() => {
    loadRuns(true)
    if (detailVisible.value && detail.value?.id) {
      openRunDetail({ id: detail.value.id })
    }
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function statusLabel(status) {
  switch (status) {
    case 'pending':
      return '待执行'
    case 'running':
      return '执行中'
    case 'success':
      return '成功'
    case 'partial_success':
      return '部分成功'
    case 'failed':
      return '失败'
    case 'skipped':
      return '已跳过'
    case 'candidate':
      return '待加入'
    case 'already_active':
      return '已在活动中'
    default:
      return status || '-'
  }
}

function statusTagType(status) {
  switch (status) {
    case 'success':
    case 'already_active':
      return 'success'
    case 'partial_success':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'running':
      return 'primary'
    default:
      return 'info'
  }
}
</script>

<style scoped>
.auto-promotion-add {
  min-height: 100%;
}

.page-actions {
  display: flex;
  gap: 10px;
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.actions-grid,
.history-card {
  margin-top: 16px;
}

.config-form {
  padding-right: 8px;
}

.form-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary);
}

.hint-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-secondary);
}

.action-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.action-checkbox {
  margin-right: 0;
}

.action-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.action-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-meta,
.count-line {
  font-size: 12px;
  color: var(--text-secondary);
}

.count-line.success {
  color: #15803d;
}

.error-text {
  color: #b91c1c;
}

.detail-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--text-secondary);
}

.result-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }

  .page-actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}
</style>
