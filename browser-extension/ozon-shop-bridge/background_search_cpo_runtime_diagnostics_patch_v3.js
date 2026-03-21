const SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3 = '2026-03-21-a'

function runtimeV3DescribeSearchCPOValueKind(value) {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

function runtimeV3ReadSearchCPONumber(value, fallback = 0) {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : fallback
}

function runtimeV3ShortText(value, limit = 80) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  if (!text) return ''
  const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 80
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

async function runtimeV3FetchSearchCPOAvailabilityInPage(sourceSKUs) {
  const normalizeText = (value) => String(value || '').trim()
  const normalizeOrganisation = (value) => normalizeText(value)
  const trimExcerpt = (value, limit = 240) => {
    const text = String(value || '').replace(/\s+/g, ' ').trim()
    if (!text) return ''
    const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 240
    return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
  }
  const describeKind = (value) => {
    if (value === null) return 'null'
    if (Array.isArray(value)) return 'array'
    return typeof value
  }
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const buildContextExcerpt = () => {
    const parts = [
      `href=${trimExcerpt(window.location?.href || '', 120)}`,
      `readyState=${trimExcerpt(document?.readyState || '', 40)}`,
      `title=${trimExcerpt(document?.title || '', 80)}`,
    ]
    return parts.join(', ')
  }
  const buildHeaders = () => {
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

  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => normalizeText(item)).filter(Boolean) : []
  if (skus.length === 0) {
    return {
      data: {},
      ok: true,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'empty_request',
      response_length: 0,
      response_parse_error: '',
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }

  try {
    const response = await fetch('/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/list/search_promo_availability', {
      method: 'POST',
      credentials: 'include',
      headers: buildHeaders(),
      body: JSON.stringify({ skus }),
      referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
    })
    const contentType = String(response.headers?.get('content-type') || '').trim()
    const rawText = await response.text().catch(() => '')
    const trimmedText = String(rawText || '').trim()
    if (!response.ok) {
      return {
        data: {},
        ok: false,
        http_status: response.status,
        http_status_text: String(response.statusText || '').trim(),
        content_type: contentType,
        response_kind: 'http_error',
        response_length: trimmedText.length,
        response_parse_error: `HTTP ${response.status} ${String(response.statusText || '').trim()}`.trim(),
        response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
        script_result_type: 'object',
      }
    }
    let data = {}
    let parseError = ''
    let responseKind = trimmedText ? 'text' : 'empty'
    if (trimmedText) {
      try {
        data = JSON.parse(trimmedText)
        responseKind = describeKind(data)
      } catch (error) {
        parseError = String(error?.message || error || '').trim()
      }
    }
    return {
      data,
      ok: true,
      http_status: response.status,
      http_status_text: String(response.statusText || '').trim(),
      content_type: contentType,
      response_kind: responseKind,
      response_length: trimmedText.length,
      response_parse_error: parseError,
      response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
      script_result_type: 'object',
    }
  } catch (error) {
    return {
      data: {},
      ok: false,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'script_error',
      response_length: 0,
      response_parse_error: String(error?.message || error || '').trim(),
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }
}

async function runtimeV3ReadSearchCPOExecutionContext() {
  return {
    href: String(window.location?.href || '').trim(),
    ready_state: String(document?.readyState || '').trim(),
    title: String(document?.title || '').trim(),
  }
}

function runtimeV3BuildSyntheticAvailabilityRaw(rawValue, contextValue) {
  const context = contextValue && typeof contextValue === 'object' ? contextValue : {}
  const contextExcerpt = [
    `href=${runtimeV3ShortText(context?.href, 120)}`,
    `readyState=${runtimeV3ShortText(context?.ready_state, 40)}`,
    `title=${runtimeV3ShortText(context?.title, 80)}`,
  ].filter((item) => !item.endsWith('=')).join(', ')
  return {
    data: {},
    ok: false,
    http_status: 0,
    http_status_text: '',
    content_type: '',
    response_kind: 'script_result_missing',
    response_length: 0,
    response_parse_error: `executeScript returned ${runtimeV3DescribeSearchCPOValueKind(rawValue)}`,
    response_excerpt: contextExcerpt,
    script_result_type: runtimeV3DescribeSearchCPOValueKind(rawValue),
  }
}

function runtimeV3BuildSearchCPOAvailabilityDiagnostics(rawValue, sourceMap, sourceSKU, targetSKU, containers, availabilityMaps, reasonMaps) {
  return {
    parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
    build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
    requested_sku: targetSKU,
    source_sku: sourceSKU,
    response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
    sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
    availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
    reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
    response_http_status: runtimeV3ReadSearchCPONumber(sourceMap?.http_status ?? sourceMap?.response_http_status),
    response_http_status_text: firstNonEmptySearchCPOText(sourceMap?.http_status_text, sourceMap?.response_http_status_text),
    response_content_type: firstNonEmptySearchCPOText(sourceMap?.content_type, sourceMap?.response_content_type),
    response_parse_error: firstNonEmptySearchCPOText(sourceMap?.response_parse_error, sourceMap?.parse_error),
    response_excerpt: firstNonEmptySearchCPOText(sourceMap?.response_excerpt),
    response_length: runtimeV3ReadSearchCPONumber(sourceMap?.response_length),
    response_kind: firstNonEmptySearchCPOText(sourceMap?.response_kind),
    script_result_type: firstNonEmptySearchCPOText(sourceMap?.script_result_type, runtimeV3DescribeSearchCPOValueKind(rawValue)),
  }
}

function runtimeV3FormatSearchCPOAvailabilityError(kind, diagnostics) {
  const prefix = kind === 'missing_availability'
    ? 'search_promo_availability 响应缺少 availability 字段'
    : '未匹配到 search_promo_availability 响应'
  const parts = [
    `source_sku=${firstNonEmptySearchCPOText(diagnostics?.source_sku, '-')}`,
    `requested_sku=${firstNonEmptySearchCPOText(diagnostics?.requested_sku, '-')}`,
    `parser_revision=${firstNonEmptySearchCPOText(diagnostics?.parser_revision, '-')}`,
    `availability_keys=${diagnostics?.availability_map_key_count ?? 0}`,
    `reason_keys=${diagnostics?.reason_map_key_count ?? 0}`,
    `root_keys=${runtimeJoinSearchCPOKeys(diagnostics?.response_root_keys)}`,
    `sample_keys=${runtimeJoinSearchCPOKeys(diagnostics?.sample_response_keys)}`,
  ]
  const scriptResultType = firstNonEmptySearchCPOText(diagnostics?.script_result_type)
  if (scriptResultType) {
    parts.push(`result_type=${scriptResultType}`)
  }
  const responseKind = firstNonEmptySearchCPOText(diagnostics?.response_kind)
  if (responseKind) {
    parts.push(`response_kind=${responseKind}`)
  }
  const responseStatus = runtimeV3ReadSearchCPONumber(diagnostics?.response_http_status)
  if (responseStatus > 0) {
    parts.push(`http_status=${responseStatus}`)
  }
  const contentType = firstNonEmptySearchCPOText(diagnostics?.response_content_type)
  if (contentType) {
    parts.push(`content_type=${runtimeV3ShortText(contentType, 60)}`)
  }
  const parseError = firstNonEmptySearchCPOText(diagnostics?.response_parse_error)
  if (parseError) {
    parts.push(`parse_error=${runtimeV3ShortText(parseError, 80)}`)
  }
  return `${prefix} (${parts.join(', ')})`
}

function runtimeV3BuildSearchCPOStepDiagnostics(rawValue, sourceMap, sourceSKU, targetSKU, containers, responseItems) {
  return {
    parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
    build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
    requested_sku: targetSKU,
    source_sku: sourceSKU,
    response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
    sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
    response_http_status: runtimeV3ReadSearchCPONumber(sourceMap?.http_status ?? sourceMap?.response_http_status),
    response_http_status_text: firstNonEmptySearchCPOText(sourceMap?.http_status_text, sourceMap?.response_http_status_text),
    response_content_type: firstNonEmptySearchCPOText(sourceMap?.content_type, sourceMap?.response_content_type),
    response_parse_error: firstNonEmptySearchCPOText(sourceMap?.response_parse_error, sourceMap?.parse_error),
    response_excerpt: firstNonEmptySearchCPOText(sourceMap?.response_excerpt),
    response_length: runtimeV3ReadSearchCPONumber(sourceMap?.response_length),
    response_kind: firstNonEmptySearchCPOText(sourceMap?.response_kind),
    script_result_type: firstNonEmptySearchCPOText(sourceMap?.script_result_type, runtimeV3DescribeSearchCPOValueKind(rawValue)),
    response_item_count: Array.isArray(responseItems) ? responseItems.length : 0,
  }
}

function runtimeV3FormatSearchCPOStepError(prefix, diagnostics) {
  const parts = [
    `source_sku=${firstNonEmptySearchCPOText(diagnostics?.source_sku, '-')}`,
    `requested_sku=${firstNonEmptySearchCPOText(diagnostics?.requested_sku, '-')}`,
    `parser_revision=${firstNonEmptySearchCPOText(diagnostics?.parser_revision, '-')}`,
    `item_count=${diagnostics?.response_item_count ?? 0}`,
    `root_keys=${runtimeJoinSearchCPOKeys(diagnostics?.response_root_keys)}`,
    `sample_keys=${runtimeJoinSearchCPOKeys(diagnostics?.sample_response_keys)}`,
  ]
  const scriptResultType = firstNonEmptySearchCPOText(diagnostics?.script_result_type)
  if (scriptResultType) {
    parts.push(`result_type=${scriptResultType}`)
  }
  const responseKind = firstNonEmptySearchCPOText(diagnostics?.response_kind)
  if (responseKind) {
    parts.push(`response_kind=${responseKind}`)
  }
  const responseStatus = runtimeV3ReadSearchCPONumber(diagnostics?.response_http_status)
  if (responseStatus > 0) {
    parts.push(`http_status=${responseStatus}`)
  }
  const contentType = firstNonEmptySearchCPOText(diagnostics?.response_content_type)
  if (contentType) {
    parts.push(`content_type=${runtimeV3ShortText(contentType, 60)}`)
  }
  const parseError = firstNonEmptySearchCPOText(diagnostics?.response_parse_error)
  if (parseError) {
    parts.push(`parse_error=${runtimeV3ShortText(parseError, 80)}`)
  }
  return `${prefix} (${parts.join(', ')})`
}

function runtimeV3FindSearchCPOEnableItems(containers) {
  for (const container of containers || []) {
    if (Array.isArray(container?.bids)) return container.bids
    if (Array.isArray(container?.items)) return container.items
  }
  return []
}

function runtimeV3FindSearchCPOMorkovskItems(containers) {
  const results = []
  for (const container of containers || []) {
    if (runtimeSearchCPOIsPlainObject(container?.skuToInfo)) {
      results.push(...patchedCollectSearchCPOMapItems(container.skuToInfo))
    }
    if (runtimeSearchCPOIsPlainObject(container?.sku_to_info)) {
      results.push(...patchedCollectSearchCPOMapItems(container.sku_to_info))
    }
    if (Array.isArray(container?.items)) {
      results.push(...container.items)
    }
  }
  return results
}

normalizeSearchCPOAvailabilityItems = self.normalizeSearchCPOAvailabilityItems = function normalizeSearchCPOAvailabilityItemsRuntimePatchedV3(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const candidates = runtimeFindSearchCPOCandidateArray(containers)
  const candidateLookup = patchedBuildSearchCPOItemLookup(candidates)
  const directMaps = containers
  const availabilityMaps = runtimeCollectSearchCPONamedObjects(containers, [
    'skuToIsSearchPromoAvailable',
    'sku_to_is_search_promo_available',
  ])
  const reasonMaps = runtimeCollectSearchCPONamedObjects(containers, [
    'skuToIsSearchPromoAvailabilityWithReason',
    'sku_to_is_search_promo_availability_with_reason',
  ])

  return pairs.map(({ sourceSKU, targetSKU }) => {
    const diagnostics = runtimeV3BuildSearchCPOAvailabilityDiagnostics(
      raw,
      sourceMap,
      sourceSKU,
      targetSKU,
      containers,
      availabilityMaps,
      reasonMaps,
    )
    const keys = patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU)
    let matched = patchedCloneSearchCPOPlainObject(patchedFindSearchCPOItemByKeys(candidateLookup, keys))
    matched = patchedMergeSearchCPOPayload(matched, patchedFindSearchCPOMapValue(directMaps, keys))
    matched = patchedMergeSearchCPOPayload(matched, patchedFindSearchCPOMapValue(reasonMaps, keys))
    matched = patchedMergeSearchCPOPayload(matched, patchedFindSearchCPOMapValue(availabilityMaps, keys))
    if (matched) {
      if (!normalizeSKU(matched?.sku)) {
        matched.sku = firstNonEmptySearchCPOText(targetSKU, sourceSKU)
      }
      if (!normalizeSKU(matched?.source_sku) && !normalizeSKU(matched?.sourceSku)) {
        matched.source_sku = sourceSKU
      }
    }
    const availability = readSearchCPOAvailabilityValue(matched)
    const error = matched
      ? availability === null
        ? runtimeV3FormatSearchCPOAvailabilityError('missing_availability', diagnostics)
        : ''
      : runtimeV3FormatSearchCPOAvailabilityError('missing_match', diagnostics)
    return {
      source_sku: sourceSKU,
      sku: firstNonEmptySearchCPOText(matched?.sku, targetSKU, sourceSKU),
      search_promo_status: firstNonEmptySearchCPOText(matched?.searchPromoStatus, matched?.search_promo_status),
      carrots_status: firstNonEmptySearchCPOText(matched?.carrotsStatus, matched?.carrots_status),
      availability_promo: availability,
      error,
      payload: {
        ...(matched || {}),
        ...diagnostics,
      },
    }
  })
}

executeSyncSearchCPOAvailability = self.executeSyncSearchCPOAvailability = async function executeSyncSearchCPOAvailabilityRuntimePatchedV3(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  let raw = await runScript(tabID, runtimeV3FetchSearchCPOAvailabilityInPage, [requestSKUs])
  if (!raw || typeof raw !== 'object') {
    const context = await runScript(tabID, runtimeV3ReadSearchCPOExecutionContext, [])
    raw = runtimeV3BuildSyntheticAvailabilityRaw(raw, context)
  }
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const availabilityMaps = runtimeCollectSearchCPONamedObjects(containers, [
    'skuToIsSearchPromoAvailable',
    'sku_to_is_search_promo_available',
  ])
  const reasonMaps = runtimeCollectSearchCPONamedObjects(containers, [
    'skuToIsSearchPromoAvailabilityWithReason',
    'sku_to_is_search_promo_availability_with_reason',
  ])
  const items = normalizeSearchCPOAvailabilityItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, !row?.error, row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: {
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      request_skus: requestSKUs,
      response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
      sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
      availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
      reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
      items,
    },
  }
}

normalizeSearchCPOEnableItems = self.normalizeSearchCPOEnableItems = function normalizeSearchCPOEnableItemsRuntimePatchedV3(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const items = runtimeV3FindSearchCPOEnableItems(containers)
  const itemBySKU = patchedBuildSearchCPOItemLookup(items)
  const fallbackMessage = firstNonEmptySearchCPOText(sourceMap?.message, sourceMap?.data?.message)
  return pairs.map(({ sourceSKU, targetSKU }) => {
    const diagnostics = runtimeV3BuildSearchCPOStepDiagnostics(raw, sourceMap, sourceSKU, targetSKU, containers, items)
    const current = patchedFindSearchCPOItemByKeys(itemBySKU, patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU))
    const error = current
      ? patchedReadSearchCPOEnableError(current)
      : firstNonEmptySearchCPOText(fallbackMessage, runtimeV3FormatSearchCPOStepError('未匹配到 Search CPO 开启响应', diagnostics))
    return {
      source_sku: sourceSKU,
      sku: targetSKU,
      status: patchedResolveSearchCPOStepStatus(current, error),
      error,
      message: firstNonEmptySearchCPOText(current?.message, fallbackMessage, error),
      ...diagnostics,
    }
  })
}

