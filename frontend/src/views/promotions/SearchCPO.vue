<template>
  <div class="search-cpo-page">
    <div class="hero-shell">
      <div class="hero-copy">
        <h2 class="gradient">CPO 商品批量报名</h2>
        <p class="hero-subtitle">
          从 Seller 隐藏 CPO 接口拉取商品，优先筛选“已关闭”商品后执行报名，过程可回溯。
        </p>
        <div class="hero-metrics">
          <div class="metric-pill">
            <span class="metric-label">缓存商品</span>
            <span class="metric-value">{{ products.length }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">当前筛选</span>
            <span class="metric-value">{{ filteredItems.length }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">默认活动</span>
            <span class="metric-value">{{ config.official_action_ids.length + config.shop_action_ids.length }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">最近刷新</span>
            <span class="metric-value metric-time">{{ lastSynced || '-' }}</span>
          </div>
        </div>
      </div>
      <div class="page-actions">
        <el-button :loading="refreshing" @click="handleRefreshProducts">刷新隐藏商品</el-button>
        <el-button :loading="savingConfig" @click="handleSaveConfig">保存默认活动</el-button>
        <el-button type="primary" :loading="running" :disabled="filteredItems.length === 0" @click="handleRunNow">
          报名当前筛选结果
        </el-button>
      </div>
    </div>

    <div class="bento-grid--2col">
      <BentoCard title="执行说明" :icon="InfoFilled" size="1x1">
        <div class="hint-list">
          <div>1. 先点击“刷新隐藏商品”，从 Seller 隐藏接口拉取最新 CPO 商品。</div>
          <div>2. 推荐先把“推广状态”筛到“已关闭”，执行范围固定为“当前筛选结果”。</div>
          <div>3. 首次进入不会自动勾选活动；请勾选后点击“保存默认活动”。</div>
        </div>
      </BentoCard>

      <BentoCard title="执行范围" :icon="List" size="1x1">
        <div class="scope-line">缓存总商品：{{ products.length }}</div>
        <div class="scope-line">当前筛选：{{ filteredItems.length }}</div>
        <div class="scope-line">默认官方活动：{{ config.official_action_ids.length }}</div>
        <div class="scope-line">默认店铺活动：{{ config.shop_action_ids.length }}</div>
      </BentoCard>
    </div>

    <div class="bento-grid--2col actions-grid">
      <BentoCard title="官方活动" :icon="Flag" size="1x1">
        <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
        <el-checkbox-group v-else v-model="config.official_action_ids" class="action-group">
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
        <el-checkbox-group v-else v-model="config.shop_action_ids" class="action-group">
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

    <BentoCard title="本地筛选" :icon="Filter" size="4x1">
      <el-form :inline="true" class="filter-form">
        <el-form-item label="关键词">
          <el-input
            v-model="filters.keyword"
            placeholder="标题 / source_sku / sku / 类目"
            clearable
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item label="推广状态">
          <el-select v-model="filters.promoStatus" style="width: 210px">
            <el-option label="全部" value="all" />
            <el-option label="已开启" value="SEARCH_PROMO_STATUS_ENABLED" />
            <el-option label="已关闭" value="SEARCH_PROMO_STATUS_DISABLED" />
          </el-select>
        </el-form-item>
        <el-form-item label="库存">
          <el-select v-model="filters.stockStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="有库存" value="in_stock" />
            <el-option label="无库存" value="out_of_stock" />
          </el-select>
        </el-form-item>
        <el-form-item label="收藏">
          <el-select v-model="filters.favoriteStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="已收藏" value="favorite" />
            <el-option label="未收藏" value="not_favorite" />
          </el-select>
        </el-form-item>
      </el-form>
    </BentoCard>

    <BentoCard title="商品列表" :icon="Goods" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="pagedItems" v-loading="productsLoading">
          <el-table-column label="图片" width="88" align="center">
            <template #default="{ row }">
              <el-image
                v-if="row.image_url"
                :src="row.image_url"
                class="thumb"
                fit="cover"
                :preview-src-list="[row.image_url]"
                preview-teleported
              />
              <span v-else class="no-data">-</span>
            </template>
          </el-table-column>

          <el-table-column label="商品信息" min-width="320">
            <template #default="{ row }">
              <div class="name">{{ row.title || '-' }}</div>
              <div class="meta">source_sku: {{ row.source_sku || '-' }}</div>
              <div class="meta">sku: {{ row.sku || '-' }}</div>
              <div class="meta">类目: {{ row.category_name || '-' }}</div>
            </template>
          </el-table-column>

          <el-table-column label="价格 / 库存" min-width="160" align="center">
            <template #default="{ row }">
              <div class="meta">价格: {{ formatMoney(row.price) }}</div>
              <div class="meta">库存总量: {{ row.stock_total ?? 0 }}</div>
              <el-tag size="small" :type="row.is_in_stock ? 'success' : 'warning'">
                {{ row.is_in_stock ? '有库存' : '无库存' }}
              </el-tag>
            </template>
          </el-table-column>

        <el-table-column label="推广状态" width="160" align="center">
          <template #default="{ row }">
              <el-tag :type="row.search_promo_status === 'SEARCH_PROMO_STATUS_ENABLED' ? 'success' : 'info'">
                {{ row.search_promo_status === 'SEARCH_PROMO_STATUS_ENABLED' ? '已开启' : '已关闭' }}
              </el-tag>
          </template>
        </el-table-column>

          <el-table-column label="核心指标" min-width="180" align="center">
            <template #default="{ row }">
              <div class="meta">订单: {{ row.orders ?? 0 }}</div>
              <div class="meta">点击: {{ row.clicks ?? 0 }}</div>
              <div class="meta">花费: {{ formatMoney(row.spent) }}</div>
              <div class="meta">CTR: {{ formatPercent(row.ctr_percent) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <div class="pager-line">
          <span>共 {{ filteredItems.length }} 条，当前页 {{ pager.page }}</span>
          <el-pagination
            v-model:current-page="pager.page"
            v-model:page-size="pager.pageSize"
            :total="filteredItems.length"
            layout="prev, pager, next, sizes"
            :page-sizes="[20, 50, 100]"
            small
          />
        </div>
      </template>
    </BentoCard>

    <BentoCard title="执行历史" :icon="Clock" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="runs" v-loading="runsLoading">
          <el-table-column prop="id" label="任务ID" width="90" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="统计" min-width="260">
            <template #default="{ row }">
              <div class="meta">缓存 {{ row.total_fetched }} / 筛选 {{ row.total_selected }} / 处理 {{ row.total_processed }}</div>
              <div class="meta">成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column prop="error_message" label="错误摘要" min-width="250">
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
      </div>
    </BentoCard>

    <el-dialog v-model="detailVisible" title="运行详情" width="1100px">
      <div v-if="detail" class="detail-summary">
        <el-tag :type="statusTagType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
        <span>筛选商品 {{ detail.source_skus?.length || 0 }} 件</span>
        <span>成功 {{ detail.success_items }} / 失败 {{ detail.failed_items }} / 跳过 {{ detail.skipped_items }}</span>
      </div>

      <el-table v-if="detail" :data="detail.items || []" max-height="520" v-loading="detailLoading">
        <el-table-column prop="source_sku" label="source_sku" width="160" />
        <el-table-column prop="sku" label="sku" width="130" />
        <el-table-column prop="title" label="商品" min-width="220" />
        <el-table-column label="总状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="官方结果" min-width="220">
          <template #default="{ row }">
            <div class="result-lines">
              <div v-for="item in row.official_results" :key="`official-${row.id}-${item.promotion_action_id}`">
                {{ item.title }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.official_results || row.official_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="店铺结果" min-width="240">
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
  getSearchCPOConfig,
  updateSearchCPOConfig,
  listSearchCPOProducts,
  refreshSearchCPOProducts,
  startSearchCPORun,
  listSearchCPORuns,
  getSearchCPORunDetail
} from '@/api/promotion'
import { BentoCard } from '@/components/bento'
import { Clock, Discount, Filter, Flag, Goods, InfoFilled, List } from '@element-plus/icons-vue'

const userStore = useUserStore()

const actions = ref([])
const products = ref([])
const runs = ref([])
const detail = ref(null)
const detailVisible = ref(false)

const actionsLoading = ref(false)
const productsLoading = ref(false)
const runsLoading = ref(false)
const detailLoading = ref(false)
const savingConfig = ref(false)
const refreshing = ref(false)
const running = ref(false)

const config = reactive({
  official_action_ids: [],
  shop_action_ids: []
})

const filters = reactive({
  keyword: '',
  promoStatus: 'all',
  stockStatus: 'all',
  favoriteStatus: 'all'
})

const pager = reactive({
  page: 1,
  pageSize: 20
})

const lastSynced = ref('')
let runPollTimer = null

const officialActions = computed(() => actions.value.filter(action => action.source === 'official'))
const shopActions = computed(() => actions.value.filter(action => action.source === 'shop'))

const filteredItems = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  return products.value.filter(item => {
    if (filters.promoStatus !== 'all' && item.search_promo_status !== filters.promoStatus) return false
    if (filters.stockStatus === 'in_stock' && !item.is_in_stock) return false
    if (filters.stockStatus === 'out_of_stock' && item.is_in_stock) return false
    if (filters.favoriteStatus === 'favorite' && !item.is_favorite) return false
    if (filters.favoriteStatus === 'not_favorite' && item.is_favorite) return false
    if (!keyword) return true
    const blob = `${item.title || ''} ${item.source_sku || ''} ${item.sku || ''} ${item.category_name || ''}`.toLowerCase()
    return blob.includes(keyword)
  })
})

const pagedItems = computed(() => {
  const start = (pager.page - 1) * pager.pageSize
  return filteredItems.value.slice(start, start + pager.pageSize)
})

watch(filteredItems, () => {
  pager.page = 1
})

watch(
  () => userStore.currentShopId,
  () => {
    resetState()
    loadPageData()
  }
)

onMounted(() => {
  loadPageData()
})

onUnmounted(() => {
  stopRunPolling()
})

function resetState() {
  products.value = []
  runs.value = []
  config.official_action_ids = []
  config.shop_action_ids = []
  detail.value = null
  detailVisible.value = false
  lastSynced.value = ''
  stopRunPolling()
}

async function loadPageData() {
  const shopId = userStore.currentShopId
  if (!shopId) return
  try {
    await Promise.all([
      loadActions(),
      loadConfig(),
      loadProducts(),
      loadRuns()
    ])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '加载 CPO 页面失败')
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

  const res = await getSearchCPOConfig(shopId)
  const data = res.data || {}
  config.official_action_ids = Array.isArray(data.official_action_ids) ? data.official_action_ids : []
  config.shop_action_ids = Array.isArray(data.shop_action_ids) ? data.shop_action_ids : []
}

async function loadProducts() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  productsLoading.value = true
  try {
    const res = await listSearchCPOProducts(shopId)
    const data = res.data || {}
    products.value = data.items || []
    lastSynced.value = data.last_synced || ''
  } finally {
    productsLoading.value = false
  }
}

async function loadRuns(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) runsLoading.value = true
  try {
    const res = await listSearchCPORuns({ shop_id: shopId, page: 1, page_size: 20 })
    runs.value = res.data?.items || []
    updateRunPollingState()
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

  savingConfig.value = true
  try {
    await updateSearchCPOConfig({
      shop_id: shopId,
      official_action_ids: config.official_action_ids,
      shop_action_ids: config.shop_action_ids
    })
    ElMessage.success('默认活动已保存')
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存默认活动失败')
  } finally {
    savingConfig.value = false
  }
}

async function handleRefreshProducts() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  refreshing.value = true
  try {
    await refreshSearchCPOProducts({ shop_id: shopId })
    ElMessage.success('刷新完成')
    await loadProducts()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '刷新失败')
  } finally {
    refreshing.value = false
  }
}

