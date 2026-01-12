<template>
  <div class="reprice">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">改价推广</h2>
    </div>

    <!-- 导入区域 -->
    <div class="glass-card">
      <div class="card-header">
        <span class="card-title">导入改价数据</span>
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

        <div class="divider">
          <span>或手动添加</span>
        </div>

        <div class="manual-input">
          <el-form :inline="true" :model="manualForm">
            <el-form-item label="SKU">
              <el-input v-model="manualForm.source_sku" placeholder="输入商品 SKU" style="width: 180px" />
            </el-form-item>
            <el-form-item label="新价格">
              <el-input-number v-model="manualForm.new_price" :min="0" :precision="2" style="width: 140px" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="addManualItem">
                <el-icon><Plus /></el-icon>
                添加
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>

    <!-- 待处理商品 -->
    <div v-if="products.length > 0" class="glass-card products-card">
      <div class="card-header">
        <span class="card-title">待处理商品</span>
        <div class="header-actions">
          <el-tag type="info" effect="plain">共 {{ products.length }} 个</el-tag>
          <el-button type="danger" text size="small" @click="clearProducts">
            <el-icon><Delete /></el-icon>
            清空
          </el-button>
        </div>
      </div>
      <div class="card-body">
        <el-table :data="products" size="small">
          <el-table-column prop="source_sku" label="SKU">
            <template #default="{ row }">
              <span class="sku-text">{{ row.source_sku }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="new_price" label="新价格" align="right">
            <template #default="{ row }">
              <span class="price-text">¥{{ row.new_price.toFixed(2) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ $index }">
              <el-button type="danger" size="small" text @click="removeProduct($index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 重新推广活动选择 -->
        <div class="reenroll-section">
          <h4 class="section-title">重新推广活动（可选）</h4>
          <el-skeleton v-if="actionsLoading" :rows="2" animated />
          <el-empty v-else-if="actions.length === 0" description="暂无可用活动" :image-size="40">
            <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </el-empty>
          <div v-else class="action-selector">
            <el-checkbox-group v-model="selectedActionIds">
              <el-checkbox
                v-for="action in actions"
                :key="action.action_id"
                :value="action.action_id"
                class="action-checkbox"
              >
                <div class="action-item">
                  <span class="action-title">{{ action.title || `活动 #${action.action_id}` }}</span>
                  <span class="action-id">ID: {{ action.action_id }}</span>
                  <el-tag v-if="action.is_elastic_boost" type="success" size="small">弹性</el-tag>
                  <el-tag v-if="action.is_discount_28" type="warning" size="small">28折</el-tag>
                </div>
              </el-checkbox>
            </el-checkbox-group>
            <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
              <el-icon><Refresh /></el-icon>
              刷新列表
            </el-button>
          </div>
          <div class="reenroll-hint">
            留空则不重新添加推广，仅执行：取消促销 → 更新价格
          </div>
        </div>

        <div class="action-section">
          <el-button
            type="primary"
            size="large"
            :loading="processing"
            @click="handleProcess"
          >
            <el-icon v-if="!processing"><Check /></el-icon>
            {{ processing ? '处理中...' : '开始处理' }}
          </el-button>
          <div class="process-hint">
            <el-icon><InfoFilled /></el-icon>
            处理流程：取消促销 → 更新价格{{ selectedActionIds.length > 0 ? ' → 重新添加推广' : '' }}
          </div>
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
          <div class="step-item">
            <div class="step-icon success">
              <el-icon><CircleCheckFilled /></el-icon>
            </div>
            <div class="step-info">
              <span class="step-title">取消促销</span>
              <span class="step-count">{{ result.remove_count }} 个商品</span>
            </div>
          </div>
          <div class="step-line"></div>
          <div class="step-item">
            <div class="step-icon success">
              <el-icon><CircleCheckFilled /></el-icon>
            </div>
            <div class="step-info">
              <span class="step-title">更新价格</span>
              <span class="step-count">{{ result.price_update_count }} 个商品</span>
            </div>
          </div>
          <div class="step-line"></div>
          <div class="step-item">
            <div class="step-icon success">
              <el-icon><CircleCheckFilled /></el-icon>
            </div>
            <div class="step-info">
              <span class="step-title">重新推广</span>
              <span class="step-count">{{ result.promote_count }} 个商品</span>
            </div>
          </div>
        </div>

        <!-- 失败详情 -->
        <div v-if="result.failed_items && result.failed_items.length > 0" class="failed-section">
          <h4 class="failed-title">
            <el-icon><WarningFilled /></el-icon>
            失败详情
          </h4>
          <el-table :data="result.failed_items" size="small" max-height="300">
            <el-table-column prop="sku" label="SKU" width="150">
              <template #default="{ row }">
                <span class="sku-text">{{ row.sku }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="step" label="失败步骤" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="warning">{{ row.step }}</el-tag>
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
import {
  UploadFilled,
  Plus,
  Delete,
  Check,
  InfoFilled,
  CircleCheckFilled,
  WarningFilled,
  Refresh
} from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getActions, removeRepricePromoteV2 } from '@/api/promotion'
import * as XLSX from 'xlsx'

const userStore = useUserStore()

const uploadRef = ref(null)
const processing = ref(false)
const products = ref([])
const result = ref(null)
const actionsLoading = ref(false)
const actions = ref([])
const selectedActionIds = ref([])

const manualForm = reactive({
  source_sku: '',
  new_price: 0
})

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

    const parsed = json.map(row => ({
      source_sku: String(row.source_sku || row.sku || row.SKU || ''),
      new_price: parseFloat(row.new_price || row.price || row.新价格 || 0)
    })).filter(row => row.source_sku && row.new_price > 0)

    if (parsed.length === 0) {
      ElMessage.warning('未找到有效数据，请检查文件格式')
      return
    }

    const existingSkus = new Set(products.value.map(p => p.source_sku))
    const newItems = parsed.filter(p => !existingSkus.has(p.source_sku))
    products.value = [...products.value, ...newItems]
    ElMessage.success(`成功导入 ${newItems.length} 个商品`)
  } catch (error) {
    console.error(error)
    ElMessage.error('解析文件失败')
  }
}

