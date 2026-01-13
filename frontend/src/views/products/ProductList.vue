<template>
  <div class="product-list">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">商品列表</h2>
      <el-button type="primary" :loading="syncing" @click="handleSync">
        <el-icon><Refresh /></el-icon>
        同步商品
      </el-button>
    </div>

    <!-- 筛选器 -->
    <div class="glass-card filter-card">
      <div class="card-body">
        <el-form :inline="true" :model="filters" class="filter-form">
          <el-form-item label="亏损状态">
            <el-select v-model="filters.is_loss" placeholder="全部" clearable style="width: 140px">
              <el-option label="亏损商品" :value="true" />
              <el-option label="正常商品" :value="false" />
            </el-select>
          </el-form-item>
          <el-form-item label="推广状态">
            <el-select v-model="filters.is_promoted" placeholder="全部" clearable style="width: 140px">
              <el-option label="已推广" :value="true" />
              <el-option label="未推广" :value="false" />
            </el-select>
          </el-form-item>
          <el-form-item label="搜索">
            <el-input
              v-model="filters.keyword"
              placeholder="商品名称 / SKU"
              clearable
              style="width: 200px"
              @keyup.enter="handleSearch"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
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
      </div>
    </div>

    <!-- 数据表格 -->
    <div class="glass-card table-card">
      <div class="card-body">
        <el-table :data="products" v-loading="loading">
          <el-table-column prop="source_sku" label="SKU" width="150">
            <template #default="{ row }">
              <span class="sku-text">{{ row.source_sku }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="商品名称" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="product-name">{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="current_price" label="当前价格" width="110" align="right">
            <template #default="{ row }">
              <span class="price-text">¥{{ row.current_price?.toFixed(2) || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="160" align="center">
            <template #default="{ row }">
              <div class="status-tags">
                <el-tag v-if="row.is_loss" type="danger" size="small" effect="dark">
                  <el-icon><WarningFilled /></el-icon>
                  亏损
                </el-tag>
                <el-tag v-if="row.is_promoted" type="success" size="small" effect="dark">
                  <el-icon><Promotion /></el-icon>
                  推广中
                </el-tag>
                <span v-if="!row.is_loss && !row.is_promoted" class="status-normal">正常</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="参与的促销" min-width="200">
            <template #default="{ row }">
              <template v-if="row.promotions && row.promotions.length > 0">
                <div class="promo-tags">
                  <el-tag
                    v-for="promo in row.promotions"
                    :key="promo.action_id"
                    size="small"
                    :type="getPromoTagType(promo.type)"
                  >
                    {{ promo.title }}
                  </el-tag>
                </div>
              </template>
              <span v-else class="no-data">-</span>
            </template>
          </el-table-column>
          <el-table-column label="亏损信息" width="140">
            <template #default="{ row }">
              <template v-if="row.loss_info">
                <div class="loss-info">
                  <div class="loss-row">
                    <span class="loss-label">原价</span>
                    <span class="loss-value old">¥{{ row.loss_info.original_price }}</span>
                  </div>
                  <div class="loss-row">
                    <span class="loss-label">新价</span>
                    <span class="loss-value new">¥{{ row.loss_info.new_price }}</span>
                  </div>
                </div>
              </template>
              <span v-else class="no-data">-</span>
            </template>
          </el-table-column>
        </el-table>

        <div class="table-footer">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.page_size"
            :total="pagination.total"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getProducts, syncProducts } from '@/api/product'
import {
  Refresh,
  RefreshLeft,
  Search,
  WarningFilled,
  Promotion
} from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const syncing = ref(false)
const products = ref([])

const filters = reactive({
  is_loss: null,
  is_promoted: null,
  keyword: ''
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

watch(() => userStore.currentShopId, () => {
  fetchProducts()
})

onMounted(() => {
  fetchProducts()
})

async function fetchProducts() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  loading.value = true
  try {
    const params = {
      shop_id: shopId,
      page: pagination.page,
      page_size: pagination.page_size
    }

    if (filters.is_loss !== null) {
      params.is_loss = filters.is_loss
    }
    if (filters.is_promoted !== null) {
      params.is_promoted = filters.is_promoted
    }
    if (filters.keyword) {
      params.keyword = filters.keyword
    }

    const res = await getProducts(params)
    products.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (error) {
    console.error(error)
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
    await syncProducts(shopId)
    ElMessage.success('同步成功')
    await fetchProducts()
  } catch (error) {
    console.error(error)
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  fetchProducts()
}

function handleReset() {
  filters.is_loss = null
  filters.is_promoted = null
  filters.keyword = ''
  pagination.page = 1
  fetchProducts()
}

function handleSizeChange() {
  pagination.page = 1
  fetchProducts()
}

function handlePageChange() {
  fetchProducts()
}

function getPromoTagType(type) {
  return 'info'
}
</script>

<style scoped>
.product-list {
  min-height: 100%;
}

.filter-card {
  margin-bottom: 24px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-end;
}

.table-card .card-body {
  padding: 0;
}

.table-footer {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid var(--glass-border);
}

.sku-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.product-name {
  color: var(--text-primary);
}

.price-text {
  font-weight: 600;
  color: var(--text-primary);
}

.status-tags {
  display: flex;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}

.status-tags .el-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.status-normal {
  color: var(--text-muted);
  font-size: 13px;
}

.promo-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.no-data {
  color: var(--text-disabled);
}

.loss-info {
  font-size: 12px;
}

.loss-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 2px;
}

.loss-label {
  color: var(--text-muted);
}

.loss-value {
  font-weight: 500;

  &.old {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  &.new {
    color: var(--warning);
  }
}
</style>
