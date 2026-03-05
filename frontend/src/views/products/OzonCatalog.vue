<template>
  <div class="ozon-catalog">
    <div class="page-header">
      <h2 class="gradient">Ozon 商品列表（实时）</h2>
      <div class="header-actions">
        <el-tag v-if="refreshStatus.running" type="warning">刷新中...</el-tag>
        <el-button :loading="refreshing" type="primary" @click="handleManualRefresh">
          <el-icon><Refresh /></el-icon>
          刷新 Ozon
        </el-button>
      </div>
    </div>

    <BentoCard title="筛选条件" :icon="Filter" size="2x1">
      <el-form :inline="true" class="filter-form">
        <el-form-item label="可见性">
          <el-select v-model="filters.visibility" style="width: 130px">
            <el-option label="全部" value="ALL" />
            <el-option label="可见" value="VISIBLE" />
            <el-option label="不可见" value="INVISIBLE" />
            <el-option label="归档" value="ARCHIVED" />
          </el-select>
        </el-form-item>

        <el-form-item label="Offer IDs">
          <el-input
            v-model="filters.offer_ids"
            placeholder="逗号分隔"
            style="width: 220px"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>

        <el-form-item label="Product IDs">
          <el-input
            v-model="filters.product_ids"
            placeholder="逗号分隔"
            style="width: 220px"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>

        <el-form-item label="上架日期">
          <el-date-picker
            v-model="filters.listed_range"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>

        <el-form-item label="日期来源">
          <el-select v-model="filters.listing_date_source" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="Ozon 时间" value="ozon" />
            <el-option label="本地同步时间" value="local_sync" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="handleReset">
            <el-icon><RefreshLeft /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
      <div class="sync-hint">
        <span>最近缓存同步：{{ lastCacheSyncAt || '-' }}</span>
        <span v-if="refreshStatus.last_error" class="error-text">最近错误：{{ refreshStatus.last_error }}</span>
      </div>
    </BentoCard>

    <BentoCard title="商品数据" :icon="List" size="4x1" class="table-card" no-padding>
      <el-table :data="items" v-loading="loading" row-key="ozon_product_id">
        <el-table-column label="图片" width="88" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.primary_image_url"
              :src="row.primary_image_url"
              class="thumb"
              fit="cover"
              :preview-src-list="[row.primary_image_url]"
              preview-teleported
            />
            <span v-else class="no-data">-</span>
          </template>
        </el-table-column>

        <el-table-column label="商品信息" min-width="280">
          <template #default="{ row }">
            <div class="name">{{ row.name || '-' }}</div>
            <div class="meta">Offer: {{ row.offer_id || '-' }}</div>
            <div class="meta">Product: {{ row.ozon_product_id || '-' }}</div>
            <div class="meta">SKU: {{ row.sku || '-' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="价格" min-width="170" align="right">
          <template #default="{ row }">
            <div class="price">现价 {{ formatMoney(row.price, row.currency) }}</div>
            <div class="meta">原价 {{ formatMoney(row.old_price, row.currency) }}</div>
            <div class="meta">营销 {{ formatMoney(row.marketing_price, row.currency) }}</div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="150" align="center">
          <template #default="{ row }">
            <div class="status-tags">
              <el-tag size="small" effect="plain">{{ row.visibility || '-' }}</el-tag>
              <el-tag size="small" type="info">{{ row.status || '-' }}</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="库存" width="160" align="center">
          <template #default="{ row }">
            <div class="meta">总 {{ row.stock_total ?? 0 }}</div>
            <div class="meta">FBO {{ row.stock_fbo ?? 0 }}</div>
            <div class="meta">FBS {{ row.stock_fbs ?? 0 }}</div>
          </template>
        </el-table-column>

        <el-table-column label="上架日期" width="170" align="center">
          <template #default="{ row }">
            <div>{{ row.listing_date || '-' }}</div>
            <el-tag :type="row.listing_date_source === 'ozon' ? 'success' : 'warning'" size="small">
              {{ row.listing_date_source === 'ozon' ? 'ozon' : 'local_sync' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="last_synced_at" label="最近同步" width="170" align="center" />
      </el-table>

      <template #footer>
        <div class="cursor-pager">
          <div class="pager-left">共 {{ total }} 条，当前第 {{ pageIndex + 1 }} 页</div>
          <div class="pager-right">
            <el-button :disabled="pageIndex === 0 || loading" @click="handlePrevPage">上一页</el-button>
            <el-button :disabled="!hasNext || loading" type="primary" @click="handleNextPage">下一页</el-button>
          </div>
        </div>
      </template>
    </BentoCard>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getOzonCatalog, refreshOzonCatalog } from '@/api/product'
import { BentoCard } from '@/components/bento'
import { Filter, List, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const refreshing = ref(false)
const items = ref([])
const total = ref(0)
const hasNext = ref(false)
const nextCursor = ref('')
const pageIndex = ref(0)
const cursorTrail = ref([''])
const lastCacheSyncAt = ref('')
const refreshStatus = reactive({
  running: false,
  last_started_at: '',
  last_finished_at: '',
  last_error: ''
})
let pollTimer = null

const filters = reactive({
  visibility: 'ALL',
  offer_ids: '',
  product_ids: '',
  listed_range: [],
  listing_date_source: 'all'
})

const pageSize = 20

watch(
  () => userStore.currentShopId,
  () => {
    resetPager()
    fetchCatalog()
    triggerRefresh('page_enter', false)
  }
)

onMounted(() => {
  fetchCatalog()
  triggerRefresh('page_enter', false)
})

onUnmounted(() => {
  clearPoll()
})

async function fetchCatalog(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) loading.value = true
  try {
    const cursor = cursorTrail.value[pageIndex.value] || ''
    const params = {
      shop_id: shopId,
      page_size: pageSize,
      cursor,
      visibility: filters.visibility,
      listing_date_source: filters.listing_date_source
    }
    if (filters.offer_ids.trim()) {
      params.offer_ids = filters.offer_ids.trim()
    }
    if (filters.product_ids.trim()) {
      params.product_ids = filters.product_ids.trim()
    }
    if (filters.listed_range && filters.listed_range.length === 2) {
      params.listed_from = filters.listed_range[0]
      params.listed_to = filters.listed_range[1]
    }

    const res = await getOzonCatalog(params)
    const data = res.data || {}
    items.value = data.items || []
    total.value = data.total || 0
    nextCursor.value = data.next_cursor || ''
    hasNext.value = !!data.has_next
    lastCacheSyncAt.value = data.last_cache_sync_at || ''
    applyRefreshStatus(data.refresh_status || {})
  } catch (error) {
    console.error(error)
    if (!silent) ElMessage.error(error.response?.data?.message || '加载 Ozon 商品失败')
  } finally {
    if (!silent) loading.value = false
  }
}

function applyRefreshStatus(status) {
  refreshStatus.running = !!status.running
  refreshStatus.last_started_at = status.last_started_at || ''
  refreshStatus.last_finished_at = status.last_finished_at || ''
  refreshStatus.last_error = status.last_error || ''
}

async function triggerRefresh(reason, force) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  try {
    const res = await refreshOzonCatalog({
      shop_id: shopId,
      reason,
      force
    })
    const status = res.data?.status || ''
    if (status === 'started' || status === 'running') {
      startPolling()
    }
  } catch (error) {
    console.error(error)
    if (reason === 'manual') {
      ElMessage.error(error.response?.data?.message || '触发刷新失败')
    }
  }
}

function startPolling() {
  clearPoll()
  const loop = async (attempt = 0) => {
    if (attempt > 40) {
      clearPoll()
      return
    }
    await fetchCatalog(true)
    if (!refreshStatus.running) {
      clearPoll()
      return
    }
    pollTimer = setTimeout(() => loop(attempt + 1), 2500)
  }
  pollTimer = setTimeout(() => loop(0), 1200)
}

function clearPoll() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

async function handleManualRefresh() {
  refreshing.value = true
  try {
    await triggerRefresh('manual', true)
    ElMessage.success('已触发刷新')
  } finally {
    refreshing.value = false
  }
}

function resetPager() {
  pageIndex.value = 0
  cursorTrail.value = ['']
}

function handleSearch() {
  resetPager()
  fetchCatalog()
}

function handleReset() {
  filters.visibility = 'ALL'
  filters.offer_ids = ''
  filters.product_ids = ''
  filters.listed_range = []
  filters.listing_date_source = 'all'
  resetPager()
  fetchCatalog()
}

function handlePrevPage() {
  if (pageIndex.value <= 0) return
  pageIndex.value -= 1
  fetchCatalog()
}

function handleNextPage() {
  if (!hasNext.value || !nextCursor.value) return
  if (pageIndex.value === cursorTrail.value.length - 1) {
    cursorTrail.value.push(nextCursor.value)
  }
  pageIndex.value += 1
  fetchCatalog()
}

function formatMoney(value, currency) {
  const number = Number(value)
  if (!Number.isFinite(number) || number <= 0) return '-'
  const suffix = currency ? ` ${currency}` : ''
  return `${number.toFixed(2)}${suffix}`
}
</script>

<style scoped>
.ozon-catalog {
  min-height: 100%;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
}

.sync-hint {
  margin-top: 6px;
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-secondary);
}

.error-text {
  color: #dc2626;
}

.table-card {
  margin-top: 16px;
}

.thumb {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  background: #f5f5f5;
}

.name {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.price {
  font-weight: 600;
  color: var(--text-primary);
}

.meta {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.status-tags {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.cursor-pager {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.pager-left {
  font-size: 13px;
  color: var(--text-secondary);
}

.pager-right {
  display: flex;
  gap: 8px;
}

.no-data {
  color: var(--text-tertiary);
}

@media (max-width: 900px) {
  .sync-hint {
    flex-direction: column;
    gap: 4px;
  }
}
</style>