function addManualItem() {
  if (!manualForm.source_sku) {
    ElMessage.warning('请输入 SKU')
    return
  }
  if (manualForm.new_price <= 0) {
    ElMessage.warning('请输入有效价格')
    return
  }

  const exists = products.value.some(p => p.source_sku === manualForm.source_sku)
  if (exists) {
    ElMessage.warning('该 SKU 已存在')
    return
  }

  products.value.push({
    source_sku: manualForm.source_sku,
    new_price: manualForm.new_price
  })

  manualForm.source_sku = ''
  manualForm.new_price = 0
}

function removeProduct(index) {
  products.value.splice(index, 1)
}

function clearProducts() {
  products.value = []
}

async function handleProcess() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  if (products.value.length === 0) {
    ElMessage.warning('请先添加商品')
    return
  }

  const actionText = selectedActionIds.value.length > 0
    ? '取消促销 → 更新价格 → 重新添加推广'
    : '取消促销 → 更新价格'

  try {
    await ElMessageBox.confirm(
      `确定要处理 ${products.value.length} 个商品吗？此操作将：${actionText}`,
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

  processing.value = true
  result.value = null

  try {
    const res = await removeRepricePromoteV2({
      shop_id: shopId,
      products: products.value,
      reenroll_action_ids: selectedActionIds.value
    })
    result.value = res.data
    if (res.data.success) {
      ElMessage.success('处理完成')
      products.value = []
    } else {
      ElMessage.warning('部分商品处理失败，请查看详情')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('处理失败')
  } finally {
    processing.value = false
  }
}

watch(() => userStore.currentShopId, () => {
  selectedActionIds.value = []
  fetchActions()
})

onMounted(() => {
  fetchActions()
})
</script>

<style scoped>
.reprice {
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

.divider {
  display: flex;
  align-items: center;
  margin: 24px 0;

  &::before,
  &::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--glass-border);
  }

  span {
    padding: 0 16px;
    font-size: 13px;
    color: var(--text-muted);
  }
}

.manual-input {
  display: flex;
  justify-content: center;
}

.products-card,
.result-card {
  margin-top: 24px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
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

.action-section {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.process-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);

  .el-icon {
    color: var(--primary);
  }
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

.reenroll-section {
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

.reenroll-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
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
</style>
