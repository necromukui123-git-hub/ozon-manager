const $ = (id) => document.getElementById(id)

function setConnection(text, tone = 'neutral') {
  const node = $('connection')
  node.textContent = text
  node.dataset.tone = tone
}

function setStatus(text) {
  $('status').textContent = text
}

function getFormData() {
  const shopID = Number($('shopId').value || 0)
  return {
    enabled: $('enabled').checked,
    apiBaseUrl: $('apiBaseUrl').value.trim(),
    adminOrigin: $('adminOrigin').value.trim(),
    authToken: $('token').value.trim(),
    shopId: Number.isFinite(shopID) && shopID > 0 ? shopID : null,
  }
}

function applyState(state) {
  $('enabled').checked = Boolean(state?.enabled)
  $('apiBaseUrl').value = state?.apiBaseUrl || ''
  $('adminOrigin').value = state?.adminOrigin || ''
  $('token').value = state?.authToken || ''
  $('shopId').value = state?.shopId || ''
}

function readStoredAuthSync(state) {
  return {
    auth_sync_status: state?.authSyncStatus || 'unknown',
    sync_source: state?.authSyncSource || 'none',
    admin_origin: state?.authSyncOrigin || '',
    token_present: Boolean(state?.authToken),
    shop_present: Boolean(state?.shopId),
    message: state?.authSyncMessage || '等待检测管理端连接状态',
    checked_at: state?.authSyncCheckedAt || '',
  }
}

function resolveAuthSync(state, authSync) {
  return authSync && authSync.auth_sync_status ? authSync : readStoredAuthSync(state)
}

async function loadState() {
  return chrome.runtime.sendMessage({ type: 'OZON_MANAGER_GET_STATE' })
}

async function checkAuthSync(requestPermission = false) {
  const response = await chrome.runtime.sendMessage({
    type: 'OZON_MANAGER_CHECK_AUTH_SYNC',
    payload: { requestPermission },
  })
  if (!response?.ok) {
    throw new Error(response?.error || 'auth sync check failed')
  }
  return response
}

async function saveState() {
  const response = await chrome.runtime.sendMessage({
    type: 'OZON_MANAGER_SET_CONFIG',
    payload: getFormData(),
  })
  if (!response?.ok) {
    throw new Error(response?.error || 'save failed')
  }
  return response
}

function buildConnectionView(auth) {
  const status = String(auth?.auth_sync_status || 'unknown').toLowerCase()
  const source = String(auth?.sync_source || 'none').toLowerCase()
  const message = String(auth?.message || '').trim() || '等待检测管理端连接状态'

  if (status === 'connected') {
    return { tone: 'success', text: message }
  }
  if (status === 'permission_required') {
    return { tone: 'warning', text: message }
  }
  if (status === 'missing_login' || status === 'missing_shop' || status === 'sync_unavailable') {
    return {
      tone: source === 'manual' ? 'warning' : 'danger',
      text: message,
    }
  }
  if (status === 'failed') {
    return { tone: 'danger', text: message }
  }
  return { tone: 'neutral', text: message }
}

function getSourceLabel(source) {
  if (source === 'auto') return '自动连接'
  if (source === 'manual') return '已保存配置兜底'
  return '未就绪'
}

function buildSaveSummary(sync) {
  if (!sync || sync.ok === undefined) {
    return '保存成功'
  }
  if (sync.ok) {
    if (sync.hasJob && String(sync.status || '').toLowerCase() === 'failed') {
      return `保存成功，但立即同步失败：${sync.error || '任务执行失败'}`
    }
    if (sync.hasJob) return '保存成功，已立即同步一次（有任务）'
    return '保存成功，已立即同步一次（当前无待执行任务）'
  }
  if (sync.skipped) {
    return `保存成功，未执行立即同步：${sync.error || '条件不满足'}`
  }

  const message = String(sync.error || '未知错误')
  if (message.includes('认证令牌已过期')) {
    return '保存成功，但立即同步失败：认证令牌已过期，请先在管理端重新登录'
  }
  return `保存成功，但立即同步失败：${message}`
}

function buildStatusText(state, auth, saveSummary = '') {
  const lines = []
  if (saveSummary) {
    lines.push(saveSummary)
    lines.push('')
  }

  lines.push(`连接来源: ${getSourceLabel(String(auth?.sync_source || 'none').toLowerCase())}`)
  lines.push(`管理端 Origin: ${auth?.admin_origin || state?.authSyncOrigin || state?.adminOrigin || '-'}`)
  lines.push(`当前 shop_id: ${state?.shopId || '-'}`)
  lines.push(`后端地址: ${state?.apiBaseUrl || '-'}`)
  lines.push(`最近检测: ${auth?.checked_at || state?.authSyncCheckedAt || '-'}`)
  lines.push(`last_run: ${state?.lastRunAt || '-'}`)
  lines.push(`last_error: ${state?.lastError || '-'}`)
  return lines.join('\n')
}

function render(state, authSync, saveSummary = '') {
  applyState(state)
  const auth = resolveAuthSync(state, authSync)
  const connection = buildConnectionView(auth)
  setConnection(connection.text, connection.tone)
  setStatus(buildStatusText(state, auth, saveSummary))
}

async function bootstrap() {
  const state = await loadState()
  render(state, null)
  const auth = await checkAuthSync(false)
  render(auth.state, auth.result)
}

async function handleDetectClick() {
  $('detectBtn').disabled = true
  setConnection('正在重新检测管理端连接状态...', 'neutral')
  try {
    const response = await checkAuthSync(true)
    render(response.state, response.result)
  } catch (error) {
    setConnection(`重新检测失败：${error?.message || error}`, 'danger')
  } finally {
    $('detectBtn').disabled = false
  }
}

async function handleSaveClick() {
  $('saveBtn').disabled = true
  setStatus('正在保存配置并执行一次立即同步...')
  try {
    const response = await saveState()
    render(response.state, response.authSync, buildSaveSummary(response.sync))
  } catch (error) {
    setStatus(`保存失败: ${error?.message || error}`)
  } finally {
    $('saveBtn').disabled = false
  }
}

document.addEventListener('DOMContentLoaded', async () => {
  try {
    await bootstrap()
  } catch (error) {
    setConnection(`加载失败：${error?.message || error}`, 'danger')
    setStatus('请检查插件后台是否正常运行。')
  }
})

$('detectBtn').addEventListener('click', handleDetectClick)
$('saveBtn').addEventListener('click', handleSaveClick)
