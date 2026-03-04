const $ = (id) => document.getElementById(id)

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
  $('enabled').checked = Boolean(state.enabled)
  $('apiBaseUrl').value = state.apiBaseUrl || ''
  $('adminOrigin').value = state.adminOrigin || ''
  $('token').value = state.authToken || ''
  $('shopId').value = state.shopId || ''

  const lines = [
    `enabled: ${state.enabled ? 'yes' : 'no'}`,
    `shop_id: ${state.shopId || '-'}`,
    `admin_origin: ${state.adminOrigin || '-'}`,
    `last_run: ${state.lastRunAt || '-'}`,
    `last_error: ${state.lastError || '-'}`,
  ]
  setStatus(lines.join('\n'))
}

async function loadState() {
  return chrome.runtime.sendMessage({ type: 'OZON_MANAGER_GET_STATE' })
}

async function saveState() {
  const payload = getFormData()
  const response = await chrome.runtime.sendMessage({
    type: 'OZON_MANAGER_SET_CONFIG',
    payload,
  })
  if (!response?.ok) {
    throw new Error(response?.error || 'save failed')
  }
  return response
}

function buildSaveSummary(sync) {
  if (!sync || sync.ok === undefined) {
    return '保存成功'
  }
  if (sync.ok) {
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

document.addEventListener('DOMContentLoaded', async () => {
  try {
    const state = await loadState()
    applyState(state)
  } catch (error) {
    setStatus(`加载失败: ${error?.message || error}`)
  }
})

$('saveBtn').addEventListener('click', async () => {
  try {
    const response = await saveState()
    applyState(response.state)
    const summary = buildSaveSummary(response.sync)
    setStatus(`${summary}\n${$('status').textContent}`)
  } catch (error) {
    setStatus(`保存失败: ${error?.message || error}`)
  }
})
