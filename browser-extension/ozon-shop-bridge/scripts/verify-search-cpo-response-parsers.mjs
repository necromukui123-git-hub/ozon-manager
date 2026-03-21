import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '../../..')

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

function loadPatchContext() {
  const sources = [
    'browser-extension/ozon-shop-bridge/background_search_cpo_response_patch.js',
    'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch.js',
    'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v2.js',
    'browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js',
  ].map((relativePath) => ({
    relativePath,
    source: readFileSync(path.join(repoRoot, relativePath), 'utf8'),
  }))
  const context = {
    console,
    self: {},
    normalizeSKU,
    firstNonEmptySearchCPOText,
  }
  context.self = context
  vm.createContext(context)
  for (const entry of sources) {
    new vm.Script(entry.source, { filename: path.basename(entry.relativePath) }).runInContext(context)
  }
  return context
}

function readDocResponse(relativePath) {
  const absolutePath = path.join(repoRoot, relativePath)
  const raw = readFileSync(absolutePath, 'utf8')
  const marker = '响应数据：'
  const index = raw.indexOf(marker)
  if (index < 0) {
    throw new Error(`未在 ${relativePath} 中找到“响应数据：”段落`)
  }
  return JSON.parse(raw.slice(index + marker.length).trim())
}

const context = loadPatchContext()
const {
  normalizeSearchCPOAvailabilityItems,
  normalizeSearchCPOEnableItems,
  normalizeSearchCPOMorkovskItems,
} = context

assert.equal(typeof normalizeSearchCPOAvailabilityItems, 'function', 'availability parser missing')
assert.equal(typeof normalizeSearchCPOEnableItems, 'function', 'enable parser missing')
assert.equal(typeof normalizeSearchCPOMorkovskItems, 'function', 'morkovsk parser missing')

const availabilityResponse = readDocResponse('doc/按订单付费推广商品操作/search_promo_availability.txt')
const availabilityPairs = Object.keys(availabilityResponse.skuToIsSearchPromoAvailable).map((sku) => ({
  sourceSKU: sku,
  targetSKU: sku,
}))
const availabilityItems = normalizeSearchCPOAvailabilityItems({ data: availabilityResponse }, availabilityPairs)
assert.equal(availabilityItems.length, availabilityPairs.length, 'availability item count mismatch')
const availabilityBySKU = Object.fromEntries(availabilityItems.map((item) => [item.source_sku, item]))
assert.equal(availabilityBySKU['3323213720'].availability_promo, true, 'expected sku 3323213720 availability=true')
assert.equal(availabilityBySKU['3328977168'].availability_promo, false, 'expected sku 3328977168 availability=false')
assert.equal(availabilityBySKU['3328977168'].payload.unavailableReason, 'PROMOTION_UNAVAILABLE_REASON_NO_SALES')
assert.equal(availabilityBySKU['3323213720'].payload.parser_revision, '2026-03-21-a')
assert.equal(availabilityBySKU['3323213720'].payload.build_revision, '2026-03-21-a')
assert.equal(availabilityBySKU['3323213720'].payload.requested_sku, '3323213720')
assert.ok(availabilityItems.every((item) => item.error === ''), 'availability parser should not report missing matches for fixture')

const envelopeItems = normalizeSearchCPOAvailabilityItems(
  { data: { response: availabilityResponse } },
  [{ sourceSKU: '3323213720', targetSKU: '3323213720' }],
)
assert.equal(envelopeItems[0].availability_promo, true, 'nested response envelope should still resolve availability')
assert.equal(envelopeItems[0].error, '', 'nested response envelope should not produce an error')

const mixedAvailability = normalizeSearchCPOAvailabilityItems(
  { data: availabilityResponse },
  [
    { sourceSKU: '3323213720', targetSKU: '3323213720' },
    { sourceSKU: 'offer-missing', targetSKU: '9999999999' },
  ],
)
assert.equal(mixedAvailability[0].error, '', 'known sku should still succeed in mixed response')
assert.match(mixedAvailability[1].error, /requested_sku=9999999999/)
assert.match(mixedAvailability[1].error, /parser_revision=2026-03-21-a/)
assert.match(mixedAvailability[1].error, /result_type=object/)

