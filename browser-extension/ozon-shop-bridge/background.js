const SELLER_BASE_URL = 'https://seller.ozon.ru'
const POLL_ALARM = 'ozon_manager_extension_poll'
const AUTH_SYNC_SCRIPT_ID = 'ozon_manager_auth_sync_dynamic'
const DEFAULT_POLL_INTERVAL_MS = 5000

const DEFAULT_STATE = {
  enabled: true,
  apiBaseUrl: 'http://127.0.0.1:8080',
  adminOrigin: '',
  authToken: '',
  shopId: null,
  extensionId: '',
  workerTabId: null,
  pollIntervalMs: DEFAULT_POLL_INTERVAL_MS,
  lastRunAt: '',
  lastError: '',
}

let pollInFlight = false

chrome.runtime.onInstalled.addListener(async () => {
  const state = await initializeState()
  await ensureAuthSyncContentScript(state, false)
  await ensurePollingAlarm()
})

chrome.runtime.onStartup.addListener(async () => {
  const state = await initializeState()
  await ensureAuthSyncContentScript(state, false)
  await ensurePollingAlarm()
})

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name !== POLL_ALARM) return
  await pollOnce()
})

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  const type = message?.type
  if (type === 'OZON_MANAGER_AUTH_SYNC') {
    handleAuthSync(message?.payload)
      .then(() => sendResponse({ ok: true }))
      .catch((error) => sendResponse({ ok: false, error: error?.message || String(error) }))
    return true
  }

  if (type === 'OZON_MANAGER_GET_STATE') {
    readState()
      .then((state) => sendResponse(state))
      .catch((error) => sendResponse({ ...DEFAULT_STATE, lastError: error?.message || String(error) }))
    return true
  }

  if (type === 'OZON_MANAGER_SET_CONFIG') {
    saveStatePatch(message?.payload || {})
      .then(async () => {
        const state = await readState()
        await ensureAuthSyncContentScript(state, true)
        await ensurePollingAlarm()
        await pollOnce()
        sendResponse({ ok: true, state })
      })
      .catch((error) => sendResponse({ ok: false, error: error?.message || String(error) }))
    return true
  }

  return false
})

async function initializeState() {
  const current = await chrome.storage.local.get(Object.keys(DEFAULT_STATE))
  const patch = {}
  for (const [key, value] of Object.entries(DEFAULT_STATE)) {
    if (current[key] === undefined) patch[key] = value
  }
  if (!current.extensionId) {
    patch.extensionId = createExtensionID()
  }
  if (Object.keys(patch).length > 0) {
    await chrome.storage.local.set(patch)
  }
  return readState()
}

async function readState() {
  const state = await chrome.storage.local.get(Object.keys(DEFAULT_STATE))
  return { ...DEFAULT_STATE, ...state }
}

async function saveStatePatch(patch) {
  const next = {}
  if (typeof patch.enabled === 'boolean') next.enabled = patch.enabled
  if (typeof patch.apiBaseUrl === 'string') next.apiBaseUrl = patch.apiBaseUrl.trim()
  if (typeof patch.adminOrigin === 'string') next.adminOrigin = normalizeHTTPOrigin(patch.adminOrigin)
  if (typeof patch.authToken === 'string') next.authToken = patch.authToken.trim()
  if (patch.shopId !== undefined) {
    const shopID = Number(patch.shopId || 0)
    next.shopId = Number.isFinite(shopID) && shopID > 0 ? shopID : null
  }
  if (patch.workerTabId !== undefined) {
    const tabID = Number(patch.workerTabId || 0)
    next.workerTabId = Number.isFinite(tabID) && tabID > 0 ? tabID : null
  }
  if (patch.pollIntervalMs !== undefined) {
    const poll = Number(patch.pollIntervalMs || 0)
    next.pollIntervalMs = Number.isFinite(poll) && poll >= 3000 ? poll : DEFAULT_POLL_INTERVAL_MS
  }
  if (typeof patch.lastRunAt === 'string') next.lastRunAt = patch.lastRunAt
  if (typeof patch.lastError === 'string') next.lastError = patch.lastError
  if (Object.keys(next).length > 0) {
    await chrome.storage.local.set(next)
  }
}

async function handleAuthSync(payload) {
  if (!payload || typeof payload !== 'object') return

  const patch = {}
  if (typeof payload.token === 'string' && payload.token.trim()) {
    patch.authToken = payload.token.trim()
  }
  const shopID = Number(payload.shop_id || 0)
  if (Number.isFinite(shopID) && shopID > 0) {
    patch.shopId = shopID
  }

  if (Object.keys(patch).length > 0) {
    await chrome.storage.local.set(patch)
  }
}

async function ensurePollingAlarm() {
  const state = await readState()
  const periodInMinutes = Math.max(0.5, Number(state.pollIntervalMs || DEFAULT_POLL_INTERVAL_MS) / 60000)
  await chrome.alarms.clear(POLL_ALARM)
  await chrome.alarms.create(POLL_ALARM, {
    periodInMinutes,
    delayInMinutes: 0.1,
  })
}

async function pollOnce() {
  if (pollInFlight) return
  pollInFlight = true
  try {
    const state = await readState()
    if (!state.enabled) return
    if (!state.authToken || !state.shopId || !state.apiBaseUrl) return

    await registerExtension(state)

    const pollData = await apiPost(
      state.apiBaseUrl,
      state.authToken,
      '/api/v1/extension/poll',
      {
        shop_id: state.shopId,
        extension_id: state.extensionId,
      },
    )

    const job = pollData?.job
    if (!job) {
      await saveStatePatch({
        lastRunAt: new Date().toISOString(),
        lastError: '',
      })
      return
    }

    const run = await executeJob(job, state)
    await apiPost(
      state.apiBaseUrl,
      state.authToken,
      '/api/v1/extension/report',
      {
        shop_id: state.shopId,
        extension_id: state.extensionId,
        job_id: job.job_id,
        status: run.status,
        results: run.results,
        meta: run.meta || {},
      },
    )

    await saveStatePatch({
      lastRunAt: new Date().toISOString(),
      lastError: '',
    })
  } catch (error) {
    await saveStatePatch({
      lastRunAt: new Date().toISOString(),
      lastError: error?.message || String(error),
    })
  } finally {
    pollInFlight = false
  }
}

