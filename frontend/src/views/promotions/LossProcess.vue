<template>
  <div class="loss-process">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">亏损商品处理</h2>
    </div>

    <!-- Bento Grid 布局 -->
    <div class="bento-grid--2col">
      <!-- 上传区域卡片 -->
      <BentoCard title="导入亏损商品" :icon="UploadFilled" size="1x1">
        <template #actions>
          <el-button text size="small" @click="downloadTemplate">
            <el-icon><Download /></el-icon>
            下载模板
          </el-button>
        </template>

        <el-upload
          ref="uploadRef"
          class="upload-area"
          drag
          :auto-upload="false"
          :limit="1"
          accept=".xlsx,.xls"
          :on-change="handleFileChange"
          :on-exceed="handleExceed"
        >
          <div class="upload-content">
            <div class="upload-icon">
              <el-icon><UploadFilled /></el-icon>
            </div>
            <div class="upload-text">
              <p>拖拽 Excel 文件到此处</p>
              <p class="sub">或 <em>点击上传</em></p>
            </div>
          </div>
        </el-upload>
        <div class="upload-tip">
          支持 .xlsx / .xls 格式，需包含 source_sku、new_price 列
        </div>
      </BentoCard>

      <!-- 重新报名活动选择 -->
      <BentoCard title="重新报名活动" :icon="Ticket" size="1x1">
        <template #actions>
          <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </template>

        <div class="rejoin-info">
          <el-icon><InfoFilled /></el-icon>
          <span>处理流程：退出促销 → 改价 → 重新报名</span>
        </div>

        <el-select
          v-model="rejoinActionId"
          placeholder="选择重新报名的活动（可选）"
          clearable
          style="width: 100%"
          :loading="actionsLoading"
        >
          <el-option
            v-for="action in actions"
            :key="action.action_id"
            :label="`${action.display_name || action.title || '活动 #' + action.action_id}`"
            :value="action.action_id"
          />
        </el-select>
        <div class="rejoin-hint">留空则不重新报名，仅执行退出促销和改价</div>
      </BentoCard>
    </div>

    <!-- 预览数据 -->
    <BentoCard
      v-if="previewData.length > 0"
      title="预览数据"
      :icon="Document"
      size="4x1"
      no-padding
      class="preview-card"
    >
      <template #actions>
        <el-tag type="info" effect="plain" size="small">共 {{ previewData.length }} 条</el-tag>
      </template>
      <el-table :data="previewData.slice(0, 10)" size="small">
        <el-table-column prop="source_sku" label="SKU">
          <template #default="{ row }">
            <span class="sku-text">{{ row.source_sku }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="new_price" label="新价格" align="right">
          <template #default="{ row }">
            <span class="price-text">¥{{ row.new_price }}</span>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <div class="preview-footer">
          <span v-if="previewData.length > 10" class="preview-hint">
            仅显示前 10 条，共 {{ previewData.length }} 条数据
          </span>
          <el-button
            type="primary"
            :loading="importing"
            :disabled="!file"
            @click="handleImport"
          >
            <el-icon v-if="!importing"><Upload /></el-icon>
            {{ importing ? '处理中...' : '导入并处理' }}
          </el-button>
        </div>
      </template>
    </BentoCard>

    <!-- 处理结果 -->
    <div v-if="result" class="result-section">
      <div class="result-header">
        <h3>处理结果</h3>
        <el-tag :type="result.success ? 'success' : 'danger'" effect="dark">
          {{ result.success ? '成功' : '失败' }}
        </el-tag>
      </div>

      <!-- 处理步骤 -->
      <div class="process-steps">
        <div class="step-item">
          <div class="step-icon success">
            <el-icon><CircleCheckFilled /></el-icon>
          </div>
          <div class="step-info">
            <span class="step-title">退出促销</span>
            <span class="step-count">{{ result.steps?.exit_promotion?.success || 0 }} 成功</span>
          </div>
        </div>
        <div class="step-line"></div>
        <div class="step-item">
          <div class="step-icon success">
            <el-icon><CircleCheckFilled /></el-icon>
          </div>
          <div class="step-info">
            <span class="step-title">更新价格</span>
            <span class="step-count">{{ result.steps?.price_update?.success || 0 }} 成功</span>
          </div>
        </div>
        <div class="step-line"></div>
        <div class="step-item">
          <div class="step-icon success">
            <el-icon><CircleCheckFilled /></el-icon>
          </div>
          <div class="step-info">
            <span class="step-title">重新报名</span>
            <span class="step-count">{{ result.steps?.rejoin_promotions?.success || 0 }} 成功</span>
          </div>
        </div>
      </div>

      <div class="result-summary">
        共处理 {{ result.processed_count || 0 }} 个商品
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  UploadFilled, Upload, Download, CircleCheckFilled, Ticket,
  Refresh, InfoFilled, Document
} from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { importLoss, processLossV2, getActions } from '@/api/promotion'
import { BentoCard } from '@/components/bento'
import * as XLSX from 'xlsx'

const userStore = useUserStore()

