<template>
  <div class="reprice">
    <h2>改价推广</h2>

    <el-card>
      <template #header>
        <span>导入改价数据</span>
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
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          将Excel文件拖到此处，或<em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            请上传包含改价数据的Excel文件（.xlsx/.xls），文件需包含：source_sku、new_price 列
          </div>
        </template>
      </el-upload>

      <el-divider>或手动添加</el-divider>

      <div class="manual-input">
        <el-form :inline="true" :model="manualForm">
          <el-form-item label="SKU">
            <el-input v-model="manualForm.source_sku" placeholder="输入商品SKU" />
          </el-form-item>
          <el-form-item label="新价格">
            <el-input-number v-model="manualForm.new_price" :min="0" :precision="2" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="addManualItem">添加</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>

    <el-card v-if="products.length > 0" class="products-card">
      <template #header>
        <div class="card-header">
          <span>待处理商品（共 {{ products.length }} 个）</span>
          <el-button type="danger" size="small" @click="clearProducts">清空</el-button>
        </div>
      </template>

      <el-table :data="products" size="small">
        <el-table-column prop="source_sku" label="SKU" />
        <el-table-column prop="new_price" label="新价格">
          <template #default="{ row }">
            ¥{{ row.new_price.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ $index }">
            <el-button type="danger" size="small" text @click="removeProduct($index)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="action-buttons">
        <el-button type="primary" :loading="processing" @click="handleProcess">
          开始处理
        </el-button>
        <div class="process-hint">
          处理流程：取消促销 → 更新价格 → 重新添加推广
        </div>
      </div>
    </el-card>

    <el-card v-if="result" class="result-card">
      <template #header>
        <span>处理结果</span>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="状态">
          <el-tag :type="result.success ? 'success' : 'danger'">
            {{ result.success ? '成功' : '失败' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="处理商品数">
          {{ result.affected_count }}
        </el-descriptions-item>
        <el-descriptions-item label="取消促销">
          {{ result.remove_count }}
        </el-descriptions-item>
        <el-descriptions-item label="改价成功">
          {{ result.price_update_count }}
        </el-descriptions-item>
        <el-descriptions-item label="重新推广">
          {{ result.promote_count }}
        </el-descriptions-item>
      </el-descriptions>

      <el-steps :active="3" finish-status="success" class="process-steps">
        <el-step title="取消促销" :description="`${result.remove_count} 个商品`" />
        <el-step title="更新价格" :description="`${result.price_update_count} 个商品`" />
        <el-step title="重新推广" :description="`${result.promote_count} 个商品`" />
      </el-steps>

      <div v-if="result.failed_items && result.failed_items.length > 0" class="failed-list">
        <h4>失败详情</h4>
        <el-table :data="result.failed_items" size="small" max-height="300">
          <el-table-column prop="sku" label="SKU" width="150" />
          <el-table-column prop="step" label="失败步骤" width="100" />
          <el-table-column prop="error" label="错误信息" />
        </el-table>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { importReprice } from '@/api/promotion'
import * as XLSX from 'xlsx'

const userStore = useUserStore()

const uploadRef = ref(null)
const processing = ref(false)
const products = ref([])
const result = ref(null)

const manualForm = reactive({
  source_sku: '',
  new_price: 0
})

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

    // 合并到现有列表，去重
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
    ElMessage.warning('请输入SKU')
    return
  }
  if (manualForm.new_price <= 0) {
    ElMessage.warning('请输入有效价格')
    return
  }

  const exists = products.value.some(p => p.source_sku === manualForm.source_sku)
  if (exists) {
    ElMessage.warning('该SKU已存在')
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

  try {
    await ElMessageBox.confirm(
      `确定要处理 ${products.value.length} 个商品吗？此操作将：取消促销 → 更新价格 → 重新添加推广`,
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
    const formData = new FormData()
    formData.append('shop_id', shopId)
    formData.append('products', JSON.stringify(products.value))

    const res = await importReprice(formData)
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
</script>

<style scoped>
.reprice {
  padding: 20px;
}

.reprice h2 {
  margin-bottom: 20px;
}

.upload-area {
  width: 100%;
}

.manual-input {
  margin-top: 20px;
}

.products-card {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.action-buttons {
  margin-top: 20px;
  display: flex;
  align-items: center;
  gap: 20px;
}

.process-hint {
  color: #909399;
  font-size: 12px;
}

.result-card {
  margin-top: 20px;
}

.process-steps {
  margin-top: 20px;
}

.failed-list {
  margin-top: 20px;
}

.failed-list h4 {
  margin-bottom: 10px;
  color: #F56C6C;
}
</style>