async function registerExtension(state) {
  await apiPost(
    state.apiBaseUrl,
    state.authToken,
    '/api/v1/extension/register',
    {
      shop_id: state.shopId,
      extension_id: state.extensionId,
      name: 'Chrome Extension',
      version: chrome.runtime.getManifest().version,
    },
  )
}

async function apiPost(baseUrl, token, path, payload) {
  const endpoint = joinURL(baseUrl, path)
  const response = await fetch(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload || {}),
  })

  let body = null
  try {
    body = await response.json()
  } catch {
    // no-op
  }

  if (!response.ok) {
    const message = body?.message || `${response.status} ${response.statusText}`
    throw new Error(`API 请求失败: ${message}`)
  }
  if (!body || body.code !== 200) {
    throw new Error(body?.message || 'API 响应异常')
  }
  return body.data || {}
}

function joinURL(base, path) {
  const left = String(base || '').replace(/\/+$/, '')
  const right = String(path || '').replace(/^\/+/, '')
  return `${left}/${right}`
}

function normalizeHTTPOrigin(raw) {
  const input = String(raw || '').trim()
  if (!input) return ''
  try {
    const url = new URL(input)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    return url.origin
  } catch {
    return ''
  }
}

function isLocalOrigin(origin) {
  if (!origin) return false
  try {
    const url = new URL(origin)
    return url.hostname === 'localhost' || url.hostname === '127.0.0.1'
  } catch {
    return false
  }
}

function buildAuthSyncOrigins(state) {
  const origins = new Set([
    'http://localhost',
    'https://localhost',
    'http://127.0.0.1',
  ])

  const backendOrigin = normalizeHTTPOrigin(state?.apiBaseUrl || '')
  if (backendOrigin) {
    origins.add(backendOrigin)
  }

  const adminOrigin = normalizeHTTPOrigin(state?.adminOrigin || '')
  if (adminOrigin) {
    origins.add(adminOrigin)
  }

  return Array.from(origins)
}

async function ensureAuthSyncContentScript(state, requestPermission) {
  const allOrigins = buildAuthSyncOrigins(state)
  const dynamicOrigins = allOrigins.filter((origin) => !isLocalOrigin(origin))
  const dynamicMatches = dynamicOrigins.map((origin) => `${origin}/*`)

  if (dynamicMatches.length === 0) {
    try {
      await chrome.scripting.unregisterContentScripts({ ids: [AUTH_SYNC_SCRIPT_ID] })
    } catch {
      // ignore if not registered
    }
    return
  }

  const hasPermission = await chrome.permissions.contains({ origins: dynamicMatches })
  if (!hasPermission) {
    if (!requestPermission) {
      return
    }
    const granted = await chrome.permissions.request({ origins: dynamicMatches })
    if (!granted) {
      throw new Error('未授予管理端域名权限，无法自动同步 token/shop_id')
    }
  }

  try {
    await chrome.scripting.unregisterContentScripts({ ids: [AUTH_SYNC_SCRIPT_ID] })
  } catch {
    // ignore if not registered
  }

  await chrome.scripting.registerContentScripts([
    {
      id: AUTH_SYNC_SCRIPT_ID,
      matches: dynamicMatches,
      js: ['content-auth-sync.js'],
      runAt: 'document_idle',
      persistAcrossSessions: true,
    },
  ])
}

async function executeJob(job, state) {
  const tab = await ensureSellerLoggedInTab()
  try {
    switch (job.job_type) {
      case 'sync_shop_actions':
        return await executeSyncShopActions(tab.id)
      case 'sync_action_products':
        return await executeSyncActionProducts(tab.id, job)
      case 'shop_action_declare':
        return await executeSingleShopAction(tab.id, job, 'declare')
      case 'shop_action_remove':
        return await executeSingleShopAction(tab.id, job, 'remove')
      case 'promo_unified_enroll':
        return await executeUnifiedShopActions(tab.id, job, 'declare')
      case 'promo_unified_remove':
        return await executeUnifiedShopActions(tab.id, job, 'remove')
      case 'remove_reprice_readd':
        return await executeRemoveRepriceReadd(tab.id, job, state)
      default:
        return {
          status: 'failed',
          results: buildJobFailureResults(job, `插件不支持该任务类型: ${job.job_type}`),
          meta: {},
        }
    }
  } catch (error) {
    return {
      status: 'failed',
      results: buildJobFailureResults(job, error?.message || String(error)),
      meta: {},
    }
  }
}

async function ensureSellerLoggedInTab() {
  const tab = await getOrCreateSellerTab(false)
  await ensureSellerPage(tab.id, `${SELLER_BASE_URL}/app/dashboard`)

  const current = await chrome.tabs.get(tab.id)
  if (!isAuthURL(current.url || '')) {
    return current
  }

  const loginTab = await getOrCreateSellerTab(true)
  await chrome.tabs.update(loginTab.id, {
    url: `${SELLER_BASE_URL}/login`,
    active: true,
  })

  const deadline = Date.now() + 5 * 60 * 1000
  while (Date.now() < deadline) {
    await sleep(3000)
    const latest = await chrome.tabs.get(loginTab.id)
    if (!isAuthURL(latest.url || '')) {
      await ensureSellerPage(latest.id, `${SELLER_BASE_URL}/app/dashboard`)
      return await chrome.tabs.get(latest.id)
    }
  }

  throw new Error('AUTH_REQUIRED: 未检测到 Ozon Seller 登录，请先完成登录')
}