async function handleRunNow() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }
  if (filteredItems.value.length === 0) {
    ElMessage.warning('当前筛选结果为空')
    return
  }
  if (config.official_action_ids.length + config.shop_action_ids.length === 0) {
    ElMessage.warning('请先选择活动并保存默认活动')
    return
  }

  try {
    await ElMessageBox.confirm(
      `将对当前筛选结果 ${filteredItems.value.length} 个商品执行报名，是否继续？`,
      '确认执行',
      {
        confirmButtonText: '开始执行',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  running.value = true
  try {
    await startSearchCPORun({
      shop_id: shopId,
      source_skus: filteredItems.value.map(item => item.source_sku).filter(Boolean),
      official_action_ids: config.official_action_ids,
      shop_action_ids: config.shop_action_ids
    })
    ElMessage.success('已创建 CPO 报名任务')
    await loadRuns()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '创建任务失败')
  } finally {
    running.value = false
  }
}

function updateRunPollingState() {
  const hasRunning = runs.value.some(row => ['pending', 'running'].includes(row.status))
  if (hasRunning) {
    startRunPolling()
  } else {
    stopRunPolling()
  }
}

function startRunPolling() {
  if (runPollTimer) return
  runPollTimer = setInterval(() => loadRuns(true), 2500)
}

function stopRunPolling() {
  if (runPollTimer) {
    clearInterval(runPollTimer)
    runPollTimer = null
  }
}

async function openRunDetail(row) {
  const shopId = userStore.currentShopId
  if (!shopId || !row?.id) return

  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res = await getSearchCPORunDetail(row.id, shopId)
    detail.value = res.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '获取运行详情失败')
  } finally {
    detailLoading.value = false
  }
}

