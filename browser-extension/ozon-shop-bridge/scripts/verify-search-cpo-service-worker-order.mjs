import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '../../..')

function loadSource(relativePath) {
  return readFileSync(path.join(repoRoot, relativePath), 'utf8')
}

function createEvent() {
  return {
    addListener() {},
    removeListener() {},
    hasListener() { return false },
  }
}

const fetchCalls = []
const chrome = {
  runtime: {
    getManifest: () => ({ version: '0.1.0' }),
    id: 'test-extension',
    onInstalled: createEvent(),
    onMessage: createEvent(),
    onStartup: createEvent(),
    lastError: null,
  },
  storage: {
    local: { get: async () => ({}), set: async () => ({}), remove: async () => ({}) },
    session: { get: async () => ({}), set: async () => ({}), remove: async () => ({}) },
    onChanged: createEvent(),
  },
  alarms: { create() {}, clear: async () => true, get: async () => null, onAlarm: createEvent() },
  tabs: {
    query: async () => [],
    get: async () => null,
    update: async () => ({}),
    create: async () => ({}),
    remove: async () => ({}),
    sendMessage: async () => ({}),
    onRemoved: createEvent(),
    onUpdated: createEvent(),
  },
  scripting: {
    executeScript: async () => [{ result: null }],
    registerContentScripts: async () => ({}),
    unregisterContentScripts: async () => ({}),
  },
  webRequest: { onBeforeSendHeaders: createEvent() },
  permissions: { contains: async () => true, request: async () => true },
}

const context = {
  console,
  setTimeout,
  clearTimeout,
  setInterval,
  clearInterval,
  URL,
  URLSearchParams,
  TextEncoder,
  TextDecoder,
  atob: (input) => Buffer.from(input, 'base64').toString('binary'),
  btoa: (input) => Buffer.from(input, 'binary').toString('base64'),
  crypto: { randomUUID: () => 'uuid-fixed' },
  fetch: async (url, options = {}) => {
    fetchCalls.push({ url, options })
    return {
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: { get: () => 'application/json; charset=utf-8' },
      text: async () => JSON.stringify({ code: 200, data: {} }),
      json: async () => ({ code: 200, data: {} }),
    }
  },
  chrome,
  self: null,
}
context.self = context
vm.createContext(context)

const sources = [
  'browser-extension/ozon-shop-bridge/background_search_cpo.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_response_patch.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_fetch_diagnostics_patch.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v2.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js',
  'browser-extension/ozon-shop-bridge/background_search_cpo_remove_result_patch.js',
]
for (const relativePath of sources) {
  new vm.Script(loadSource(relativePath), { filename: path.basename(relativePath) }).runInContext(context)
}

assert.equal(typeof context.executeSyncSearchCPOAvailability, 'function', 'availability executor missing after worker load')
assert.equal(typeof context.registerExtension, 'function', 'registerExtension missing after worker load')
assert.match(String(context.normalizeSearchCPOAvailabilityItems), /runtimeV3FormatSearchCPOAvailabilityError/)
assert.equal(typeof context.normalizeSearchCPORemoveOperationMessage, 'function', 'remove message normalizer missing after worker load')
assert.match(
  context.normalizeSearchCPORemoveOperationMessage(
    'remove',
    '3231133',
    'API error (status 404): {"code":5,"message":"rpc error: code = NotFound desc = Resource not found"}',
  ),
  /^__SKIPPED__:action_not_found:/,
)

const availabilityFixture = {
  skuToIsSearchPromoAvailable: {
    '3323213720': true,
  },
  skuToIsSearchPromoAvailabilityWithReason: {
    '3323213720': {
      isAvailable: true,
      unavailableReason: 'PROMOTION_UNAVAILABLE_REASON_UNSPECIFIED',
    },
  },
}

context.prepareSearchCPOJobExecution = async () => ({ tabID: 1 })
context.runScript = async () => ({ data: { response: availabilityFixture } })

const execution = await context.executeSyncSearchCPOAvailability({
  items: [{ source_sku: 'offer-1' }],
  meta: { sku_map: { 'offer-1': '3323213720' } },
})
assert.equal(execution.status, 'success')
assert.equal(execution.meta.parser_revision, '2026-03-20-d')
assert.equal(execution.meta.build_revision, '2026-03-20-d')
assert.equal(execution.meta.items[0].availability_promo, true)
assert.equal(execution.meta.items[0].payload.requested_sku, '3323213720')
assert.equal(execution.meta.items[0].payload.parser_revision, '2026-03-20-d')
assert.equal(execution.meta.items[0].payload.build_revision, '2026-03-20-d')

await context.registerExtension({
  apiBaseUrl: 'http://127.0.0.1:8080',
  authToken: 'token',
  shopId: 2,
  extensionId: 'ext-1',
})
const registerCall = fetchCalls.find((call) => String(call.url).endsWith('/api/v1/extension/register'))
assert.ok(registerCall, 'register call missing')
const registerPayload = JSON.parse(registerCall.options.body)
assert.equal(registerPayload.version, '0.1.0')
assert.equal(registerPayload.build_revision, '2026-03-20-d')

fetchCalls.length = 0
context.readState = async () => ({
  enabled: true,
  apiBaseUrl: 'http://127.0.0.1:8080',
  authToken: 'token',
  shopId: 2,
  extensionId: 'ext-1',
})
context.saveStatePatch = async () => ({})
const pollResult = await context.pollOnce()
assert.equal(pollResult.ok, true)
const pollCall = fetchCalls.find((call) => String(call.url).endsWith('/api/v1/extension/poll'))
assert.ok(pollCall, 'poll call missing')
const pollPayload = JSON.parse(pollCall.options.body)
assert.equal(pollPayload.build_revision, '2026-03-20-d')

console.log('Search CPO service worker order checks passed')