async function getOrCreateSellerTab(active) {
  const state = await readState()
  const storedTabID = Number(state.workerTabId || 0)
  if (storedTabID > 0) {
    try {
      const tab = await chrome.tabs.get(storedTabID)
      if (String(tab.url || '').startsWith(SELLER_BASE_URL)) {
        if (active) {
          await chrome.tabs.update(tab.id, { active: true })
        }
        return tab
      }
    } catch {
      // worker tab was closed, create a new one.
    }
  }

  const tab = await chrome.tabs.create({
    url: `${SELLER_BASE_URL}/app/dashboard`,
    active: Boolean(active),
  })
  await saveStatePatch({ workerTabId: tab.id })
  return tab
}

async function ensureSellerPage(tabID, url) {
  const current = await chrome.tabs.get(tabID)
  const currentURL = String(current.url || '')
  const shouldNavigate = !currentURL.startsWith(SELLER_BASE_URL) || isAuthURL(currentURL)
  if (shouldNavigate) {
    await chrome.tabs.update(tabID, { url })
    await waitTabLoaded(tabID, 15000)
    await sleep(600)
  }
}

function isAuthURL(url) {
  const lower = String(url || '').toLowerCase()
  return lower.includes('/login') || lower.includes('/auth')
}

function waitTabLoaded(tabID, timeoutMs) {
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      chrome.tabs.onUpdated.removeListener(listener)
      resolve()
    }, timeoutMs)

    function listener(updatedTabID, info) {
      if (updatedTabID !== tabID) return
      if (info.status !== 'complete') return
      clearTimeout(timer)
      chrome.tabs.onUpdated.removeListener(listener)
      resolve()
    }

    chrome.tabs.onUpdated.addListener(listener)
  })
}

async function executeSyncShopActions(tabID) {
  const payloads = await runScript(tabID, scriptFetchShopActionsPayloads, [])
  let actions = []
  for (const packet of payloads || []) {
    actions = actions.concat(extractShopActionsFromPayload(packet?.data || {}, packet?.endpoint || ''))
  }
  actions = uniqueBy(actions, (item) => item.source_action_id)

  const success = actions.length > 0
  const status = success ? 'success' : 'failed'
  return {
    status,
    results: [buildSyncResult('__sync_shop_actions__', success, success ? '' : '未获取到店铺活动数据')],
    meta: {
      actions,
      payload_count: (payloads || []).length,
    },
  }
}

async function executeSyncActionProducts(tabID, job) {
  const sourceActionID = String(job?.meta?.source_action_id || '').trim()
  const payloads = await runScript(tabID, scriptFetchActionProductsPayloads, [sourceActionID])
  let items = []
  for (const packet of payloads || []) {
    items = items.concat(extractActionProductsFromPayload(packet?.data || {}, packet?.endpoint || ''))
  }
  items = uniqueBy(items, (item) => buildActionProductDedupKey(item))

  const success = items.length > 0
  const status = success ? 'success' : 'failed'
  return {
    status,
    results: [buildSyncResult('__sync_action_products__', success, success ? '' : '未获取到活动商品数据')],
    meta: {
      source_action_id: sourceActionID,
      items,
      payload_count: (payloads || []).length,
    },
  }
}

async function executeSingleShopAction(tabID, job, operation) {
  const sourceActionID = String(job?.meta?.source_action_id || '').trim()
  const sourceSKUs = (job?.items || []).map((item) => normalizeSKU(item?.source_sku)).filter(Boolean)
  if (!sourceActionID) {
    return {
      status: 'failed',
      results: sourceSKUs.map((sku) => makeActionResult(sku, operation, false, '任务缺少 source_action_id')),
      meta: {},
    }
  }

  const errorsBySKU = await executeActionOperation(tabID, sourceActionID, sourceSKUs, operation)
  const results = sourceSKUs.map((sku) => makeActionResult(sku, operation, !errorsBySKU[sku], errorsBySKU[sku] || ''))

  return {
    status: summarizeStatus(results),
    results,
    meta: {
      operation,
      source_action_id: sourceActionID,
    },
  }
}

async function executeUnifiedShopActions(tabID, job, operation) {
  const actionMeta = Array.isArray(job?.meta?.shop_actions) ? job.meta.shop_actions : []
  const sourceActionIDs = actionMeta
    .map((item) => String(item?.source_action_id || '').trim())
    .filter(Boolean)

  const sourceSKUs = (job?.items || []).map((item) => normalizeSKU(item?.source_sku)).filter(Boolean)
  if (sourceActionIDs.length === 0) {
    return {
      status: 'failed',
      results: sourceSKUs.map((sku) => makeActionResult(sku, operation, false, '任务缺少店铺活动列表')),
      meta: {},
    }
  }

  const mergedErrors = {}
  for (const actionID of sourceActionIDs) {
    const errors = await executeActionOperation(tabID, actionID, sourceSKUs, operation)
    for (const [sku, message] of Object.entries(errors)) {
      if (!mergedErrors[sku]) {
        mergedErrors[sku] = message
      } else if (!mergedErrors[sku].includes(message)) {
        mergedErrors[sku] = `${mergedErrors[sku]}; ${message}`
      }
    }
  }

  const results = sourceSKUs.map((sku) => makeActionResult(sku, operation, !mergedErrors[sku], mergedErrors[sku] || ''))
  return {
    status: summarizeStatus(results),
    results,
    meta: {
      operation,
      source_action_ids: sourceActionIDs,
      failed_items: results.filter((item) => item.overall_status === 'failed').length,
    },
  }
}

