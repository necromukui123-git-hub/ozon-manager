<template>
  <div class="product-list">
    <div class="page-header">
      <h2>商品列表</h2>
      <el-button type="primary" :loading="syncing" @click="handleSync">
        同步商品
      </el-button>
    </div>

    <el-card class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="亏损状态">
          <el-select v-model="filters.is_loss" placeholder="全部" clearable>
            <el-option label="亏损商品" :value="true" />
            <el-option label="正常商品" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="推广状态">
          <el-select v-model="filters.is_promoted" placeholder="全部" clearable>
            <el-option label="已推广" :value="true" />
            <el-option label="未推广" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="搜索">
          <el-input
            v-model="filters.keyword"
            placeholder="商品名称/SKU"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card">
      <el-table :data="products" v-loading="loading" stripe>
        <el-table-column prop="source_sku" label="SKU" width="150" />
        <el-table-column prop="name" label="商品名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="current_price" label="当前价格" width="100">
          <template #default="{ row }">
            ¥{{ row.current_price?.toFixed(2) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="亏损" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_loss" type="danger" size="small">亏损</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="推广" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_promoted" type="success" size="small">已推广</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="参与的促销活动" min-width="200">
          <template #default="{ row }">
            <template v-if="row.promotions && row.promotions.length > 0">
              <el-tag
                v-for="promo in row.promotions"
                :key="promo.action_id"
                size="small"
                :type="getPromoTagType(promo.type)"
                style="margin-right: 5px; margin-bottom: 5px;"
              >
                {{ promo.title }}
              </el-tag>
            </template>
            <span v-else class="no-promo">-</span>
          </template>
        </el-table-column>
        <el-table-column label="亏损信息" width="150">
          <template #default="{ row }">
            <template v-if="row.loss_info">
              <div class="loss-info">
                <div>原价: ¥{{ row.loss_info.original_price }}</div>
                <div>新价: ¥{{ row.loss_info.new_price }}</div>
              </div>
            </template>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="pagination"
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.page_size"
        :total="pagination.total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getProducts, syncProducts } from '@/api/product'

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
  if (type === 'elastic_boost') return 'primary'
  if (type === 'discount_28') return 'success'
  return 'info'
}
</script>

<style scoped>
.product-list {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.filter-card {
  margin-bottom: 20px;
}

.table-card {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.no-promo {
  color: #909399;
}

.loss-info {
  font-size: 12px;
  color: #E6A23C;
}
</style>
