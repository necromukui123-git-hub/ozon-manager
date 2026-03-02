const fs = require('fs')
const path = require('path')

const { loadSettings } = require('./flow-settings')

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function firstVisible(page, selectors, timeoutMs = 5000) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    for (const selector of selectors) {
      try {
        const locator = page.locator(selector).first()
        if (await locator.isVisible({ timeout: 300 })) {
          return locator
        }
      } catch (_) {}
    }
    await sleep(200)
  }
  return null
}

async function ensureLoggedIn(page) {
  const loginLocator = page.locator('input[name="login"], input[type="email"], input[type="password"]').first()
  const visible = await loginLocator.isVisible().catch(() => false)
  if (visible) {
    throw new Error('not logged in: please login manually using persistent browser profile')
  }
}

async function openAndSearchBySku(page, sectionConfig, sourceSKU, globalSettings) {
  const targetUrl = new URL(sectionConfig.url, globalSettings.sellerBaseUrl).toString()
  await page.goto(targetUrl, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await ensureLoggedIn(page)

  const searchInput = await firstVisible(page, sectionConfig.searchInputSelectors, globalSettings.searchTimeoutMs)
  if (!searchInput) {
    throw new Error(`search input not found on ${targetUrl}`)
  }

  await searchInput.fill('')
  await searchInput.type(sourceSKU, { delay: 30 })

  const searchSubmit = await firstVisible(page, sectionConfig.searchSubmitSelectors || [], 1500)
  if (searchSubmit) {
    await searchSubmit.click({ timeout: 2000 }).catch(() => {})
  } else {
    await page.keyboard.press('Enter').catch(() => {})
  }

  await sleep(1200)
}

async function clickFirstActionByText(page, actionSelectors, sourceSKU) {
  for (const selector of actionSelectors) {
    const locator = page.locator(selector).first()
    const visible = await locator.isVisible().catch(() => false)
    if (!visible) {
      continue
    }

    await locator.click({ timeout: 4000 })
    return
  }

  throw new Error(`action button not found for SKU ${sourceSKU}`)
}

async function fillFirstInput(page, selectors, value, label) {
  for (const selector of selectors) {
    const input = page.locator(selector).first()
    const visible = await input.isVisible().catch(() => false)
    if (!visible) {
      continue
    }
    await input.fill('')
    await input.type(String(value), { delay: 30 })
    return
  }

  throw new Error(`${label} input not found`)
}

async function safeScreenshot(page, artifactDir, fileName) {
  fs.mkdirSync(artifactDir, { recursive: true })
  const filePath = path.resolve(artifactDir, fileName)
  await page.screenshot({ path: filePath, fullPage: true }).catch(() => {})
  return filePath
}

async function executeExitPromotion(page, settings, input) {
  await openAndSearchBySku(page, settings.exitPromotion, input.sourceSKU, settings)
  await clickFirstActionByText(page, settings.exitPromotion.removeButtonSelectors, input.sourceSKU)

  const confirm = await firstVisible(page, settings.exitPromotion.confirmButtonSelectors, 4000)
  if (confirm) {
    await confirm.click({ timeout: 4000 })
  }
}

async function executeReprice(page, settings, input) {
  await openAndSearchBySku(page, settings.reprice, input.sourceSKU, settings)
  await fillFirstInput(page, settings.reprice.priceInputSelectors, input.targetPrice, 'price')

  const saveButton = await firstVisible(page, settings.reprice.saveButtonSelectors, 4000)
  if (!saveButton) {
    throw new Error('save button not found in reprice step')
  }

  await saveButton.click({ timeout: 4000 })
  await sleep(800)
}

async function executeReaddPromotion(page, settings, input) {
  await openAndSearchBySku(page, settings.readdPromotion, input.sourceSKU, settings)
  await clickFirstActionByText(page, settings.readdPromotion.addButtonSelectors, input.sourceSKU)

  const hasPromoPriceInput = Array.isArray(settings.readdPromotion.promoPriceInputSelectors)
    && settings.readdPromotion.promoPriceInputSelectors.length > 0

  if (hasPromoPriceInput) {
    await fillFirstInput(page, settings.readdPromotion.promoPriceInputSelectors, input.targetPrice, 'promo price')
  }

  const confirm = await firstVisible(page, settings.readdPromotion.confirmButtonSelectors, 4000)
  if (confirm) {
    await confirm.click({ timeout: 4000 })
  }
}

async function runActionFlow(context, input) {
  const settings = loadSettings()
  const page = await context.newPage()

  const artifactPrefix = `${Date.now()}_${input.sourceSKU}`

  let stepExitStatus = 'skipped'
  let stepRepriceStatus = 'skipped'
  let stepReaddStatus = 'skipped'
  let stepExitError = ''
  let stepRepriceError = ''
  let stepReaddError = ''

  try {
    await executeExitPromotion(page, settings, input)
    stepExitStatus = 'success'
  } catch (error) {
    stepExitStatus = 'failed'
    stepExitError = error?.message || 'exit promotion failed'
    const shot = await safeScreenshot(page, settings.artifactDir, `${artifactPrefix}_exit_failed.png`)
    stepExitError = `${stepExitError}; screenshot=${shot}`
  }

  if (stepExitStatus !== 'failed') {
    try {
      await executeReprice(page, settings, input)
      stepRepriceStatus = 'success'
    } catch (error) {
      stepRepriceStatus = 'failed'
      stepRepriceError = error?.message || 'reprice failed'
      const shot = await safeScreenshot(page, settings.artifactDir, `${artifactPrefix}_reprice_failed.png`)
      stepRepriceError = `${stepRepriceError}; screenshot=${shot}`
    }
  }

  if (stepExitStatus !== 'failed' && stepRepriceStatus !== 'failed') {
    try {
      await executeReaddPromotion(page, settings, input)
      stepReaddStatus = 'success'
    } catch (error) {
      stepReaddStatus = 'failed'
      stepReaddError = error?.message || 're-add promotion failed'
      const shot = await safeScreenshot(page, settings.artifactDir, `${artifactPrefix}_readd_failed.png`)
      stepReaddError = `${stepReaddError}; screenshot=${shot}`
    }
  }

  await page.close()

  const failed = [stepExitStatus, stepRepriceStatus, stepReaddStatus].some((status) => status === 'failed')

  return {
    source_sku: input.sourceSKU,
    overall_status: failed ? 'failed' : 'success',
    step_exit_status: stepExitStatus,
    step_reprice_status: stepRepriceStatus,
    step_readd_status: stepReaddStatus,
    step_exit_error: stepExitError,
    step_reprice_error: stepRepriceError,
    step_readd_error: stepReaddError,
  }
}

module.exports = {
  runActionFlow,
}

