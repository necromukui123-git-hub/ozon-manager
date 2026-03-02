/**
 * 店铺促销活动 API 封装
 * 
 * 参考 Discount28Campaign 的实现，通过 page.evaluate() 在 seller.ozon.ru 域下
 * 调用 Ozon 卖家后台的内部 API，实现店铺促销活动的候选商品获取、申报和退出。
 * 
 * API 端点:
 * - POST /api/site/own-seller-products/v1/action/{id}/candidate — 获取候选商品
 * - POST /api/site/own-seller-products/v1/action/{id}/activate  — 申报商品
 * - POST /api/site/own-seller-products/v1/action/{id}/deactivate — 退出商品
 */

const SELLER_BASE_URL = process.env.OZON_SELLER_BASE_URL || 'https://seller.ozon.ru'

/**
 * 确保页面在 seller.ozon.ru 并已登录
 */
async function ensureSellerReady(page) {
  const currentUrl = page.url()
  if (!currentUrl.includes('seller.ozon.ru')) {
    await page.goto(`${SELLER_BASE_URL}/app/dashboard`, {
      waitUntil: 'domcontentloaded',
      timeout: 30000,
    })
  }

  const url = page.url().toLowerCase()
  if (url.includes('login') || url.includes('auth')) {
    throw new Error('Agent 浏览器未登录 Ozon Seller，请先手动登录')
  }

  // 等页面稳定
  await page.waitForTimeout(1000)
}

/**
 * 获取店铺活动的所有候选商品（含已参与的）
 * 使用 cursor 分页迭代获取全部
 * 
 * @param {import('playwright').Page} page
 * @param {string} actionId - 店铺活动原始 ID (source_action_id)
 * @returns {Promise<Array>} 商品列表
 */
async function fetchCandidates(page, actionId) {
  await ensureSellerReady(page)

  const result = await page.evaluate(async ({ actionId, baseUrl }) => {
    const allProducts = []
    let hasNext = true
    let cursor = ''
    let pageCount = 0

    while (hasNext && pageCount < 100) {
      const body = { limit: 100 }
      if (cursor) {
        body.cursor = cursor
      }

      const response = await fetch(
        `${baseUrl}/api/site/own-seller-products/v1/action/${actionId}/candidate`,
        {
          method: 'POST',
          headers: {
            'accept': 'application/json',
            'content-type': 'application/json',
          },
          body: JSON.stringify(body),
          credentials: 'include',
        }
      )

      if (!response.ok) {
        throw new Error(`API 请求失败: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()

      if (data.products && data.products.length > 0) {
        allProducts.push(...data.products)
      }

      hasNext = Boolean(data.has_next)
      cursor = data.cursor || ''
      pageCount++

      if (hasNext) {
        await new Promise((r) => setTimeout(r, 100))
      }
    }

    return allProducts
  }, { actionId, baseUrl: SELLER_BASE_URL })

  return result
}

/**
 * 批量申报商品到店铺促销活动
 * 
 * @param {import('playwright').Page} page
 * @param {string} actionId - 店铺活动原始 ID
 * @param {Array} products - 要申报的商品（从 fetchCandidates 返回的格式）
 * @returns {Promise<{success: boolean, message: string}>}
 */
async function activateProducts(page, actionId, products) {
  await ensureSellerReady(page)

  const result = await page.evaluate(async ({ actionId, products, baseUrl }) => {
    // 构建请求 payload
    const payload = products.map((p) => ({
      product_id: Number(p.product_id || p.id),
      skus: (p.skus || []).map(Number),
      action_price: p.action_price || { currency_code: '', nanos: 0, units: '0' },
      discount_percent: p.discount_percent || 0,
      currency: p.currency || '',
    }))

    const response = await fetch(
      `${baseUrl}/api/site/own-seller-products/v1/action/${actionId}/activate`,
      {
        method: 'POST',
        headers: {
          'accept': 'application/json',
          'content-type': 'application/json',
        },
        body: JSON.stringify({ products: payload }),
        credentials: 'include',
      }
    )

    if (!response.ok) {
      const text = await response.text().catch(() => '')
      throw new Error(`申报失败: ${response.status} ${response.statusText} - ${text}`)
    }

    return { success: true, message: `成功申报 ${products.length} 个商品` }
  }, { actionId, products, baseUrl: SELLER_BASE_URL })

  return result
}

/**
 * 批量退出商品从店铺促销活动
 * 
 * @param {import('playwright').Page} page
 * @param {string} actionId - 店铺活动原始 ID
 * @param {string[]} skus - 要退出的商品 SKU 列表
 * @returns {Promise<{success: boolean, message: string}>}
 */
async function deactivateProducts(page, actionId, skus) {
  await ensureSellerReady(page)

  const result = await page.evaluate(async ({ actionId, skus, baseUrl }) => {
    const response = await fetch(
      `${baseUrl}/api/site/own-seller-products/v1/action/${actionId}/deactivate`,
      {
        method: 'POST',
        headers: {
          'accept': 'application/json',
          'content-type': 'application/json',
        },
        body: JSON.stringify({ skus }),
        credentials: 'include',
      }
    )

    if (!response.ok) {
      const text = await response.text().catch(() => '')
      throw new Error(`退出失败: ${response.status} ${response.statusText} - ${text}`)
    }

    return { success: true, message: `成功退出 ${skus.length} 个商品` }
  }, { actionId, skus, baseUrl: SELLER_BASE_URL })

  return result
}

module.exports = {
  fetchCandidates,
  activateProducts,
  deactivateProducts,
  ensureSellerReady,
}