const missingAvailability = normalizeSearchCPOAvailabilityItems(
  {
    response_kind: 'text',
    response_content_type: 'text/html; charset=utf-8',
    response_parse_error: "Unexpected token '<'",
    response_excerpt: '<!doctype html><html>challenge</html>',
  },
  [{ sourceSKU: '999', targetSKU: '999' }],
)
assert.match(missingAvailability[0].error, /未匹配到 search_promo_availability 响应/)
assert.match(missingAvailability[0].error, /content_type=text\/html/)
assert.match(missingAvailability[0].error, /parse_error=Unexpected token/)
assert.equal(missingAvailability[0].payload.response_kind, 'text')
assert.equal(missingAvailability[0].payload.response_parse_error, "Unexpected token '<'")
assert.equal(missingAvailability[0].payload.response_excerpt, '<!doctype html><html>challenge</html>')

const enableResponse = readDocResponse('doc/按订单付费推广商品操作/enable.txt')
const enablePairs = enableResponse.bids.map((item) => ({ sourceSKU: item.sku, targetSKU: item.sku }))
const enableItems = normalizeSearchCPOEnableItems({ data: enableResponse }, enablePairs)
assert.ok(enableItems.every((item) => item.status === 'success' && item.error === ''), 'enable fixture should be all success')

const wrappedEnableItems = normalizeSearchCPOEnableItems(
  { data: { response: { result: enableResponse } } },
  [{ sourceSKU: enablePairs[0].sourceSKU, targetSKU: enablePairs[0].targetSKU }],
)
assert.equal(wrappedEnableItems[0].status, 'success', 'wrapped enable response should still resolve item')
assert.equal(wrappedEnableItems[0].error, '', 'wrapped enable response should not produce an error')
assert.equal(wrappedEnableItems[0].parser_revision, '2026-03-21-a')

const missingEnable = normalizeSearchCPOEnableItems(
  {
    response_kind: 'text',
    response_content_type: 'text/html; charset=utf-8',
    response_parse_error: "Unexpected token '<'",
    response_excerpt: '<!doctype html><html>challenge</html>',
  },
  [{ sourceSKU: '1966971285-N0wL', targetSKU: '3371928661' }],
)
assert.match(missingEnable[0].error, /未匹配到 Search CPO 开启响应/)
assert.match(missingEnable[0].error, /requested_sku=3371928661/)
assert.match(missingEnable[0].error, /source_sku=1966971285-N0wL/)
assert.match(missingEnable[0].error, /parser_revision=2026-03-21-a/)
assert.match(missingEnable[0].error, /content_type=text\/html/)
assert.match(missingEnable[0].error, /parse_error=Unexpected token/)

const failedEnable = normalizeSearchCPOEnableItems(
  {
    data: {
      bids: [{
        sku: '100',
        bid: 0,
        error: 'validation failed',
        errorReason: 'BID_ERROR_INVALID',
        unavailableReason: 'PROMOTION_UNAVAILABLE_REASON_NO_SALES',
      }],
    },
  },
  [{ sourceSKU: '100', targetSKU: '100' }],
)
assert.equal(failedEnable[0].status, 'failed', 'enable business failure should be marked failed')
assert.match(failedEnable[0].error, /validation failed/)

const morkovskResponse = readDocResponse('doc/按订单付费推广商品操作/batch_enable.txt')
const morkovskPairs = Object.keys(morkovskResponse.skuToInfo).map((sku) => ({ sourceSKU: sku, targetSKU: sku }))
const morkovskItems = normalizeSearchCPOMorkovskItems({ data: morkovskResponse }, morkovskPairs)
assert.ok(morkovskItems.every((item) => item.status === 'success' && item.error === ''), 'morkovsk fixture should be all success')

const wrappedMorkovskItems = normalizeSearchCPOMorkovskItems(
  { data: { response: { result: morkovskResponse } } },
  [{ sourceSKU: morkovskPairs[0].sourceSKU, targetSKU: morkovskPairs[0].targetSKU }],
)
assert.equal(wrappedMorkovskItems[0].status, 'success', 'wrapped morkovsk response should still resolve item')
assert.equal(wrappedMorkovskItems[0].error, '', 'wrapped morkovsk response should not produce an error')
assert.equal(wrappedMorkovskItems[0].parser_revision, '2026-03-21-a')

const failedMorkovsk = normalizeSearchCPOMorkovskItems(
  {
    data: {
      skuToInfo: {
        '200': {
          isEnabled: false,
          error: 'CARROT_ERROR_BID_TOO_LOW',
          bidPercent: 0,
        },
      },
    },
  },
  [{ sourceSKU: '200', targetSKU: '200' }],
)
assert.equal(failedMorkovsk[0].status, 'failed', 'morkovsk business failure should be marked failed')
assert.match(failedMorkovsk[0].error, /CARROT_ERROR_BID_TOO_LOW/)

console.log('Search CPO parser checks passed')




