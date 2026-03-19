const SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2 = '2026-03-20-b'

function runtimeV2DescribeSearchCPOValueKind(value) {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

function runtimeV2ReadSearchCPONumber(value, fallback = 0) {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : fallback
}

function runtimeV2ShortText(value, limit = 80) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  if (!text) return ''
  const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 80
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

function runtimeV2BuildSearchCPOAvailabilityDiagnostics(rawValue, sourceMap, sourceSKU, targetSKU, containers, availabilityMaps, reasonMaps) {
  return {
    parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
    build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
    requested_sku: targetSKU,
    source_sku: sourceSKU,
    response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
    sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
    availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
    reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
    response_http_status: runtimeV2ReadSearchCPONumber(sourceMap?.http_status ?? sourceMap?.response_http_status),
    response_http_status_text: firstNonEmptySearchCPOText(sourceMap?.http_status_text, sourceMap?.response_http_status_text),
    response_content_type: firstNonEmptySearchCPOText(sourceMap?.content_type, sourceMap?.response_content_type),
    response_parse_error: firstNonEmptySearchCPOText(sourceMap?.response_parse_error, sourceMap?.parse_error),
    response_excerpt: firstNonEmptySearchCPOText(sourceMap?.response_excerpt),
    response_length: runtimeV2ReadSearchCPONumber(sourceMap?.response_length),
    response_kind: firstNonEmptySearchCPOText(sourceMap?.response_kind),
    script_result_type: runtimeV2DescribeSearchCPOValueKind(rawValue),
  }
}

function runtimeV2FormatSearchCPOAvailabilityError(kind, diagnostics) {
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
  const responseStatus = runtimeV2ReadSearchCPONumber(diagnostics?.response_http_status)
  if (responseStatus > 0) {
    parts.push(`http_status=${responseStatus}`)
  }
  const contentType = firstNonEmptySearchCPOText(diagnostics?.response_content_type)
  if (contentType) {
    parts.push(`content_type=${runtimeV2ShortText(contentType, 60)}`)
  }
  const parseError = firstNonEmptySearchCPOText(diagnostics?.response_parse_error)
  if (parseError) {
    parts.push(`parse_error=${runtimeV2ShortText(parseError, 80)}`)
  }
  return `${prefix} (${parts.join(', ')})`
}

normalizeSearchCPOAvailabilityItems = self.normalizeSearchCPOAvailabilityItems = function normalizeSearchCPOAvailabilityItemsRuntimePatchedV2(raw, skuPairs) {
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
    const diagnostics = runtimeV2BuildSearchCPOAvailabilityDiagnostics(
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
        ? runtimeV2FormatSearchCPOAvailabilityError('missing_availability', diagnostics)
        : ''
      : runtimeV2FormatSearchCPOAvailabilityError('missing_match', diagnostics)
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

executeSyncSearchCPOAvailability = self.executeSyncSearchCPOAvailability = async function executeSyncSearchCPOAvailabilityRuntimePatchedV2(job) {
  const { tabID } = await prepareSearchCPOJobExecution()
  const skuPairs = buildSearchCPOSkuPairs(job)
  const requestSKUs = skuPairs.map((pair) => pair.targetSKU)
  const raw = await runScript(tabID, scriptFetchSearchCPOAvailability, [requestSKUs])
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
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
      parser_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
      request_skus: requestSKUs,
      response_root_keys: runtimeCollectSearchCPOUniqueKeys([sourceMap], 8),
      sample_response_keys: runtimeCollectSearchCPOUniqueKeys(containers, 8),
      availability_map_key_count: runtimeCollectSearchCPOUniqueKeys(availabilityMaps).length,
      reason_map_key_count: runtimeCollectSearchCPOUniqueKeys(reasonMaps).length,
      items,
    },
  }
}

registerExtension = self.registerExtension = async function registerExtensionRuntimePatchedV2(state) {
  await apiPost(
    state.apiBaseUrl,
    state.authToken,
    '/api/v1/extension/register',
    {
      shop_id: state.shopId,
      extension_id: state.extensionId,
      name: 'Chrome Extension',
      version: chrome.runtime.getManifest().version,
      build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
    },
  )
}

pollOnce = self.pollOnce = async function pollOnceRuntimePatchedV2() {
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
        build_revision: SEARCH_CPO_RUNTIME_DIAGNOSTICS_REVISION_V2,
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


