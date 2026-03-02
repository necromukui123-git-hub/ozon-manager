<template>
  <div class="action-products">
    <div class="page-header">
      <div>
        <h2 class="gradient">活动商品</h2>
        <p class="subtitle">{{ actionTitle || `活动 #${actionId}` }} · {{ actionSourceLabel }}</p>
      </div>
      <div class="header-actions">
        <el-button @click="goBack">返回活动列表</el-button>
        <el-button type="primary" :loading="refreshing" @click="fetchProducts(true)">刷新</el-button>
      </div>
    </div>

    <div class="table-card" v-loading="loading">
      <el-table :data="items" stripe border>
        <el-table-column prop="source_sku" label="SKU" min-width="140" />
        <el-table-column prop="name" label="商品名称" min-width="220" />
        <el-table-column prop="ozon_product_id" label="Ozon Product ID" min-width="140" />
        <el-table-column prop="price" label="原价" min-width="100" />
        <el-table-column prop="action_price" label="活动价" min-width="100" />
        <el-table-column prop="stock" label="库存" min-width="80" />
        <el-table-column prop="status" label="状态" min-width="100" />
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
import { computed, onMounted, ref } from 'vue'
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

const actionId = computed(() => Number(route.params.id))
const shopId = computed(() => Number(route.query.shop_id || 0))
const actionTitle = computed(() => String(route.query.title || ''))
const actionSource = computed(() => String(route.query.source || 'official'))
const actionSourceLabel = computed(() => actionSource.value === 'shop' ? '店铺促销' : '官方促销')

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
    const res = await getActionProducts(actionId.value, shopId.value, {
      page: page.value,
      page_size: pageSize.value,
      force_refresh: forceRefresh
    })
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

function goBack() {
  router.push({ name: 'ActionList' })
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
}

.subtitle {
  margin-top: 6px;
  color: var(--text-secondary);
}

.table-card {
  background: var(--bg-secondary);
  border: 2px solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: var(--neo-shadow);
  padding: 12px;
}

.pager-wrap {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}
</style>
