<template>
  <div class="section-shell">
    <div class="section-bar">
      <div class="section-copy">
        <div class="section-title">状态迁移自动化</div>
        <div class="section-tip">
          自动触发和“手动执行一次”共用同一条迁移链路：先刷新搜索推广商品，再同步可推进状态，最后按 state1 到 state3 规则收口。
        </div>
        <div class="section-meta">
          <span class="section-chip">自动触发：{{ config.auto_enabled ? '已启用' : '未启用' }}</span>
          <span class="section-chip">触发时间：{{ config.schedule_time || '09:05' }}</span>
          <span class="section-chip">最近任务：{{ latestRunText }}</span>
        </div>
      </div>

      <div class="section-actions">
        <el-button :loading="savingConfig" @click="emit('save-automation')">保存自动化设置</el-button>
        <el-button
          type="primary"
          plain
          :loading="automationRunning"
          :disabled="totalDefaultActions === 0"
          @click="emit('start-run')"
        >
          手动执行一次
        </el-button>
      </div>
    </div>

    <div class="bento-grid--2col">
      <BentoCard title="自动执行设置" :icon="Clock" size="1x1">
        <el-form label-width="90px" class="automation-form">
          <el-form-item label="自动触发">
            <div class="inline-row">
              <el-switch v-model="config.auto_enabled" />
              <span class="inline-tip">{{ config.auto_enabled ? '已启用定时调度' : '当前仅支持手动触发' }}</span>
            </div>
          </el-form-item>
          <el-form-item label="触发时间">
            <div class="inline-row">
              <el-time-select
                v-model="config.schedule_time"
                start="00:00"
                step="00:05"
                end="23:55"
                style="width: 160px"
              />
              <span class="inline-tip">按服务器分钟粒度执行</span>
            </div>
          </el-form-item>
          <el-form-item>
            <div class="form-tip">
              默认活动与“商品池与手动报名”共享同一份配置；这里只保存自动触发开关和时间。
            </div>
          </el-form-item>
        </el-form>
      </BentoCard>

      <BentoCard title="默认活动摘要" :icon="Flag" size="1x1">
        <div v-if="totalDefaultActions === 0" class="empty-summary">
          <div class="empty-title">当前还没有默认活动</div>
          <div class="empty-desc">状态迁移自动化会直接复用手动报名标签里保存的默认活动。</div>
          <el-button type="primary" link @click="emit('configure-actions')">去配置默认活动</el-button>
        </div>
        <div v-else class="summary-groups">
          <div class="summary-group">
            <div class="summary-title">官方活动</div>
            <div class="summary-tags">
              <el-tag v-for="item in selectedOfficialActions" :key="`official-${item.id}`" effect="plain">
                {{ item.label }}
              </el-tag>
              <span v-if="selectedOfficialActions.length === 0" class="summary-empty">-</span>
            </div>
          </div>

          <div class="summary-group">
            <div class="summary-title">店铺活动</div>
            <div class="summary-tags">
              <el-tag v-for="item in selectedShopActions" :key="`shop-${item.id}`" effect="plain" type="warning">
                {{ item.label }}
              </el-tag>
              <span v-if="selectedShopActions.length === 0" class="summary-empty">-</span>
            </div>
          </div>

          <el-button type="primary" link @click="emit('configure-actions')">返回手动报名标签调整活动</el-button>
        </div>
      </BentoCard>
    </div>

    <BentoCard title="规则状态概览" :icon="List" size="4x1">
      <div class="rule-stats-grid">
        <div v-for="item in ruleStats" :key="item.key" class="rule-stat-card">
          <div class="rule-stat-label">{{ item.label }}</div>
          <div class="rule-stat-value">{{ item.value }}</div>
        </div>
      </div>
    </BentoCard>

    <BentoCard title="状态迁移历史" :icon="Clock" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="automationRuns" v-loading="automationRunsLoading">
          <el-table-column prop="id" label="任务ID" width="90" />
          <el-table-column label="触发方式" width="110">
            <template #default="{ row }">
              <el-tag size="small" :type="row.trigger_mode === 'scheduled' ? 'warning' : 'primary'">
                {{ triggerModeLabel(row.trigger_mode) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态统计" min-width="280">
            <template #default="{ row }">
              <div class="meta">抓取 {{ row.total_fetched }} / State1 {{ row.total_state1 }} / State2 迁移 {{ row.total_state2 }} / State3 兜底 {{ row.total_state3_trigger }}</div>
              <div class="meta">处理 {{ row.total_processed }} / 成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column prop="error_message" label="错误摘要" min-width="250">
            <template #default="{ row }">
              <span class="error-text">{{ row.error_message || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button text type="primary" @click="emit('open-automation-detail', row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </BentoCard>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { BentoCard } from '@/components/bento'
import { Clock, Flag, List } from '@element-plus/icons-vue'
import { statusLabel, statusTagType, triggerModeLabel } from '@/views/promotions/search-cpo/ui'

const props = defineProps({
  actions: {
    type: Array,
    default: () => []
  },
  savingConfig: {
    type: Boolean,
    default: false
  },
  automationRunning: {
    type: Boolean,
    default: false
  },
  automationRunsLoading: {
    type: Boolean,
    default: false
  },
  config: {
    type: Object,
    default: () => ({ official_action_ids: [], shop_action_ids: [], auto_enabled: false, schedule_time: '09:05' })
  },
  products: {
    type: Array,
    default: () => []
  },
  automationRuns: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['save-automation', 'start-run', 'configure-actions', 'open-automation-detail'])

const totalDefaultActions = computed(() => {
  return (props.config?.official_action_ids?.length || 0) + (props.config?.shop_action_ids?.length || 0)
})

const latestRunText = computed(() => {
  const latest = props.automationRuns[0]
  if (!latest) return '暂无记录'
  return `${triggerModeLabel(latest.trigger_mode)} / ${statusLabel(latest.status)}`
})

const selectedOfficialActions = computed(() => {
  return buildSelectedActions(props.config?.official_action_ids, 'official', 'action_id')
})

const selectedShopActions = computed(() => {
  return buildSelectedActions(props.config?.shop_action_ids, 'shop', 'source_action_id')
})

const ruleStats = computed(() => {
  const summary = {
    state1: 0,
    state2: 0,
    state3_trigger: 0,
    morkovsk_joined: 0,
    other: 0
  }

  props.products.forEach(item => {
    const state = item.rule_state || 'other'
    if (Object.prototype.hasOwnProperty.call(summary, state)) {
      summary[state] += 1
      return
    }
    summary.other += 1
  })

  return [
    { key: 'state1', label: 'State 1', value: summary.state1 },
    { key: 'state2', label: 'State 2', value: summary.state2 },
    { key: 'state3_trigger', label: 'State 3', value: summary.state3_trigger },
    { key: 'morkovsk_joined', label: '已加入 Morkovsk', value: summary.morkovsk_joined },
    { key: 'other', label: '其它 / 未识别', value: summary.other }
  ]
})

function buildSelectedActions(ids = [], source, idField) {
  return (Array.isArray(ids) ? ids : []).map(id => {
    const action = props.actions.find(item => item.id === id && item.source === source)
    if (!action) {
      return {
        id,
        label: `活动 #${id}`
      }
    }
    const sourceID = action[idField]
    const label = action.display_name || action.title || (sourceID ? `活动 #${sourceID}` : `活动 #${id}`)
    return {
      id,
      label
    }
  })
}
</script>

<style scoped>
.section-shell {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-bar {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border-radius: 16px;
  border: 1px solid rgba(5, 150, 105, 0.14);
  background: rgba(255, 255, 255, 0.72);
}

.section-copy {
  flex: 1;
  min-width: 260px;
}

.section-title {
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 18px;
  color: #064e3b;
}

.section-tip {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.55;
  color: #33695c;
}

.section-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.section-chip {
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(236, 253, 245, 0.95);
  border: 1px solid rgba(16, 185, 129, 0.14);
  font-size: 12px;
  color: #0f766e;
}

.section-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: flex-end;
  gap: 10px;
  max-width: 320px;
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.automation-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.inline-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.inline-tip,
.form-tip {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.empty-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--text-secondary);
}

.empty-title {
  font-weight: 600;
  color: var(--text-primary);
}

.empty-desc,
.summary-empty {
  font-size: 12px;
  color: var(--text-muted);
}

.summary-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.summary-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-title {
  font-weight: 600;
  color: var(--text-primary);
}

.summary-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.rule-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.rule-stat-card {
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(5, 150, 105, 0.12);
  background: rgba(255, 255, 255, 0.86);
}

.rule-stat-label {
  font-size: 12px;
  color: #0f766e;
}

.rule-stat-value {
  margin-top: 6px;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 20px;
  color: #064e3b;
}

.table-shell {
  overflow-x: auto;
}

.meta {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
}

.error-text {
  color: #d14343;
}

@media (max-width: 1180px) {
  .section-bar {
    flex-direction: column;
  }

  .section-actions {
    max-width: none;
    justify-content: flex-start;
  }
}

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }
}
</style>
