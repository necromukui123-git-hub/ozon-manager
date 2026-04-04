export function formatMoney(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '-'
  return amount.toFixed(2)
}

export function formatPercent(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '0%'
  return `${amount.toFixed(2)}%`
}

export function statusLabel(status) {
  switch (status) {
    case 'success':
      return '成功'
    case 'partial_success':
      return '部分成功'
    case 'failed':
      return '失败'
    case 'running':
      return '执行中'
    case 'pending':
      return '等待中'
    case 'skipped':
      return '跳过'
    default:
      return status || '-'
  }
}

export function statusTagType(status) {
  switch (status) {
    case 'success':
      return 'success'
    case 'partial_success':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'running':
      return 'primary'
    case 'pending':
      return 'info'
    case 'skipped':
      return ''
    default:
      return 'info'
  }
}

export function actionResultLabel(item) {
  const title = String(item?.title || '').trim() || '未命名活动'
  const source = String(item?.source || '').trim()
  const sourceActionID = String(item?.source_action_id || '').trim()
  if (source === 'shop' && sourceActionID) {
    return `${title} (source_action_id=${sourceActionID})`
  }
  return title
}

export function availabilityLabel(value) {
  if (value === true) return '可进入下一阶段'
  if (value === false) return '暂不可推进'
  return '待检测'
}

export function availabilityTagType(value) {
  if (value === true) return 'success'
  if (value === false) return 'info'
  return 'warning'
}

export function joinAvailabilityDiagnosticKeys(keys) {
  return Array.isArray(keys) && keys.length > 0 ? keys.join(' | ') : '-'
}

export function formatAvailabilityHTTP(diagnostics) {
  const status = Number(diagnostics?.response_http_status || 0)
  const statusText = String(diagnostics?.response_http_status_text || '').trim()
  if (status > 0 && statusText) return `${status} ${statusText}`
  if (status > 0) return String(status)
  return '-'
}

export function searchPromoStatusLabel(status) {
  if (status === 'SEARCH_PROMO_STATUS_ENABLED') return '已开启'
  if (status === 'SEARCH_PROMO_STATUS_DISABLED') return '已关闭'
  return '状态未知'
}

export function searchPromoStatusTagType(status) {
  if (status === 'SEARCH_PROMO_STATUS_ENABLED') return 'success'
  if (status === 'SEARCH_PROMO_STATUS_DISABLED') return 'info'
  return 'warning'
}

export function carrotsStatusLabel(status) {
  if (status === 'CARROTS_STATUS_ENABLED') return 'Carrots 已开启'
  if (status === 'CARROTS_STATUS_DISABLED') return 'Carrots 已关闭'
  return status || 'Carrots 未知'
}

export function carrotsStatusTagType(status) {
  if (status === 'CARROTS_STATUS_ENABLED') return 'success'
  if (status === 'CARROTS_STATUS_DISABLED') return 'info'
  return 'warning'
}

export function ruleStateLabel(state) {
  switch (state) {
    case 'state1':
      return '状态1：推广已关闭'
    case 'state2':
      return '状态2：可加入推广'
    case 'state3':
      return '状态3：已加入推广，未加入 Morkovsk'
    case 'state3_trigger':
      return '状态3：已加入推广，未加入 Morkovsk'
    case 'state4':
      return '状态4：已加入推广，已加入 Morkovsk'
    case 'morkovsk_joined':
      return '状态4：已加入推广，已加入 Morkovsk'
    case 'other':
      return '其它'
    default:
      return state || '-'
  }
}

export function ruleStateTagType(state) {
  switch (state) {
    case 'state1':
      return 'warning'
    case 'state2':
      return 'primary'
    case 'state3':
    case 'state3_trigger':
      return 'info'
    case 'state4':
    case 'morkovsk_joined':
      return 'success'
    default:
      return 'info'
  }
}

export function triggerModeLabel(mode) {
  switch (mode) {
    case 'scheduled':
      return '自动触发'
    case 'manual':
      return '手动触发'
    default:
      return mode || '-'
  }
}
