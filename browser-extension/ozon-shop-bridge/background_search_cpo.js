const SELLER_BASE_URL = 'https://seller.ozon.ru'
const POLL_ALARM = 'ozon_manager_extension_poll'
const AUTH_SYNC_SCRIPT_ID = 'ozon_manager_auth_sync_dynamic'
const DEFAULT_POLL_INTERVAL_MS = 5000
const SEARCH_CPO_CONTEXT_STORAGE_KEY = 'searchCpoRequestContext'
const SEARCH_CPO_CONTEXT_MAX_AGE_MS = 30 * 60 * 1000
const SEARCH_CPO_REQUEST_URL_PATTERN = `${SELLER_BASE_URL}/performance-api/seller-api/search-performance-cpo/*`
const AUTH_SYNC_STATUS_UNKNOWN = 'unknown'
const AUTH_SYNC_STATUS_CONNECTED = 'connected'
const AUTH_SYNC_STATUS_MISSING_LOGIN = 'missing_login'
const AUTH_SYNC_STATUS_MISSING_SHOP = 'missing_shop'
const AUTH_SYNC_STATUS_PERMISSION_REQUIRED = 'permission_required'
const AUTH_SYNC_STATUS_SYNC_UNAVAILABLE = 'sync_unavailable'
const AUTH_SYNC_STATUS_FAILED = 'failed'
const AUTH_SYNC_SOURCE_AUTO = 'auto'
const AUTH_SYNC_SOURCE_MANUAL = 'manual'
const AUTH_SYNC_SOURCE_NONE = 'none'


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
  authSyncStatus: AUTH_SYNC_STATUS_UNKNOWN,
  authSyncSource: AUTH_SYNC_SOURCE_NONE,
  authSyncMessage: '等待检测管理端连接状态',
  authSyncOrigin: '',
  authSyncCheckedAt: '',
}

let pollInFlight = false

function handleSearchCPORequestHeaders(details) {
  const context = extractSearchCPORequestContext(details)
  if (!context) return
  void persistSearchCPORequestContext(context)
}