async function executeRemoveRepriceReadd(tabID, job, state) {
  const actionMeta = Array.isArray(job?.meta?.shop_actions) ? job.meta.shop_actions : []
  const sourceActionIDs = actionMeta
    .map((item) => String(item?.source_action_id || '').trim())
    .filter(Boolean)

  const items = Array.isArray(job?.items) ? job.items : []
  if (items.length === 0) {
    return {
      status: 'failed',
      results: [],
      meta: { error: '任务没有可处理商品' },
    }
  }
  if (sourceActionIDs.length === 0) {
    return {
      status: 'failed',
      results: items.map((item) =>
        makeRemoveRepriceReaddResult(normalizeSKU(item?.source_sku), false, false, false, '缺少店铺活动列表'),
      ),
      meta: { error: '任务缺少 shop_actions' },
    }
  }

  const results = []
  for (const item of items) {
    const sourceSKU = normalizeSKU(item?.source_sku)
    const targetPrice = Number(item?.target_price || 0)
    if (!sourceSKU || !Number.isFinite(targetPrice) || targetPrice <= 0) {
      results.push(makeRemoveRepriceReaddResult(sourceSKU || '-', false, false, false, '任务参数异常'))
      continue
    }

    let exitSuccess = true
    let repriceSuccess = false
    let readdSuccess = false
    let exitError = ''
    let repriceError = ''
    let readdError = ''

    for (const actionID of sourceActionIDs) {
      const removeErrors = await executeActionOperation(tabID, actionID, [sourceSKU], 'remove')
      if (removeErrors[sourceSKU]) {
        exitSuccess = false
        exitError = mergeError(exitError, removeErrors[sourceSKU])
      }
    }

    if (exitSuccess) {
      try {
        await repriceByBackend(state, job.shop_id, sourceSKU, targetPrice)
        repriceSuccess = true
      } catch (error) {
        repriceSuccess = false
        repriceError = error?.message || String(error)
      }
    } else {
      repriceError = '退出活动失败，跳过改价'
    }

    if (exitSuccess && repriceSuccess) {
      let allAdded = true
      for (const actionID of sourceActionIDs) {
        const addErrors = await executeActionOperation(tabID, actionID, [sourceSKU], 'declare')
        if (addErrors[sourceSKU]) {
          allAdded = false
          readdError = mergeError(readdError, addErrors[sourceSKU])
        }
      }
      readdSuccess = allAdded
    } else if (!repriceSuccess) {
      readdError = '改价失败，跳过重新报名'
    }

    results.push(makeRemoveRepriceReaddResult(sourceSKU, exitSuccess, repriceSuccess, readdSuccess, '', exitError, repriceError, readdError))
  }

  return {
    status: summarizeStatus(results),
    results,
    meta: {
      source_action_ids: sourceActionIDs,
      failed_items: results.filter((item) => item.overall_status === 'failed').length,
    },
  }
}

async function executeActionOperation(tabID, sourceActionID, sourceSKUs, operation) {
  const candidates = await runScript(tabID, scriptFetchCandidates, [sourceActionID])
  const matched = []
  const errorsBySKU = {}

  for (const sku of sourceSKUs) {
    const candidate = findCandidateBySKU(candidates || [], sku)
    if (!candidate) {
      errorsBySKU[sku] = '未找到活动候选商品'
      continue
    }
    matched.push({ sourceSKU: sku, candidate })
  }

  if (matched.length === 0) {
    return errorsBySKU
  }

  try {
    if (operation === 'declare') {
      await runScript(tabID, scriptActivateProducts, [sourceActionID, matched.map((entry) => entry.candidate)])
    } else {
      const skus = matched.map((entry) => {
        if (Array.isArray(entry.candidate?.skus) && entry.candidate.skus.length > 0) {
          return String(entry.candidate.skus[0] || '').trim()
        }
        return normalizeSKU(entry.sourceSKU)
      })
      await runScript(tabID, scriptDeactivateProducts, [sourceActionID, skus])
    }
  } catch (error) {
    const message = error?.message || '调用店铺活动接口失败'
    for (const entry of matched) {
      errorsBySKU[entry.sourceSKU] = message
    }
  }

  return errorsBySKU
}

async function runScript(tabID, func, args) {
  const output = await chrome.scripting.executeScript({
    target: { tabId: tabID },
    func,
    args,
    world: 'MAIN',
  })
  return output?.[0]?.result
}

function buildJobFailureResults(job, message) {
  if (job?.job_type === 'sync_shop_actions') {
    return [buildSyncResult('__sync_shop_actions__', false, message)]
  }
  if (job?.job_type === 'sync_action_products') {
    return [buildSyncResult('__sync_action_products__', false, message)]
  }
  if (job?.job_type === 'remove_reprice_readd') {
    const sourceSKUs = (job?.items || []).map((item) => normalizeSKU(item?.source_sku)).filter(Boolean)
    return sourceSKUs.map((sku) => makeRemoveRepriceReaddResult(sku, false, false, false, message))
  }
  const operation = job?.job_type === 'shop_action_remove' || job?.job_type === 'promo_unified_remove'
    ? 'remove'
    : 'declare'
  const sourceSKUs = (job?.items || []).map((item) => normalizeSKU(item?.source_sku)).filter(Boolean)
  return sourceSKUs.map((sku) => makeActionResult(sku, operation, false, message))
}

function buildSyncResult(sourceSKU, success, errorMessage) {
  return {
    source_sku: sourceSKU,
    overall_status: success ? 'success' : 'failed',
    step_exit_status: success ? 'success' : 'failed',
    step_reprice_status: success ? 'success' : 'failed',
    step_readd_status: success ? 'success' : 'failed',
    step_exit_error: success ? '' : errorMessage,
    step_reprice_error: success ? '' : errorMessage,
    step_readd_error: success ? '' : errorMessage,
  }
}

function summarizeStatus(results) {
  if (!Array.isArray(results) || results.length === 0) return 'failed'
  let successCount = 0
  let failedCount = 0
  for (const item of results) {
    if (item.overall_status === 'success' || item.overall_status === 'skipped') {
      successCount += 1
    } else {
      failedCount += 1
    }
  }
  if (failedCount === 0) return 'success'
  if (successCount === 0) return 'failed'
  return 'partial_success'
}

