(() => {
  const PUSH_INTERVAL_MS = 15000

  function parseShopID(raw) {
    const parsed = Number(raw || '')
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null
  }

  function pushAuthSnapshot() {
    try {
      const token = localStorage.getItem('token') || ''
      const shopID = parseShopID(localStorage.getItem('currentShopId'))
      chrome.runtime.sendMessage({
        type: 'OZON_MANAGER_AUTH_SYNC',
        payload: {
          token,
          shop_id: shopID,
          origin: window.location.origin,
        },
      })
    } catch (error) {
      // Content script should never break page rendering.
      console.debug('[OzonBridge] auth sync skipped:', error?.message || error)
    }
  }

  pushAuthSnapshot()

  window.addEventListener('storage', (event) => {
    if (event.key === 'token' || event.key === 'currentShopId') {
      pushAuthSnapshot()
    }
  })

  window.setInterval(pushAuthSnapshot, PUSH_INTERVAL_MS)
})()
