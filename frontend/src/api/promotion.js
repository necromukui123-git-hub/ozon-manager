import request from '@/utils/request'

// 批量报名促销活动
export function batchEnroll(data) {
  return request.post('/promotions/batch-enroll', data)
}

// 处理亏损商品
export function processLoss(data) {
  return request.post('/promotions/process-loss', data)
}

// 移除-改价-重新推广
export function removeRepricePromote(data) {
  return request.post('/promotions/remove-reprice-promote', data)
}

// 同步促销活动
export function syncActions(shopId) {
  return request.post('/promotions/sync-actions', { shop_id: shopId })
}

// 导入亏损商品
export function importLoss(formData) {
  return request.post('/excel/import-loss', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// 导入改价商品
export function importReprice(formData) {
  return request.post('/excel/import-reprice', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// 导出可推广商品
export function exportPromotable(shopId) {
  return request.get('/excel/export-promotable', {
    params: { shop_id: shopId },
    responseType: 'blob'
  })
}

// 下载模板
export function downloadTemplate() {
  return request.get('/excel/template/loss', {
    responseType: 'blob'
  })
}
