<template>
  <div class="section-shell">
    <div class="section-bar">
      <div class="section-copy">
        <div class="section-title">商品池与手动报名</div>
        <div class="section-tip">
          先刷新搜索推广商品，再保存默认活动；真正执行时只处理当前页面筛选出的商品。
        </div>
        <div class="section-meta">
          <span class="section-chip">最近同步：{{ lastSynced || '未同步' }}</span>
          <span class="section-chip">官方活动 {{ config.official_action_ids.length }}</span>
          <span class="section-chip">店铺活动 {{ config.shop_action_ids.length }}</span>
        </div>
      </div>

      <div class="section-actions">
        <el-button :loading="refreshing" @click="emit('refresh-products')">刷新搜索推广商品</el-button>
        <el-button :loading="savingConfig" @click="emit('save-config')">保存默认活动</el-button>
        <el-button
          type="primary"
          :loading="running"
          :disabled="filteredItems.length === 0 || totalDefaultActions === 0"
          @click="emit('run-manual')"
        >
          报名当前筛选结果
        </el-button>
      </div>
    </div>

    <div class="bento-grid--2col actions-grid">
      <BentoCard title="官方活动" :icon="Flag" size="1x1">
        <div class="action-panel" v-loading="actionsLoading">
          <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
          <el-checkbox-group v-else v-model="config.official_action_ids" class="action-group">
            <el-checkbox
              v-for="action in officialActions"
              :key="action.id"
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
      </BentoCard>

      <BentoCard title="店铺活动" :icon="Discount" size="1x1">
        <div class="action-panel" v-loading="actionsLoading">
          <el-empty v-if="!actionsLoading && shopActions.length === 0" description="暂无店铺活动" />
          <el-checkbox-group v-else v-model="config.shop_action_ids" class="action-group">
            <el-checkbox
              v-for="action in shopActions"
              :key="action.id"
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
      </BentoCard>
    </div>

    <BentoCard title="本地筛选" :icon="Filter" size="4x1">
      <el-form :inline="true" class="filter-form">
        <el-form-item label="关键词">
          <el-input
            v-model="filters.keyword"
            placeholder="标题 / source_sku / sku / 类目"
            clearable
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item label="推广状态">
          <el-select v-model="filters.promoStatus" style="width: 210px">
            <el-option label="全部" value="all" />
            <el-option label="已开启" value="SEARCH_PROMO_STATUS_ENABLED" />
            <el-option label="已关闭" value="SEARCH_PROMO_STATUS_DISABLED" />
          </el-select>
        </el-form-item>
        <el-form-item label="规则状态">
          <el-select v-model="filters.ruleState" style="width: 180px">
            <el-option label="全部" value="all" />
            <el-option label="State 1" value="state1" />
            <el-option label="State 2（迁移触发）" value="state2" />
            <el-option label="State 3 兜底" value="state3_trigger" />
            <el-option label="已加入 Morkovsk" value="morkovsk_joined" />
            <el-option label="其它 / 未识别" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="库存">
          <el-select v-model="filters.stockStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="有库存" value="in_stock" />
            <el-option label="无库存" value="out_of_stock" />
          </el-select>
        </el-form-item>
        <el-form-item label="收藏">
          <el-select v-model="filters.favoriteStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="已收藏" value="favorite" />
            <el-option label="未收藏" value="not_favorite" />
          </el-select>
        </el-form-item>
      </el-form>
    </BentoCard>

    <BentoCard title="商品列表" :icon="Goods" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="pagedItems" v-loading="productsLoading">
          <el-table-column label="图片" width="88" align="center">
            <template #default="{ row }">
              <el-image
                v-if="row.image_url"
                :src="row.image_url"
                class="thumb"
                fit="cover"
                :preview-src-list="[row.image_url]"
                preview-teleported
              />
              <span v-else class="no-data">-</span>
            </template>
          </el-table-column>

          <el-table-column label="商品信息" min-width="320">
            <template #default="{ row }">
              <div class="name">{{ row.title || '-' }}</div>
              <div class="meta">source_sku: {{ row.source_sku || '-' }}</div>
              <div class="meta">sku: {{ row.sku || '-' }}</div>
              <div class="meta">类目: {{ row.category_name || '-' }}</div>
            </template>
          </el-table-column>

          <el-table-column label="价格 / 库存" min-width="160" align="center">
            <template #default="{ row }">
              <div class="meta">价格: {{ formatMoney(row.price) }}</div>
              <div class="meta">库存总量: {{ row.stock_total ?? 0 }}</div>
              <el-tag size="small" :type="row.is_in_stock ? 'success' : 'warning'">
                {{ row.is_in_stock ? '有库存' : '无库存' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="搜索推广状态" width="190" align="center">
            <template #default="{ row }">
              <div class="status-stack">
                <el-tag :type="searchPromoStatusTagType(row.search_promo_status)">
                  {{ searchPromoStatusLabel(row.search_promo_status) }}
                </el-tag>
                <el-tag size="small" :type="carrotsStatusTagType(row.carrots_status)">
                  {{ carrotsStatusLabel(row.carrots_status) }}
                </el-tag>
                <el-tag size="small" :type="availabilityTagType(row.availability_promo)">
                  {{ availabilityLabel(row.availability_promo) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="规则状态" min-width="190" align="center">
            <template #default="{ row }">
              <div class="meta-block">
                <el-tag :type="ruleStateTagType(row.rule_state)">{{ ruleStateLabel(row.rule_state) }}</el-tag>
                <div class="meta" v-if="row.availability_checked_at">检测: {{ row.availability_checked_at }}</div>
                <div class="meta" v-if="row.state2_detected_at">State2: {{ row.state2_detected_at }}</div>
                <div class="meta" v-if="row.morkovsk_joined_at">Morkovsk: {{ row.morkovsk_joined_at }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="核心指标" min-width="180" align="center">
            <template #default="{ row }">
              <div class="meta">订单: {{ row.orders ?? 0 }}</div>
              <div class="meta">点击: {{ row.clicks ?? 0 }}</div>
              <div class="meta">花费: {{ formatMoney(row.spent) }}</div>
              <div class="meta">CTR: {{ formatPercent(row.ctr_percent) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <div class="pager-line">
          <span>共 {{ filteredItems.length }} 条，当前页 {{ pager.page }}</span>
          <el-pagination
            v-model:current-page="pager.page"
            v-model:page-size="pager.pageSize"
            :total="filteredItems.length"
            layout="prev, pager, next, sizes"
            :page-sizes="[20, 50, 100]"
            small
          />
        </div>
      </template>
    </BentoCard>

    <BentoCard title="手动报名历史" :icon="List" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="runs" v-loading="runsLoading">
          <el-table-column prop="id" label="任务ID" width="90" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="统计" min-width="260">
            <template #default="{ row }">
              <div class="meta">缓存 {{ row.total_fetched }} / 筛选 {{ row.total_selected }} / 处理 {{ row.total_processed }}</div>
              <div class="meta">成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
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
              <el-button text type="primary" @click="emit('open-run-detail', row)">详情</el-button>
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
import { Discount, Filter, Flag, Goods, List } from '@element-plus/icons-vue'
import {
  availabilityLabel,
  availabilityTagType,
  carrotsStatusLabel,
  carrotsStatusTagType,
  formatMoney,
  formatPercent,
  ruleStateLabel,
  ruleStateTagType,
  searchPromoStatusLabel,
  searchPromoStatusTagType,
  statusLabel,
  statusTagType
} from '@/views/promotions/search-cpo/ui'

const props = defineProps({
  actions: {
    type: Array,
    default: () => []
  },
  actionsLoading: {
    type: Boolean,
    default: false
  },
  productsLoading: {
    type: Boolean,
    default: false
  },
  runsLoading: {
    type: Boolean,
    default: false
  },
  savingConfig: {
    type: Boolean,
    default: false
  },
  refreshing: {
    type: Boolean,
    default: false
  },
  running: {
    type: Boolean,
    default: false
  },
  config: {
    type: Object,
    default: () => ({ official_action_ids: [], shop_action_ids: [] })
  },
  filters: {
    type: Object,
    default: () => ({})
  },
  pager: {
    type: Object,
    default: () => ({ page: 1, pageSize: 20 })
  },
  filteredItems: {
    type: Array,
    default: () => []
  },
  pagedItems: {
    type: Array,
    default: () => []
  },
  runs: {
    type: Array,
    default: () => []
  },
  lastSynced: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['save-config', 'refresh-products', 'run-manual', 'open-run-detail'])

const officialActions = computed(() => props.actions.filter(action => action.source === 'official'))
const shopActions = computed(() => props.actions.filter(action => action.source === 'shop'))
const totalDefaultActions = computed(() => {
  return (props.config?.official_action_ids?.length || 0) + (props.config?.shop_action_ids?.length || 0)
})
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
  max-width: 380px;
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.actions-grid {
  margin-top: 2px;
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

.filter-form {
  row-gap: 8px;
}

.table-shell {
  overflow-x: auto;
}

.thumb {
  width: 56px;
  height: 56px;
  border-radius: 6px;
}

.name {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.meta {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
}

.meta-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.status-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.no-data {
  color: var(--text-muted);
}

.pager-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
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

@media (max-width: 640px) {
  .pager-line {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
