<template>
  <div class="reprice">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="gradient">改价推广</h2>
    </div>

    <!-- Bento Grid 布局 -->
    <div class="bento-grid--2col">
      <!-- 导入区域 -->
      <BentoCard title="导入改价数据" :icon="UploadFilled" size="1x1">
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

        <div class="divider">
          <span>或手动添加</span>
        </div>

        <div class="manual-input">
          <el-input v-model="manualForm.source_sku" placeholder="SKU" style="width: 140px" />
          <el-input-number v-model="manualForm.new_price" :min="0" :precision="2" placeholder="新价格" style="width: 120px" />
          <el-button type="primary" @click="addManualItem">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>
      </BentoCard>

      <!-- 重新推广活动选择 -->
      <BentoCard title="重新推广活动" :icon="Ticket" size="1x1">
        <template #actions>
          <el-button type="primary" text size="small" @click="fetchActions" :loading="actionsLoading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </template>

        <div class="process-info">
          <el-icon><InfoFilled /></el-icon>
          <span>流程：取消促销 → 更新价格 → 重新推广</span>
        </div>

        <div v-if="actionsLoading" class="loading-state">
          <el-skeleton :rows="2" animated />
        </div>
        <div v-else-if="actions.length === 0" class="empty-state-small">
          <el-icon><Box /></el-icon>
          <p>暂无可用活动</p>
        </div>
        <div v-else class="action-list">
          <el-checkbox-group v-model="selectedActionIds">
            <el-checkbox
              v-for="action in actions"
              :key="action.action_id"
              :value="action.action_id"
              class="action-checkbox"
            >
              <div class="action-item">
                <span class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</span>
                <span class="action-id">ID: {{ action.action_id }}</span>
              </div>
            </el-checkbox>
          </el-checkbox-group>
        </div>
        <div class="action-hint">留空则不重新添加推广，仅执行取消促销和更新价格</div>
      </BentoCard>
    </div>

    <!-- 待处理商品 -->
    <BentoCard
      v-if="products.length > 0"
      title="待处理商品"
      :icon="Goods"
      size="4x1"
      no-padding
      class="products-card"
    >
      <template #actions>
        <el-tag type="info" effect="plain" size="small">共 {{ products.length }} 个</el-tag>
        <el-button type="danger" text size="small" @click="clearProducts">
          <el-icon><Delete /></el-icon>
          清空
        </el-button>
      </template>

      <el-table :data="products" size="small" max-height="300">
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

      <template #footer>
        <div class="action-footer">
          <div class="process-hint">
            <el-icon><InfoFilled /></el-icon>
            处理流程：取消促销 → 更新价格{{ selectedActionIds.length > 0 ? ' → 重新添加推广' : '' }}
          </div>
          <el-button
            type="primary"
            size="large"
            :loading="processing"
            @click="handleProcess"
          >
            <el-icon v-if="!processing"><Check /></el-icon>
            {{ processing ? '处理中...' : '开始处理' }}
          </el-button>
        </div>
      </template>
    </BentoCard>

    <!-- 处理结果 -->
    <div v-if="result" class="bento-grid">
      <StatCard
        :value="result.remove_count"
        label="取消促销"
        :icon="CircleCloseFilled"
        variant="warning"
      />
      <StatCard
        :value="result.price_update_count"
        label="更新价格"
        :icon="PriceTag"
        variant="primary"
      />
      <StatCard
        :value="result.promote_count"
        label="重新推广"
        :icon="Promotion"
        variant="success"
      />
      <StatCard
        :value="result.success ? '成功' : '失败'"
        label="执行状态"
        :icon="result.success ? CircleCheckFilled : WarningFilled"
        :variant="result.success ? 'success' : 'danger'"
      />

      <!-- 失败详情 -->
      <BentoCard
        v-if="result.failed_items && result.failed_items.length > 0"
        title="失败详情"
        :icon="WarningFilled"
        size="4x1"
        no-padding
      >
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
      </BentoCard>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UploadFilled, Plus, Delete, Check, InfoFilled, Refresh, Box,
  CircleCheckFilled, CircleCloseFilled, WarningFilled, PriceTag,
  Promotion, Goods, Ticket
} from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getActions, removeRepricePromoteV2 } from '@/api/promotion'
import { StatCard, BentoCard } from '@/components/bento'
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
  padding: 24px;
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
  gap: 10px;
}

.upload-icon {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.upload-icon .el-icon {
  font-size: 20px;
  color: white;
}

.upload-text {
  text-align: center;
}

.upload-text p {
  color: var(--text-primary);
  font-size: 13px;
  margin: 0;
}

.upload-text p.sub {
  color: var(--text-muted);
  font-size: 12px;
  margin-top: 2px;
}

.upload-text em {
  color: var(--primary);
  font-style: normal;
  cursor: pointer;
}

.divider {
  display: flex;
  align-items: center;
  margin: 16px 0;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--surface-border);
}

.divider span {
  padding: 0 12px;
  font-size: 12px;
  color: var(--text-muted);
}

.manual-input {
  display: flex;
  gap: 8px;
  align-items: center;
}

.process-info {
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

.loading-state {
  padding: 16px 0;
}

.empty-state-small {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  color: var(--text-muted);
}

.empty-state-small .el-icon {
  font-size: 28px;
  opacity: 0.5;
}

.empty-state-small p {
  font-size: 12px;
  margin: 0;
}

.action-list {
  max-height: 150px;
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
  gap: 10px;
}

.action-title {
  font-weight: 500;
  font-size: 12px;
}

.action-id {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 10px;
  color: var(--text-muted);
}

.action-hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-muted);
}

.products-card {
  margin-bottom: 24px;
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

.action-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.process-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
}

.process-hint .el-icon {
  color: var(--primary);
}

/* 响应式 */
@media (max-width: 768px) {
  .action-footer {
    flex-direction: column;
    gap: 12px;
  }

  .manual-input {
    flex-wrap: wrap;
  }
}
</style>
