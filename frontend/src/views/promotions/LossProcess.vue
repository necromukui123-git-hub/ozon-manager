<template>
  <div class="loss-process">
    <h2>亏损商品处理</h2>

    <el-card>
      <template #header>
        <span>导入亏损商品</span>
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
            请上传包含亏损商品的Excel文件（.xlsx/.xls），文件需包含：source_sku、new_price 列
          </div>
        </template>
      </el-upload>

      <div class="action-buttons">
        <el-button type="primary" :loading="importing" :disabled="!file" @click="handleImport">
          导入并处理
        </el-button>
        <el-button @click="downloadTemplate">
          下载模板
        </el-button>
      </div>
    </el-card>

    <el-card v-if="previewData.length > 0" class="preview-card">
      <template #header>
        <span>预览数据（共 {{ previewData.length }} 条）</span>
      </template>
      <el-table :data="previewData.slice(0, 10)" size="small">
        <el-table-column prop="source_sku" label="SKU" />
        <el-table-column prop="new_price" label="新价格">
          <template #default="{ row }">
            ¥{{ row.new_price }}
          </template>
        </el-table-column>
      </el-table>
      <div v-if="previewData.length > 10" class="preview-hint">
        仅显示前10条，共 {{ previewData.length }} 条数据
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
        <el-descriptions-item label="退出促销">
          {{ result.exit_count }}
        </el-descriptions-item>
        <el-descriptions-item label="改价成功">
          {{ result.price_update_count }}
        </el-descriptions-item>
        <el-descriptions-item label="重新报名">
          {{ result.rejoin_count }}
        </el-descriptions-item>
      </el-descriptions>

      <el-steps :active="3" finish-status="success" class="process-steps">
        <el-step title="退出促销" :description="`${result.exit_count} 个商品`" />
        <el-step title="更新价格" :description="`${result.price_update_count} 个商品`" />
        <el-step title="重新报名28%" :description="`${result.rejoin_count} 个商品`" />
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
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { importLoss } from '@/api/promotion'
import * as XLSX from 'xlsx'

const userStore = useUserStore()

const uploadRef = ref(null)
const file = ref(null)
const importing = ref(false)
const previewData = ref([])
const result = ref(null)

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

    const res = await importLoss(formData)
    result.value = res.data
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
</script>

<style scoped>
.loss-process {
  padding: 20px;
}

.loss-process h2 {
  margin-bottom: 20px;
}

.upload-area {
  width: 100%;
}

.action-buttons {
  margin-top: 20px;
  display: flex;
  gap: 10px;
}

.preview-card {
  margin-top: 20px;
}

.preview-hint {
  margin-top: 10px;
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
