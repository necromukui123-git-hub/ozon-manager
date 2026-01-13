<template>
  <div class="loss-process">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">亏损商品处理</h2>
    </div>

    <!-- 上传区域 -->
    <div class="glass-card">
      <div class="card-header">
        <span class="card-title">导入亏损商品</span>
        <el-button text size="small" @click="downloadTemplate">
          <el-icon><Download /></el-icon>
          下载模板
        </el-button>
      </div>
      <div class="card-body">
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
              <p>将 Excel 文件拖到此处</p>
              <p class="sub">或 <em>点击上传</em></p>
            </div>
          </div>
        </el-upload>
        <div class="upload-tip">
          支持 .xlsx / .xls 格式，文件需包含 source_sku、new_price 列
        </div>

        <!-- 重新报名活动选择 -->
        <div class="rejoin-section">
          <h4 class="section-title">重新报名活动（可选）</h4>
          <el-select
            v-model="rejoinActionId"
            placeholder="选择重新报名的活动（留空则不重新报名）"
            clearable
            style="width: 100%; max-width: 400px"
            :loading="actionsLoading"
          >
            <el-option
              v-for="action in actions"
              :key="action.action_id"
              :label="`${action.title || '活动 #' + action.action_id} (ID: ${action.action_id})`"
              :value="action.action_id"
            />
          </el-select>
          <div class="rejoin-hint">
            处理流程：退出促销 → 改价 → 重新报名选定活动
          </div>
        </div>

        <div class="action-buttons">
          <el-button
            type="primary"
            size="large"
            :loading="importing"
            :disabled="!file"
            @click="handleImport"
          >
            <el-icon v-if="!importing"><Upload /></el-icon>
            {{ importing ? '处理中...' : '导入并处理' }}
          </el-button>
        </div>
      </div>
    </div>

    <!-- 预览数据 -->
    <div v-if="previewData.length > 0" class="glass-card preview-card">
      <div class="card-header">
        <span class="card-title">预览数据</span>
        <el-tag type="info" effect="plain">共 {{ previewData.length }} 条</el-tag>
      </div>
      <div class="card-body">
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
        <div v-if="previewData.length > 10" class="preview-hint">
          仅显示前 10 条，共 {{ previewData.length }} 条数据
        </div>
      </div>
    </div>

    <!-- 处理结果 -->
    <div v-if="result" class="glass-card result-card">
      <div class="card-header">
        <span class="card-title">处理结果</span>
        <el-tag :type="result.success ? 'success' : 'danger'" effect="dark">
          {{ result.success ? '成功' : '失败' }}
        </el-tag>
      </div>
      <div class="card-body">
        <!-- 处理步骤 -->
        <div class="process-steps">
          <div class="step-item" :class="{ active: true }">
            <div class="step-icon success">
              <el-icon><CircleCheckFilled /></el-icon>
            </div>
            <div class="step-info">
              <span class="step-title">退出促销</span>
              <span class="step-count">{{ result.steps?.exit_promotion?.success || 0 }} 成功</span>
            </div>
          </div>
          <div class="step-line"></div>
          <div class="step-item" :class="{ active: true }">
            <div class="step-icon success">
              <el-icon><CircleCheckFilled /></el-icon>
            </div>
            <div class="step-info">
              <span class="step-title">更新价格</span>
              <span class="step-count">{{ result.steps?.price_update?.success || 0 }} 成功</span>
            </div>
          </div>
          <div class="step-line"></div>
          <div class="step-item" :class="{ active: true }">
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
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Upload, Download, CircleCheckFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { importLoss, processLossV2, getActions } from '@/api/promotion'
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
    // 先导入亏损商品
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

    // 然后处理亏损商品
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

.upload-area {
  width: 100%;
}

.upload-area :deep(.el-upload-dragger) {
  background: var(--glass-bg);
  border: 2px dashed var(--glass-border);
  border-radius: var(--radius-lg);
  padding: 40px;
  transition: all var(--transition-normal);

  &:hover {
    border-color: var(--primary);
    background: var(--glass-bg-hover);
  }
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.upload-icon {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.2), rgba(139, 92, 246, 0.1));
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;

  .el-icon {
    font-size: 28px;
    color: var(--primary);
  }
}

.upload-text {
  text-align: center;

  p {
    color: var(--text-primary);
    font-size: 15px;
    margin: 0;

    &.sub {
      color: var(--text-muted);
      font-size: 13px;
      margin-top: 4px;
    }

    em {
      color: var(--primary);
      font-style: normal;
      cursor: pointer;
    }
  }
}

.upload-tip {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
}

.rejoin-section {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--glass-border);
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
  padding-left: 12px;
  border-left: 3px solid var(--primary);
}

.rejoin-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}

.action-buttons {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}

.preview-card,
.result-card {
  margin-top: 24px;
}

.preview-hint {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
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

  .step-icon {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--glass-bg);
    border: 2px solid var(--glass-border);

    .el-icon {
      font-size: 24px;
      color: var(--text-muted);
    }

    &.success {
      background: rgba(16, 185, 129, 0.15);
      border-color: var(--success);

      .el-icon {
        color: var(--success);
      }
    }
  }

  .step-info {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;

    .step-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .step-count {
      font-size: 12px;
      color: var(--text-muted);
    }
  }
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
  border-top: 1px solid var(--glass-border);
}
</style>
