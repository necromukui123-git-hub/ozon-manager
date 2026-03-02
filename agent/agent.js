require('dotenv').config()
const axios = require('axios')
const os = require('os')

const { createMockExecutor } = require('./executors/mock-executor')
const { createPlaywrightExecutor } = require('./executors/playwright-executor')

const baseURL = process.env.BASE_URL || 'http://127.0.0.1:8080'
const agentKey = process.env.AGENT_KEY || 'local-agent-001'
const agentName = process.env.AGENT_NAME || 'Local Agent'
const agentHostname = process.env.AGENT_HOSTNAME || os.hostname()
const pollIntervalMs = Number(process.env.POLL_INTERVAL_MS || 8000)
const mode = (process.env.AGENT_MODE || 'mock').toLowerCase()

const client = axios.create({
  baseURL,
  timeout: 30000,
})

let executor = null
let isRunning = false

function getExecutor() {
  if (executor) return executor

  if (mode === 'playwright') {
    executor = createPlaywrightExecutor()
  } else {
    executor = createMockExecutor()
  }

  return executor
}

async function heartbeat() {
  const currentExecutor = getExecutor()
  await client.post('/api/v1/automation/agent/heartbeat', {
    agent_key: agentKey,
    name: agentName,
    hostname: agentHostname,
    capabilities: {
      mode,
      browser: mode === 'playwright',
      version: 'm2.5',
      executor: currentExecutor.name,
    },
  })
}

async function pollJob() {
  const { data } = await client.post('/api/v1/automation/agent/poll', {
    agent_key: agentKey,
  })
  return data?.data?.job || null
}

async function reportJob(job, results, status) {
  let meta = {}
  if (results && results.__meta) {
    meta = results.__meta
  }
  const normalizedResults = Array.isArray(results) ? results : []
  await client.post('/api/v1/automation/agent/report', {
    agent_key: agentKey,
    job_id: job.job_id,
    status,
    results: normalizedResults,
    meta,
  })
}

function summarizeStatus(results) {
  const list = Array.isArray(results) ? results : []
  const failedCount = list.filter((item) => item.overall_status === 'failed').length
  if (failedCount === 0) return 'success'
  if (failedCount === list.length) return 'failed'
  return 'partial_success'
}

async function executeJob(job) {
  const currentExecutor = getExecutor()
  const results = await currentExecutor.executeJob(job)
  const status = summarizeStatus(results)
  await reportJob(job, results, status)
}

async function loop() {
  if (isRunning) return
  isRunning = true

  try {
    await heartbeat()
    const job = await pollJob()

    if (!job) {
      return
    }

    console.log(`[Agent] picked job #${job.job_id}, items=${job.items?.length || 0}`)
    await executeJob(job)
    console.log(`[Agent] reported job #${job.job_id}`)
  } catch (error) {
    const message = error?.response?.data?.message || error.message
    console.error('[Agent] loop error:', message)
  } finally {
    isRunning = false
  }
}

async function shutdown() {
  try {
    if (executor && typeof executor.close === 'function') {
      await executor.close()
    }
  } finally {
    process.exit(0)
  }
}

process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)

console.log(`[Agent] start: ${agentName} (${agentKey}) -> ${baseURL}, mode=${mode}`)
setInterval(loop, pollIntervalMs)
loop()
