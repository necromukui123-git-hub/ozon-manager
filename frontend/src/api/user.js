import request from '@/utils/request'

// 获取用户列表
export function getUsers() {
  return request.get('/users')
}

// 创建用户
export function createUser(data) {
  return request.post('/users', data)
}

// 更新用户状态
export function updateUserStatus(id, status) {
  return request.put(`/users/${id}/status`, { status })
}

// 重置用户密码
export function updateUserPassword(id, newPassword) {
  return request.put(`/users/${id}/password`, { new_password: newPassword })
}

// 更新用户店铺权限
export function updateUserShops(id, shopIds) {
  return request.put(`/users/${id}/shops`, { shop_ids: shopIds })
}

// 修改当前用户密码
export function changePassword(oldPassword, newPassword) {
  return request.put('/auth/password', { old_password: oldPassword, new_password: newPassword })
}
