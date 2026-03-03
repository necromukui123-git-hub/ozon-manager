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
  return response.state
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
    const state = await saveState()
    applyState(state)
    setStatus(`保存成功\n${$('status').textContent}`)
  } catch (error) {
    setStatus(`保存失败: ${error?.message || error}`)
  }
})