function makeActionResult(sourceSKU, operation, success, errorMessage) {
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

function makeRemoveRepriceReaddResult(
  sourceSKU,
  exitSuccess,
  repriceSuccess,
  readdSuccess,
  sharedError = '',
  exitError = '',
  repriceError = '',
  readdError = '',
) {
  const stepExitStatus = exitSuccess ? 'success' : 'failed'
  const stepRepriceStatus = exitSuccess ? (repriceSuccess ? 'success' : 'failed') : 'skipped'
  const stepReaddStatus = exitSuccess && repriceSuccess ? (readdSuccess ? 'success' : 'failed') : 'skipped'
  const overallFailed = stepExitStatus === 'failed' || stepRepriceStatus === 'failed' || stepReaddStatus === 'failed'
  return {
    source_sku: sourceSKU,
    overall_status: overallFailed ? 'failed' : 'success',
    step_exit_status: stepExitStatus,
    step_reprice_status: stepRepriceStatus,
    step_readd_status: stepReaddStatus,
    step_exit_error: exitError || sharedError,
    step_reprice_error: repriceError || sharedError,
    step_readd_error: readdError || sharedError,
  }
}

async function repriceByBackend(state, shopID, sourceSKU, newPrice) {
  if (!state?.apiBaseUrl || !state?.authToken) {
    throw new Error('缺少后端地址或登录 token，无法改价')
  }
  await apiPost(
    state.apiBaseUrl,
    state.authToken,
    '/api/v1/extension/reprice',
    {
      shop_id: shopID,
      source_sku: sourceSKU,
      new_price: Number(newPrice),
    },
  )
}

function mergeError(existing, next) {
  const current = String(existing || '').trim()
  const incoming = String(next || '').trim()
  if (!incoming) return current
  if (!current) return incoming
  if (current.includes(incoming)) return current
  return `${current}; ${incoming}`
}

function normalizeSKU(value) {
  return String(value || '').trim()
}

function findCandidateBySKU(candidates, sourceSKU) {
  const needle = normalizeSKU(sourceSKU)
  if (!needle) return null

  for (const item of candidates || []) {
    const productID = normalizeSKU(item?.product_id || item?.id)
    const offerID = normalizeSKU(item?.offer_id || item?.offerID)
    const skus = Array.isArray(item?.skus) ? item.skus.map((sku) => normalizeSKU(sku)) : []
    if (productID === needle || offerID.includes(needle) || skus.includes(needle)) {
      return item
    }
  }
  return null
}

function uniqueBy(items, keyGetter) {
  const seen = new Set()
  const output = []
  for (const item of items || []) {
    const key = keyGetter(item)
    if (!key || seen.has(key)) continue
    seen.add(key)
    output.push(item)
  }
  return output
}

function toNumber(value, fallback = 0) {
  if (value === null || value === undefined || value === '') return fallback
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPriceNumber(value, fallback = 0) {
  if (value === null || value === undefined || value === '') return fallback
  if (typeof value === 'number' || typeof value === 'string') {
    return toNumber(value, fallback)
  }
  if (typeof value === 'object') {
    const units = toNumber(value.units ?? value.unit ?? 0, 0)
    const nanos = toNumber(value.nanos ?? 0, 0)
    if (Number.isFinite(units) && Number.isFinite(nanos)) {
      return units + nanos / 1e9
    }
  }
  return fallback
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

function getFirstPresent(values) {
  for (const value of values) {
    if (value !== undefined && value !== null && value !== '') {
      return value
    }
  }
  return undefined
}

function pickValue(raw, actionParameters, rawKeys, actionParameterKeys = rawKeys) {
  const direct = getFirstDefined(raw, rawKeys)
  if (direct !== undefined) return direct
  return getFirstDefined(actionParameters, actionParameterKeys)
}

function toNullableNumber(value) {
  if (value === undefined || value === null || value === '') return null
  const parsed = toNumber(value, Number.NaN)
  if (!Number.isFinite(parsed)) return null
  return parsed
}

function toNullableBoolean(value) {
  if (value === undefined || value === null || value === '') return null
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (['true', '1', 'yes', 'y'].includes(normalized)) return true
    if (['false', '0', 'no', 'n'].includes(normalized)) return false
  }
  return null
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

  const actionParameters = getFirstPresent([
    raw.actionParameters,
    raw.action_parameters,
  ])
  const normalizedActionParameters = actionParameters && typeof actionParameters === 'object'
    ? actionParameters
    : {}

  const dateStartValue = getFirstPresent([
    pickValue(raw, normalizedActionParameters, [
      'date_start',
      'dateStart',
      'start_at',
      'startAt',
      'starts_at',
      'startsAt',
      'from',
    ]),
    getFirstDefined(normalizedActionParameters, [
      'date_start',
      'dateStart',
      'start_at',
      'startAt',
      'starts_at',
      'startsAt',
      'from',
    ]),
  ])

  const dateEndValue = getFirstPresent([
    pickValue(raw, normalizedActionParameters, [
      'date_end',
      'dateEnd',
      'end_at',
      'endAt',
      'ends_at',
      'endsAt',
      'to',
    ]),
    getFirstDefined(normalizedActionParameters, [
      'date_end',
      'dateEnd',
      'end_at',
      'endAt',
      'ends_at',
      'endsAt',
      'to',
    ]),
  ])

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
    'action_title',
    'actionTitle',
    'campaign_title',
    'campaignTitle',
  ]) || getFirstDefined(normalizedActionParameters, [
    'title',
    'name',
    'action_name',
    'actionName',
    'campaign_name',
    'campaignName',
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
    ]) || getFirstDefined(normalizedActionParameters, [
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
      'skuCount',
      'products_count',
      'product_count',
      'items_count',
      'joined_products_count',
    ]) || getFirstDefined(normalizedActionParameters, [
      'participating_products_count',
      'participatingCount',
      'skuCount',
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
    ]) || getFirstDefined(normalizedActionParameters, [
      'potential_products_count',
      'potentialCount',
      'available_products_count',
      'availableCount',
      'total_products_count',
      'all_products_count',
    ]), 0),
    date_start: toNullableDate(dateStartValue),
    date_end: toNullableDate(dateEndValue),
    discount_type: String(pickValue(raw, normalizedActionParameters, [
      'discount_type',
      'discountType',
    ], [
      'discount_type',
      'discountType',
    ]) || ''),
    minimal_action_percent: toNullableNumber(pickValue(raw, normalizedActionParameters, [
      'minimal_action_percent',
      'minimalActionPercent',
    ], [
      'minimal_action_percent',
      'minimalActionPercent',
    ])),
    budget_spent: toNullableNumber(pickValue(raw, normalizedActionParameters, [
      'action_budget_spent',
      'actionBudgetSpent',
      'budget_spent',
      'budgetSpent',
    ], [
      'action_budget_spent',
      'actionBudgetSpent',
      'budget_spent',
      'budgetSpent',
    ])),
    currency: String(pickValue(raw, normalizedActionParameters, [
      'currency',
    ], [
      'currency',
    ]) || ''),
    promotion_company_status: String(pickValue(raw, normalizedActionParameters, [
      'promotion_company_status',
      'promotionCompanyStatus',
    ], [
      'promotion_company_status',
      'promotionCompanyStatus',
    ]) || ''),
    is_editable: toNullableBoolean(pickValue(raw, normalizedActionParameters, [
      'is_editable',
      'isEditable',
    ], [
      'is_editable',
      'isEditable',
    ])),
    can_be_updatable: toNullableBoolean(pickValue(raw, normalizedActionParameters, [
      'can_be_updatable',
      'canBeUpdatable',
    ], [
      'can_be_updatable',
      'canBeUpdatable',
    ])),
    is_participated: toNullableBoolean(pickValue(raw, normalizedActionParameters, [
      'is_participated',
      'isParticipated',
    ], [
      'is_participated',
      'isParticipated',
    ])),
    is_turn_on: toNullableBoolean(pickValue(raw, normalizedActionParameters, [
      'is_turn_on',
      'isTurnOn',
    ], [
      'is_turn_on',
      'isTurnOn',
    ])),
    is_repricer_available: toNullableBoolean(pickValue(raw, normalizedActionParameters, [
      'is_repricer_available',
      'isRepricerAvailable',
    ], [
      'is_repricer_available',
      'isRepricerAvailable',
    ])),
    highlight_url: String(pickValue(raw, normalizedActionParameters, [
      'highlight_url',
      'highlightUrl',
      'url',
    ], [
      'highlight_url',
      'highlightUrl',
      'url',
    ]) || ''),
    created_at: toNullableDate(pickValue(raw, normalizedActionParameters, [
      'created_at',
      'createdAt',
    ], [
      'created_at',
      'createdAt',
    ])),
    action_status: String(pickValue(raw, normalizedActionParameters, [
      'action_status',
      'actionStatus',
      'status',
    ], [
      'action_status',
      'actionStatus',
      'status',
    ]) || ''),
  }
}

