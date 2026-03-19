function trimSearchCPOFetchResponseExcerpt(value, limit = 240) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  if (!text) return ''
  const maxLength = Number.isFinite(limit) && limit > 0 ? Math.trunc(limit) : 240
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

function describeSearchCPOFetchValueKind(value) {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

scriptFetchSearchCPOAvailability = self.scriptFetchSearchCPOAvailability = async function scriptFetchSearchCPOAvailabilityPatched(sourceSKUs) {
  const skus = Array.isArray(sourceSKUs) ? sourceSKUs.map((item) => String(item || '').trim()).filter(Boolean) : []
  if (skus.length === 0) return { items: [] }
  const response = await fetch('/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/list/search_promo_availability', {
    method: 'POST',
    credentials: 'include',
    headers: buildSearchCPORequestHeadersInPage(),
    body: JSON.stringify({ skus }),
    referrer: 'https://seller.ozon.ru/app/advertisement/product/cpo/selected',
  })
  const contentType = String(response.headers?.get('content-type') || '').trim()
  const rawText = await response.text().catch(() => '')
  if (!response.ok) {
    throw new Error(`拉取 CPO 可推广状态失败: ${response.status} ${response.statusText} ${trimSearchCPOFetchResponseExcerpt(rawText, 400)}`)
  }
  const trimmedText = String(rawText || '').trim()
  let data = {}
  let parseError = ''
  let responseKind = trimmedText ? 'text' : 'empty'
  if (trimmedText) {
    try {
      data = JSON.parse(trimmedText)
      responseKind = describeSearchCPOFetchValueKind(data)
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
    response_excerpt: trimSearchCPOFetchResponseExcerpt(trimmedText),
  }
}
