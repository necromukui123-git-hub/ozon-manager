<template>
  <div class="search-cpo-page">
    <div class="hero-shell">
      <div class="hero-copy">
        <h2 class="gradient">搜索推广商品</h2>
        <p class="hero-subtitle">
          同一入口拆成两块工作面：前台商品池与人工批量报名，以及后台状态迁移自动化。
        </p>
      </div>

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
          <span class="metric-value">{{ totalDefaultActions }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">最近同步</span>
          <span class="metric-value metric-time">{{ lastSynced || '未同步' }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">自动化</span>
          <span class="metric-value metric-time">{{ automationMetricText }}</span>
        </div>
      </div>
    </div>

    <el-tabs v-model="activeTabName" class="search-cpo-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="商品池与手动报名" name="manual">
        <SearchCPOManualTab
          :actions="actions"
          :actions-loading="actionsLoading"
          :products-loading="productsLoading"
          :runs-loading="runsLoading"
          :saving-config="savingConfig"
          :refreshing="refreshing"
          :running="running"
          :config="config"
          :filters="filters"
          :pager="pager"
          :filtered-items="filteredItems"
          :paged-items="pagedItems"
          :runs="runs"
          :last-synced="lastSynced"
          @save-config="handleSaveDefaultActions"
          @refresh-products="handleRefreshProducts"
          @run-manual="handleRunNow"
          @open-run-detail="openRunDetail"
        />
      </el-tab-pane>

      <el-tab-pane label="状态迁移自动化" name="automation">
        <SearchCPOAutomationTab
          :actions="actions"
          :saving-config="savingConfig"
          :automation-running="automationRunning"
          :automation-runs-loading="automationRunsLoading"
          :config="config"
          :products="products"
          :automation-runs="automationRuns"
          @save-automation="handleSaveAutomationSettings"
          @start-run="handleStartAutomationRun"
          @configure-actions="switchToTab('manual')"
          @open-automation-detail="openAutomationRunDetail"
        />
      </el-tab-pane>
    </el-tabs>

    <SearchCPORunDetailDialog
      v-model:visible="detailVisible"
      :detail="detail"
      :loading="detailLoading"
    />

    <SearchCPOAutomationDetailDialog
      v-model:visible="automationDetailVisible"
      :detail="automationDetail"
      :loading="automationDetailLoading"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import {
  getActions,
  getSearchCPOAutomationRunDetail,
  getSearchCPOConfig,
  getSearchCPORunDetail,
  listSearchCPOAutomationRuns,
  listSearchCPOProducts,
  listSearchCPORuns,
  refreshSearchCPOProducts,
  startSearchCPOAutomationRun,
  startSearchCPORun,
  updateSearchCPOConfig
} from '@/api/promotion'
import SearchCPOAutomationDetailDialog from '@/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue'
import SearchCPOAutomationTab from '@/views/promotions/search-cpo/SearchCPOAutomationTab.vue'
import SearchCPOManualTab from '@/views/promotions/search-cpo/SearchCPOManualTab.vue'
import SearchCPORunDetailDialog from '@/views/promotions/search-cpo/SearchCPORunDetailDialog.vue'
import { statusLabel } from '@/views/promotions/search-cpo/ui'

const TAB_MANUAL = 'manual'
const TAB_AUTOMATION = 'automation'
const DEFAULT_SCHEDULE_TIME = '09:05'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const actions = ref([])
const products = ref([])
const runs = ref([])
const automationRuns = ref([])
const detail = ref(null)
const automationDetail = ref(null)
const detailVisible = ref(false)
const automationDetailVisible = ref(false)

const actionsLoading = ref(false)
const productsLoading = ref(false)
const runsLoading = ref(false)
const automationRunsLoading = ref(false)
const detailLoading = ref(false)
const automationDetailLoading = ref(false)
const savingConfig = ref(false)
const refreshing = ref(false)
const running = ref(false)
const automationRunning = ref(false)

const config = reactive({
  official_action_ids: [],
  shop_action_ids: [],
  auto_enabled: false,
  schedule_time: DEFAULT_SCHEDULE_TIME
})

const filters = reactive({
  keyword: '',
  promoStatus: 'all',
  ruleState: 'all',
  stockStatus: 'all',
  favoriteStatus: 'all'
})

const pager = reactive({
  page: 1,
  pageSize: 20
})

const lastSynced = ref('')
const activeTabName = ref(normalizeTabName(route.query.tab))
let runPollTimer = null

const totalDefaultActions = computed(() => config.official_action_ids.length + config.shop_action_ids.length)
const latestAutomationRun = computed(() => automationRuns.value[0] || null)
const automationMetricText = computed(() => {
  const latest = latestAutomationRun.value
  if (latest && ['pending', 'running'].includes(latest.status)) {
    return '执行中'
  }
  if (latest) {
    return `最近${statusLabel(latest.status)}`
  }
  if (config.auto_enabled) {
    return `已开启 / ${config.schedule_time || DEFAULT_SCHEDULE_TIME}`
  }
  return `未开启 / ${config.schedule_time || DEFAULT_SCHEDULE_TIME}`
})

const filteredItems = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  return products.value.filter(item => {
    if (filters.promoStatus !== 'all' && item.search_promo_status !== filters.promoStatus) return false
    if (filters.ruleState !== 'all' && (item.rule_state || 'other') !== filters.ruleState) return false
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
  () => route.query.tab,
  value => {
    activeTabName.value = normalizeTabName(value)
  }
)

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

function normalizeTabName(value) {
  return value === TAB_AUTOMATION ? TAB_AUTOMATION : TAB_MANUAL
}

function updateRouteTab(name) {
  const normalized = normalizeTabName(name)
  activeTabName.value = normalized
  if (route.query.tab === normalized) return
  router.replace({
    query: {
      ...route.query,
      tab: normalized
    }
  })
}

function handleTabChange(name) {
  updateRouteTab(String(name || TAB_MANUAL))
}

function switchToTab(name) {
  updateRouteTab(name)
}

function resetState() {
  products.value = []
  runs.value = []
  automationRuns.value = []
  config.official_action_ids = []
  config.shop_action_ids = []
  config.auto_enabled = false
  config.schedule_time = DEFAULT_SCHEDULE_TIME
  detail.value = null
  automationDetail.value = null
  detailVisible.value = false
  automationDetailVisible.value = false
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
      loadRuns(),
      loadAutomationRuns()
    ])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '加载搜索推广商品页面失败')
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
  config.auto_enabled = Boolean(data.auto_enabled)
  config.schedule_time = data.schedule_time || DEFAULT_SCHEDULE_TIME
}

