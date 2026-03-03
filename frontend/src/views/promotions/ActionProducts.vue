<template>
  <div class="action-products">
    <div class="page-header">
      <div class="header-main">
        <h2 class="gradient">活动商品</h2>
        <p class="subtitle">{{ actionTitle || `活动 #${actionId}` }} · {{ actionSourceLabel }}</p>
      </div>
      <div class="header-actions">
        <el-button @click="goBack">返回活动列表</el-button>
        <el-button type="primary" :loading="refreshing" @click="fetchProducts(true)">刷新数据</el-button>
      </div>
    </div>

    <div class="filter-card">
      <el-input
        v-model.trim="filters.keyword"
        placeholder="搜索 SKU / 中文名 / 原文名"
        clearable
        class="filter-keyword"
        @keyup.enter="handleSearch"
      />
      <el-select v-model="filters.status" class="filter-status">
        <el-option label="全部状态" value="all" />
        <el-option label="进行中" value="active" />
        <el-option label="未生效/停用" value="inactive" />
      </el-select>
      <el-button type="primary" @click="handleSearch">搜索</el-button>
      <el-button @click="resetFilters">重置</el-button>
      <div class="summary-inline">
        <span>总计 {{ total }} 条</span>
        <span>当前页进行中 {{ activeCount }} 条</span>
      </div>
    </div>

    <div class="table-card" v-loading="loading">
      <el-table :data="items" row-key="id" stripe border>
        <el-table-column label="图片" width="90" align="center">
          <template #default="{ row }">
            <div class="thumb-wrap">
              <img
                v-if="row.thumbnail_url && !row.__img_error"
                :src="row.thumbnail_url"
                alt="thumb"
                class="thumb-img"
                @error="() => onImageError(row)"
              />
              <div v-else class="thumb-placeholder">NO IMG</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="商品信息" min-width="360">
          <template #default="{ row }">
            <div class="product-main">
              <div class="name-cn" :title="displayNameCN(row)">{{ displayNameCN(row) }}</div>
              <div class="name-origin" :title="displayNameOrigin(row)">{{ displayNameOrigin(row) }}</div>
              <div class="meta-line">
                <el-tag size="small" effect="plain">{{ row.category_name || '未分类' }}</el-tag>
              </div>
              <div class="id-grid">
                <div class="id-row">
                  <span class="id-label">Offer ID</span>
                  <span class="mono">{{ row.offer_id || '-' }}</span>
                </div>
                <div class="id-row">
                  <span class="id-label">平台 SKU</span>
                  <span class="mono">{{ row.platform_sku || row.source_sku || '-' }}</span>
                </div>
                <div class="id-row">
                  <span class="id-label">Product ID</span>
                  <span class="mono">{{ row.ozon_product_id || '-' }}</span>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="价格结构" min-width="240">
          <template #default="{ row }">
            <div class="price-block">
              <div>
                <span class="label">原价</span>
                <span class="value old">{{ formatMoney(row.base_price || row.price, row.currency) }}</span>
              </div>
              <div>
                <span class="label">活动价</span>
                <span class="value">{{ formatMoney(row.action_price, row.currency) }}</span>
              </div>
              <div>
                <span class="label">折扣</span>
                <span class="value discount">{{ formatPercent(row.discount_percent) }}</span>
              </div>
              <div v-if="row.marketplace_price > 0">
                <span class="label">平台参考</span>
                <span class="value">{{ formatMoney(row.marketplace_price, row.currency) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="库存" min-width="130">
          <template #default="{ row }">
            <div class="stock-block">
              <div>卖家: <b>{{ formatStock(row.seller_stock, row.stock) }}</b></div>
              <div>平台: <b>{{ formatStock(row.ozon_stock) }}</b></div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="light">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="last_synced_at" label="同步时间" min-width="160" />
      </el-table>

      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :current-page="page"
          :page-size="pageSize"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getActionProducts } from '@/api/promotion'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const refreshing = ref(false)
const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const filters = reactive({
  keyword: '',
  status: 'all'
})

const actionId = computed(() => Number(route.params.id))
const shopId = computed(() => Number(route.query.shop_id || 0))
const actionTitle = computed(() => String(route.query.title || ''))
const actionSource = computed(() => String(route.query.source || 'official'))
const actionSourceLabel = computed(() => actionSource.value === 'shop' ? '店铺促销' : '官方促销')
const activeCount = computed(() => items.value.filter(item => String(item.status || '').toLowerCase() === 'active').length)

function buildQuery(forceRefresh) {
  const query = {
    page: page.value,
    page_size: pageSize.value,
    force_refresh: forceRefresh
  }
  if (filters.keyword) query.keyword = filters.keyword
  if (filters.status && filters.status !== 'all') query.status = filters.status
  return query
}

async function fetchProducts(forceRefresh = false) {
  if (!actionId.value || !shopId.value) {
    ElMessage.warning('参数缺失，无法加载活动商品')
    return
  }

  if (forceRefresh) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  try {
    const res = await getActionProducts(actionId.value, shopId.value, buildQuery(forceRefresh))
    const payload = res.data || {}
    items.value = payload.items || []
    total.value = payload.total || 0
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '加载活动商品失败')
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function handlePageChange(nextPage) {
  page.value = nextPage
  fetchProducts(false)
}

function handleSearch() {
  page.value = 1
  fetchProducts(false)
}

function resetFilters() {
  filters.keyword = ''
  filters.status = 'all'
  handleSearch()
}

function goBack() {
  router.push({ name: 'ActionList' })
}

function displayNameCN(row) {
  return row.category_name || row.name_cn || row.name || '-'
}

function displayNameOrigin(row) {
  return row.name_origin || row.name || '-'
}

function statusType(rawStatus) {
  const status = String(rawStatus || '').toLowerCase()
  if (status === 'active') return 'success'
  if (status === 'inactive') return 'warning'
  return 'info'
}

function statusLabel(rawStatus) {
  const status = String(rawStatus || '').toLowerCase()
  if (status === 'active') return '进行中'
  if (status === 'inactive') return '未生效'
  return status || '未知'
}

function formatMoney(raw, currency = 'CNY') {
  const amount = Number(raw || 0)
  if (!Number.isFinite(amount)) return '-'
  try {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: String(currency || 'CNY').toUpperCase(),
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${String(currency || 'CNY').toUpperCase()}`
  }
}

function formatPercent(raw) {
  const value = Number(raw || 0)
  if (!Number.isFinite(value) || value <= 0) return '-'
  return `${value.toFixed(0)}%`
}

function formatStock(primary, fallback) {
  const primaryNum = Number(primary)
  if (Number.isFinite(primaryNum)) return primaryNum
  const fallbackNum = Number(fallback)
  if (Number.isFinite(fallbackNum)) return fallbackNum
  return '-'
}

function onImageError(row) {
  row.__img_error = true
}

onMounted(() => {
  fetchProducts(false)
})
</script>

<style scoped>
.action-products {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.subtitle {
  margin-top: 6px;
  color: var(--text-secondary);
}

.filter-card {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  background: var(--bg-secondary);
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: var(--neo-shadow);
  padding: 12px;
}

.filter-keyword {
  width: 320px;
}

.filter-status {
  width: 140px;
}

.summary-inline {
  margin-left: auto;
  display: flex;
  gap: 14px;
  color: var(--text-secondary);
  font-size: 13px;
}

.table-card {
  background: var(--bg-secondary);
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: var(--neo-shadow);
  padding: 12px;
}

.thumb-wrap {
  width: 52px;
  height: 52px;
}

.thumb-img,
.thumb-placeholder {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  border: 1px solid #dcdfe6;
}

.thumb-img {
  object-fit: cover;
}

.thumb-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  color: #909399;
  font-size: 10px;
  font-weight: 600;
}

.product-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name-cn {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.name-origin {
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-line {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  font-size: 12px;
}

.id-line {
  color: var(--text-secondary);
}

.id-grid {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 2px;
}

.id-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.id-label {
  min-width: 68px;
  color: #909399;
}

.mono {
  font-family: Consolas, Monaco, monospace;
}

.price-block,
.stock-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.label {
  color: var(--text-secondary);
  margin-right: 6px;
}

.value {
  font-weight: 600;
  color: var(--text-primary);
}

.value.old {
  color: #606266;
}

.value.discount {
  color: #e6a23c;
}

.pager-wrap {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 900px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .summary-inline {
    margin-left: 0;
    width: 100%;
  }

  .filter-keyword,
  .filter-status {
    width: 100%;
  }
}
</style>