normalizeSearchCPOMorkovskItems = self.normalizeSearchCPOMorkovskItems = function normalizeSearchCPOMorkovskItemsRuntimePatchedV3(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const infoMaps = runtimeCollectSearchCPONamedObjects(containers, ['skuToInfo', 'sku_to_info'])
  const items = runtimeV3FindSearchCPOMorkovskItems(containers)
  const itemBySKU = patchedBuildSearchCPOItemLookup(items)
  const fallbackMessage = firstNonEmptySearchCPOText(sourceMap?.message, sourceMap?.data?.message)
  return pairs.map(({ sourceSKU, targetSKU }) => {
    const diagnostics = runtimeV3BuildSearchCPOStepDiagnostics(raw, sourceMap, sourceSKU, targetSKU, containers, items)
    const keys = patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU)
    let current = patchedCloneSearchCPOPlainObject(patchedFindSearchCPOItemByKeys(itemBySKU, keys))
    current = patchedMergeSearchCPOPayload(current, patchedFindSearchCPOMapValue(infoMaps, keys))
    if (current) {
      if (!normalizeSKU(current?.sku)) {
        current.sku = firstNonEmptySearchCPOText(targetSKU, sourceSKU)
      }
      if (!normalizeSKU(current?.source_sku) && !normalizeSKU(current?.sourceSku)) {
        current.source_sku = sourceSKU
      }
    }
    const error = current
      ? patchedReadSearchCPOMorkovskError(current)
      : firstNonEmptySearchCPOText(fallbackMessage, runtimeV3FormatSearchCPOStepError('未匹配到 Search CPO Morkovsk 响应', diagnostics))
    return {
      source_sku: sourceSKU,
      sku: targetSKU,
      status: patchedResolveSearchCPOStepStatus(current, error),
      error,
      message: firstNonEmptySearchCPOText(current?.message, fallbackMessage, error),
      ...diagnostics,
    }
  })
}

