const path = require('path')

const { runActionFlow } = require('../flows/ozon-action-flow')
const {
  fetchCandidates,
  activateProducts,
  deactivateProducts,
  ensureSellerReady,
} = require('./shop-promo-api')

function createPlaywrightExecutor() {
  const userDataDir = process.env.BROWSER_USER_DATA_DIR || path.resolve(__dirname, '..', 'browser-profile')
  const headless = String(process.env.BROWSER_HEADLESS || 'false').toLowerCase() === 'true'
  const browserChannel = process.env.BROWSER_CHANNEL || 'chrome'

  let playwright = null
  let context = null

  async function ensureContext() {
    if (context) {
      return context
    }

    playwright = require('playwright')
    context = await playwright.chromium.launchPersistentContext(userDataDir, {
      channel: browserChannel,
      headless,
      viewport: { width: 1440, height: 900 },
    })

    return context
  }

  return {
    name: 'playwright-executor',

    async executeJob(job) {
      const browserContext = await ensureContext()

      if (job.job_type === 'shop_action_declare') {
        return executeSingleShopActionJob(browserContext, job, 'declare')
      }

      if (job.job_type === 'shop_action_remove') {
        return executeSingleShopActionJob(browserContext, job, 'remove')
      }

      if (job.job_type === 'promo_unified_enroll') {
        return executeUnifiedShopActionsJob(browserContext, job, 'declare')
      }

      if (job.job_type === 'promo_unified_remove') {
        return executeUnifiedShopActionsJob(browserContext, job, 'remove')
      }

      if (job.job_type === 'sync_shop_actions') {
        const snapshot = await fetchShopActionsSnapshot(browserContext)
        const actions = snapshot.actions || []
        const canConfirmCapture = Number(snapshot.debug?.captured_action_responses || 0) > 0
        const isSuccess = actions.length > 0 || canConfirmCapture
        console.log(`[Executor] sync_shop_actions captured_actions=${actions.length}, captured_responses=${snapshot.debug?.captured_action_responses || 0}`)
        if (!isSuccess && snapshot.error) {
          console.warn(`[Executor] sync_shop_actions warning: ${snapshot.error}`)
        }
        const results = [{
          source_sku: '__sync_shop_actions__',
          overall_status: isSuccess ? 'success' : 'failed',
          step_exit_status: isSuccess ? 'success' : 'failed',
          step_reprice_status: isSuccess ? 'success' : 'failed',
          step_readd_status: isSuccess ? 'success' : 'failed',
          step_exit_error: isSuccess ? '' : (snapshot.error || 'shop actions capture returned empty'),
          step_reprice_error: isSuccess ? '' : (snapshot.error || 'shop actions capture returned empty'),
          step_readd_error: isSuccess ? '' : (snapshot.error || 'shop actions capture returned empty'),
        }]
        results.__meta = {
          actions,
          debug: snapshot.debug || {},
          error: snapshot.error || '',
        }
        return results
      }

      if (job.job_type === 'sync_action_products') {
        const sourceActionID = String(job.meta?.source_action_id || '')
        const snapshot = await fetchActionProductsSnapshot(browserContext, sourceActionID)
        const items = snapshot.items || []
        const canConfirmCapture = Number(snapshot.debug?.captured_product_responses || 0) > 0
        const isSuccess = items.length > 0 || canConfirmCapture
        console.log(`[Executor] sync_action_products action=${sourceActionID}, captured_items=${items.length}, captured_responses=${snapshot.debug?.captured_product_responses || 0}`)
        if (!isSuccess && snapshot.error) {
          console.warn(`[Executor] sync_action_products warning: ${snapshot.error}`)
        }
        const results = [{
          source_sku: '__sync_action_products__',
          overall_status: isSuccess ? 'success' : 'failed',
          step_exit_status: isSuccess ? 'success' : 'failed',
          step_reprice_status: isSuccess ? 'success' : 'failed',
          step_readd_status: isSuccess ? 'success' : 'failed',
          step_exit_error: isSuccess ? '' : (snapshot.error || 'action products capture returned empty'),
          step_reprice_error: isSuccess ? '' : (snapshot.error || 'action products capture returned empty'),
          step_readd_error: isSuccess ? '' : (snapshot.error || 'action products capture returned empty'),
        }]
        results.__meta = {
          items,
          debug: snapshot.debug || {},
          error: snapshot.error || '',
        }
        return results
      }

      const results = []

      for (const item of job.items || []) {
        try {
          const stepResult = await runActionFlow(browserContext, {
            shopId: job.shop_id,
            sourceSKU: item.source_sku,
            targetPrice: item.target_price,
          })

          results.push(stepResult)
        } catch (error) {
          const message = error?.message || 'unknown executor error'
          results.push({
            source_sku: item.source_sku,
            overall_status: 'failed',
            step_exit_status: 'failed',
            step_reprice_status: 'failed',
            step_readd_status: 'failed',
            step_exit_error: message,
            step_reprice_error: message,
            step_readd_error: message,
          })
        }
      }

      return results
    },

    async close() {
      if (context) {
        await context.close()
        context = null
      }
    },
  }
}