function normalizeActionProduct(raw, pathHint = '') {
  if (!raw || typeof raw !== 'object') return null

  const offerID = normalizeSKU(getFirstDefined(raw, [
    'offer_id',
    'offerID',
    'offerId',
  ]))
  const skus = Array.isArray(raw?.skus) ? raw.skus.map((sku) => normalizeSKU(sku)).filter(Boolean) : []
  const platformSKU = skus.length > 0 ? skus[0] : ''

  const sourceSKU = String(getFirstDefined(raw, [
    'source_sku',
    'sourceSku',
    'offer_id',
    'offerID',
    'offerId',
    'ozonSku',
    'vendor_code',
    'vendorCode',
    'sku',
    'sku_id',
    'skuId',
    'item_code',
    'id',
  ]) || offerID || platformSKU).trim()
  if (!sourceSKU) return null

  const keys = Object.keys(raw).map((key) => key.toLowerCase())
  const joinedHint = `${pathHint} ${keys.join(' ')}`.toLowerCase()
  const hasProductHint = joinedHint.includes('product') || joinedHint.includes('offer') || joinedHint.includes('sku') || joinedHint.includes('item')
  if (!hasProductHint) return null

  const priceNode = raw?.price || {}
  const basePriceNode = raw?.base_price || raw?.basePrice || {}
  const actionPriceNode = raw?.action_price || raw?.actionPrice || {}
  const currency = String(
    getFirstDefined(raw, ['currency']) ||
    getFirstDefined(priceNode, ['currencyCode', 'currency']) ||
    getFirstDefined(basePriceNode, ['currencyCode', 'currency']) ||
    getFirstDefined(actionPriceNode, ['currencyCode', 'currency']) ||
    '',
  ).trim()

  const nameOrigin = String(getFirstDefined(raw, [
    'name',
    'title',
    'product_name',
    'productName',
    'offer_name',
    'offerName',
    'source_sku',
    'sourceSku',
    'offerID',
  ]) || sourceSKU).trim()
  const categoryName = String(getFirstDefined(raw, [
    'item_type',
    'itemType',
    'category_name',
    'categoryName',
  ]) || '').trim()

  const statusText = String(getFirstDefined(raw, [
    'status',
    'state',
    'activity_status',
    'activityStatus',
  ]) || '').trim()
  const isActive = toNullableBoolean(getFirstDefined(raw, ['is_active', 'isActive']))
  let status = statusText || 'active'
  if (!statusText && isActive !== null) {
    status = isActive ? 'active' : 'inactive'
  }

  return {
    source_sku: sourceSKU,
    offer_id: offerID || sourceSKU,
    platform_sku: platformSKU,
    ozon_product_id: toNumber(getFirstDefined(raw, [
      'ozon_product_id',
      'ozonProductId',
      'product_id',
      'productId',
      'id',
    ]), 0),
    name: nameOrigin,
    name_cn: String(getFirstDefined(raw, [
      'name_cn',
      'nameCn',
    ]) || categoryName || nameOrigin || sourceSKU).trim(),
    name_origin: nameOrigin,
    thumbnail_url: String(getFirstDefined(raw, [
      'thumbnail',
      'thumb',
      'image',
      'image_url',
      'imageUrl',
      'picture',
    ]) || '').trim(),
    category_name: categoryName,
    currency,
    base_price: toPriceNumber(getFirstDefined(raw, [
      'base_price',
      'basePrice',
      'old_price',
      'oldPrice',
      'original_price',
      'originalPrice',
      'price',
    ]), 0),
    price: toPriceNumber(getFirstDefined(raw, [
      'price',
      'base_price',
      'basePrice',
      'old_price',
      'oldPrice',
      'original_price',
      'originalPrice',
    ]), 0),
    action_price: toPriceNumber(getFirstDefined(raw, [
      'action_price',
      'actionPrice',
      'promo_price',
      'promoPrice',
      'discount_price',
      'discountPrice',
      'price',
    ]), 0),
    marketplace_price: toPriceNumber(getFirstDefined(raw, [
      'marketplace_seller_price',
      'marketplaceSellerPrice',
    ]), 0),
    min_seller_price: toPriceNumber(getFirstDefined(raw, [
      'min_seller_price',
      'minSellerPrice',
    ]), 0),
    max_action_price: toPriceNumber(getFirstDefined(raw, [
      'max_action_price',
      'maxActionPrice',
    ]), 0),
    discount_percent: toNumber(getFirstDefined(raw, [
      'discount_percent',
      'discountPercent',
    ]), 0),
    stock: toNumber(getFirstDefined(raw, [
      'stock',
      'seller_stock',
      'sellerStock',
      'fbo_stock',
      'fbs_stock',
      'quantity',
      'qty',
      'available',
    ]), 0),
    seller_stock: toNumber(getFirstDefined(raw, [
      'seller_stock',
      'sellerStock',
      'stock',
    ]), 0),
    ozon_stock: toNumber(getFirstDefined(raw, [
      'ozon_stock',
      'ozonStock',
    ]), 0),
    status,
  }
}

