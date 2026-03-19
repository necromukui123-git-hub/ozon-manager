function patchedPickFirstSearchCPOArray() {
  for (const value of arguments) {
    if (Array.isArray(value)) {
      return value
    }
  }
  return []
}

function patchedIsSearchCPOPlainObject(value) {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function patchedHasSearchCPOOwnKey(value, key) {
  return Object.prototype.hasOwnProperty.call(value, key)
}

function patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU) {
  const keys = []
  for (const value of [targetSKU, sourceSKU]) {
    const text = normalizeSKU(value)
    if (text && !keys.includes(text)) {
      keys.push(text)
    }
  }
  return keys
}

function patchedBuildSearchCPOItemLookup(items) {
  const itemBySKU = {}
  for (const item of items || []) {
    const keys = [
      item?.source_sku,
      item?.sourceSku,
      item?.sku,
      item?.offer_id,
      item?.product_id,
      item?.id,
    ]
      .map((value) => normalizeSKU(value))
      .filter(Boolean)
    for (const key of keys) {
      if (!itemBySKU[key]) {
        itemBySKU[key] = item
      }
    }
  }
  return itemBySKU
}

function patchedFindSearchCPOItemByKeys(lookup, keys) {
  const source = lookup && typeof lookup === 'object' ? lookup : {}
  for (const key of keys || []) {
    const text = normalizeSKU(key)
    if (text && source[text]) {
      return source[text]
    }
  }
  return null
}

function patchedFindSearchCPOMapValue(containers, keys) {
  for (const container of containers || []) {
    if (!patchedIsSearchCPOPlainObject(container)) continue
    for (const key of keys || []) {
      const text = normalizeSKU(key)
      if (text && patchedHasSearchCPOOwnKey(container, text)) {
        return container[text]
      }
    }
  }
  return undefined
}

function patchedReadSearchCPOLooseBoolean(value) {
  if (typeof value === 'boolean') return value
  if (value === 1 || value === '1' || value === 'true') return true
  if (value === 0 || value === '0' || value === 'false') return false
  return null
}

function patchedMergeSearchCPOPayload(base, value) {
  const nextValue = value === undefined ? null : value
  if (nextValue === null) {
    return base || null
  }
  if (patchedIsSearchCPOPlainObject(nextValue)) {
    return {
      ...(base || {}),
      ...nextValue,
    }
  }
  const boolValue = patchedReadSearchCPOLooseBoolean(nextValue)
  if (boolValue === null) {
    return base || null
  }
  return {
    ...(base || {}),
    isAvailable: boolValue,
  }
}

function patchedCloneSearchCPOPlainObject(value) {
  return patchedIsSearchCPOPlainObject(value) ? { ...value } : null
}

function patchedCollectSearchCPOMapItems(mapValue) {
  if (!patchedIsSearchCPOPlainObject(mapValue)) {
    return []
  }
  return Object.entries(mapValue).map(([sku, value]) => {
    if (patchedIsSearchCPOPlainObject(value)) {
      return { ...value, sku: firstNonEmptySearchCPOText(value?.sku, sku) }
    }
    return { sku, value }
  })
}

function patchedResolveSearchCPOStepStatus(current, error) {
  const status = normalizeSKU(current?.status).toLowerCase()
  if (status === 'skipped') return 'skipped'
  if (status === 'failed' || status === 'error') return 'failed'
  if (!current) return 'failed'
  return error ? 'failed' : 'success'
}

function patchedReadSearchCPOEnableError(item) {
  const status = normalizeSKU(item?.status).toLowerCase()
  if (status === 'failed' || status === 'error') {
    return firstNonEmptySearchCPOText(
      item?.error,
      item?.message,
      item?.errorReason,
      item?.error_reason,
      item?.unavailableReason,
      item?.unavailable_reason,
      '开启 CPO 推广失败',
    )
  }
  const explicitError = firstNonEmptySearchCPOText(item?.error)
  if (explicitError) {
    return explicitError
  }
  const errorReason = firstNonEmptySearchCPOText(item?.errorReason, item?.error_reason)
  if (errorReason && errorReason !== 'BID_ERROR_UNSPECIFIED') {
    return errorReason
  }
  const unavailableReason = firstNonEmptySearchCPOText(item?.unavailableReason, item?.unavailable_reason)
  if (unavailableReason && unavailableReason !== 'PROMOTION_UNAVAILABLE_REASON_UNSPECIFIED') {
    return unavailableReason
  }
  return ''
}

function patchedReadSearchCPOMorkovskError(item) {
  const status = normalizeSKU(item?.status).toLowerCase()
  if (status === 'failed' || status === 'error') {
    return firstNonEmptySearchCPOText(item?.error, item?.message, '加入 Morkovsk 失败')
  }
  const explicitError = firstNonEmptySearchCPOText(item?.error)
  if (explicitError && explicitError !== 'CARROT_ERROR_NONE') {
    return explicitError
  }
  const isEnabled = patchedReadSearchCPOLooseBoolean(item?.isEnabled ?? item?.is_enabled)
  if (isEnabled === false) {
    return firstNonEmptySearchCPOText(
      explicitError === 'CARROT_ERROR_NONE' ? '' : explicitError,
      item?.message,
      '加入 Morkovsk 失败',
    )
  }
  return ''
}

