import request from '@/utils/request'

// ========== 促销活动管理 ==========

// 获取促销活动列表
export function getActions(shopId) {
  return request.get('/promotions/actions', { params: { shop_id: shopId } })
}

// 同步促销活动
export function syncActions(shopId) {
  return request.post('/promotions/sync-actions', { shop_id: shopId })
}

// 手动添加促销活动
export function createManualAction(data) {
  return request.post('/promotions/actions/manual', data)
}

// 删除促销活动
export function deleteAction(id, shopId) {
  return request.delete(`/promotions/actions/${id}`, { params: { shop_id: shopId } })
}

// 更新促销活动显示名称
export function updateActionDisplayName(id, shopId, displayName) {
  return request.put(`/promotions/actions/${id}/display-name`,
    { display_name: displayName },
    { params: { shop_id: shopId } }
  )
}

// 更新促销活动排序
export function updateActionsSortOrder(shopId, sortOrders) {
  return request.put('/promotions/actions/sort-order', {
    shop_id: shopId,
    sort_orders: sortOrders
  })
}

// ========== V1 接口（保持兼容）==========

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

// ========== V2 接口（支持选择活动）==========

// 批量报名到指定活动
export function batchEnrollV2(data) {
  return request.post('/promotions/batch-enroll-v2', data)
}

// 处理亏损商品（支持选择重新报名活动）
export function processLossV2(data) {
  return request.post('/promotions/process-loss-v2', data)
}

// 移除-改价-重新推广（支持选择活动）
export function removeRepricePromoteV2(data) {
  return request.post('/promotions/remove-reprice-promote-v2', data)
}

// ========== Excel 相关 ==========

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
