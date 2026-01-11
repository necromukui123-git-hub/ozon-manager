import request from '@/utils/request'

// 获取店铺列表
export function getShops() {
  return request.get('/shops')
}

// 获取店铺详情
export function getShop(id) {
  return request.get(`/shops/${id}`)
}

// 创建店铺
export function createShop(data) {
  return request.post('/shops', data)
}

// 更新店铺
export function updateShop(id, data) {
  return request.put(`/shops/${id}`, data)
}

// 删除店铺
export function deleteShop(id) {
  return request.delete(`/shops/${id}`)
}
