import request from '@/utils/request'

// ========== 系统管理员专用 API ==========

// 获取店铺管理员列表
export function getShopAdmins() {
  return request.get('/admin/shop-admins')
}

// 获取店铺管理员详情
export function getShopAdmin(id) {
  return request.get(`/admin/shop-admins/${id}`)
}

// 创建店铺管理员
export function createShopAdmin(data) {
  return request.post('/admin/shop-admins', data)
}

// 更新店铺管理员状态
export function updateShopAdminStatus(id, status) {
  return request.put(`/admin/shop-admins/${id}/status`, { status })
}

// 重置店铺管理员密码
export function resetShopAdminPassword(id, newPassword) {
  return request.put(`/admin/shop-admins/${id}/password`, { new_password: newPassword })
}

// 删除店铺管理员
export function deleteShopAdmin(id) {
  return request.delete(`/admin/shop-admins/${id}`)
}

// 获取系统概览
export function getSystemOverview() {
  return request.get('/admin/overview')
}
