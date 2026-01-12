import request from '@/utils/request'

// 获取店铺列表
export function getShops() {
  return request.get('/shops')
}

// 获取店铺详情
export function getShop(id) {
  return request.get(`/shops/${id}`)
}
