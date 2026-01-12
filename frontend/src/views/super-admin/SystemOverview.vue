<template>
  <div class="system-overview">
    <div class="page-header">
      <h2 class="gradient">系统概览</h2>
    </div>

    <div class="stats-grid" v-loading="loading">
      <div class="glass-card stat-card">
        <div class="stat-icon shop-admin">
          <el-icon><User /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ overview.shop_admin_count || 0 }}</div>
          <div class="stat-label">店铺管理员</div>
        </div>
      </div>

      <div class="glass-card stat-card">
        <div class="stat-icon staff">
          <el-icon><UserFilled /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ overview.staff_count || 0 }}</div>
          <div class="stat-label">员工总数</div>
        </div>
      </div>

      <div class="glass-card stat-card">
        <div class="stat-icon shop">
          <el-icon><Shop /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ overview.shop_count || 0 }}</div>
          <div class="stat-label">店铺总数</div>
        </div>
      </div>

      <div class="glass-card stat-card">
        <div class="stat-icon product">
          <el-icon><Goods /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ overview.product_count || 0 }}</div>
          <div class="stat-label">商品总数</div>
        </div>
      </div>
    </div>

    <div class="glass-card">
      <div class="card-header">
        <h3>店铺管理员列表</h3>
      </div>
      <div class="card-body">
        <el-table :data="overview.shop_admins || []" v-loading="loading">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="{ row }">
              <span class="code-text">{{ row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="display_name" label="显示名称" width="120" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="dark" size="small">
                {{ row.status === 'active' ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="店铺数量" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="primary" effect="plain" size="small">
                {{ row.shop_count || 0 }} 个
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="员工数量" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain" size="small">
                {{ row.staff_count || 0 }} 人
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_login_at" label="最后登录" width="180">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.last_login_at) || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { User, UserFilled, Shop, Goods } from '@element-plus/icons-vue'
import { getSystemOverview } from '@/api/admin'

const loading = ref(false)
const overview = reactive({
  shop_admin_count: 0,
  staff_count: 0,
  shop_count: 0,
  product_count: 0,
  shop_admins: []
})

onMounted(async () => {
  await fetchOverview()
})

async function fetchOverview() {
  loading.value = true
  try {
    const res = await getSystemOverview()
    Object.assign(overview, res.data)
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.system-overview {
  min-height: 100%;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 24px;
  gap: 20px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
}

.stat-icon.shop-admin {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.staff {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.stat-icon.shop {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
}

.stat-icon.product {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: white;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.code-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}
</style>