async function loadProducts(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) productsLoading.value = true
  try {
    const res = await listSearchCPOProducts(shopId)
    const data = res.data || {}
    products.value = data.items || []
    lastSynced.value = data.last_synced || ''
  } finally {
    if (!silent) productsLoading.value = false
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

async function loadAutomationRuns(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) automationRunsLoading.value = true
  try {
    const res = await listSearchCPOAutomationRuns({ shop_id: shopId, page: 1, page_size: 20 })
    automationRuns.value = res.data?.items || []
    updateRunPollingState()
  } finally {
    if (!silent) automationRunsLoading.value = false
  }
}

async function saveConfig(successMessage) {
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
      shop_action_ids: config.shop_action_ids,
      auto_enabled: config.auto_enabled,
      schedule_time: config.schedule_time || DEFAULT_SCHEDULE_TIME
    })
    ElMessage.success(successMessage)
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存搜索推广商品配置失败')
  } finally {
    savingConfig.value = false
  }
}

async function handleSaveDefaultActions() {
  await saveConfig('默认活动已保存')
}

async function handleSaveAutomationSettings() {
  await saveConfig('自动化设置已保存')
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
  if (totalDefaultActions.value === 0) {
    ElMessage.warning('请先选择默认活动并保存')
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
    ElMessage.success('已创建搜索推广商品报名任务')
    await loadRuns()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '创建任务失败')
  } finally {
    running.value = false
  }
}

async function handleStartAutomationRun() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }
  if (totalDefaultActions.value === 0) {
    ElMessage.warning('请先到“商品池与手动报名”配置默认活动')
    return
  }

  try {
    await ElMessageBox.confirm(
      '自动化执行会刷新商品、同步可推进状态，并按 state1 / state2 / state3 规则推进迁移，是否继续？',
      '确认执行自动化',
      {
        confirmButtonText: '开始执行',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  automationRunning.value = true
  try {
    await startSearchCPOAutomationRun({ shop_id: shopId })
    ElMessage.success('已创建状态迁移自动化任务')
    await Promise.all([loadAutomationRuns(), loadProducts()])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '创建自动化任务失败')
  } finally {
    automationRunning.value = false
  }
}

function updateRunPollingState() {
  const hasRunning = [...runs.value, ...automationRuns.value].some(row => ['pending', 'running'].includes(row.status))
  if (hasRunning) {
    startRunPolling()
  } else {
    stopRunPolling()
  }
}

function startRunPolling() {
  if (runPollTimer) return
  runPollTimer = setInterval(async () => {
    await Promise.allSettled([
      loadRuns(true),
      loadAutomationRuns(true),
      loadProducts(true)
    ])
  }, 2500)
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
    ElMessage.error(error.response?.data?.message || '获取手动报名详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function openAutomationRunDetail(row) {
  const shopId = userStore.currentShopId
  if (!shopId || !row?.id) return

  automationDetailVisible.value = true
  automationDetailLoading.value = true
  automationDetail.value = null
  try {
    const res = await getSearchCPOAutomationRunDetail(row.id, shopId)
    automationDetail.value = res.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '获取状态迁移详情失败')
  } finally {
    automationDetailLoading.value = false
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
  flex-direction: column;
  gap: 16px;
  margin-bottom: 18px;
  padding: 20px 22px;
  border-radius: 18px;
  border: 1px solid var(--cpo-border);
  background: linear-gradient(120deg, rgba(5, 150, 105, 0.08), rgba(249, 115, 22, 0.08));
}

.hero-copy h2 {
  margin: 0;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
}

.hero-subtitle {
  margin: 8px 0 0;
  line-height: 1.6;
  color: var(--cpo-text);
  opacity: 0.9;
  max-width: 820px;
}

.hero-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px;
}

.metric-pill {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 66px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(5, 150, 105, 0.14);
  transition: border-color 0.24s ease, transform 0.24s ease;
}

.metric-pill:hover {
  border-color: rgba(5, 150, 105, 0.34);
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
  line-height: 1.5;
}

.search-cpo-tabs {
  padding-bottom: 8px;
}

.search-cpo-tabs :deep(.el-tabs__header) {
  margin-bottom: 18px;
}

.search-cpo-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: rgba(15, 118, 110, 0.12);
}

.search-cpo-tabs :deep(.el-tabs__item) {
  font-weight: 600;
}

@media (max-width: 640px) {
  .hero-shell {
    padding: 18px 16px;
  }
}
</style>
