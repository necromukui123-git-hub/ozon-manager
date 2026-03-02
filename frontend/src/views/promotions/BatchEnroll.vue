<template>
  <div class="batch-enroll">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">批量报名促销</h2>
    </div>

    <!-- Bento Grid 布局 -->
    <div class="bento-grid--2col">
      <!-- 筛选条件卡片 -->
      <BentoCard title="筛选条件" :icon="Filter" size="1x1">
        <div class="filter-options">
          <div class="filter-item">
            <div class="filter-label">
              <el-icon><RemoveFilled /></el-icon>
              <span>排除亏损商品</span>
            </div>
            <el-switch v-model="form.exclude_loss" />
          </div>
          <div class="filter-desc">开启后将跳过标记为亏损的商品</div>

          <div class="filter-item">
            <div class="filter-label">
              <el-icon><CircleCloseFilled /></el-icon>
              <span>排除已推广商品</span>
            </div>
            <el-switch v-model="form.exclude_promoted" />
          </div>
          <div class="filter-desc">开启后将跳过已参与推广活动的商品</div>
        </div>
      </BentoCard>

      <!-- 选择活动卡片 -->
      <BentoCard title="选择促销活动" :icon="Ticket" size="1x1">
        <template #actions>
          <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </template>

        <div v-if="actionsLoading" class="loading-state">
          <el-skeleton :rows="3" animated />
        </div>
        <div v-else-if="actions.length === 0" class="empty-state-small">
          <el-icon><Box /></el-icon>
          <p>暂无促销活动</p>
          <el-button type="primary" size="small" @click="$router.push('/promotions/actions')">
            前往活动管理
          </el-button>
        </div>
        <div v-else class="action-list">
          <el-checkbox-group v-model="form.action_ids">
            <el-checkbox
              v-for="action in actions"
              :key="action.id"
              :value="action.id"
              class="action-checkbox"
            >
              <div class="action-item">
                <span class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</span>
                <span class="action-id">ID: {{ action.source === 'shop' ? action.source_action_id : action.action_id }}</span>
              </div>
            </el-checkbox>
          </el-checkbox-group>
        </div>
      </BentoCard>
    </div>

    <!-- 操作按钮 -->
    <div class="action-bar">
      <div class="selected-info">
        <el-icon><Checked /></el-icon>
        <span>已选择 <strong>{{ form.action_ids.length }}</strong> 个活动</span>
      </div>
      <el-button
        type="primary"
        size="large"
        :loading="loading"
        :disabled="form.action_ids.length === 0"
        @click="handleSubmit"
      >
        <el-icon v-if="!loading"><Upload /></el-icon>
        {{ loading ? '处理中...' : '开始报名' }}
      </el-button>
    </div>

    <!-- 执行结果 -->
    <div v-if="result" class="bento-grid">
      <!-- 结果统计卡片 -->
      <StatCard
        :value="result.enrolled_count + result.failed_count"
        label="处理商品"
        :icon="Goods"
        variant="primary"
      />
      <StatCard
        :value="result.enrolled_count"
        label="成功"
        :icon="CircleCheckFilled"
        variant="success"
      />
      <StatCard
        :value="result.failed_count"
        label="失败"
        :icon="CircleCloseFilled"
        variant="danger"
      />
      <StatCard
        :value="result.success ? '成功' : '失败'"
        label="执行状态"
        :icon="result.success ? CircleCheckFilled : WarningFilled"
        :variant="result.success ? 'success' : 'danger'"
      />

      <!-- 失败详情 -->
      <BentoCard
        v-if="result.details && result.details.filter(d => d.status === 'failed').length > 0"
        title="失败详情"
        :icon="WarningFilled"
        size="4x1"
        no-padding
      >
        <el-table :data="result.details.filter(d => d.status === 'failed')" size="small" max-height="300">
          <el-table-column prop="source_sku" label="SKU" width="180">
            <template #default="{ row }">
              <span class="sku-text">{{ row.source_sku }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="error" label="错误信息" />
        </el-table>
      </BentoCard>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { unifiedEnroll, getActions } from '@/api/promotion'
import { getJobDetail } from '@/api/automation'
import { StatCard, BentoCard } from '@/components/bento'
import {
  Upload, WarningFilled, Refresh, Filter, Ticket, Box, Checked,
  CircleCheckFilled, CircleCloseFilled, RemoveFilled, Goods
} from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const actionsLoading = ref(false)
const result = ref(null)
const actions = ref([])
const pollTimer = ref(null)

const form = reactive({
  exclude_loss: true,
  exclude_promoted: true,
  action_ids: []
})

async function fetchActions() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  actionsLoading.value = true
  try {
    const res = await getActions(shopId)
    actions.value = res.data || []
    if (form.action_ids.length === 0 && actions.value.length > 0) {
      form.action_ids = actions.value.map(a => a.id)
    }
  } catch (error) {
    console.error(error)
  } finally {
    actionsLoading.value = false
  }
}