function buildActionProductDedupKey(item) {
  const offerID = normalizeSKU(item?.offer_id)
  const sourceSKU = normalizeSKU(item?.source_sku)
  const ozonProductID = normalizeSKU(item?.ozon_product_id)
  if (offerID && ozonProductID) {
    return `${offerID}:${ozonProductID}`
  }
  if (sourceSKU && ozonProductID) {
    return `${sourceSKU}:${ozonProductID}`
  }
  return offerID || sourceSKU || ozonProductID
}

function extractShopActionsFromPayload(payload, pathHint = '') {
  const walked = walkPayload(payload, pathHint)
  const candidates = []
  for (const entry of walked) {
    if (entry.kind !== 'array') continue
    for (const item of entry.value) {
      const normalized = normalizeShopAction(item, entry.path)
      if (normalized) candidates.push(normalized)
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
      if (normalized) candidates.push(normalized)
    }
  }
  return uniqueBy(candidates, (item) => buildActionProductDedupKey(item))
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function createExtensionID() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `ext_${Date.now()}_${Math.floor(Math.random() * 100000)}`
}

// ===== Functions executed inside seller tab =====

async function scriptFetchShopActionsPayloads() {
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const companyId = readCookie('sc_company_id')
  const language = readCookie('x-o3-language') || 'zh-Hans'
  const requestHeaders = { accept: 'application/json' }
  if (companyId) requestHeaders['x-o3-company-id'] = companyId
  if (language) requestHeaders['x-o3-language'] = language

  const packets = []

  // New endpoint used by current Seller activity list page.
  try {
    const limit = 50
    let offset = 0
    for (let page = 0; page < 20; page += 1) {
      const endpoint = `/api/site/marketplace-seller-actions/v2/action/list?offset=${offset}&limit=${limit}&skipVouchersCount=true`
      const response = await fetch(endpoint, {
        method: 'GET',
        credentials: 'include',
        headers: requestHeaders,
      })
      if (!response.ok) break
      const contentType = String(response.headers.get('content-type') || '').toLowerCase()
      if (!contentType.includes('json')) break

      const data = await response.json()
      packets.push({ endpoint, data })

      const actions = Array.isArray(data?.actions) ? data.actions : []
      const total = toNumber(data?.total, 0)
      if (actions.length === 0) break

      offset += actions.length
      if ((total > 0 && offset >= total) || actions.length < limit) {
        break
      }
    }
  } catch {
    // fall through to legacy endpoints
  }

  if (packets.length > 0) {
    return packets
  }

  const endpoints = [
    '/api/seller-actions/list',
    '/api/promotions/actions/list',
    '/api/actions/list',
    '/api/v1/promotions/list',
    '/api/v2/promotions/list',
    '/api/campaigns/list',
  ]
  for (const endpoint of endpoints) {
    try {
      const response = await fetch(endpoint, {
        method: 'GET',
        credentials: 'include',
        headers: requestHeaders,
      })
      if (!response.ok) continue
      const contentType = String(response.headers.get('content-type') || '').toLowerCase()
      if (!contentType.includes('json')) continue
      const data = await response.json()
      packets.push({ endpoint, data })
    } catch {
      // ignore
    }
  }
  return packets
}

