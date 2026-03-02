const fs = require('fs')
const path = require('path')

function getDefaultSettings() {
  return {
    sellerBaseUrl: process.env.OZON_SELLER_BASE_URL || 'https://seller.ozon.ru',
    artifactDir: process.env.ARTIFACT_DIR || path.resolve(__dirname, '..', 'artifacts'),
    searchTimeoutMs: Number(process.env.SEARCH_TIMEOUT_MS || 15000),

    common: {
      rowSelectors: ['tr', '[data-test-id*="row"]', '[class*="row"]'],
      actionButtonsSelectors: ['button', '[role="button"]', 'a'],
    },

    exitPromotion: {
      url: '/app/promotions',
      searchInputSelectors: [
        'input[placeholder*="SKU"]',
        'input[placeholder*="Артикул"]',
        'input[type="search"]',
      ],
      searchSubmitSelectors: [
        'button:has-text("Найти")',
        'button:has-text("Search")',
      ],
      removeButtonSelectors: [
        'button:has-text("Удалить")',
        'button:has-text("Исключить")',
        'button:has-text("Remove")',
      ],
      confirmButtonSelectors: [
        'button:has-text("Подтвердить")',
        'button:has-text("Удалить")',
        'button:has-text("Confirm")',
      ],
    },

    reprice: {
      url: '/app/prices',
      searchInputSelectors: [
        'input[placeholder*="SKU"]',
        'input[placeholder*="Артикул"]',
        'input[type="search"]',
      ],
      searchSubmitSelectors: [
        'button:has-text("Найти")',
        'button:has-text("Search")',
      ],
      priceInputSelectors: [
        'input[name*="price"]',
        'input[placeholder*="Цена"]',
        'input[type="number"]',
      ],
      saveButtonSelectors: [
        'button:has-text("Сохранить")',
        'button:has-text("Save")',
      ],
    },

    readdPromotion: {
      url: '/app/promotions',
      searchInputSelectors: [
        'input[placeholder*="SKU"]',
        'input[placeholder*="Артикул"]',
        'input[type="search"]',
      ],
      searchSubmitSelectors: [
        'button:has-text("Найти")',
        'button:has-text("Search")',
      ],
      addButtonSelectors: [
        'button:has-text("Добавить")',
        'button:has-text("Участвовать")',
        'button:has-text("Add")',
      ],
      promoPriceInputSelectors: [
        'input[name*="price"]',
        'input[placeholder*="Цена"]',
        'input[type="number"]',
      ],
      confirmButtonSelectors: [
        'button:has-text("Подтвердить")',
        'button:has-text("Сохранить")',
        'button:has-text("Confirm")',
      ],
    },
  }
}

function mergeDeep(base, extra) {
  if (!extra || typeof extra !== 'object') {
    return base
  }

  const result = { ...base }

  for (const key of Object.keys(extra)) {
    const current = extra[key]
    if (
      current &&
      typeof current === 'object' &&
      !Array.isArray(current) &&
      result[key] &&
      typeof result[key] === 'object' &&
      !Array.isArray(result[key])
    ) {
      result[key] = mergeDeep(result[key], current)
    } else {
      result[key] = current
    }
  }

  return result
}

function loadSettings() {
  const defaults = getDefaultSettings()
  const configPath = process.env.OZON_FLOW_CONFIG_PATH

  if (!configPath) {
    return defaults
  }

  const absolutePath = path.isAbsolute(configPath)
    ? configPath
    : path.resolve(process.cwd(), configPath)

  if (!fs.existsSync(absolutePath)) {
    throw new Error(`flow config not found: ${absolutePath}`)
  }

  const raw = fs.readFileSync(absolutePath, 'utf8')
  const parsed = JSON.parse(raw)
  return mergeDeep(defaults, parsed)
}

module.exports = {
  loadSettings,
}

