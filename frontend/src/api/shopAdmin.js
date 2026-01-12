import request from '@/utils/request'

// ========== 店铺管理员专用 API ==========

// ----- 店铺管理 -----

// 获取自己的店铺列表
export function getMyShops() {
  return request.get('/my/shops')
}

// 创建店铺
export function createMyShop(data) {
  return request.post('/my/shops', data)
}

// 更新店铺
export function updateMyShop(id, data) {
  return request.put(`/my/shops/${id}`, data)
}

// 删除店铺
export function deleteMyShop(id) {
  return request.delete(`/my/shops/${id}`)
}

// ----- 员工管理 -----

// 获取自己的员工列表
export function getMyStaff() {
  return request.get('/my/staff')
}

// 创建员工
export function createStaff(data) {
  return request.post('/my/staff', data)
}

// 更新员工状态
export function updateStaffStatus(id, status) {
  return request.put(`/my/staff/${id}/status`, { status })
}

// 重置员工密码
export function resetStaffPassword(id, newPassword) {
  return request.put(`/my/staff/${id}/password`, { new_password: newPassword })
}

// 更新员工可访问的店铺
export function updateStaffShops(id, shopIds) {
  return request.put(`/my/staff/${id}/shops`, { shop_ids: shopIds })
}

// 删除员工
export function deleteStaff(id) {
  return request.delete(`/my/staff/${id}`)
}
