<template>
  <div class="batch-enroll">
    <h2>批量报名促销</h2>

    <el-card>
      <el-form :model="form" label-width="140px">
        <el-form-item label="排除亏损商品">
          <el-switch v-model="form.exclude_loss" />
          <span class="form-hint">开启后将跳过标记为亏损的商品</span>
        </el-form-item>

        <el-form-item label="排除已推广商品">
          <el-switch v-model="form.exclude_promoted" />
          <span class="form-hint">开启后将跳过已参与推广活动的商品</span>
        </el-form-item>

        <el-form-item label="报名弹性提升">
          <el-switch v-model="form.enroll_elastic_boost" />
          <span class="form-hint">Ozon官方"弹性提升"促销活动</span>
        </el-form-item>

        <el-form-item label="报名折扣28%">
          <el-switch v-model="form.enroll_discount_28" />
          <span class="form-hint">店铺"折扣28%"促销活动</span>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleSubmit">
            开始报名
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="result" class="result-card">
      <template #header>
        <span>执行结果</span>
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
        <el-descriptions-item label="成功数">
          {{ result.success_count }}
        </el-descriptions-item>
        <el-descriptions-item label="失败数">
          {{ result.failed_count }}
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="result.failed_items && result.failed_items.length > 0" class="failed-list">
        <h4>失败详情</h4>
        <el-table :data="result.failed_items" size="small" max-height="300">
          <el-table-column prop="sku" label="SKU" width="150" />
          <el-table-column prop="error" label="错误信息" />
        </el-table>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { batchEnroll } from '@/api/promotion'

const userStore = useUserStore()

const loading = ref(false)
const result = ref(null)

const form = reactive({
  exclude_loss: true,
  exclude_promoted: true,
  enroll_elastic_boost: true,
  enroll_discount_28: true
})

async function handleSubmit() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  if (!form.enroll_elastic_boost && !form.enroll_discount_28) {
    ElMessage.warning('请至少选择一个促销活动')
    return
  }

  try {
    await ElMessageBox.confirm(
      '确定要批量报名促销活动吗？此操作可能需要一些时间。',
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

  loading.value = true
  result.value = null

  try {
    const res = await batchEnroll({
      shop_id: shopId,
      ...form
    })
    result.value = res.data
    if (res.data.success) {
      ElMessage.success(`批量报名完成，成功 ${res.data.success_count} 个商品`)
    } else {
      ElMessage.warning('部分商品报名失败，请查看详情')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('操作失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.batch-enroll {
  padding: 20px;
}

.batch-enroll h2 {
  margin-bottom: 20px;
}

.form-hint {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}

.result-card {
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