async function startPolling(jobId, shopId) {
  loading.value = true
  
  if (pollTimer.value) clearInterval(pollTimer.value)
  ElMessage.info('已提交后台异步处理，正在获取执行结果...')

  pollTimer.value = setInterval(async () => {
    try {
      const res = await getJobDetail(jobId, shopId)
      const job = res.data

      if (['success', 'partial_success', 'failed', 'canceled'].includes(job.status)) {
        clearInterval(pollTimer.value)
        pollTimer.value = null
        loading.value = false

        // 组装统一 result 对象给界面渲染
        const isSuccess = job.status === 'success' || job.status === 'partial_success'
        result.value = {
          success: isSuccess,
          enrolled_count: job.success_items || 0,
          failed_count: job.failed_items || 0,
          details: (job.items || []).filter(i => i.overall_status === 'failed').map(i => ({
            source_sku: i.source_sku,
            status: 'failed',
            error: i.step_exit_error || i.step_reprice_error || i.step_readd_error || '店铺任务执行失败'
          }))
        }

        if (job.status === 'success') {
          ElMessage.success(`异步执行完成，成功 ${job.success_items} 个商品`)
        } else {
          ElMessage.warning(`异步执行完成，部分商品失败。成功 ${job.success_items} 个`)
        }
      }
    } catch (e) {
      console.error('Job polling failed', e)
      clearInterval(pollTimer.value)
      pollTimer.value = null
      loading.value = false
      ElMessage.error('无法轮询任务状态，请重试')
    }
  }, 3000)
}

async function handleSubmit() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  if (form.action_ids.length === 0) {
    ElMessage.warning('请至少选择一个促销活动')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要对所选活动执行自动改价并报名吗？\n包含官方活动时将同步处理，包含店铺活动时将异步处理。`,
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  loading.value = true
  result.value = null

  try {
    const res = await unifiedEnroll({
      shop_id: shopId,
      action_ids: form.action_ids,
      exclude_loss: form.exclude_loss,
      exclude_promoted: form.exclude_promoted
    })

    if (res.data && res.data.mode === 'sync') {
      result.value = res.data.results
      if (result.value.success) {
        ElMessage.success(`同步处理完成，成功 ${result.value.enrolled_count} 个商品`)
      } else {
        ElMessage.warning('同步处理发生部分失败，请查看明细')
      }
      loading.value = false
    } else if (res.data && res.data.mode === 'async') {
      // 进入轮询
      startPolling(res.data.job_id, shopId)
    } else {
      loading.value = false
      ElMessage.error('未知响应模式')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('操作提交失败')
    loading.value = false
  }
}

watch(() => userStore.currentShopId, () => {
  form.action_ids = []
  fetchActions()
})

onMounted(() => {
  fetchActions()
})
</script>

<style scoped>
.batch-enroll {
  min-height: 100%;
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }
}

.filter-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: var(--bg-tertiary);
  border-radius: var(--neo-radius);
  border: 2px solid var(--neo-border-color);
  box-shadow: 2px 2px 0 var(--neo-border-color);
}

.filter-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: var(--text-primary);
}

.filter-label .el-icon {
  color: var(--primary);
}

.filter-desc {
  font-size: 12px;
  color: var(--text-muted);
  padding-left: 12px;
  margin-bottom: 8px;
}

.loading-state {
  padding: 20px 0;
}

.empty-state-small {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px;
  color: var(--text-muted);
}

.empty-state-small .el-icon {
  font-size: 32px;
  opacity: 0.5;
}

.empty-state-small p {
  font-size: 13px;
  margin: 0;
}

.action-list {
  max-height: 200px;
  overflow-y: auto;
}

.action-checkbox {
  display: block;
  margin-bottom: 8px;
  margin-right: 0;
}

.action-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-title {
  font-weight: 500;
  font-size: 13px;
}

.action-id {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 11px;
  color: var(--text-muted);
}

.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--bg-secondary);
  border: var(--neo-border-width) solid var(--neo-border-color);
  border-radius: var(--neo-radius);
  box-shadow: var(--neo-shadow-offset) var(--neo-shadow-offset) 0 var(--neo-border-color);
  margin-bottom: 24px;
}

.selected-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
}

.selected-info .el-icon {
  color: var(--primary);
}

.selected-info strong {
  color: var(--primary);
  font-weight: 600;
}

.sku-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

/* 响应式 */
@media (max-width: 768px) {
  .action-bar {
    flex-direction: column;
    gap: 16px;
  }
}
</style>
