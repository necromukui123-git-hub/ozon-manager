const SEARCH_CPO_REMOVE_RESULT_PATCH_REVISION = '2026-03-20-d'

function runtimeV4NormalizeSearchCPOText(value) {
  return String(value || '').trim()
}

function runtimeV4BuildSearchCPORemoveNotFoundMessage(sourceActionID, rawMessage) {
  const parts = ['action_not_found']
  const actionID = runtimeV4NormalizeSearchCPOText(sourceActionID)
  if (actionID) {
    parts.push(`source_action_id=${actionID}`)
  }
  const message = runtimeV4NormalizeSearchCPOText(rawMessage)
  if (message) {
    parts.push(message)
  }
  return `__SKIPPED__:${parts.join(': ')}`
}

function normalizeSearchCPORemoveOperationMessage(operation, sourceActionID, rawMessage) {
  const message = runtimeV4NormalizeSearchCPOText(rawMessage)
  if (operation !== 'remove' || !message) {
    return message
  }
  const lowered = message.toLowerCase()
  if (lowered.includes('商品当前不在活动中')) {
    return '__SKIPPED__:商品当前不在活动中'
  }
  const isActionNotFound = lowered.includes('action_not_found')
    || lowered.includes('status 404')
    || lowered.includes('404 not found')
    || lowered.includes('rpc error: code = notfound desc = resource not found')
    || (lowered.includes('resource not found') && lowered.includes('notfound'))
  if (isActionNotFound) {
    return runtimeV4BuildSearchCPORemoveNotFoundMessage(sourceActionID, message)
  }
  return message
}

normalizeSearchCPORemoveOperationMessage = self.normalizeSearchCPORemoveOperationMessage = normalizeSearchCPORemoveOperationMessage

executeActionOperation = self.executeActionOperation = async function executeActionOperationPatchedV4(tabID, sourceActionID, sourceSKUs, operation) {
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
    const message = normalizeSearchCPORemoveOperationMessage(operation, sourceActionID, error?.message || '调用店铺活动接口失败')
    for (const entry of matched) {
      errorsBySKU[entry.sourceSKU] = message
    }
  }

  return errorsBySKU
}