if (chrome.webRequest?.onBeforeSendHeaders && !chrome.webRequest.onBeforeSendHeaders.hasListener(handleSearchCPORequestHeaders)) {
  chrome.webRequest.onBeforeSendHeaders.addListener(
    handleSearchCPORequestHeaders,
    { urls: [SEARCH_CPO_REQUEST_URL_PATTERN] },
    ['requestHeaders', 'extraHeaders'],
  )
}


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

  if (type === 'OZON_MANAGER_CHECK_AUTH_SYNC') {
    checkAuthSync(message?.payload || {})
      .then(async (result) => {
        const state = await readState()
        sendResponse({ ok: true, result, state })
      })
      .catch((error) => sendResponse({ ok: false, error: error?.message || String(error) }))
    return true
  }

  if (type === 'OZON_MANAGER_SET_CONFIG') {
    saveStatePatch(message?.payload || {})
      .then(async () => {
        const authSync = await checkAuthSync({ requestPermission: true })
        await ensurePollingAlarm()
        const sync = await pollOnce()
        const latestState = await readState()
        sendResponse({ ok: true, state: latestState, sync, authSync })
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
  if (typeof patch.authSyncStatus === 'string') next.authSyncStatus = patch.authSyncStatus || AUTH_SYNC_STATUS_UNKNOWN
  if (typeof patch.authSyncSource === 'string') next.authSyncSource = patch.authSyncSource || AUTH_SYNC_SOURCE_NONE
  if (typeof patch.authSyncMessage === 'string') next.authSyncMessage = patch.authSyncMessage
  if (typeof patch.authSyncOrigin === 'string') next.authSyncOrigin = normalizeHTTPOrigin(patch.authSyncOrigin)
  if (typeof patch.authSyncCheckedAt === 'string') next.authSyncCheckedAt = patch.authSyncCheckedAt
  if (Object.keys(next).length > 0) {
    await chrome.storage.local.set(next)
  }
}

function hasSavedAuthConfig(state) {
  return Boolean(state?.authToken && state?.shopId)
}

function getSavedAuthSource(state) {
  return hasSavedAuthConfig(state) ? AUTH_SYNC_SOURCE_MANUAL : AUTH_SYNC_SOURCE_NONE
}

function buildAuthSyncResult(status, source, message, extra = {}) {
  return {
    auth_sync_status: status || AUTH_SYNC_STATUS_UNKNOWN,
    sync_source: source || AUTH_SYNC_SOURCE_NONE,
    admin_origin: normalizeHTTPOrigin(extra.admin_origin || ''),
    token_present: Boolean(extra.token_present),
    shop_present: Boolean(extra.shop_present),
    message: String(message || '').trim() || '管理端连接状态未知',
    checked_at: String(extra.checked_at || '').trim() || new Date().toISOString(),
  }
}

function buildAuthSyncStoragePatch(result, snapshot) {
  const patch = {
    authSyncStatus: result?.auth_sync_status || AUTH_SYNC_STATUS_UNKNOWN,
    authSyncSource: result?.sync_source || AUTH_SYNC_SOURCE_NONE,
    authSyncMessage: result?.message || '',
    authSyncOrigin: result?.admin_origin || '',
    authSyncCheckedAt: result?.checked_at || new Date().toISOString(),
  }
  if (snapshot?.token) {
    patch.authToken = snapshot.token
  }
  if (snapshot?.shop_id) {
    patch.shopId = snapshot.shop_id
  }
  return patch
}

async function persistAuthSyncResult(result, snapshot) {
  await saveStatePatch(buildAuthSyncStoragePatch(result, snapshot))
}

function normalizeManagerAuthSnapshot(raw) {
  if (!raw || typeof raw !== 'object') return null

  const token = typeof raw.token === 'string' ? raw.token.trim() : ''
  const shopID = Number(raw.shop_id || 0)
  return {
    token,
    shop_id: Number.isFinite(shopID) && shopID > 0 ? shopID : null,
    origin: normalizeHTTPOrigin(raw.origin || ''),
  }
}

function makeAuthSyncResultFromSnapshot(snapshot, state) {
  const tokenPresent = Boolean(snapshot?.token)
  const shopPresent = Boolean(snapshot?.shop_id)
  const savedSource = getSavedAuthSource(state)
  const adminOrigin = snapshot?.origin || ''

  if (tokenPresent && shopPresent) {
    return buildAuthSyncResult(
      AUTH_SYNC_STATUS_CONNECTED,
      AUTH_SYNC_SOURCE_AUTO,
      '已连接管理端，可自动执行任务',
      {
        admin_origin: adminOrigin,
        token_present: true,
        shop_present: true,
      },
    )
  }

  if (!tokenPresent) {
    return buildAuthSyncResult(
      AUTH_SYNC_STATUS_MISSING_LOGIN,
      savedSource,
      savedSource === AUTH_SYNC_SOURCE_MANUAL
        ? '当前管理端未登录，插件将继续使用已保存配置'
        : '请先在管理端登录',
      {
        admin_origin: adminOrigin,
        token_present: Boolean(state?.authToken),
        shop_present: Boolean(state?.shopId),
      },
    )
  }

  return buildAuthSyncResult(
    AUTH_SYNC_STATUS_MISSING_SHOP,
    savedSource === AUTH_SYNC_SOURCE_MANUAL ? AUTH_SYNC_SOURCE_MANUAL : AUTH_SYNC_SOURCE_AUTO,
    savedSource === AUTH_SYNC_SOURCE_MANUAL
      ? '当前管理端已登录但未选择店铺，插件将继续使用已保存配置'
      : '请先在管理端选择店铺',
    {
      admin_origin: adminOrigin,
      token_present: true,
      shop_present: Boolean(state?.shopId),
    },
  )
}

function buildAuthSyncTabPatterns(state) {
  return buildAuthSyncOrigins(state).map((origin) => `${origin}/*`)
}

function getPreferredAuthSyncOrigin(state) {
  const adminOrigin = normalizeHTTPOrigin(state?.adminOrigin || '')
  if (adminOrigin) return adminOrigin
  return normalizeHTTPOrigin(state?.apiBaseUrl || '')
}

function normalizeSearchCPOText(value) {
  if (value === null || value === undefined) return ''
  return String(value).trim()
}

function normalizeSearchCPOOrganisation(value) {
  if (value && typeof value === 'object') {
    const direct = [
      value.id,
      value.organizationId,
      value.organisationId,
      value.currentOrganizationId,
      value.currentOrganisationId,
      value.companyId,
      value.value,
    ]
    for (const candidate of direct) {
      const text = normalizeSearchCPOText(candidate)
      if (/^\d+$/.test(text)) return text
    }
  }

  const text = normalizeSearchCPOText(value)
  if (!text) return ''
  if (/^\d+$/.test(text)) return text
  try {
    const parsed = JSON.parse(text)
    return normalizeSearchCPOOrganisation(parsed)
  } catch {
    return ''
  }
}

function normalizeSearchCPOContext(raw) {
  if (!raw || typeof raw !== 'object') return null

  const context = {
    tabId: Number(raw.tabId || 0),
    url: normalizeSearchCPOText(raw.url),
    capturedAt: Number(raw.capturedAt || 0),
    companyId: normalizeSearchCPOOrganisation(raw.companyId),
    advOrganisation: normalizeSearchCPOOrganisation(raw.advOrganisation),
    language: normalizeSearchCPOText(raw.language) || 'zh-Hans',
    appName: normalizeSearchCPOText(raw.appName) || 'performance-sc',
  }

  if (!context.companyId || !context.advOrganisation) return null
  if (!Number.isFinite(context.capturedAt) || context.capturedAt <= 0) {
    context.capturedAt = Date.now()
  }
  return context
}

function extractSearchCPORequestContext(details) {
  const headers = Array.isArray(details?.requestHeaders) ? details.requestHeaders : []
  if (headers.length === 0) return null

  const headerMap = {}
  for (const header of headers) {
    const name = normalizeSearchCPOText(header?.name).toLowerCase()
    if (!name) continue
    headerMap[name] = normalizeSearchCPOText(header?.value)
  }

  return normalizeSearchCPOContext({
    tabId: details?.tabId,
    url: details?.initiator || details?.documentUrl || details?.url || '',
    capturedAt: Date.now(),
    companyId: headerMap['x-o3-company-id'],
    advOrganisation: headerMap['x-adv-current-organisation'],
    language: headerMap['x-o3-language'],
    appName: headerMap['x-o3-app-name'],
  })
}

async function persistSearchCPORequestContext(rawContext) {
  const context = normalizeSearchCPOContext(rawContext)
  if (!context) return
  await chrome.storage.local.set({
    [SEARCH_CPO_CONTEXT_STORAGE_KEY]: context,
  })
}

async function readSearchCPORequestContext() {
  const data = await chrome.storage.local.get([SEARCH_CPO_CONTEXT_STORAGE_KEY])
  const context = normalizeSearchCPOContext(data?.[SEARCH_CPO_CONTEXT_STORAGE_KEY])
  if (!context) return null
  if (Date.now() - context.capturedAt > SEARCH_CPO_CONTEXT_MAX_AGE_MS) {
    return null
  }
  return context
}

function selectSearchCPORequestContextForTab(context, tabID) {
  if (!context) return null
  if (Number(context.tabId || 0) !== Number(tabID || 0)) return null
  return context
}

async function handleAuthSync(payload) {
  if (!payload || typeof payload !== 'object') return

  const state = await readState()
  const snapshot = normalizeManagerAuthSnapshot(payload)
  if (!snapshot) return

  const result = makeAuthSyncResultFromSnapshot(snapshot, state)
  await persistAuthSyncResult(result, snapshot)
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
  if (pollInFlight) {
    return {
      ok: false,
      skipped: true,
      error: '已有同步任务进行中，请稍后重试',
    }
  }
  pollInFlight = true
  try {
    const state = await readState()
    if (!state.enabled) {
      return {
        ok: false,
        skipped: true,
        error: '轮询已关闭，未执行立即同步',
      }
    }
    if (!state.authToken || !state.shopId || !state.apiBaseUrl) {
      return {
        ok: false,
        skipped: true,
        error: '配置不完整，请先连接管理端，或在高级设置填写后端地址与兜底配置',
      }
    }

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
      return {
        ok: true,
        hasJob: false,
      }
    }

    const run = await executeJob(job, state)
    let runError = ''
    if (run?.status === 'failed') {
      const failed = Array.isArray(run?.results)
        ? run.results.filter((item) => String(item?.overall_status || '').toLowerCase() === 'failed')
        : []
      const messages = failed
        .map((item) => String(item?.error_message || '').trim())
        .filter(Boolean)
      runError = messages[0] || '任务执行失败'
    }
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
    return {
      ok: true,
      hasJob: true,
      status: run.status,
      error: runError,
    }
  } catch (error) {
    const message = error?.message || String(error)
    await saveStatePatch({
      lastRunAt: new Date().toISOString(),
      lastError: message,
    })
    return {
      ok: false,
      skipped: false,
      error: message,
    }
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

async function findManagementTabs(state) {
  const patterns = buildAuthSyncTabPatterns(state)
  if (patterns.length === 0) return []

  const tabs = await chrome.tabs.query({ url: patterns })
  const seen = new Set()
  return (tabs || [])
    .filter((tab) => {
      const tabID = Number(tab?.id || 0)
      if (tabID <= 0 || seen.has(tabID)) return false
      seen.add(tabID)
      return true
    })
    .sort(compareTabPriority)
}

async function checkAuthSync(options = {}) {
  const state = await readState()
  const requestPermission = Boolean(options?.requestPermission)
  const preferredOrigin = getPreferredAuthSyncOrigin(state)
  const savedSource = getSavedAuthSource(state)
  const permission = await ensureAuthSyncContentScript(state, requestPermission)

  if (!permission.ok) {
    const result = buildAuthSyncResult(
      AUTH_SYNC_STATUS_PERMISSION_REQUIRED,
      savedSource,
      savedSource === AUTH_SYNC_SOURCE_MANUAL
        ? '未授予管理端页面权限，当前继续使用已保存配置'
        : permission.message || '需要授权访问管理端页面',
      {
        admin_origin: preferredOrigin,
        token_present: Boolean(state?.authToken),
        shop_present: Boolean(state?.shopId),
      },
    )
    await persistAuthSyncResult(result)
    return result
  }

  const tabs = await findManagementTabs(state)
  if (tabs.length === 0) {
    const result = buildAuthSyncResult(
      AUTH_SYNC_STATUS_SYNC_UNAVAILABLE,
      savedSource,
      savedSource === AUTH_SYNC_SOURCE_MANUAL
        ? '未检测到已打开的管理端页面，当前继续使用已保存配置'
        : '请先打开管理端页面后重试',
      {
        admin_origin: preferredOrigin,
        token_present: Boolean(state?.authToken),
        shop_present: Boolean(state?.shopId),
      },
    )
    await persistAuthSyncResult(result)
    return result
  }

  let bestResult = null
  let bestSnapshot = null
  let lastError = ''
  for (const tab of tabs) {
    try {
      const snapshot = normalizeManagerAuthSnapshot(await runScript(tab.id, scriptReadManagerAuthSnapshot, []))
      if (!snapshot) continue

      const result = makeAuthSyncResultFromSnapshot(snapshot, state)
      if (result.auth_sync_status === AUTH_SYNC_STATUS_CONNECTED) {
        await persistAuthSyncResult(result, snapshot)
        return result
      }
      if (
        !bestResult ||
        (bestResult.auth_sync_status === AUTH_SYNC_STATUS_MISSING_LOGIN &&
          result.auth_sync_status === AUTH_SYNC_STATUS_MISSING_SHOP)
      ) {
        bestResult = result
        bestSnapshot = snapshot
      }
    } catch (error) {
      lastError = error?.message || String(error)
    }
  }

  if (bestResult) {
    await persistAuthSyncResult(bestResult, bestSnapshot)
    return bestResult
  }

  const result = buildAuthSyncResult(
    lastError ? AUTH_SYNC_STATUS_FAILED : AUTH_SYNC_STATUS_SYNC_UNAVAILABLE,
    savedSource,
    lastError
      ? (savedSource === AUTH_SYNC_SOURCE_MANUAL
          ? `读取管理端登录态失败，当前继续使用已保存配置：${lastError}`
          : `读取管理端登录态失败：${lastError}`)
      : (savedSource === AUTH_SYNC_SOURCE_MANUAL
          ? '未能从管理端页面读取登录态，当前继续使用已保存配置'
          : '暂时无法读取管理端登录态，请打开管理端页面后重试'),
    {
      admin_origin: preferredOrigin,
      token_present: Boolean(state?.authToken),
      shop_present: Boolean(state?.shopId),
    },
  )
  await persistAuthSyncResult(result)
  return result
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
    return { ok: true, message: '' }
  }

  const hasPermission = await chrome.permissions.contains({ origins: dynamicMatches })
  if (!hasPermission) {
    if (!requestPermission) {
      return {
        ok: false,
        message: '需要授权访问管理端页面',
      }
    }
    const granted = await chrome.permissions.request({ origins: dynamicMatches })
    if (!granted) {
      return {
        ok: false,
        message: '未授予管理端页面权限，请点击“重新检测”后允许访问',
      }
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

  return { ok: true, message: '' }
}

async function executeJob(job, state) {
  try {
    // Search CPO jobs must stay on the current CPO page/context instead of the generic seller worker tab.
    if (isSearchCPOJobType(job?.job_type)) {
      return await executeSearchCPOJob(job)
    }

    const tab = await ensureSellerLoggedInTab()
    switch (job.job_type) {
      case 'sync_shop_actions':
        return await executeSyncShopActions(tab.id)
      case 'sync_action_candidates':
        return await executeSyncActionCandidates(tab.id, job)
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

function isSearchCPOJobType(jobType) {
  return [
    'sync_search_cpo_products',
    'sync_search_cpo_availability',
    'search_cpo_enable_products',
    'search_cpo_batch_enable_morkovsk',
  ].includes(String(jobType || '').trim())
}

async function executeSearchCPOJob(job) {
  switch (job?.job_type) {
    case 'sync_search_cpo_products':
      return await executeSyncSearchCPOProducts()
    case 'sync_search_cpo_availability':
      return await executeSyncSearchCPOAvailability(job)
    case 'search_cpo_enable_products':
      return await executeSearchCPOEnableProducts(job)
    case 'search_cpo_batch_enable_morkovsk':
      return await executeSearchCPOMorkovskBatchEnable(job)
    default:
      return {
        status: 'failed',
        results: buildJobFailureResults(job, `插件不支持该任务类型: ${job?.job_type || ''}`),
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

function isSearchCPOURL(url) {
  const lower = String(url || '').toLowerCase()
  return lower.startsWith(`${SELLER_BASE_URL.toLowerCase()}/`) && lower.includes('/app/advertisement/product/cpo')
}

function compareTabPriority(left, right) {
  const leftCurrentWindow = left?.currentWindow ? 1 : 0
  const rightCurrentWindow = right?.currentWindow ? 1 : 0
  if (leftCurrentWindow !== rightCurrentWindow) {
    return rightCurrentWindow - leftCurrentWindow
  }

  const leftActive = left?.active ? 1 : 0
  const rightActive = right?.active ? 1 : 0
  if (leftActive !== rightActive) {
    return rightActive - leftActive
  }

  const leftAccessed = Number(left?.lastAccessed || 0)
  const rightAccessed = Number(right?.lastAccessed || 0)
  return rightAccessed - leftAccessed
}

async function findReusableSearchCPOTab() {
  const tabs = await chrome.tabs.query({})
  const candidates = (tabs || [])
    .filter((tab) => isSearchCPOURL(tab?.url || '') && !isAuthURL(tab?.url || ''))
    .sort(compareTabPriority)

  if (candidates.length === 0) {
    throw new Error('未找到已打开的 Seller 推广按订单付费页面，请先打开后重试')
  }

  const capturedContext = await readSearchCPORequestContext()
  let missingAdvOrganisation = false
  let missingCompanyContext = false

  for (const tab of candidates) {
    try {
      const context = await runScript(tab.id, scriptReadSearchCPOContext, [])
      const companyId = String(context?.companyId || '').trim()
      const advOrganisation = String(context?.advOrganisation || '').trim()

      if (companyId && advOrganisation) {
        return { tab, context }
      }
      if (!companyId) {
        missingCompanyContext = true
      }
      if (!advOrganisation) {
        missingAdvOrganisation = true
      }
    } catch {
      // Ignore transient tab/script failures and continue scanning other tabs.
    }

    const capturedForTab = selectSearchCPORequestContextForTab(capturedContext, tab.id)
    if (capturedForTab) {
      return { tab, context: capturedForTab }
    }
  }

  if (missingAdvOrganisation) {
    throw new Error('当前 CPO 页面未提供组织上下文，请在该页面刷新或重新进入后重试；如刚重载插件，请先手动刷新该页面一次')
  }
  if (missingCompanyContext) {
    throw new Error('当前 CPO 页面未提供登录上下文，请确认 Seller 登录状态后重试')
  }
  throw new Error('未找到可用的 Seller 推广按订单付费页面，请先打开后重试')
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

async function executeSyncActionCandidates(tabID, job) {
  const sourceActionID = String(job?.meta?.source_action_id || '').trim()
  if (!sourceActionID) {
    return {
      status: 'failed',
      results: [buildSyncResult('__sync_action_candidates__', false, '任务缺少 source_action_id')],
      meta: {},
    }
  }

  const rawItems = await runScript(tabID, scriptFetchCandidates, [sourceActionID])
  const items = uniqueBy(
    (rawItems || [])
      .map((item) => normalizeActionProduct(item, 'candidate'))
      .filter(Boolean),
    (item) => buildActionProductDedupKey(item),
  )

  return {
    status: 'success',
    results: [buildSyncResult('__sync_action_candidates__', true, '')],
    meta: {
      source_action_id: sourceActionID,
      items,
      candidate_count: items.length,
    },
  }
}

async function executeSyncSearchCPOProducts() {
  const selection = await findReusableSearchCPOTab()
  const tab = selection?.tab
  const context = selection?.context
  const tabID = tab.id

  if (context) {
    await runScript(tabID, scriptPrimeSearchCPOContext, [context])
  }

  const rawItems = await runScript(tabID, scriptFetchSearchCPOProducts, [])
  const items = uniqueBy(
    (rawItems || [])
      .map((item) => normalizeSearchCPOProduct(item))
      .filter(Boolean),
    (item) => normalizeSKU(item.source_sku || item.sku),
  )

  return {
    status: 'success',
    results: [buildSyncResult('__sync_search_cpo_products__', true, '')],
    meta: {
      items,
      total: items.length,
    },
  }
}

async function prepareSearchCPOJobExecution() {
  const selection = await findReusableSearchCPOTab()
  const tab = selection?.tab
  const context = selection?.context
  const tabID = tab.id
  if (context) {
    await runScript(tabID, scriptPrimeSearchCPOContext, [context])
  }
  return { tabID, context }
}

async function executeSyncSearchCPOAvailability(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptFetchSearchCPOAvailability, [requestSKUs])
  const items = normalizeSearchCPOAvailabilityItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, !row?.error, row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: { items },
  }
}

async function executeSearchCPOEnableProducts(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptEnableSearchCPOProducts, [requestSKUs])
  const items = normalizeSearchCPOStepItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, String(row?.status || '') !== 'failed', row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: { items },
  }
}

async function executeSearchCPOMorkovskBatchEnable(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptBatchEnableSearchCPOMorkovsk, [requestSKUs])
  const items = normalizeSearchCPOStepItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, String(row?.status || '') !== 'failed', row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: { items },
  }
}

function normalizeSearchCPOAvailabilityItems(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const data = sourceMap?.data || sourceMap
  const candidates = Array.isArray(data?.items) ? data.items : Array.isArray(data?.products) ? data.products : []

  return pairs.map(({ sourceSKU, targetSKU }) => {
    let matched = data?.[targetSKU] || data?.result?.[targetSKU] || data?.[sourceSKU] || data?.result?.[sourceSKU] || null
    if (!matched) {
      matched = candidates.find((item) => {
        const keys = [item?.sku, item?.source_sku, item?.sourceSku, item?.offer_id, item?.product_id, item?.id]
          .map((value) => normalizeSKU(value))
          .filter(Boolean)
        return keys.includes(targetSKU) || keys.includes(sourceSKU)
      }) || null
    }
    const availability = readSearchCPOAvailabilityValue(matched)
    const error = matched
      ? availability === null
        ? 'search_promo_availability 响应缺少 availability 字段'
        : ''
      : '未匹配到 search_promo_availability 响应'
    return {
      source_sku: sourceSKU,
      sku: firstNonEmptySearchCPOText(matched?.sku, targetSKU, sourceSKU),
      search_promo_status: firstNonEmptySearchCPOText(matched?.searchPromoStatus, matched?.search_promo_status),
      carrots_status: firstNonEmptySearchCPOText(matched?.carrotsStatus, matched?.carrots_status),
      availability_promo: availability,
      error,
      payload: matched || { requested_sku: targetSKU, source_sku: sourceSKU, error },
    }
  })
}

function normalizeSearchCPOStepItems(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const items = Array.isArray(sourceMap?.items)
    ? sourceMap.items
    : Array.isArray(sourceMap?.data?.items)
      ? sourceMap.data.items
      : []
  const itemBySKU = {}
  for (const item of items) {
    const keys = [item?.source_sku, item?.sourceSku, item?.sku, item?.offer_id, item?.product_id, item?.id]
      .map((value) => normalizeSKU(value))
      .filter(Boolean)
    for (const key of keys) {
      if (!itemBySKU[key]) {
        itemBySKU[key] = item
      }
    }
  }
  const fallbackMessage = firstNonEmptySearchCPOText(sourceMap?.message, sourceMap?.data?.message)
  return pairs.map(({ sourceSKU, targetSKU }) => {
    const current = itemBySKU[targetSKU] || itemBySKU[sourceSKU] || null
    const error = firstNonEmptySearchCPOText(current?.error, current ? "" : `未匹配到 Search CPO 执行响应: ${targetSKU}`)
    return {
      source_sku: sourceSKU,
      sku: targetSKU,
      status: firstNonEmptySearchCPOText(current?.status, error ? 'failed' : current ? 'success' : 'failed') || 'failed',
      error,
      message: firstNonEmptySearchCPOText(current?.message, fallbackMessage, error),
    }
  })
}

function makeSearchCPOStepJobResult(sourceSKU, success, errorMessage) {
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
  const results = sourceSKUs.map((sku) => makeActionResultByMessage(sku, operation, errorsBySKU[sku] || ''))

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

  const results = sourceSKUs.map((sku) => makeActionResultByMessage(sku, operation, mergedErrors[sku] || ''))
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
  const candidates = operation === 'declare'
    ? await runScript(tabID, scriptFetchCandidates, [sourceActionID])
    : []
  const activeItems = operation === 'remove'
    ? uniqueBy(
        ((await runScript(tabID, scriptFetchActionProductsPayloads, [sourceActionID])) || [])
          .flatMap((packet) => extractActionProductsFromPayload(packet?.data || {}, packet?.endpoint || '')),
        (item) => buildActionProductDedupKey(item),
      )
    : []
  const matched = []
  const errorsBySKU = {}

  for (const sku of sourceSKUs) {
    if (operation === 'declare') {
      const candidate = findCandidateBySKU(candidates || [], sku)
      if (!candidate) {
        errorsBySKU[sku] = '未找到活动候选商品'
        continue
      }
      matched.push({ sourceSKU: sku, candidate })
      continue
    }

    const activeItem = findActionProductBySKU(activeItems || [], sku)
    if (!activeItem) {
      errorsBySKU[sku] = '__SKIPPED__:商品当前不在活动中'
      continue
    }
    matched.push({ sourceSKU: sku, activeItem })
  }

  if (matched.length === 0) {
    return errorsBySKU
  }

  try {
    if (operation === 'declare') {
      await runScript(tabID, scriptActivateProducts, [sourceActionID, matched.map((entry) => entry.candidate)])
    } else {
      const skus = matched.map((entry) => normalizeSKU(entry.sourceSKU)).filter(Boolean)
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
  if (job?.job_type === 'sync_action_candidates') {
    return [buildSyncResult('__sync_action_candidates__', false, message)]
  }
  if (job?.job_type === 'sync_search_cpo_products') {
    return [buildSyncResult('__sync_search_cpo_products__', false, message)]
  }
  if (job?.job_type === 'sync_search_cpo_availability' || job?.job_type === 'search_cpo_enable_products' || job?.job_type === 'search_cpo_batch_enable_morkovsk') {
    const sourceSKUs = (job?.items || []).map((item) => normalizeSKU(item?.source_sku)).filter(Boolean)
    return sourceSKUs.map((sku) => makeSearchCPOStepJobResult(sku, false, message))
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

function makeSkippedActionResult(sourceSKU, operation, message) {
  if (operation === 'declare') {
    return {
      source_sku: sourceSKU,
      overall_status: 'skipped',
      step_exit_status: 'skipped',
      step_reprice_status: 'skipped',
      step_readd_status: 'skipped',
      step_exit_error: '',
      step_reprice_error: '',
      step_readd_error: String(message || '').trim(),
    }
  }
  return {
    source_sku: sourceSKU,
    overall_status: 'skipped',
    step_exit_status: 'skipped',
    step_reprice_status: 'skipped',
    step_readd_status: 'skipped',
    step_exit_error: String(message || '').trim(),
    step_reprice_error: '',
    step_readd_error: '',
  }
}

function makeActionResultByMessage(sourceSKU, operation, message) {
  const text = String(message || '').trim()
  if (!text) {
    return makeActionResult(sourceSKU, operation, true, '')
  }
  if (text.startsWith('__SKIPPED__:')) {
    return makeSkippedActionResult(sourceSKU, operation, text.replace('__SKIPPED__:', '').trim())
  }
  return makeActionResult(sourceSKU, operation, false, text)
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

function firstNonEmptySearchCPOText(...values) {
  for (const value of values) {
    const text = normalizeSKU(value)
    if (text) return text
  }
  return ''
}

function normalizeSearchCPOSkuMap(meta) {
  const rawMap = meta && typeof meta === 'object' && meta.sku_map && typeof meta.sku_map === 'object'
    ? meta.sku_map
    : {}
  const normalized = {}
  for (const [sourceSKU, targetSKU] of Object.entries(rawMap)) {
    const normalizedSource = normalizeSKU(sourceSKU)
    const normalizedTarget = normalizeSKU(targetSKU)
    if (!normalizedSource || !normalizedTarget) continue
    normalized[normalizedSource] = normalizedTarget
  }
  return normalized
}

function buildSearchCPOSkuPairs(job) {
  const skuMap = normalizeSearchCPOSkuMap(job?.meta)
  const pairs = []
  const seen = new Set()
  for (const item of job?.items || []) {
    const sourceSKU = normalizeSKU(item?.source_sku)
    if (!sourceSKU || seen.has(sourceSKU)) continue
    seen.add(sourceSKU)
    const targetSKU = normalizeSKU(skuMap[sourceSKU] || sourceSKU)
    if (!targetSKU) continue
    pairs.push({ sourceSKU, targetSKU })
  }
  return pairs
}

function readSearchCPOAvailabilityValue(item) {
  const candidates = [
    item?.promo,
    item?.isPromo,
    item?.availabilityPromo,
    item?.availability_promo,
    item?.available,
    item?.is_available,
  ]
  for (const value of candidates) {
    if (typeof value === 'boolean') return value
    if (value === 1 || value === '1') return true
    if (value === 0 || value === '0') return false
  }
  return null
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

function findActionProductBySKU(items, sourceSKU) {
  const needle = normalizeSKU(sourceSKU)
  if (!needle) return null

  for (const item of items || []) {
    const candidates = [
      normalizeSKU(item?.source_sku),
      normalizeSKU(item?.offer_id),
      normalizeSKU(item?.platform_sku),
      normalizeSKU(item?.ozon_product_id),
      normalizeSKU(item?.sku),
    ].filter(Boolean)
    if (candidates.includes(needle)) {
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

function normalizeSearchCPOProduct(raw) {
  if (!raw || typeof raw !== 'object') return null

  const sku = normalizeSKU(raw?.sku)
  const sourceSKU = normalizeSKU(raw?.sourceSku || raw?.source_sku || raw?.offer_id || sku)
  if (!sourceSKU) return null

  const metrics = raw?.metrics && typeof raw.metrics === 'object' ? raw.metrics : {}
  const stocks = Array.isArray(raw?.stocks) ? raw.stocks : []
  let stockTotal = 0
  for (const stock of stocks) {
    stockTotal += Math.max(0, Math.trunc(toNumber(stock?.count ?? stock?.stock ?? 0, 0)))
  }

  return {
    sku,
    source_sku: sourceSKU,
    image_url: String(raw?.imageUrl || raw?.image_url || '').trim(),
    title: String(raw?.title || '').trim(),
    category_name: String(raw?.categoryName || raw?.category_name || '').trim(),
    price: toNumber(raw?.price, 0),
    is_in_stock: Boolean(raw?.isInStock),
    search_promo_status: String(raw?.searchPromoStatus || raw?.search_promo_status || '').trim(),
    carrots_status: String(raw?.carrotsStatus || raw?.carrots_status || '').trim(),
    is_favorite: Boolean(raw?.isFavorite),
    orders: Math.max(0, Math.trunc(toNumber(metrics?.orders, 0))),
    spent: toNumber(metrics?.spent, 0),
    clicks: Math.max(0, Math.trunc(toNumber(metrics?.clicks, 0))),
    ctr_percent: toNumber(metrics?.ctrPercent, 0),
    stock_total: stockTotal,
    payload: raw,
  }
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

function scriptReadManagerAuthSnapshot() {
  const token = localStorage.getItem('token') || ''
  const rawShopID = localStorage.getItem('currentShopId')
  const parsedShopID = Number(rawShopID || '')
  return {
    origin: window.location.origin,
    token: String(token || '').trim(),
    shop_id: Number.isFinite(parsedShopID) && parsedShopID > 0 ? parsedShopID : null,
  }
}

function scriptPrimeSearchCPOContext(context) {
  const current = window.__ozonManagerSearchCPOContext && typeof window.__ozonManagerSearchCPOContext === 'object'
    ? window.__ozonManagerSearchCPOContext
    : {}
  window.__ozonManagerSearchCPOContext = {
    ...current,
    ...(context && typeof context === 'object' ? context : {}),
  }
  return window.__ozonManagerSearchCPOContext
}

function scriptReadSearchCPOContext() {
  const normalizeText = (value) => {
    if (value === null || value === undefined) return ''
    return String(value).trim()
  }

  const normalizeNonEmpty = (value) => {
    if (value && typeof value === 'object') {
      const direct = [value.value, value.name, value.code, value.lang, value.language]
      for (const candidate of direct) {
        const text = normalizeText(candidate)
        if (text) return text
      }
    }
    return normalizeText(value)
  }

  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }

  const normalizeOrganisation = (value) => {
    if (value && typeof value === 'object') {
      const direct = [
        value.id,
        value.organizationId,
        value.organisationId,
        value.currentOrganizationId,
        value.currentOrganisationId,
        value.companyId,
        value.value,
      ]
      for (const candidate of direct) {
        const candidateText = normalizeText(candidate)
        if (/^\d+$/.test(candidateText)) return candidateText
      }
    }

    const text = normalizeText(value)
    if (!text) return ''
    if (/^\d+$/.test(text)) return text
    try {
      const parsed = JSON.parse(text)
      return normalizeOrganisation(parsed)
    } catch {
      return ''
    }
  }

  const readExactKeys = (target, keys, normalize) => {
    if (!target || typeof target !== 'object') return ''
    for (const key of keys) {
      try {
        const normalized = normalize(target[key])
        if (normalized) return normalized
      } catch {
        // ignore getter failures
      }
    }
    return ''
  }

  const enqueue = (queue, seen, value) => {
    if (!value || typeof value !== 'object') return
    if (seen.has(value)) return
    queue.push(value)
  }

  const findInObjectGraph = (keys, normalize, looseMatcher) => {
    const queue = []
    const seen = new Set()
    const rootCandidates = [
      window.__ozonManagerSearchCPOContext,
      window.__INITIAL_STATE__,
      window.__PRELOADED_STATE__,
      window.__NEXT_DATA__,
      window.__NUXT__,
      window.__PINIA__,
      window.__APOLLO_STATE__,
      window.__APP_STATE__,
      window.__APP_CONFIG__,
      window.__STORE__,
      window.__REDUX_STATE__,
      window.store,
      window.app,
      window.App,
      document.body?.__vueParentComponent,
      document.documentElement?.__vueParentComponent,
      document.querySelector('#app')?.__vue_app__,
      document.querySelector('#__next'),
      document.querySelector('#__nuxt'),
    ]

    for (const candidate of rootCandidates) {
      enqueue(queue, seen, candidate)
    }

    let discoveredWindowRoots = 0
    for (const key of Object.getOwnPropertyNames(window)) {
      if (discoveredWindowRoots >= 80) break
      if (!/(^__|state|store|config|headers|request|client|api|advert|promo|performance|query|pinia|redux|apollo)/i.test(key)) {
        continue
      }
      try {
        enqueue(queue, seen, window[key])
        discoveredWindowRoots += 1
      } catch {
        // ignore cross-origin / getter failures
      }
    }

    enqueue(queue, seen, window)

    let scanned = 0
    while (queue.length > 0 && scanned < 2000) {
      const current = queue.shift()
      if (!current || typeof current !== 'object') continue
      if (seen.has(current)) continue
      seen.add(current)
      scanned += 1

      const exact = readExactKeys(current, keys, normalize)
      if (exact) return exact

      for (const containerKey of ['headers', 'header', 'requestHeaders', 'defaultHeaders', 'common', 'config', 'defaults', 'options']) {
        let nested = null
        try {
          nested = current[containerKey]
        } catch {
          nested = null
        }
        const nestedExact = readExactKeys(nested, keys, normalize)
        if (nestedExact) return nestedExact
      }

      let objectKeys = []
      try {
        objectKeys = Array.isArray(current) ? current.keys?.() : Object.keys(current)
      } catch {
        objectKeys = []
      }
      if (!Array.isArray(objectKeys)) {
        objectKeys = Array.from(objectKeys || [])
      }

      for (const key of objectKeys.slice(0, 120)) {
        let value
        try {
          value = current[key]
        } catch {
          continue
        }

        if (typeof looseMatcher === 'function') {
          const maybe = looseMatcher(key, value, normalize)
          if (maybe) return maybe
        }

        if (value && typeof value === 'object' && queue.length < 2500) {
          enqueue(queue, seen, value)
        }
      }
    }

    return ''
  }

  const readAdvOrganisation = () => {
    const keys = [
      'advCurrentOrganisation',
      'adv_current_organisation',
      'adv_current_organization',
      'advertisementCurrentOrganisation',
      'advertisement_current_organisation',
      'x-adv-current-organisation',
    ]

    for (const key of keys) {
      const fromLocal = normalizeOrganisation(window.localStorage?.getItem(key))
      if (fromLocal) return fromLocal
      const fromSession = normalizeOrganisation(window.sessionStorage?.getItem(key))
      if (fromSession) return fromSession
    }

    const fromCookie = normalizeOrganisation(readCookie('adv_current_organisation'))
    if (fromCookie) return fromCookie

    return findInObjectGraph(
      keys,
      normalizeOrganisation,
      (key, value, normalize) => {
        const lower = normalizeText(key).toLowerCase()
        if (
          (lower.includes('organisation') || lower.includes('organization')) &&
          (lower.includes('adv') || lower.includes('advert') || lower.includes('current'))
        ) {
          return normalize(value)
        }
        return ''
      },
    )
  }

  const readCompanyID = () => {
    const cookieValue = normalizeOrganisation(readCookie('sc_company_id') || readCookie('x-o3-company-id'))
    if (cookieValue) return cookieValue

    return findInObjectGraph(
      ['x-o3-company-id', 'x_o3_company_id', 'companyId', 'company_id', 'currentCompanyId'],
      normalizeOrganisation,
      (key, value, normalize) => {
        const lower = normalizeText(key).toLowerCase()
        if (lower.includes('company') && lower.includes('id')) {
          return normalize(value)
        }
        return ''
      },
    )
  }

  const readLanguage = () => {
    const cookieValue = normalizeNonEmpty(readCookie('x-o3-language'))
    if (cookieValue) return cookieValue

    return (
      findInObjectGraph(
        ['x-o3-language', 'x_o3_language', 'language', 'lang', 'locale'],
        normalizeNonEmpty,
        (key, value, normalize) => {
          const lower = normalizeText(key).toLowerCase()
          if (lower === 'language' || lower === 'lang' || lower === 'locale' || lower.endsWith('language')) {
            return normalize(value)
          }
          return ''
        },
      ) || 'zh-Hans'
    )
  }

  const readAppName = () => {
    return (
      findInObjectGraph(
        ['x-o3-app-name', 'x_o3_app_name', 'appName'],
        normalizeNonEmpty,
        (key, value, normalize) => {
          const lower = normalizeText(key).toLowerCase()
          if (lower === 'appname' || (lower.includes('app') && lower.includes('name'))) {
            return normalize(value)
          }
          return ''
        },
      ) || 'performance-sc'
    )
  }

  const cached = window.__ozonManagerSearchCPOContext && typeof window.__ozonManagerSearchCPOContext === 'object'
    ? window.__ozonManagerSearchCPOContext
    : {}

  const resolved = {
    companyId: normalizeOrganisation(cached.companyId) || readCompanyID(),
    advOrganisation: normalizeOrganisation(cached.advOrganisation) || readAdvOrganisation(),
    language: normalizeNonEmpty(cached.language) || readLanguage(),
    appName: normalizeNonEmpty(cached.appName) || readAppName(),
  }

  try {
    window.__ozonManagerSearchCPOContext = resolved
  } catch {
    // ignore page write failures
  }

  return resolved
}

async function scriptFetchSearchCPOProducts() {
  const normalizeText = (value) => {
    if (value === null || value === undefined) return ''
    return String(value).trim()
  }

  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }

  const normalizeOrganisation = (value) => {
    if (value && typeof value === 'object') {
      const direct = [
        value.id,
        value.organizationId,
        value.organisationId,
        value.currentOrganizationId,
        value.currentOrganisationId,
        value.companyId,
        value.value,
      ]
      for (const candidate of direct) {
        const candidateText = normalizeText(candidate)
        if (/^\d+$/.test(candidateText)) return candidateText
      }
    }

    const text = normalizeText(value)
    if (!text) return ''
    if (/^\d+$/.test(text)) return text
    try {
      const parsed = JSON.parse(text)
      return normalizeOrganisation(parsed)
    } catch {
      return ''
    }
  }

  const readAdvOrganisation = () => {
    const keys = [
      'advCurrentOrganisation',
      'adv_current_organisation',
      'adv_current_organization',
      'advertisementCurrentOrganisation',
      'advertisement_current_organisation',
      'x-adv-current-organisation',
    ]

    for (const key of keys) {
      const fromLocal = normalizeOrganisation(window.localStorage?.getItem(key))
      if (fromLocal) return fromLocal
      const fromSession = normalizeOrganisation(window.sessionStorage?.getItem(key))
      if (fromSession) return fromSession
    }

    const fromCookie = normalizeOrganisation(readCookie('adv_current_organisation'))
    if (fromCookie) return fromCookie
    return ''
  }

  const pad2 = (value) => String(value).padStart(2, '0')
  const buildTimeBounds = () => {
    const now = new Date()
    const nowUTC = now.getTime() + now.getTimezoneOffset() * 60000
    const nowMSK = new Date(nowUTC + 3 * 60 * 60 * 1000)
    const toDayUTC = Date.UTC(nowMSK.getUTCFullYear(), nowMSK.getUTCMonth(), nowMSK.getUTCDate() + 1)
    const fromDayUTC = toDayUTC - 7 * 24 * 60 * 60 * 1000

    const format = (ms) => {
      const d = new Date(ms)
      return `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}T00:00:00+03:00`
    }
    return { from: format(fromDayUTC), to: format(toDayUTC) }
  }

  const cachedContext = window.__ozonManagerSearchCPOContext && typeof window.__ozonManagerSearchCPOContext === 'object'
    ? window.__ozonManagerSearchCPOContext
    : {}

  const companyId = normalizeOrganisation(cachedContext.companyId) || normalizeOrganisation(readCookie('sc_company_id') || readCookie('x-o3-company-id'))
  const language = normalizeText(cachedContext.language || readCookie('x-o3-language') || 'zh-Hans') || 'zh-Hans'
  const advOrganisation = normalizeOrganisation(cachedContext.advOrganisation) || readAdvOrganisation()
  const appName = normalizeText(cachedContext.appName || 'performance-sc') || 'performance-sc'

  if (!companyId) {
    throw new Error('未找到 x-o3-company-id（sc_company_id），请先确认 Seller 登录态有效')
  }
  if (!advOrganisation) {
    throw new Error('未找到 x-adv-current-organisation，请先打开 Seller 推广按订单付费页面并完整加载后重试')
  }

  const requestHeaders = {
    accept: 'application/json, text/plain, */*',
    'content-type': 'application/json',
    'x-o3-app-name': String(appName),
    'x-o3-company-id': String(companyId),
    'x-o3-language': String(language),
    'x-adv-current-organisation': String(advOrganisation),
  }

  const endpoint = '/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/list'
  const bounds = buildTimeBounds()
  const pageSize = 50
  const allProducts = []
  let total = 0

  for (let page = 1; page <= 200; page += 1) {
    const body = {
      completeFilter: {
        filter: {
          searchPromoStatus: ['SEARCH_PROMO_STATUS_ENABLED', 'SEARCH_PROMO_STATUS_DISABLED'],
        },
        metricsTimeBounds: bounds,
      },
      page: String(page),
      pageSize: String(pageSize),
      sort: [{ desc: false, field: 'LIST_PRODUCTS_SORT_FIELD_INVALID' }],
    }

    const response = await fetch(endpoint, {
      method: 'POST',
      credentials: 'include',
      headers: requestHeaders,
      body: JSON.stringify(body),
      referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
    })

    if (!response.ok) {
      const text = await response.text().catch(() => '')
      throw new Error(`拉取 CPO 商品失败: ${response.status} ${response.statusText} ${text}`)
    }

    const data = await response.json()
    const products = Array.isArray(data?.products) ? data.products : []
    if (page === 1) {
      total = Math.max(0, Math.trunc(Number(data?.total || products.length || 0)))
    }
    if (products.length > 0) {
      allProducts.push(...products)
    }

    if (products.length === 0) break
    if (total > 0 && allProducts.length >= total) break
    if (products.length < pageSize) break

    await new Promise((resolve) => setTimeout(resolve, 80))
  }

  return allProducts
}

function buildSearchCPORequestHeadersInPage() {
  const normalizeText = (value) => {
    if (value === null || value === undefined) return ''
    return String(value).trim()
  }

  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }

  const normalizeOrganisation = (value) => {
    const text = normalizeText(value)
    if (!text) return ''
    return text
  }

  const cachedContext = window.__ozonManagerSearchCPOContext && typeof window.__ozonManagerSearchCPOContext === 'object'
    ? window.__ozonManagerSearchCPOContext
    : {}

  const companyId = normalizeOrganisation(cachedContext.companyId || readCookie('sc_company_id') || readCookie('x-o3-company-id'))
  const language = normalizeText(cachedContext.language || readCookie('x-o3-language') || 'zh-Hans') || 'zh-Hans'
  const advOrganisation = normalizeOrganisation(cachedContext.advOrganisation || readCookie('adv_current_organisation'))
  const appName = normalizeText(cachedContext.appName || 'performance-sc') || 'performance-sc'

  if (!companyId) {
    throw new Error('未找到 x-o3-company-id（sc_company_id），请先确认 Seller 登录态有效')
  }
  if (!advOrganisation) {
    throw new Error('未找到 x-adv-current-organisation，请先打开 Seller 推广按订单付费页面并完整加载后重试')
  }

  return {
    accept: 'application/json, text/plain, */*',
    'content-type': 'application/json',
    'x-o3-app-name': String(appName),
    'x-o3-company-id': String(companyId),
    'x-o3-language': String(language),
    'x-adv-current-organisation': String(advOrganisation),
  }
}

async function scriptFetchSearchCPOAvailability(sourceSKUs) {
  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => String(item || '').trim()).filter(Boolean) : []
  if (skus.length === 0) return { items: [] }
  const response = await fetch('/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/list/search_promo_availability', {
    method: 'POST',
    credentials: 'include',
    headers: buildSearchCPORequestHeadersInPage(),
    body: JSON.stringify({ skus }),
    referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
  })
  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(`拉取 CPO 可推广状态失败: ${response.status} ${response.statusText} ${text}`)
  }
  const data = await response.json().catch(() => ({}))
  return { data }
}

async function scriptEnableSearchCPOProducts(sourceSKUs) {
  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => String(item || '').trim()).filter(Boolean) : []
  if (skus.length === 0) return { items: [] }
  const response = await fetch('/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/enable', {
    method: 'POST',
    credentials: 'include',
    headers: buildSearchCPORequestHeadersInPage(),
    body: JSON.stringify({ productsSelectorSkus: { skus } }),
    referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
  })
  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(`开启 CPO 推广失败: ${response.status} ${response.statusText} ${text}`)
  }
  const data = await response.json().catch(() => ({}))
  return {
    message: String(data?.message || '').trim(),
    items: skus.map((sku) => ({ source_sku: sku, status: 'success' })),
    data,
  }
}

async function scriptBatchEnableSearchCPOMorkovsk(sourceSKUs) {
  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => String(item || '').trim()).filter(Boolean) : []
  if (skus.length === 0) return { items: [] }
  const response = await fetch('/performance-api/seller-api/search-performance-cpo/carrots/batch_enable', {
    method: 'POST',
    credentials: 'include',
    headers: buildSearchCPORequestHeadersInPage(),
    body: JSON.stringify({ productsSelectorSkus: { skus } }),
    referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
  })
  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(`加入 Morkovsk 失败: ${response.status} ${response.statusText} ${text}`)
  }
  const data = await response.json().catch(() => ({}))
  return {
    message: String(data?.message || '').trim(),
    items: skus.map((sku) => ({ source_sku: sku, status: 'success' })),
    data,
  }
}
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