async function scriptFetchActionProductsPayloads(sourceActionID) {
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const companyId = readCookie('sc_company_id')
  const language = readCookie('x-o3-language') || 'zh-Hans'
  const requestHeaders = { accept: 'application/json', 'Content-Type': 'application/json' }
  if (companyId) requestHeaders['x-o3-company-id'] = companyId
  if (language) requestHeaders['x-o3-language'] = language

  const actionID = String(sourceActionID || '').trim()
  const packets = []

  // Prefer active products endpoint; fallback to candidate only if active endpoints are unavailable.
  const cursorEndpoints = [
    `/api/site/own-seller-products/v2/action/${actionID}/active`,
    `/api/site/own-seller-products/v2/action/${actionID}/active-search`,
    `/api/site/own-seller-products/v1/action/${actionID}/candidate`,
  ]

  for (const endpoint of cursorEndpoints) {
    let hasSuccessfulResponse = false
    const endpointPackets = []
    try {
      let cursor = ''
      for (let page = 0; page < 100; page += 1) {
        const body = { limit: 100 }
        if (cursor) {
          body.cursor = cursor
        }
        const response = await fetch(endpoint, {
          method: 'POST',
          credentials: 'include',
          headers: requestHeaders,
          body: JSON.stringify(body),
        })
        if (!response.ok) break
        hasSuccessfulResponse = true

        const contentType = String(response.headers.get('content-type') || '').toLowerCase()
        if (!contentType.includes('json')) break

        const data = await response.json()
        endpointPackets.push({ endpoint, data })

        const hasNext = Boolean(data?.has_next)
        cursor = String(data?.cursor || '')
        if (!hasNext) {
          break
        }
      }
    } catch {
      // try next endpoint
    }

    if (hasSuccessfulResponse) {
      packets.push(...endpointPackets)
      return packets
    }
  }

  if (packets.length > 0) {
    return packets
  }

  const endpoints = [
    { url: '/api/seller-actions/products', body: { action_id: actionID, limit: 500, offset: 0 } },
    { url: '/api/promotions/actions/products', body: { action_id: actionID, limit: 500, offset: 0 } },
    { url: '/api/actions/products', body: { action_id: actionID, limit: 500, offset: 0 } },
    { url: '/api/v1/promotions/products', body: { action_id: actionID, limit: 500, offset: 0 } },
    { url: '/api/v2/promotions/products', body: { action_id: actionID, limit: 500, offset: 0 } },
  ]
  for (const endpoint of endpoints) {
    try {
      const response = await fetch(endpoint.url, {
        method: 'POST',
        credentials: 'include',
        headers: requestHeaders,
        body: JSON.stringify(endpoint.body),
      })
      if (!response.ok) continue
      const contentType = String(response.headers.get('content-type') || '').toLowerCase()
      if (!contentType.includes('json')) continue
      const data = await response.json()
      packets.push({ endpoint: endpoint.url, data })
    } catch {
      // ignore
    }
  }
  return packets
}

async function scriptFetchCandidates(actionID) {
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const companyId = readCookie('sc_company_id')
  const language = readCookie('x-o3-language') || 'zh-Hans'
  const requestHeaders = { accept: 'application/json', 'content-type': 'application/json' }
  if (companyId) requestHeaders['x-o3-company-id'] = companyId
  if (language) requestHeaders['x-o3-language'] = language

  const allProducts = []
  let hasNext = true
  let cursor = ''
  let pageCount = 0

  while (hasNext && pageCount < 100) {
    const body = { limit: 100 }
    if (cursor) body.cursor = cursor

    const response = await fetch(
      `/api/site/own-seller-products/v1/action/${actionID}/candidate`,
      {
        method: 'POST',
        headers: requestHeaders,
        body: JSON.stringify(body),
        credentials: 'include',
      },
    )
    if (!response.ok) {
      throw new Error(`获取候选商品失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    if (Array.isArray(data?.products) && data.products.length > 0) {
      allProducts.push(...data.products)
    }

    hasNext = Boolean(data?.has_next)
    cursor = data?.cursor || ''
    pageCount += 1

    if (hasNext) {
      await new Promise((resolve) => setTimeout(resolve, 100))
    }
  }

  return allProducts
}

async function scriptActivateProducts(actionID, products) {
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const companyId = readCookie('sc_company_id')
  const language = readCookie('x-o3-language') || 'zh-Hans'
  const requestHeaders = { accept: 'application/json', 'content-type': 'application/json' }
  if (companyId) requestHeaders['x-o3-company-id'] = companyId
  if (language) requestHeaders['x-o3-language'] = language

  const payload = (products || []).map((item) => ({
    product_id: Number(item?.product_id || item?.id),
    skus: Array.isArray(item?.skus) ? item.skus.map((sku) => Number(sku)) : [],
    action_price: item?.action_price || { currency_code: '', nanos: 0, units: '0' },
    discount_percent: item?.discount_percent || 0,
    currency: item?.currency || '',
  }))

  const response = await fetch(
    `/api/site/own-seller-products/v1/action/${actionID}/activate`,
    {
      method: 'POST',
      headers: requestHeaders,
      body: JSON.stringify({ products: payload }),
      credentials: 'include',
    },
  )

  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(`申报失败: ${response.status} ${response.statusText} ${text}`)
  }
  return { success: true }
}

async function scriptDeactivateProducts(actionID, skus) {
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const companyId = readCookie('sc_company_id')
  const language = readCookie('x-o3-language') || 'zh-Hans'
  const requestHeaders = { accept: 'application/json', 'content-type': 'application/json' }
  if (companyId) requestHeaders['x-o3-company-id'] = companyId
  if (language) requestHeaders['x-o3-language'] = language

  const response = await fetch(
    `/api/site/own-seller-products/v1/action/${actionID}/deactivate`,
    {
      method: 'POST',
      headers: requestHeaders,
      body: JSON.stringify({ skus: skus || [] }),
      credentials: 'include',
    },
  )

  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(`退出失败: ${response.status} ${response.statusText} ${text}`)
  }
  return { success: true }
}
