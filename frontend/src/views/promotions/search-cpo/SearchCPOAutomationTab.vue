<template>
  <div class="section-shell">
    <BentoCard title="自动执行设置" :icon="Clock" size="4x1">
      <el-form label-width="92px" class="automation-form">
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
        <el-form-item label="最近任务">
          <span class="inline-tip">{{ latestRunText }}</span>
        </el-form-item>
        <el-form-item>
          <div class="section-actions">
            <el-button :loading="savingConfig" @click="emit('save-automation')">保存</el-button>
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
        </el-form-item>
      </el-form>
    </BentoCard>

    <BentoCard title="默认活动" :icon="Flag" size="4x1">
      <div class="summary-note">推广已关闭的商品需要添加的促销活动</div>
      <div class="bento-grid--2col actions-grid">
        <div class="action-column">
          <div class="action-column-title">官方活动</div>
          <div class="action-panel" v-loading="actionsLoading">
            <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
            <el-checkbox-group v-else v-model="config.official_action_ids" class="action-group">
              <el-checkbox
                v-for="action in officialActions"
                :key="`default-official-${action.id}`"
                :value="action.id"
                class="action-checkbox"
              >
                <div class="action-item">
                  <div class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</div>
                  <div class="action-meta">官方 ID: {{ action.action_id }}</div>
                </div>
              </el-checkbox>
            </el-checkbox-group>
          </div>
        </div>

        <div class="action-column">
          <div class="action-column-title">店铺活动</div>
          <div class="action-panel" v-loading="actionsLoading">
            <el-empty v-if="!actionsLoading && shopActions.length === 0" description="暂无店铺活动" />
            <el-checkbox-group v-else v-model="config.shop_action_ids" class="action-group">
              <el-checkbox
                v-for="action in shopActions"
                :key="`default-shop-${action.id}`"
                :value="action.id"
                class="action-checkbox"
              >
                <div class="action-item">
                  <div class="action-title">{{ action.display_name || action.title || `活动 #${action.source_action_id}` }}</div>
                  <div class="action-meta">店铺 ID: {{ action.source_action_id }}</div>
                </div>
              </el-checkbox>
            </el-checkbox-group>
          </div>
        </div>
      </div>
    </BentoCard>

    <BentoCard title="退出活动" :icon="Flag" size="4x1">
      <div class="summary-note">所有可加入推广、已加入推广的商品需要退出的促销活动</div>
      <div v-if="totalExitActions === 0" class="empty-tip">
        当前未配置退出活动，将不会执行退出步骤。
      </div>
      <div class="bento-grid--2col actions-grid">
        <div class="action-column">
          <div class="action-column-title">官方活动</div>
          <div class="action-panel" v-loading="actionsLoading">
            <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
            <el-checkbox-group v-else v-model="config.exit_official_action_ids" class="action-group">
              <el-checkbox
                v-for="action in officialActions"
                :key="`exit-official-${action.id}`"
                :value="action.id"
                class="action-checkbox"
              >
                <div class="action-item">
                  <div class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</div>
                  <div class="action-meta">官方 ID: {{ action.action_id }}</div>
                </div>
              </el-checkbox>
            </el-checkbox-group>
          </div>
        </div>

        <div class="action-column">
          <div class="action-column-title">店铺活动</div>
          <div class="action-panel" v-loading="actionsLoading">
            <el-empty v-if="!actionsLoading && shopActions.length === 0" description="暂无店铺活动" />
            <el-checkbox-group v-else v-model="config.exit_shop_action_ids" class="action-group">
              <el-checkbox
                v-for="action in shopActions"
                :key="`exit-shop-${action.id}`"
                :value="action.id"
                class="action-checkbox"
              >
                <div class="action-item">
                  <div class="action-title">{{ action.display_name || action.title || `活动 #${action.source_action_id}` }}</div>
                  <div class="action-meta">店铺 ID: {{ action.source_action_id }}</div>
                </div>
              </el-checkbox>
            </el-checkbox-group>
          </div>
        </div>
      </div>
    </BentoCard>

    <BentoCard title="状态概览" :icon="List" size="4x1">
      <div class="rule-stats-grid">
        <div v-for="item in ruleStats" :key="item.key" class="rule-stat-card">
          <div class="rule-stat-label">{{ item.label }}</div>
          <div class="rule-stat-value">{{ item.value }}</div>
        </div>
      </div>
    </BentoCard>

    <BentoCard title="执行历史" :icon="Clock" size="4x1" no-padding>
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
              <div class="meta">抓取 {{ row.total_fetched || 0 }} / 状态1 {{ row.total_state1 || 0 }} / 状态2 {{ row.total_state2 || 0 }} / 状态3 {{ row.total_state3 || 0 }} / 状态4 {{ row.total_state4 || 0 }}</div>
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
  actionsLoading: {
    type: Boolean,
    default: false
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
    default: () => ({
      official_action_ids: [],
      shop_action_ids: [],
      exit_official_action_ids: [],
      exit_shop_action_ids: [],
      auto_enabled: false,
      schedule_time: '09:05'
    })
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

const emit = defineEmits(['save-automation', 'start-run', 'open-automation-detail'])

const totalDefaultActions = computed(() => {
  return (props.config?.official_action_ids?.length || 0) + (props.config?.shop_action_ids?.length || 0)
})
const totalExitActions = computed(() => {
  return (props.config?.exit_official_action_ids?.length || 0) + (props.config?.exit_shop_action_ids?.length || 0)
})

const latestRunText = computed(() => {
  const latest = props.automationRuns[0]
  if (!latest) return '暂无记录'
  return `${triggerModeLabel(latest.trigger_mode)} / ${statusLabel(latest.status)}`
})

const officialActions = computed(() => props.actions.filter(action => action.source === 'official'))
const shopActions = computed(() => props.actions.filter(action => action.source === 'shop'))

const ruleStats = computed(() => {
  const summary = {
    state1: 0,
    state2: 0,
    state3: 0,
    state4: 0,
    other: 0
  }

  props.products.forEach(item => {
    const state = normalizeRuleState(item.rule_state)
    if (Object.prototype.hasOwnProperty.call(summary, state)) {
      summary[state] += 1
      return
    }
    summary.other += 1
  })

  return [
    { key: 'state1', label: '状态1：推广已关闭', value: summary.state1 },
    { key: 'state2', label: '状态2：可加入推广', value: summary.state2 },
    { key: 'state3', label: '状态3：已加入推广，未加入 Morkovsk', value: summary.state3 },
    { key: 'state4', label: '状态4：已加入推广，已加入 Morkovsk', value: summary.state4 },
    { key: 'other', label: '其它', value: summary.other }
  ]
})

function normalizeRuleState(state) {
  if (state === 'state3_trigger') return 'state3'
  if (state === 'morkovsk_joined') return 'state4'
  if (['state1', 'state2', 'state3', 'state4'].includes(state)) return state
  return 'other'
}
</script>

<style scoped>
.section-shell {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
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

.inline-tip {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.summary-note {
  margin-bottom: 12px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.empty-tip {
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgba(249, 115, 22, 0.2);
  background: rgba(255, 247, 237, 0.7);
  color: #9a3412;
  font-size: 12px;
  line-height: 1.5;
}

.actions-grid {
  margin-top: 2px;
}

.action-column {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-column-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-panel {
  min-height: 72px;
}

.action-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-checkbox {
  width: 100%;
  margin-right: 0;
}

.action-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.action-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-meta {
  color: var(--text-muted);
  font-size: 12px;
}

.rule-stats-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.rule-stat-card {
  flex: 1 1 180px;
  min-width: 180px;
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

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }
}
</style>