function formatMoney(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '-'
  return amount.toFixed(2)
}

function formatPercent(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '0%'
  return `${amount.toFixed(2)}%`
}

function statusLabel(status) {
  switch (status) {
    case 'success':
      return '成功'
    case 'partial_success':
      return '部分成功'
    case 'failed':
      return '失败'
    case 'running':
      return '执行中'
    case 'pending':
      return '等待中'
    case 'skipped':
      return '跳过'
    default:
      return status || '-'
  }
}

function statusTagType(status) {
  switch (status) {
    case 'success':
      return 'success'
    case 'partial_success':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'running':
      return 'primary'
    case 'pending':
      return 'info'
    case 'skipped':
      return ''
    default:
      return 'info'
  }
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Nunito+Sans:wght@400;500;600;700&family=Rubik:wght@500;600;700&display=swap');

.search-cpo-page {
  min-height: 100%;
  --cpo-primary: #059669;
  --cpo-secondary: #10b981;
  --cpo-cta: #f97316;
  --cpo-bg-soft: #ecfdf5;
  --cpo-text: #064e3b;
  --cpo-border: #c7f0dd;
  font-family: 'Nunito Sans', sans-serif;
  background:
    radial-gradient(1200px 440px at -8% -20%, rgba(16, 185, 129, 0.16), transparent 62%),
    radial-gradient(980px 440px at 108% -18%, rgba(249, 115, 22, 0.12), transparent 64%);
}

.hero-shell {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  padding: 18px 20px;
  border-radius: 16px;
  border: 1px solid var(--cpo-border);
  background: linear-gradient(120deg, rgba(5, 150, 105, 0.08), rgba(249, 115, 22, 0.08));
}

.hero-copy {
  flex: 1;
  min-width: 320px;
}

.hero-subtitle {
  margin: 8px 0 12px;
  line-height: 1.55;
  color: var(--cpo-text);
  opacity: 0.9;
}

.hero-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.metric-pill {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 62px;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(5, 150, 105, 0.16);
  transition: border-color 0.24s ease, transform 0.24s ease;
}

.metric-pill:hover {
  border-color: rgba(5, 150, 105, 0.36);
  transform: translateY(-1px);
}

.metric-label {
  color: #0f766e;
  font-size: 12px;
}

.metric-value {
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 15px;
  color: #064e3b;
}

.metric-time {
  font-size: 12px;
  line-height: 1.4;
}

.page-header {
  margin-bottom: 10px;
}

.page-header h2,
.hero-shell h2 {
  margin: 0;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  max-width: 340px;
  justify-content: flex-end;
}

.page-actions :deep(.el-button) {
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.page-actions :deep(.el-button:hover) {
  transform: translateY(-1px);
}

.page-actions :deep(.el-button--primary) {
  box-shadow: 0 8px 18px rgba(249, 115, 22, 0.18);
}

.hint-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.scope-line {
  margin-bottom: 8px;
  color: var(--text-secondary);
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.actions-grid {
  margin-top: 6px;
}

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }
}

.action-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-checkbox {
  width: 100%;
  margin-right: 0;
}

.action-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.action-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-meta {
  color: var(--text-muted);
  font-size: 12px;
}

.filter-form {
  row-gap: 8px;
}

.table-shell {
  overflow-x: auto;
}

.thumb {
  width: 56px;
  height: 56px;
  border-radius: 6px;
}

.name {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.meta {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
}

.no-data {
  color: var(--text-muted);
}

.pager-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.error-text {
  color: #d14343;
}

.detail-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.result-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

@media (max-width: 1280px) {
  .hero-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 920px) {
  .hero-shell {
    flex-direction: column;
  }

  .page-actions {
    max-width: none;
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 600px) {
  .hero-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
