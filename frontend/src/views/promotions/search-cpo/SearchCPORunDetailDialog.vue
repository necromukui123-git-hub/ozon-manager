<template>
  <el-dialog
    :model-value="visible"
    title="手动报名详情"
    width="1100px"
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-if="detail" class="detail-summary">
      <el-tag :type="statusTagType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
      <span>筛选商品 {{ detail.source_skus?.length || 0 }} 件</span>
      <span>成功 {{ detail.success_items }} / 失败 {{ detail.failed_items }} / 跳过 {{ detail.skipped_items }}</span>
    </div>

    <el-table v-if="detail" :data="detail.items || []" max-height="520" v-loading="loading">
      <el-table-column prop="source_sku" label="source_sku" width="160" />
      <el-table-column prop="sku" label="sku" width="130" />
      <el-table-column prop="title" label="商品" min-width="220" />
      <el-table-column label="总状态" width="110">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="官方结果" min-width="220">
        <template #default="{ row }">
          <div class="result-lines">
            <div v-for="item in row.official_results" :key="`official-${row.id}-${item.promotion_action_id}`">
              {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
            </div>
            <div v-if="!row.official_results || row.official_results.length === 0">-</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="店铺结果" min-width="240">
        <template #default="{ row }">
          <div class="result-lines">
            <div v-for="item in row.shop_results" :key="`shop-${row.id}-${item.promotion_action_id}`">
              {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
            </div>
            <div v-if="!row.shop_results || row.shop_results.length === 0">-</div>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup>
import { actionResultLabel, statusLabel, statusTagType } from '@/views/promotions/search-cpo/ui'

defineProps({
  visible: {
    type: Boolean,
    default: false
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
</script>

<style scoped>
.detail-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.result-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}
</style>