executeSearchCPOEnableProducts = self.executeSearchCPOEnableProducts = async function executeSearchCPOEnableProductsRuntimePatchedV3(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  let raw = await runScript(tabID, runtimeV3FetchSearchCPOEnableInPage, [requestSKUs])
  if (!raw || typeof raw !== 'object') {
    const context = await runScript(tabID, runtimeV3ReadSearchCPOExecutionContext, [])
    raw = runtimeV3BuildSyntheticAvailabilityRaw(raw, context)
  }
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const items = normalizeSearchCPOEnableItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, String(row?.status || '') !== 'failed', row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: {
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      request_skus: requestSKUs,
      response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
      sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
      response_item_count: runtimeV3FindSearchCPOEnableItems(containers).length,
      items,
    },
  }
}

executeSearchCPOMorkovskBatchEnable = self.executeSearchCPOMorkovskBatchEnable = async function executeSearchCPOMorkovskBatchEnableRuntimePatchedV3(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  let raw = await runScript(tabID, runtimeV3FetchSearchCPOMorkovskInPage, [requestSKUs])
  if (!raw || typeof raw !== 'object') {
    const context = await runScript(tabID, runtimeV3ReadSearchCPOExecutionContext, [])
    raw = runtimeV3BuildSyntheticAvailabilityRaw(raw, context)
  }
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const containers = runtimeCollectSearchCPOAvailabilityContainers(sourceMap)
  const items = normalizeSearchCPOMorkovskItems(raw, skuPairs)
  const itemBySKU = Object.fromEntries(items.map((item) => [item.source_sku, item]))
  const results = skuPairs.map(({ sourceSKU }) => {
    const row = itemBySKU[sourceSKU]
    return makeSearchCPOStepJobResult(sourceSKU, String(row?.status || '') !== 'failed', row?.error || '')
  })
  return {
    status: summarizeStatus(results),
    results,
    meta: {
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
      request_skus: requestSKUs,
      response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
      sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
      response_item_count: runtimeV3FindSearchCPOMorkovskItems(containers).length,
      items,
    },
  }
}

