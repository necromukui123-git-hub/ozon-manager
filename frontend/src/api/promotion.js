import request from '@/utils/request'

// ========== 促销活动管理 ==========

// 获取促销活动列表
export function getActions(shopId) {
  return request.get('/promotions/actions', { params: { shop_id: shopId } })
}

// 同步促销活动
export function syncActions(shopId) {
  return request({
    url: '/promotions/sync-actions',
    method: 'post',
    data: { shop_id: shopId }
  })
}

export function updateActionsSortOrder(shopId, sortOrders) {
  return request.put('/promotions/actions/sort-order', {
    shop_id: shopId,
    sort_orders: sortOrders
  })
}

// 获取活动的推广商品
export function getActionProducts(actionId, shopId, params) {
  return request.get(`/promotions/actions/${actionId}/products`, {
    params: { shop_id: shopId, ...params }
  })
}

// 更新活动显示名称
export function updateActionDisplayName(id, shopId, displayName) {
  return request.put(`/promotions/actions/${id}/display-name`, {
    shop_id: shopId,
    display_name: displayName
  })
}

// 新增手动活动
export function createManualAction(data) {
  return request.post('/promotions/actions/manual', data)
}

// 删除活动
export function deleteAction(id, shopId) {
  return request.delete(`/promotions/actions/${id}`, {
    params: { shop_id: shopId }
  })
}

// ========== V1 老接口 ==============

// 批量报名（自动降价）
export function batchEnroll(data) {
  return request.post('/promotions/batch-enroll', data)
}

// 处理亏损商品（重新调价并报名）
export function processLoss(data) {
  return request.post('/promotions/process-loss', data)
}

// 移除-改价-重新推广
export function removeRepricePromote(data) {
  return request.post('/promotions/remove-reprice-promote', data)
}

// ========== V2 接口（支持选择活动） ==============

// 批量报名（自动降价）- V2
export function batchEnrollV2(data) {
  return request.post('/promotions/batch-enroll-v2', data)
}

// 处理亏损商品（重新调价并报名）- V2
export function processLossV2(data) {
  return request.post('/promotions/process-loss-v2', data)
}

// 移除-改价-重新推广（支持选择活动） - V2
export function removeRepricePromoteV2(data) {
  return request.post('/promotions/remove-reprice-promote-v2', data)
}

// ========== 统一操作接口 (方案 B) ==============

// 统一报名 (自动判断官方/店铺)
export function unifiedEnroll(data) {
  return request.post('/promotions/unified-enroll', data)
}

// 统一退出 (自动判断官方/店铺)
export function unifiedRemove(data) {
  return request.post('/promotions/unified-remove', data)
}

// 统一亏损处理
export function unifiedProcessLoss(data) {
  return request.post('/promotions/unified-process-loss', data)
}

// 统一改价推广
export function unifiedRepricePromote(data) {
  return request.post('/promotions/unified-reprice-promote', data)
}

// ========== 自动加促销 ==============

export function getAutoPromotionConfig(shopId) {
  return request.get('/promotions/auto-add/config', {
    params: { shop_id: shopId }
  })
}

export function updateAutoPromotionConfig(data) {
  return request.put('/promotions/auto-add/config', data)
}

export function startAutoPromotionRun(data) {
  return request.post('/promotions/auto-add/runs', data)
}

export function listAutoPromotionRuns(params) {
  return request.get('/promotions/auto-add/runs', { params })
}

export function getAutoPromotionRunDetail(runId, shopId) {
  return request.get(`/promotions/auto-add/runs/${runId}`, {
    params: { shop_id: shopId }
  })
}

// ========== Excel 相关 ==============

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
