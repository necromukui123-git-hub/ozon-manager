<template>
  <el-dialog
    :model-value="visible"
    title="状态迁移详情"
    width="1320px"
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-if="detail" class="detail-summary detail-summary--wrap">
      <el-tag :type="statusTagType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
      <span>触发方式 {{ triggerModeLabel(detail.trigger_mode) }}</span>
      <span>状态1 {{ detail.total_state1 || 0 }} / 状态2 {{ detail.total_state2 || 0 }} / 状态3 {{ detail.total_state3 || 0 }} / 状态4 {{ detail.total_state4 || 0 }}</span>
      <span>成功 {{ detail.success_items }} / 失败 {{ detail.failed_items }} / 跳过 {{ detail.skipped_items }}</span>
    </div>

    <div v-if="detail" class="snapshot-panel">
      <div class="snapshot-title">当次配置快照</div>
      <div class="snapshot-grid">
        <div class="snapshot-group">
          <div class="snapshot-label">默认活动（官方）</div>
          <div class="snapshot-values">
            <el-tag v-for="id in normalizeIDs(detail.official_action_ids)" :key="`default-official-${id}`" size="small" effect="plain">
              {{ resolveActionLabel(id, 'official') }}
            </el-tag>
            <span v-if="normalizeIDs(detail.official_action_ids).length === 0" class="result-line-muted">-</span>
          </div>
        </div>
        <div class="snapshot-group">
          <div class="snapshot-label">默认活动（店铺）</div>
          <div class="snapshot-values">
            <el-tag
              v-for="id in normalizeIDs(detail.shop_action_ids)"
              :key="`default-shop-${id}`"
              size="small"
              effect="plain"
              type="warning"
            >
              {{ resolveActionLabel(id, 'shop') }}
            </el-tag>
            <span v-if="normalizeIDs(detail.shop_action_ids).length === 0" class="result-line-muted">-</span>
          </div>
        </div>
        <div class="snapshot-group">
          <div class="snapshot-label">退出活动（官方）</div>
          <div class="snapshot-values">
            <el-tag v-for="id in normalizeIDs(detail.exit_official_action_ids)" :key="`exit-official-${id}`" size="small" effect="plain">
              {{ resolveActionLabel(id, 'official') }}
            </el-tag>
            <span v-if="normalizeIDs(detail.exit_official_action_ids).length === 0" class="result-line-muted">-</span>
          </div>
        </div>
        <div class="snapshot-group">
          <div class="snapshot-label">退出活动（店铺）</div>
          <div class="snapshot-values">
            <el-tag
              v-for="id in normalizeIDs(detail.exit_shop_action_ids)"
              :key="`exit-shop-${id}`"
              size="small"
              effect="plain"
              type="warning"
            >
              {{ resolveActionLabel(id, 'shop') }}
            </el-tag>
            <span v-if="normalizeIDs(detail.exit_shop_action_ids).length === 0" class="result-line-muted">-</span>
          </div>
        </div>
        <div class="snapshot-group">
          <div class="snapshot-label">触发时间</div>
          <div class="snapshot-values">
            <span>{{ detail.schedule_time || '-' }}</span>
          </div>
        </div>
      </div>
    </div>

    <el-table v-if="detail" :data="detail.items || []" max-height="560" v-loading="loading">
      <el-table-column type="expand" width="48">
        <template #default="{ row }">
          <div class="availability-debug-panel">
            <div class="availability-debug-title">可推广状态诊断</div>
            <div class="availability-debug-grid">
              <div>
                <span class="availability-debug-label">可推广状态</span>
                <el-tag size="small" :type="availabilityTagType(row.availability_promo)">
                  {{ availabilityLabel(row.availability_promo) }}
                </el-tag>
              </div>
              <div>
                <span class="availability-debug-label">检查时间</span>
                <span>{{ row.availability_checked_at || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">requested_sku</span>
                <span>{{ row.availability_diagnostics?.requested_sku || row.sku || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">parser_revision</span>
                <span>{{ row.availability_diagnostics?.parser_revision || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">build_revision</span>
                <span>{{ row.availability_diagnostics?.build_revision || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">脚本结果类型</span>
                <span>{{ row.availability_diagnostics?.script_result_type || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">响应类型</span>
                <span>{{ row.availability_diagnostics?.response_kind || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">HTTP</span>
                <span>{{ formatAvailabilityHTTP(row.availability_diagnostics) }}</span>
              </div>
              <div>
                <span class="availability-debug-label">Content-Type</span>
                <span>{{ row.availability_diagnostics?.response_content_type || '-' }}</span>
              </div>
              <div>
                <span class="availability-debug-label">availability keys</span>
                <span>{{ row.availability_diagnostics?.availability_map_key_count ?? 0 }}</span>
              </div>
              <div>
                <span class="availability-debug-label">reason keys</span>
                <span>{{ row.availability_diagnostics?.reason_map_key_count ?? 0 }}</span>
              </div>
              <div>
                <span class="availability-debug-label">业务原因</span>
                <span>{{ row.availability_diagnostics?.unavailable_reason || '-' }}</span>
              </div>
            </div>
            <div class="availability-debug-block">
              <div class="availability-debug-label">root keys</div>
              <div>{{ joinAvailabilityDiagnosticKeys(row.availability_diagnostics?.response_root_keys) }}</div>
            </div>
            <div class="availability-debug-block">
              <div class="availability-debug-label">sample keys</div>
              <div>{{ joinAvailabilityDiagnosticKeys(row.availability_diagnostics?.sample_response_keys) }}</div>
            </div>
            <div v-if="row.availability_diagnostics?.response_parse_error" class="availability-debug-block">
              <div class="availability-debug-label">解析错误</div>
              <div class="error-text">{{ row.availability_diagnostics.response_parse_error }}</div>
            </div>
            <div v-if="row.availability_diagnostics?.response_excerpt" class="availability-debug-block">
              <div class="availability-debug-label">响应摘要</div>
              <pre class="availability-debug-pre">{{ row.availability_diagnostics.response_excerpt }}</pre>
            </div>
            <div v-if="!row.availability_diagnostics && !row.availability_checked_at" class="result-line-muted">
              当前没有可展示的 availability 诊断。
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="source_sku" label="source_sku" width="160" />
      <el-table-column prop="sku" label="sku" width="130" />
      <el-table-column prop="title" label="商品" min-width="220" />
      <el-table-column label="规则" width="170">
        <template #default="{ row }">
          <div class="result-lines">
            <div>进入前: {{ ruleStateLabel(row.rule_state_before) }}</div>
            <div>执行后: {{ ruleStateLabel(row.rule_state_after) }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="总状态" width="110">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="默认活动" min-width="240">
        <template #default="{ row }">
          <div class="result-lines">
            <div class="result-line-muted">步骤状态: {{ statusLabel(row.initial_status) }}</div>
            <div v-for="item in row.initial_results" :key="`init-${row.id}-${item.promotion_action_id}`">
              {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
            </div>
            <div v-if="!row.initial_results || row.initial_results.length === 0">-</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="退出活动" min-width="240">
        <template #default="{ row }">
          <div class="result-lines">
            <div class="result-line-muted">步骤状态: {{ statusLabel(row.exit_status) }}</div>
            <div v-for="item in row.exit_results" :key="`exit-${row.id}-${item.promotion_action_id}`">
              {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
            </div>
            <div v-if="!row.exit_results || row.exit_results.length === 0">-</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Enable" min-width="220">
        <template #default="{ row }">
          <div class="result-lines">
            <div>{{ statusLabel(row.enable_status) }}</div>
            <div v-if="row.enable_result?.message">{{ row.enable_result.message }}</div>
            <div v-if="row.enable_result?.error" class="error-text">{{ row.enable_result.error }}</div>
            <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
              requested_sku: {{ row.enable_result.diagnostics.requested_sku || '-' }} / parser_revision: {{ row.enable_result.diagnostics.parser_revision || '-' }}
            </div>
            <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
              HTTP: {{ formatAvailabilityHTTP(row.enable_result.diagnostics) }} / Content-Type: {{ row.enable_result.diagnostics.response_content_type || '-' }}
            </div>
            <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
              root: {{ joinAvailabilityDiagnosticKeys(row.enable_result.diagnostics.response_root_keys) }} / sample: {{ joinAvailabilityDiagnosticKeys(row.enable_result.diagnostics.sample_response_keys) }}
            </div>
            <div v-if="row.enable_result?.diagnostics?.response_parse_error" class="error-text">解析错误: {{ row.enable_result.diagnostics.response_parse_error }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Morkovsk" min-width="220">
        <template #default="{ row }">
          <div class="result-lines">
            <div>{{ statusLabel(row.morkovsk_status) }}</div>
            <div v-if="row.morkovsk_result?.message">{{ row.morkovsk_result.message }}</div>
            <div v-if="row.morkovsk_result?.error" class="error-text">{{ row.morkovsk_result.error }}</div>
            <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
              requested_sku: {{ row.morkovsk_result.diagnostics.requested_sku || '-' }} / parser_revision: {{ row.morkovsk_result.diagnostics.parser_revision || '-' }}
            </div>
            <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
              HTTP: {{ formatAvailabilityHTTP(row.morkovsk_result.diagnostics) }} / Content-Type: {{ row.morkovsk_result.diagnostics.response_content_type || '-' }}
            </div>
            <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
              root: {{ joinAvailabilityDiagnosticKeys(row.morkovsk_result.diagnostics.response_root_keys) }} / sample: {{ joinAvailabilityDiagnosticKeys(row.morkovsk_result.diagnostics.sample_response_keys) }}
            </div>
            <div v-if="row.morkovsk_result?.diagnostics?.response_parse_error" class="error-text">解析错误: {{ row.morkovsk_result.diagnostics.response_parse_error }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="message" label="备注" min-width="180" />
    </el-table>
  </el-dialog>
</template>

<script setup>
import {
  actionResultLabel,
  availabilityLabel,
  availabilityTagType,
  formatAvailabilityHTTP,
  joinAvailabilityDiagnosticKeys,
  ruleStateLabel,
  statusLabel,
  statusTagType,
  triggerModeLabel
} from '@/views/promotions/search-cpo/ui'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  actions: {
    type: Array,
    default: () => []
  },
  detail: {
    type: Object,
    default: null
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:visible'])

function normalizeIDs(ids) {
  return Array.isArray(ids) ? ids : []
}

function resolveActionLabel(id, source) {
  const numericID = Number(id)
  const action = props.actions.find(item => item.id === numericID && item.source === source)
  if (!action) {
    return `活动 #${id}`
  }
  const sourceID = source === 'shop' ? action.source_action_id : action.action_id
  return action.display_name || action.title || (sourceID ? `活动 #${sourceID}` : `活动 #${id}`)
}
</script>

<style scoped>
.detail-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.detail-summary--wrap {
  flex-wrap: wrap;
}

.snapshot-panel {
  margin-bottom: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(5, 150, 105, 0.12);
  background: rgba(236, 253, 245, 0.45);
}

.snapshot-title {
  margin-bottom: 10px;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 14px;
  color: #065f46;
}

.snapshot-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
}

.snapshot-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.snapshot-label {
  font-size: 12px;
  color: #0f766e;
}

.snapshot-values {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 22px;
  font-size: 13px;
}

.availability-debug-panel {
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(5, 150, 105, 0.05);
  border: 1px solid rgba(5, 150, 105, 0.12);
}

.availability-debug-title {
  margin-bottom: 10px;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 14px;
  color: #065f46;
}

.availability-debug-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 18px;
  margin-bottom: 12px;
}

.availability-debug-grid > div,
.availability-debug-block {
  font-size: 13px;
  line-height: 1.6;
  color: #134e4a;
}

.availability-debug-label {
  display: inline-block;
  min-width: 112px;
  margin-right: 8px;
  color: #0f766e;
}

.availability-debug-block + .availability-debug-block {
  margin-top: 8px;
}

.availability-debug-pre {
  margin: 8px 0 0;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(15, 118, 110, 0.12);
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.5;
  color: #1f2937;
}

.result-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.result-line-muted {
  color: var(--text-muted);
}

.error-text {
  color: #d14343;
}

@media (max-width: 900px) {
  .snapshot-grid {
    grid-template-columns: 1fr;
  }

  .availability-debug-grid {
    grid-template-columns: 1fr;
  }
}
</style>