async function runtimeV3FetchSearchCPOEnableInPage(sourceSKUs) {
  const normalizeText = (value) => String(value || '').trim()
  const normalizeOrganisation = (value) => normalizeText(value)
  const trimExcerpt = (value, limit = 240) => {
    const text = String(value || '').replace(/\s+/g, ' ').trim()
    if (!text) return ''
    const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 240
    return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
  }
  const describeKind = (value) => {
    if (value === null) return 'null'
    if (Array.isArray(value)) return 'array'
    return typeof value
  }
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const buildContextExcerpt = () => {
    const parts = [
      `href=${trimExcerpt(window.location?.href || '', 120)}`,
      `readyState=${trimExcerpt(document?.readyState || '', 40)}`,
      `title=${trimExcerpt(document?.title || '', 80)}`,
    ]
    return parts.join(', ')
  }
  const buildHeaders = () => {
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

  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => normalizeText(item)).filter(Boolean) : []
  if (skus.length === 0) {
    return {
      data: {},
      ok: true,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'empty_request',
      response_length: 0,
      response_parse_error: '',
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }

  try {
    const response = await fetch('/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/enable', {
      method: 'POST',
      credentials: 'include',
      headers: buildHeaders(),
      body: JSON.stringify({ productsSelectorSkus: { skus } }),
      referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
    })
    const contentType = String(response.headers?.get('content-type') || '').trim()
    const rawText = await response.text().catch(() => '')
    const trimmedText = String(rawText || '').trim()
    if (!response.ok) {
      return {
        data: {},
        ok: false,
        http_status: response.status,
        http_status_text: String(response.statusText || '').trim(),
        content_type: contentType,
        response_kind: 'http_error',
        response_length: trimmedText.length,
        response_parse_error: `HTTP ${response.status} ${String(response.statusText || '').trim()}`.trim(),
        response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
        script_result_type: 'object',
      }
    }
    let data = {}
    let parseError = ''
    let responseKind = trimmedText ? 'text' : 'empty'
    if (trimmedText) {
      try {
        data = JSON.parse(trimmedText)
        responseKind = describeKind(data)
      } catch (error) {
        parseError = String(error?.message || error || '').trim()
      }
    }
    return {
      data,
      ok: true,
      http_status: response.status,
      http_status_text: String(response.statusText || '').trim(),
      content_type: contentType,
      response_kind: responseKind,
      response_length: trimmedText.length,
      response_parse_error: parseError,
      response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
      script_result_type: 'object',
    }
  } catch (error) {
    return {
      data: {},
      ok: false,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'script_error',
      response_length: 0,
      response_parse_error: String(error?.message || error || '').trim(),
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }
}

async function runtimeV3FetchSearchCPOMorkovskInPage(sourceSKUs) {
  const normalizeText = (value) => String(value || '').trim()
  const normalizeOrganisation = (value) => normalizeText(value)
  const trimExcerpt = (value, limit = 240) => {
    const text = String(value || '').replace(/\s+/g, ' ').trim()
    if (!text) return ''
    const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 240
    return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
  }
  const describeKind = (value) => {
    if (value === null) return 'null'
    if (Array.isArray(value)) return 'array'
    return typeof value
  }
  const readCookie = (name) => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop().split(';').shift() || ''
    }
    return ''
  }
  const buildContextExcerpt = () => {
    const parts = [
      `href=${trimExcerpt(window.location?.href || '', 120)}`,
      `readyState=${trimExcerpt(document?.readyState || '', 40)}`,
      `title=${trimExcerpt(document?.title || '', 80)}`,
    ]
    return parts.join(', ')
  }
  const buildHeaders = () => {
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

  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => normalizeText(item)).filter(Boolean) : []
  if (skus.length === 0) {
    return {
      data: {},
      ok: true,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'empty_request',
      response_length: 0,
      response_parse_error: '',
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }

  try {
    const response = await fetch('/performance-api/seller-api/search-performance-cpo/carrots/batch_enable', {
      method: 'POST',
      credentials: 'include',
      headers: buildHeaders(),
      body: JSON.stringify({ productsSelectorSkus: { skus } }),
      referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
    })
    const contentType = String(response.headers?.get('content-type') || '').trim()
    const rawText = await response.text().catch(() => '')
    const trimmedText = String(rawText || '').trim()
    if (!response.ok) {
      return {
        data: {},
        ok: false,
        http_status: response.status,
        http_status_text: String(response.statusText || '').trim(),
        content_type: contentType,
        response_kind: 'http_error',
        response_length: trimmedText.length,
        response_parse_error: `HTTP ${response.status} ${String(response.statusText || '').trim()}`.trim(),
        response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
        script_result_type: 'object',
      }
    }
    let data = {}
    let parseError = ''
    let responseKind = trimmedText ? 'text' : 'empty'
    if (trimmedText) {
      try {
        data = JSON.parse(trimmedText)
        responseKind = describeKind(data)
      } catch (error) {
        parseError = String(error?.message || error || '').trim()
      }
    }
    return {
      data,
      ok: true,
      http_status: response.status,
      http_status_text: String(response.statusText || '').trim(),
      content_type: contentType,
      response_kind: responseKind,
      response_length: trimmedText.length,
      response_parse_error: parseError,
      response_excerpt: trimExcerpt(trimmedText) || buildContextExcerpt(),
      script_result_type: 'object',
    }
  } catch (error) {
    return {
      data: {},
      ok: false,
      http_status: 0,
      http_status_text: '',
      content_type: '',
      response_kind: 'script_error',
      response_length: 0,
      response_parse_error: String(error?.message || error || '').trim(),
      response_excerpt: buildContextExcerpt(),
      script_result_type: 'object',
    }
  }
}

registerExtension = self.registerExtension = async function registerExtensionRuntimePatchedV3(state) {
  await apiPost(
    state.apiBaseUrl,
    state.authToken,
    '/api/v1/extension/register',
    {
      shop_id: state.shopId,
      extension_id: state.extensionId,
      name: 'Chrome Extension',
      version: chrome.runtime.getManifest().version,
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
    },
  )
}

pollOnce = self.pollOnce = async function pollOnceRuntimePatchedV3() {
  if (pollInFlight) {
    return {
      ok: false,
      skipped: true,
      error: '当前已有轮询执行中',
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
        build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V3,
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