function uniqStrings(values) {
  const set = new Set()
  for (const value of values || []) {
    if (!value) continue
    set.add(String(value))
  }
  return Array.from(set)
}

function toNumber(value, fallback = 0) {
  if (value === null || value === undefined || value === '') return fallback
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toNullableDate(value) {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return date.toISOString()
}

function getFirstDefined(item, keys) {
  for (const key of keys) {
    if (item[key] !== undefined && item[key] !== null && item[key] !== '') {
      return item[key]
    }
  }
  return undefined
}

function walkPayload(node, path = '$', maxDepth = 7, currentDepth = 0, collector = []) {
  if (currentDepth > maxDepth || node === null || node === undefined) {
    return collector
  }

  if (Array.isArray(node)) {
    collector.push({ kind: 'array', value: node, path })
    node.forEach((item, index) => {
      walkPayload(item, `${path}[${index}]`, maxDepth, currentDepth + 1, collector)
    })
    return collector
  }

  if (typeof node === 'object') {
    collector.push({ kind: 'object', value: node, path })
    for (const [key, value] of Object.entries(node)) {
      walkPayload(value, `${path}.${key}`, maxDepth, currentDepth + 1, collector)
    }
  }

  return collector
}

function normalizeShopAction(raw, pathHint = '') {
  if (!raw || typeof raw !== 'object') return null

  const sourceActionID = String(getFirstDefined(raw, [
    'source_action_id',
    'sourceActionId',
    'action_id',
    'actionId',
    'promotion_id',
    'promotionId',
    'campaign_id',
    'campaignId',
    'id',
    'uuid',
  ]) || '').trim()
  if (!sourceActionID) return null

  const title = String(getFirstDefined(raw, [
    'title',
    'name',
    'action_name',
    'actionName',
    'promotion_name',
    'promotionName',
    'campaign_name',
    'campaignName',
    'display_name',
    'displayName',
  ]) || '').trim()

  const keys = Object.keys(raw).map((key) => key.toLowerCase())
  const joinedHint = `${pathHint} ${title}`.toLowerCase()
  const hasPromoHint = keys.some((key) => key.includes('action') || key.includes('promo') || key.includes('campaign') || key.includes('discount')) ||
    joinedHint.includes('action') || joinedHint.includes('promo') || joinedHint.includes('campaign') || joinedHint.includes('акци')

  if (!hasPromoHint && keys.includes('price') && keys.includes('sku')) {
    return null
  }

  return {
    source_action_id: sourceActionID,
    title: title || `Shop Promo ${sourceActionID}`,
    action_type: String(getFirstDefined(raw, [
      'action_type',
      'actionType',
      'type',
      'promotion_type',
      'promotionType',
      'campaign_type',
      'campaignType',
    ]) || 'SHOP_PRIVATE_PROMO'),
    participating_products_count: toNumber(getFirstDefined(raw, [
      'participating_products_count',
      'participatingCount',
      'products_count',
      'product_count',
      'items_count',
      'joined_products_count',
    ]), 0),
    potential_products_count: toNumber(getFirstDefined(raw, [
      'potential_products_count',
      'potentialCount',
      'available_products_count',
      'availableCount',
      'total_products_count',
      'all_products_count',
    ]), 0),
    date_start: toNullableDate(getFirstDefined(raw, [
      'date_start',
      'start_at',
      'startAt',
      'starts_at',
      'startsAt',
      'from',
    ])),
    date_end: toNullableDate(getFirstDefined(raw, [
      'date_end',
      'end_at',
      'endAt',
      'ends_at',
      'endsAt',
      'to',
    ])),
  }
}

function normalizeActionProduct(raw, pathHint = '') {
  if (!raw || typeof raw !== 'object') return null

  const sourceSKU = String(getFirstDefined(raw, [
    'source_sku',
    'sourceSku',
    'offer_id',
    'offerId',
    'vendor_code',
    'vendorCode',
    'sku',
    'sku_id',
    'skuId',
    'item_code',
    'id',
  ]) || '').trim()
  if (!sourceSKU) return null

  const keys = Object.keys(raw).map((key) => key.toLowerCase())
  const joinedHint = `${pathHint} ${keys.join(' ')}`.toLowerCase()
  const hasProductHint = joinedHint.includes('product') || joinedHint.includes('offer') || joinedHint.includes('sku') || joinedHint.includes('item')
  if (!hasProductHint) return null

  return {
    source_sku: sourceSKU,
    ozon_product_id: toNumber(getFirstDefined(raw, [
      'ozon_product_id',
      'ozonProductId',
      'product_id',
      'productId',
      'id',
    ]), 0),
    name: String(getFirstDefined(raw, [
      'name',
      'title',
      'product_name',
      'productName',
      'offer_name',
      'offerName',
      'source_sku',
      'sourceSku',
    ]) || sourceSKU),
    price: toNumber(getFirstDefined(raw, [
      'price',
      'base_price',
      'basePrice',
      'old_price',
      'oldPrice',
      'original_price',
      'originalPrice',
    ]), 0),
    action_price: toNumber(getFirstDefined(raw, [
      'action_price',
      'actionPrice',
      'promo_price',
      'promoPrice',
      'discount_price',
      'discountPrice',
      'price',
    ]), 0),
    stock: toNumber(getFirstDefined(raw, [
      'stock',
      'fbo_stock',
      'fbs_stock',
      'quantity',
      'qty',
      'available',
    ]), 0),
    status: String(getFirstDefined(raw, [
      'status',
      'state',
      'activity_status',
      'activityStatus',
    ]) || 'active'),
  }
}

function uniqueBy(items, keyGetter) {
  const visited = new Set()
  const result = []
  for (const item of items || []) {
    const key = keyGetter(item)
    if (!key || visited.has(key)) continue
    visited.add(key)
    result.push(item)
  }
  return result
}

function extractShopActionsFromPayload(payload, pathHint = '') {
  const walked = walkPayload(payload, pathHint)
  const candidates = []
  for (const entry of walked) {
    if (entry.kind !== 'array') continue
    for (const item of entry.value) {
      const normalized = normalizeShopAction(item, entry.path)
      if (normalized) {
        candidates.push(normalized)
      }
    }
  }
  return uniqueBy(candidates, (item) => item.source_action_id)
}

function extractActionProductsFromPayload(payload, pathHint = '') {
  const walked = walkPayload(payload, pathHint)
  const candidates = []
  for (const entry of walked) {
    if (entry.kind !== 'array') continue
    for (const item of entry.value) {
      const normalized = normalizeActionProduct(item, entry.path)
      if (normalized) {
        candidates.push(normalized)
      }
    }
  }
  return uniqueBy(candidates, (item) => `${item.source_sku}:${item.ozon_product_id}`)
}

function looksLikeActionUrl(url) {
  const lower = String(url || '').toLowerCase()
  const hasApi = lower.includes('/api/')
  const hasPromo = lower.includes('promo') || lower.includes('action') || lower.includes('campaign') || lower.includes('marketing')
  return hasApi && hasPromo
}

function looksLikeProductUrl(url) {
  const lower = String(url || '').toLowerCase()
  const hasApi = lower.includes('/api/')
  const hasPromo = lower.includes('promo') || lower.includes('action') || lower.includes('campaign')
  const hasProduct = lower.includes('product') || lower.includes('offer') || lower.includes('item') || lower.includes('sku')
  return hasApi && hasPromo && hasProduct
}

async function collectJsonResponses(page, matcher, trigger, waitMs = 5000) {
  const captured = []
  const onResponse = async (response) => {
    try {
      const request = response.request()
      const resourceType = request.resourceType()
      if (resourceType !== 'fetch' && resourceType !== 'xhr') {
        return
      }
      const url = response.url()
      if (!matcher(url)) {
        return
      }
      const contentType = (response.headers()['content-type'] || '').toLowerCase()
      if (!contentType.includes('json')) {
        return
      }
      const body = await response.json()
      captured.push({
        url,
        status: response.status(),
        body,
      })
    } catch {
      // ignore invalid json responses
    }
  }

  page.on('response', onResponse)
  try {
    await trigger()
    await page.waitForTimeout(waitMs)
  } finally {
    page.off('response', onResponse)
  }

  return captured
}

async function ensureSellerPageReady(page, url) {
  await page.goto(url, { waitUntil: 'domcontentloaded' })
  const currentUrl = page.url().toLowerCase()
  if (currentUrl.includes('login') || currentUrl.includes('auth')) {
    throw new Error('agent browser is not logged in to Ozon Seller')
  }
}

async function fallbackFetchShopActions(page) {
  return page.evaluate(async () => {
    const tryFetch = async (url, method = 'GET', body = null) => {
      try {
        const response = await fetch(url, {
          method,
          credentials: 'include',
          headers: body ? { 'Content-Type': 'application/json' } : undefined,
          body: body ? JSON.stringify(body) : undefined,
        })
        if (!response.ok) return null
        return await response.json()
      } catch {
        return null
      }
    }

    const endpoints = [
      { url: '/api/seller-actions/list' },
      { url: '/api/promotions/actions/list' },
      { url: '/api/actions/list' },
      { url: '/api/v1/promotions/list' },
      { url: '/api/v2/promotions/list' },
      { url: '/api/campaigns/list' },
    ]

    const payloads = []
    for (const endpoint of endpoints) {
      const data = await tryFetch(endpoint.url)
      if (data) {
        payloads.push({ endpoint: endpoint.url, data })
      }
    }

    return payloads
  })
}

async function fallbackFetchActionProducts(page, sourceActionID) {
  return page.evaluate(async ({ actionId }) => {
    const tryFetch = async (url, body) => {
      try {
        const response = await fetch(url, {
          method: body ? 'POST' : 'GET',
          credentials: 'include',
          headers: body ? { 'Content-Type': 'application/json' } : undefined,
          body: body ? JSON.stringify(body) : undefined,
        })
        if (!response.ok) return null
        return await response.json()
      } catch {
        return null
      }
    }

    const endpoints = [
      { url: '/api/seller-actions/products', body: { action_id: actionId, limit: 500, offset: 0 } },
      { url: '/api/promotions/actions/products', body: { action_id: actionId, limit: 500, offset: 0 } },
      { url: '/api/actions/products', body: { action_id: actionId, limit: 500, offset: 0 } },
      { url: '/api/v1/promotions/products', body: { action_id: actionId, limit: 500, offset: 0 } },
      { url: '/api/v2/promotions/products', body: { action_id: actionId, limit: 500, offset: 0 } },
    ]

    const payloads = []
    for (const endpoint of endpoints) {
      const data = await tryFetch(endpoint.url, endpoint.body)
      if (data) {
        payloads.push({ endpoint: endpoint.url, data })
      }
    }

    return payloads
  }, { actionId: sourceActionID })
}

async function fetchShopActionsSnapshot(browserContext) {
  const page = await browserContext.newPage()
  try {
    const sellerBaseUrl = process.env.OZON_SELLER_BASE_URL || 'https://seller.ozon.ru'
    const promotionsUrl = `${sellerBaseUrl}/app/promotions`

    const responsePackets = await collectJsonResponses(
      page,
      looksLikeActionUrl,
      async () => {
        await ensureSellerPageReady(page, promotionsUrl)
        await page.waitForTimeout(2500)
      },
      4500,
    )

    let actions = []
    for (const packet of responsePackets) {
      actions = actions.concat(extractShopActionsFromPayload(packet.body, packet.url))
    }
    actions = uniqueBy(actions, (item) => item.source_action_id)

    let fallbackPayloads = []
    if (actions.length === 0) {
      fallbackPayloads = await fallbackFetchShopActions(page)
      for (const payload of fallbackPayloads) {
        actions = actions.concat(extractShopActionsFromPayload(payload.data, payload.endpoint))
      }
      actions = uniqueBy(actions, (item) => item.source_action_id)
    }

    return {
      actions,
      debug: {
        current_url: page.url(),
        captured_action_responses: responsePackets.length,
        captured_action_urls: uniqStrings(responsePackets.map((item) => item.url)).slice(0, 30),
        fallback_payload_count: fallbackPayloads.length,
      },
      error: actions.length === 0 ? 'no shop action data captured from seller page responses' : '',
    }
  } finally {
    await page.close()
  }
}

async function fetchActionProductsSnapshot(browserContext, sourceActionID) {
  const page = await browserContext.newPage()
  try {
    const sellerBaseUrl = process.env.OZON_SELLER_BASE_URL || 'https://seller.ozon.ru'
    const promotionsUrl = `${sellerBaseUrl}/app/promotions`

    const responsePackets = await collectJsonResponses(
      page,
      looksLikeProductUrl,
      async () => {
        await ensureSellerPageReady(page, promotionsUrl)
        await page.waitForTimeout(2500)
      },
      4500,
    )

    let items = []
    const actionHint = String(sourceActionID || '').trim()
    for (const packet of responsePackets) {
      if (actionHint && !packet.url.includes(actionHint)) {
        const packetText = JSON.stringify(packet.body)
        if (!packetText.includes(actionHint)) {
          continue
        }
      }
      items = items.concat(extractActionProductsFromPayload(packet.body, packet.url))
    }
    items = uniqueBy(items, (item) => `${item.source_sku}:${item.ozon_product_id}`)

    let fallbackPayloads = []
    if (items.length === 0) {
      fallbackPayloads = await fallbackFetchActionProducts(page, sourceActionID)
      for (const payload of fallbackPayloads) {
        items = items.concat(extractActionProductsFromPayload(payload.data, payload.endpoint))
      }
      items = uniqueBy(items, (item) => `${item.source_sku}:${item.ozon_product_id}`)
    }

    return {
      items,
      debug: {
        source_action_id: sourceActionID,
        current_url: page.url(),
        captured_product_responses: responsePackets.length,
        captured_product_urls: uniqStrings(responsePackets.map((item) => item.url)).slice(0, 30),
        fallback_payload_count: fallbackPayloads.length,
      },
      error: items.length === 0 ? 'no action products data captured from seller page responses' : '',
    }
  } finally {
    await page.close()
  }
}

function normalizeSku(value) {
  return String(value || '').trim()
}

function findCandidateBySKU(candidates, sourceSKU) {
  const needle = normalizeSku(sourceSKU)
  if (!needle) return null

  return (candidates || []).find((item) => {
    const productID = normalizeSku(item.product_id || item.id)
    const offerID = normalizeSku(item.offer_id || item.offerID)
    const skus = Array.isArray(item.skus) ? item.skus.map((sku) => normalizeSku(sku)) : []
    return productID === needle || offerID.includes(needle) || skus.includes(needle)
  }) || null
}

function makeResult(sourceSKU, operation, success, errorMessage = '') {
  if (operation === 'declare') {
    return {
      source_sku: sourceSKU,
      overall_status: success ? 'success' : 'failed',
      step_exit_status: 'skipped',
      step_reprice_status: 'skipped',
      step_readd_status: success ? 'success' : 'failed',
      step_exit_error: '',
      step_reprice_error: '',
      step_readd_error: success ? '' : errorMessage,
    }
  }

  return {
    source_sku: sourceSKU,
    overall_status: success ? 'success' : 'failed',
    step_exit_status: success ? 'success' : 'failed',
    step_reprice_status: 'skipped',
    step_readd_status: 'skipped',
    step_exit_error: success ? '' : errorMessage,
    step_reprice_error: '',
    step_readd_error: '',
  }
}

async function executeActionOperation(page, sourceActionID, sourceSKUs, operation) {
  const candidates = await fetchCandidates(page, sourceActionID)
  const matched = []
  const errorsBySKU = new Map()

  for (const sku of sourceSKUs) {
    const candidate = findCandidateBySKU(candidates, sku)
    if (!candidate) {
      errorsBySKU.set(sku, '未找到活动候选商品')
      continue
    }
    matched.push({ sourceSKU: sku, candidate })
  }

  if (matched.length === 0) {
    return errorsBySKU
  }

  try {
    if (operation === 'declare') {
      await activateProducts(page, sourceActionID, matched.map((entry) => entry.candidate))
    } else {
      const skusToRemove = matched.map((entry) => {
        if (Array.isArray(entry.candidate.skus) && entry.candidate.skus.length > 0) {
          return normalizeSku(entry.candidate.skus[0])
        }
        return normalizeSku(entry.sourceSKU)
      })
      await deactivateProducts(page, sourceActionID, skusToRemove)
    }
  } catch (error) {
    const message = error?.message || '调用店铺活动接口失败'
    for (const entry of matched) {
      errorsBySKU.set(entry.sourceSKU, message)
    }
  }

  return errorsBySKU
}

async function executeSingleShopActionJob(browserContext, job, operation) {
  const sourceActionID = String(job.meta?.source_action_id || '').trim()
  const sourceSKUs = (job.items || []).map((item) => normalizeSku(item.source_sku)).filter(Boolean)

  if (!sourceActionID) {
    return sourceSKUs.map((sku) => makeResult(sku, operation, false, '任务缺少 source_action_id'))
  }

  const page = await browserContext.newPage()
  try {
    await ensureSellerReady(page)
    const errorsBySKU = await executeActionOperation(page, sourceActionID, sourceSKUs, operation)
    return sourceSKUs.map((sku) => makeResult(sku, operation, !errorsBySKU.has(sku), errorsBySKU.get(sku) || ''))
  } finally {
    await page.close()
  }
}

async function executeUnifiedShopActionsJob(browserContext, job, operation) {
  const actionMeta = Array.isArray(job.meta?.shop_actions) ? job.meta.shop_actions : []
  const sourceActionIDs = actionMeta
    .map((entry) => String(entry?.source_action_id || '').trim())
    .filter(Boolean)
  const sourceSKUs = (job.items || []).map((item) => normalizeSku(item.source_sku)).filter(Boolean)

  if (sourceActionIDs.length === 0) {
    return sourceSKUs.map((sku) => makeResult(sku, operation, false, '任务缺少店铺活动列表'))
  }

  const page = await browserContext.newPage()
  try {
    await ensureSellerReady(page)

    const errorsBySKU = new Map()
    for (const actionID of sourceActionIDs) {
      const actionErrors = await executeActionOperation(page, actionID, sourceSKUs, operation)
      for (const [sku, error] of actionErrors.entries()) {
        const prev = errorsBySKU.get(sku)
        errorsBySKU.set(sku, prev ? `${prev}; ${error}` : error)
      }
    }

    const results = sourceSKUs.map((sku) => makeResult(sku, operation, !errorsBySKU.has(sku), errorsBySKU.get(sku) || ''))
    results.__meta = {
      operation,
      source_action_ids: sourceActionIDs,
      failed_items: results.filter((item) => item.overall_status === 'failed').length,
    }
    return results
  } finally {
    await page.close()
  }
}

module.exports = {
  createPlaywrightExecutor,
}
