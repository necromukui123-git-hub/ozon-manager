<template>
  <div class="batch-enroll">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">批量报名促销</h2>
    </div>

    <!-- 配置表单 -->
    <div class="glass-card">
      <div class="card-header">
        <span class="card-title">报名配置</span>
      </div>
      <div class="card-body">
        <el-form :model="form" label-width="160px" class="config-form">
          <div class="form-section">
            <h4 class="section-title">筛选条件</h4>
            <el-form-item label="排除亏损商品">
              <div class="switch-wrapper">
                <el-switch v-model="form.exclude_loss" />
                <span class="form-hint">开启后将跳过标记为亏损的商品</span>
              </div>
            </el-form-item>

            <el-form-item label="排除已推广商品">
              <div class="switch-wrapper">
                <el-switch v-model="form.exclude_promoted" />
                <span class="form-hint">开启后将跳过已参与推广活动的商品</span>
              </div>
            </el-form-item>
          </div>

          <div class="form-section">
            <h4 class="section-title">选择促销活动</h4>
            <el-form-item v-if="actionsLoading" label="加载中">
              <el-skeleton :rows="2" animated />
            </el-form-item>
            <el-form-item v-else-if="actions.length === 0" label="无可用活动">
              <el-empty description="暂无促销活动，请先同步或添加活动" :image-size="60">
                <el-button type="primary" size="small" @click="$router.push('/promotions/actions')">
                  前往活动管理
                </el-button>
              </el-empty>
            </el-form-item>
            <el-form-item v-else label="促销活动">
              <div class="action-selector">
                <el-checkbox-group v-model="form.action_ids">
                  <el-checkbox
                    v-for="action in actions"
                    :key="action.action_id"
                    :value="action.action_id"
                    class="action-checkbox"
                  >
                    <div class="action-item">
                      <span class="action-title">{{ action.title || `活动 #${action.action_id}` }}</span>
                      <span class="action-id">ID: {{ action.action_id }}</span>
                    </div>
                  </el-checkbox>
                </el-checkbox-group>
                <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
                  <el-icon><Refresh /></el-icon>
                  刷新列表
                </el-button>
              </div>
            </el-form-item>
          </div>

          <el-form-item class="form-actions">
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
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 执行结果 -->
    <div v-if="result" class="glass-card result-card">
      <div class="card-header">
        <span class="card-title">执行结果</span>
        <el-tag :type="result.success ? 'success' : 'danger'" effect="dark">
          {{ result.success ? '成功' : '失败' }}
        </el-tag>
      </div>
      <div class="card-body">
        <div class="result-stats">
          <div class="stat-item">
            <span class="stat-number">{{ result.enrolled_count + result.failed_count }}</span>
            <span class="stat-text">处理商品</span>
          </div>
          <div class="stat-item success">
            <span class="stat-number">{{ result.enrolled_count }}</span>
            <span class="stat-text">成功</span>
          </div>
          <div class="stat-item danger">
            <span class="stat-number">{{ result.failed_count }}</span>
            <span class="stat-text">失败</span>
          </div>
        </div>

        <div v-if="result.details && result.details.filter(d => d.status === 'failed').length > 0" class="failed-section">
          <h4 class="failed-title">
            <el-icon><WarningFilled /></el-icon>
            失败详情
          </h4>
          <el-table :data="result.details.filter(d => d.status === 'failed')" size="small" max-height="300">
            <el-table-column prop="source_sku" label="SKU" width="180">
              <template #default="{ row }">
                <span class="sku-text">{{ row.source_sku }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="error" label="错误信息" />
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { batchEnrollV2, getActions } from '@/api/promotion'
import { Upload, WarningFilled, Refresh } from '@element-plus/icons-vue'

const userStore = useUserStore()

const loading = ref(false)
const actionsLoading = ref(false)
const result = ref(null)
const actions = ref([])

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
    // 默认选中所有活动
    if (form.action_ids.length === 0 && actions.value.length > 0) {
      form.action_ids = actions.value.map(a => a.action_id)
    }
  } catch (error) {
    console.error(error)
  } finally {
    actionsLoading.value = false
  }
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
      `确定要批量报名 ${form.action_ids.length} 个促销活动吗？此操作可能需要一些时间。`,
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
    const res = await batchEnrollV2({
      shop_id: shopId,
      action_ids: form.action_ids,
      exclude_loss: form.exclude_loss,
      exclude_promoted: form.exclude_promoted
    })
    result.value = res.data
    if (res.data.success) {
      ElMessage.success(`批量报名完成，成功 ${res.data.enrolled_count} 个商品`)
    } else {
      ElMessage.warning('部分商品报名失败，请查看详情')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('操作失败')
  } finally {
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

.config-form {
  max-width: 700px;
}

.form-section {
  margin-bottom: 32px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 20px;
  padding-left: 12px;
  border-left: 3px solid var(--primary);
}

.switch-wrapper {
  display: flex;
  align-items: center;
  gap: 16px;
}

.form-hint {
  font-size: 13px;
  color: var(--text-muted);
}

.form-actions {
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid var(--glass-border);
}

.action-selector {
  width: 100%;
}

.action-checkbox {
  display: block;
  margin-bottom: 12px;
  margin-right: 0;
}

.action-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-title {
  font-weight: 500;
}

.action-id {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: var(--text-muted);
}

.result-card {
  margin-top: 24px;
}

.result-stats {
  display: flex;
  gap: 40px;
  margin-bottom: 24px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;

  .stat-number {
    font-size: 36px;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1;
  }

  .stat-text {
    font-size: 13px;
    color: var(--text-muted);
    margin-top: 8px;
  }

  &.success .stat-number {
    color: var(--success);
  }

  &.danger .stat-number {
    color: var(--danger);
  }
}

.failed-section {
  padding-top: 20px;
  border-top: 1px solid var(--glass-border);
}

.failed-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--danger);
  margin-bottom: 16px;
}

.sku-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}
</style>
