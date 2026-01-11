<template>
  <div class="dashboard">
    <h2>仪表盘</h2>

    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ stats.totalProducts }}</div>
          <div class="stat-label">商品总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ stats.promotedProducts }}</div>
          <div class="stat-label">已推广商品</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card warning">
          <div class="stat-value">{{ stats.lossProducts }}</div>
          <div class="stat-label">亏损商品</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card success">
          <div class="stat-value">{{ stats.promotableProducts }}</div>
          <div class="stat-label">可推广商品</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="action-row">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>快捷操作</span>
          </template>
          <div class="quick-actions">
            <el-button type="primary" @click="$router.push('/promotions/batch-enroll')">
              批量报名促销
            </el-button>
            <el-button type="warning" @click="$router.push('/promotions/loss-process')">
              处理亏损商品
            </el-button>
            <el-button type="success" @click="$router.push('/promotions/reprice')">
              改价推广
            </el-button>
            <el-button @click="handleExport">
              导出可推广商品
            </el-button>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>最近操作</span>
          </template>
          <el-table :data="recentLogs" size="small" max-height="300">
            <el-table-column prop="operation_type" label="操作类型" width="120">
              <template #default="{ row }">
                {{ formatOperationType(row.operation_type) }}
              </template>
            </el-table-column>
            <el-table-column prop="affected_count" label="影响数量" width="80" />
            <el-table-column prop="status" label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getProducts } from '@/api/product'
import { exportPromotable } from '@/api/promotion'

const userStore = useUserStore()

const stats = ref({
  totalProducts: 0,
  promotedProducts: 0,
  lossProducts: 0,
  promotableProducts: 0
})

const recentLogs = ref([])

onMounted(async () => {
  await fetchStats()
})

async function fetchStats() {
  try {
    const shopId = userStore.currentShopId
    if (!shopId) return

    // 获取商品统计
    const res = await getProducts({ shop_id: shopId, page_size: 1 })
    stats.value.totalProducts = res.data.total

    // 获取已推广商品数
    const promotedRes = await getProducts({ shop_id: shopId, is_promoted: true, page_size: 1 })
    stats.value.promotedProducts = promotedRes.data.total

    // 获取亏损商品数
    const lossRes = await getProducts({ shop_id: shopId, is_loss: true, page_size: 1 })
    stats.value.lossProducts = lossRes.data.total

    // 可推广 = 总数 - 已推广 - 亏损
    stats.value.promotableProducts = stats.value.totalProducts - stats.value.promotedProducts - stats.value.lossProducts
  } catch (error) {
    console.error(error)
  }
}

async function handleExport() {
  try {
    const shopId = userStore.currentShopId
    if (!shopId) {
      ElMessage.warning('请先选择店铺')
      return
    }

    const res = await exportPromotable(shopId)
    // 处理文件下载
    const blob = new Blob([res], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `可推广商品_${new Date().toISOString().split('T')[0]}.xlsx`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (error) {
    console.error(error)
    ElMessage.error('导出失败')
  }
}

function formatOperationType(type) {
  const map = {
    'batch_enroll': '批量报名',
    'process_loss': '处理亏损',
    'remove_reprice_promote': '改价推广',
    'sync_products': '同步商品'
  }
  return map[type] || type
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.dashboard {
  padding: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
  padding: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409EFF;
}

.stat-card.warning .stat-value {
  color: #E6A23C;
}

.stat-card.success .stat-value {
  color: #67C23A;
}

.stat-label {
  margin-top: 10px;
  color: #909399;
}

.action-row {
  margin-top: 20px;
}

.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
</style>
