const SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION = '2026-03-20-a'

function runtimeSearchCPOIsPlainObject(value) {
  if (typeof patchedIsSearchCPOPlainObject === 'function') {
    return patchedIsSearchCPOPlainObject(value)
  }
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function runtimeCollectSearchCPOAvailabilityContainers(sourceMap) {
  const queue = [sourceMap]
  const containers = []
  const seen = new Set()
  while (queue.length > 0 && containers.length < 16) {
    const current = queue.shift()
    if (!runtimeSearchCPOIsPlainObject(current) || seen.has(current)) {
      continue
    }
    seen.add(current)
    containers.push(current)
    for (const key of ['data', 'result', 'payload', 'response']) {
      const next = current[key]
      if (runtimeSearchCPOIsPlainObject(next) && !seen.has(next)) {
        queue.push(next)
      }
    }
  }
  return containers
}

function runtimeCollectSearchCPONamedObjects(containers, names) {
  const items = []
  for (const container of containers || []) {
    if (!runtimeSearchCPOIsPlainObject(container)) continue
    for (const name of names || []) {
      const value = container?.[name]
      if (runtimeSearchCPOIsPlainObject(value)) {
        items.push(value)
      }
    }
  }
  return items
}

function runtimeFindSearchCPOCandidateArray(containers) {
  for (const container of containers || []) {
    if (Array.isArray(container?.items)) return container.items
    if (Array.isArray(container?.products)) return container.products
  }
  return []
}

function runtimeCollectSearchCPOUniqueKeys(containers, limit) {
  const values = []
  const seen = new Set()
  const hardLimit = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : Number.POSITIVE_INFINITY
  for (const container of containers || []) {
    if (!runtimeSearchCPOIsPlainObject(container)) continue
    for (const key of Object.keys(container)) {
      const normalized = normalizeSKU(key)
      if (!normalized || seen.has(normalized)) continue
      seen.add(normalized)
      values.push(normalized)
      if (values.length >= hardLimit) {
        return values
      }
    }
  }
  return values
}

function runtimeJoinSearchCPOKeys(keys) {
  const values = Array.isArray(keys) ? keys.filter(Boolean) : []
  return values.length > 0 ? values.join('|') : '-'
}

function runtimeBuildSearchCPOAvailabilityDiagnostics(sourceMap, sourceSKU, targetSKU, containers, availabilityMaps, reasonMaps) {
  return {
    parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION,
    requested_sku: targetSKU,
    source_sku: sourceSKU,
    response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
    sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
    availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
    reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
  }
}

function runtimeFormatSearchCPOAvailabilityError(kind, diagnostics) {
  const prefix = kind === 'missing_availability'
    ? 'search_promo_availability 响应缺少 availability 字段'
    : '未匹配到 search_promo_availability 响应'
  return `${prefix} (source_sku=${firstNonEmptySearchCPOText(diagnostics?.source_sku, '-')}, requested_sku=${firstNonEmptySearchCPOText(diagnostics?.requested_sku, '-')}, parser_revision=${firstNonEmptySearchCPOText(diagnostics?.parser_revision, '-')}, availability_keys=${diagnostics?.availability_map_key_count ?? 0}, reason_keys=${diagnostics?.reason_map_key_count ?? 0}, root_keys=${runtimeJoinSearchCPOKeys(diagnostics?.response_root_keys)}, sample_keys=${runtimeJoinSearchCPOKeys(diagnostics?.sample_response_keys)})`
}

normalizeSearchCPOAvailabilityItems = self.normalizeSearchCPOAvailabilityItems = function normalizeSearchCPOAvailabilityItemsRuntimePatched(raw, skuPairs) {
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
    const diagnostics = runtimeBuildSearchCPOAvailabilityDiagnostics(
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
        ? runtimeFormatSearchCPOAvailabilityError('missing_availability', diagnostics)
        : ''
      : runtimeFormatSearchCPOAvailabilityError('missing_match', diagnostics)
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

executeSyncSearchCPOAvailability = self.executeSyncSearchCPOAvailability = async function executeSyncSearchCPOAvailabilityRuntimePatched(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptFetchSearchCPOAvailability, [requestSKUs])
  const containers = runtimeCollectSearchCPOAvailabilityContainers(raw && typeof raw === 'object' ? raw : {})
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
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION,
      parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION,
      request_skus: requestSKUs,
      response_root_keys: runtimeCollectSearchCPOUniqueKeys([raw && typeof raw === 'object' ? raw : {}], 8),
      sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
      availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
      reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
      items,
    },
  }
}

registerExtension = self.registerExtension = async function registerExtensionRuntimePatched(state) {
  await apiPost(
    state.apiBaseUrl,
    state.authToken,
    '/api/v1/extension/register',
    {
      shop_id: state.shopId,
      extension_id: state.extensionId,
      name: 'Chrome Extension',
      version: chrome.runtime.getManifest().version,
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION,
    },
  )
}
pollOnce = self.pollOnce = async function pollOnceRuntimePatched() {
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
        build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION,
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