const uploadRef = ref(null)
const file = ref(null)
const importing = ref(false)
const previewData = ref([])
const result = ref(null)
const actionsLoading = ref(false)
const actions = ref([])
const rejoinActionId = ref(null)

async function fetchActions() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  actionsLoading.value = true
  try {
    const res = await getActions(shopId)
    actions.value = res.data || []
  } catch (error) {
    console.error(error)
  } finally {
    actionsLoading.value = false
  }
}

function handleFileChange(uploadFile) {
  file.value = uploadFile.raw
  parseExcel(uploadFile.raw)
}

function handleExceed() {
  ElMessage.warning('只能上传一个文件')
}

async function parseExcel(file) {
  try {
    const data = await file.arrayBuffer()
    const workbook = XLSX.read(data, { type: 'array' })
    const sheetName = workbook.SheetNames[0]
    const sheet = workbook.Sheets[sheetName]
    const json = XLSX.utils.sheet_to_json(sheet)

    previewData.value = json.map(row => ({
      source_sku: row.source_sku || row.sku || row.SKU || '',
      new_price: parseFloat(row.new_price || row.price || row.新价格 || 0)
    })).filter(row => row.source_sku && row.new_price > 0)

    if (previewData.value.length === 0) {
      ElMessage.warning('未找到有效数据，请检查文件格式')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('解析文件失败')
  }
}

async function handleImport() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  if (!file.value) {
    ElMessage.warning('请先选择文件')
    return
  }

  importing.value = true
  result.value = null

  try {
    const formData = new FormData()
    formData.append('file', file.value)
    formData.append('shop_id', shopId)

    const importRes = await importLoss(formData)
    const lossProductIds = importRes.data.loss_product_ids || []

    if (lossProductIds.length === 0) {
      ElMessage.warning('没有找到可处理的亏损商品')
      importing.value = false
      return
    }

    const processRes = await processLossV2({
      shop_id: shopId,
      loss_product_ids: lossProductIds,
      rejoin_action_id: rejoinActionId.value
    })

    result.value = processRes.data
    ElMessage.success('处理完成')
  } catch (error) {
    console.error(error)
    ElMessage.error('处理失败')
  } finally {
    importing.value = false
  }
}

function downloadTemplate() {
  const template = [
    { source_sku: 'SKU001', new_price: 1500 },
    { source_sku: 'SKU002', new_price: 2300 }
  ]
  const ws = XLSX.utils.json_to_sheet(template)
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, '亏损商品')
  XLSX.writeFile(wb, '亏损商品模板.xlsx')
}

watch(() => userStore.currentShopId, () => {
  rejoinActionId.value = null
  fetchActions()
})

onMounted(() => {
  fetchActions()
})
</script>

<style scoped>
.loss-process {
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

.upload-area {
  width: 100%;
}

.upload-area :deep(.el-upload-dragger) {
  background: var(--bg-tertiary);
  border: 2px dashed var(--surface-border);
  border-radius: var(--radius-lg);
  padding: 30px;
  transition: all var(--transition-normal);
}

.upload-area :deep(.el-upload-dragger:hover) {
  border-color: var(--primary);
  background: var(--surface-bg-hover);
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.upload-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.upload-icon .el-icon {
  font-size: 24px;
  color: white;
}

.upload-text {
  text-align: center;
}

.upload-text p {
  color: var(--text-primary);
  font-size: 14px;
  margin: 0;
}

.upload-text p.sub {
  color: var(--text-muted);
  font-size: 12px;
  margin-top: 4px;
}

.upload-text em {
  color: var(--primary);
  font-style: normal;
  cursor: pointer;
}

.upload-tip {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
}

.rejoin-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: rgba(196, 113, 78, 0.1);
  border-radius: var(--radius-md);
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--primary);
}

.rejoin-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}

.preview-card {
  margin-bottom: 24px;
}

.preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.preview-hint {
  font-size: 12px;
  color: var(--text-muted);
}

.sku-text {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  color: var(--accent);
}

.price-text {
  font-weight: 600;
  color: var(--warning);
}

.result-section {
  background: var(--bg-secondary);
  border: 1px solid var(--surface-border);
  border-radius: var(--radius-lg);
  padding: 24px;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.result-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.process-steps {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  margin-bottom: 24px;
}

.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.step-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border: 2px solid var(--surface-border);
}

.step-icon .el-icon {
  font-size: 24px;
  color: var(--text-muted);
}

.step-icon.success {
  background: rgba(74, 150, 104, 0.15);
  border-color: var(--success);
}

.step-icon.success .el-icon {
  color: var(--success);
}

.step-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.step-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.step-count {
  font-size: 12px;
  color: var(--text-muted);
}

.step-line {
  width: 60px;
  height: 2px;
  background: var(--success);
  margin: 0 16px;
  margin-bottom: 50px;
}

.result-summary {
  text-align: center;
  font-size: 14px;
  color: var(--text-secondary);
  padding-top: 16px;
  border-top: 1px solid var(--surface-border);
}
</style>
