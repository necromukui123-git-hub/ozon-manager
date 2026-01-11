import request from '@/utils/request'

// 获取商品列表
export function getProducts(params) {
  return request.get('/products', { params })
}

// 获取商品详情
export function getProduct(id) {
  return request.get(`/products/${id}`)
}

// 同步商品
export function syncProducts(shopId) {
  return request.post('/products/sync', { shop_id: shopId })
}

// 获取统计数据
export function getStats(shopId) {
  return request.get('/stats/overview', { params: { shop_id: shopId } })
}