normalizeSearchCPOAvailabilityItems = self.normalizeSearchCPOAvailabilityItems = function normalizeSearchCPOAvailabilityItemsPatched(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const data = sourceMap?.data || sourceMap
  const candidates = patchedPickFirstSearchCPOArray(data?.items, data?.products)
  const candidateLookup = patchedBuildSearchCPOItemLookup(candidates)
  const directMaps = [data, data?.result]
  const availabilityMaps = [
    data?.skuToIsSearchPromoAvailable,
    data?.result?.skuToIsSearchPromoAvailable,
    data?.sku_to_is_search_promo_available,
    data?.result?.sku_to_is_search_promo_available,
  ]
  const reasonMaps = [
    data?.skuToIsSearchPromoAvailabilityWithReason,
    data?.result?.skuToIsSearchPromoAvailabilityWithReason,
    data?.sku_to_is_search_promo_availability_with_reason,
    data?.result?.sku_to_is_search_promo_availability_with_reason,
  ]

  return pairs.map(({ sourceSKU, targetSKU }) => {
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

function normalizeSearchCPOEnableItems(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const data = sourceMap?.data || sourceMap
  const items = patchedPickFirstSearchCPOArray(data?.bids, data?.items, sourceMap?.items)
  const itemBySKU = patchedBuildSearchCPOItemLookup(items)
  const fallbackMessage = firstNonEmptySearchCPOText(sourceMap?.message, data?.message)
  return pairs.map(({ sourceSKU, targetSKU }) => {
    const current = patchedFindSearchCPOItemByKeys(itemBySKU, patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU))
    const error = current
      ? patchedReadSearchCPOEnableError(current)
      : firstNonEmptySearchCPOText(fallbackMessage, `未匹配到 Search CPO 开启响应: ${targetSKU}`)
    return {
      source_sku: sourceSKU,
      sku: targetSKU,
      status: patchedResolveSearchCPOStepStatus(current, error),
      error,
      message: firstNonEmptySearchCPOText(current?.message, fallbackMessage, error),
    }
  })
}

function normalizeSearchCPOMorkovskItems(raw, skuPairs) {
  const pairs = Array.isArray(skuPairs) ? skuPairs : []
  const sourceMap = raw && typeof raw === 'object' ? raw : {}
  const data = sourceMap?.data || sourceMap
  const items = patchedPickFirstSearchCPOArray(
    patchedCollectSearchCPOMapItems(data?.skuToInfo),
    patchedCollectSearchCPOMapItems(data?.result?.skuToInfo),
    data?.items,
    sourceMap?.items,
  )
  const itemBySKU = patchedBuildSearchCPOItemLookup(items)
  const fallbackMessage = firstNonEmptySearchCPOText(sourceMap?.message, data?.message)
  return pairs.map(({ sourceSKU, targetSKU }) => {
    const current = patchedFindSearchCPOItemByKeys(itemBySKU, patchedBuildSearchCPOPairKeys(sourceSKU, targetSKU))
    const error = current
      ? patchedReadSearchCPOMorkovskError(current)
      : firstNonEmptySearchCPOText(fallbackMessage, `未匹配到 Search CPO Morkovsk 响应: ${targetSKU}`)
    return {
      source_sku: sourceSKU,
      sku: targetSKU,
      status: patchedResolveSearchCPOStepStatus(current, error),
      error,
      message: firstNonEmptySearchCPOText(current?.message, fallbackMessage, error),
    }
  })
}

readSearchCPOAvailabilityValue = self.readSearchCPOAvailabilityValue = function readSearchCPOAvailabilityValuePatched(item) {
  const candidates = [
    item?.promo,
    item?.isPromo,
    item?.availabilityPromo,
    item?.availability_promo,
    item?.available,
    item?.is_available,
    item?.isAvailable,
  ]
  for (const value of candidates) {
    const normalized = patchedReadSearchCPOLooseBoolean(value)
    if (normalized !== null) {
      return normalized
    }
  }
  return null
}

executeSearchCPOEnableProducts = self.executeSearchCPOEnableProducts = async function executeSearchCPOEnableProductsPatched(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptEnableSearchCPOProducts, [requestSKUs])
  const items = normalizeSearchCPOEnableItems(raw, skuPairs)
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

executeSearchCPOMorkovskBatchEnable = self.executeSearchCPOMorkovskBatchEnable = async function executeSearchCPOMorkovskBatchEnablePatched(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptBatchEnableSearchCPOMorkovsk, [requestSKUs])
  const items = normalizeSearchCPOMorkovskItems(raw, skuPairs)
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

scriptEnableSearchCPOProducts = self.scriptEnableSearchCPOProducts = async function scriptEnableSearchCPOProductsPatched(sourceSKUs) {
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
  return { data }
}

scriptBatchEnableSearchCPOMorkovsk = self.scriptBatchEnableSearchCPOMorkovsk = async function scriptBatchEnableSearchCPOMorkovskPatched(sourceSKUs) {
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
  return { data }
}
