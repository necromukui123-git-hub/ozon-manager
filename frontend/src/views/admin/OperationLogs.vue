<template>
  <div class="operation-logs">
    <h2>操作日志</h2>

    <el-card class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="操作人">
          <el-select v-model="filters.user_id" placeholder="全部" clearable>
            <el-option
              v-for="user in users"
              :key="user.id"
              :label="user.display_name"
              :value="user.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="店铺">
          <el-select v-model="filters.shop_id" placeholder="全部" clearable>
            <el-option
              v-for="shop in shops"
              :key="shop.id"
              :label="shop.name"
              :value="shop.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="操作类型">
          <el-select v-model="filters.operation_type" placeholder="全部" clearable>
            <el-option label="批量报名" value="batch_enroll" />
            <el-option label="处理亏损" value="process_loss" />
            <el-option label="改价推广" value="remove_reprice_promote" />
            <el-option label="同步商品" value="sync_products" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filters.date_range"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <el-table :data="logs" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="操作人" width="120">
          <template #default="{ row }">
            {{ row.user?.display_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="店铺" width="120">
          <template #default="{ row }">
            {{ row.shop?.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作类型" width="120">
          <template #default="{ row }">
            {{ formatOperationType(row.operation_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="affected_count" label="影响数量" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ formatStatus(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ip_address" label="IP地址" width="140" />
        <el-table-column prop="created_at" label="操作时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button type="primary" size="small" text @click="showDetail(row)">
              详情
            </el-button>
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

    <el-dialog v-model="detailVisible" title="操作详情" width="600px">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="操作人">
          {{ currentLog?.user?.display_name }}
        </el-descriptions-item>
        <el-descriptions-item label="店铺">
          {{ currentLog?.shop?.name }}
        </el-descriptions-item>
        <el-descriptions-item label="操作类型">
          {{ formatOperationType(currentLog?.operation_type) }}
        </el-descriptions-item>
        <el-descriptions-item label="影响数量">
          {{ currentLog?.affected_count }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(currentLog?.status)">
            {{ formatStatus(currentLog?.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="IP地址">
          {{ currentLog?.ip_address }}
        </el-descriptions-item>
        <el-descriptions-item label="操作时间" :span="2">
          {{ formatTime(currentLog?.created_at) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="currentLog?.error_message" label="错误信息" :span="2">
          <span class="error-message">{{ currentLog.error_message }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="currentLog?.operation_detail" class="detail-section">
        <h4>操作详情</h4>
        <pre class="detail-json">{{ JSON.stringify(currentLog.operation_detail, null, 2) }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getOperationLogs } from '@/api/log'
import { getUsers } from '@/api/user'
import { getShops } from '@/api/shop'

const loading = ref(false)
const logs = ref([])
const users = ref([])
const shops = ref([])

const filters = reactive({
  user_id: null,
  shop_id: null,
  operation_type: '',
  date_range: []
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const detailVisible = ref(false)
const currentLog = ref(null)

onMounted(async () => {
  await Promise.all([fetchLogs(), fetchUsers(), fetchShops()])
})

async function fetchLogs() {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.page_size
    }

    if (filters.user_id) {
      params.user_id = filters.user_id
    }
    if (filters.shop_id) {
      params.shop_id = filters.shop_id
    }
    if (filters.operation_type) {
      params.operation_type = filters.operation_type
    }
    if (filters.date_range && filters.date_range.length === 2) {
      params.date_from = filters.date_range[0]
      params.date_to = filters.date_range[1]
    }

    const res = await getOperationLogs(params)
    logs.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function fetchUsers() {
  try {
    const res = await getUsers()
    users.value = res.data || []
  } catch (error) {
    console.error(error)
  }
}

async function fetchShops() {
  try {
    const res = await getShops()
    shops.value = res.data || []
  } catch (error) {
    console.error(error)
  }
}

function handleSearch() {
  pagination.page = 1
  fetchLogs()
}

function handleReset() {
  filters.user_id = null
  filters.shop_id = null
  filters.operation_type = ''
  filters.date_range = []
  pagination.page = 1
  fetchLogs()
}

function handleSizeChange() {
  pagination.page = 1
  fetchLogs()
}

function handlePageChange() {
  fetchLogs()
}

function showDetail(log) {
  currentLog.value = log
  detailVisible.value = true
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

function formatStatus(status) {
  const map = {
    'pending': '进行中',
    'success': '成功',
    'failed': '失败'
  }
  return map[status] || status
}

function getStatusType(status) {
  const map = {
    'pending': 'warning',
    'success': 'success',
    'failed': 'danger'
  }
  return map[status] || 'info'
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.operation-logs {
  padding: 20px;
}

.operation-logs h2 {
  margin-bottom: 20px;
}

.filter-card {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.error-message {
  color: #F56C6C;
}

.detail-section {
  margin-top: 20px;
}

.detail-section h4 {
  margin-bottom: 10px;
}

.detail-json {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  overflow: auto;
  max-height: 300px;
  font-size: 12px;
}
</style>
