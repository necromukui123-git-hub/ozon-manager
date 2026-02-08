import { ref } from 'vue'

export const THEMES = [
  {
    value: 'neo-prime',
    label: '三原色'
  },
  {
    value: 'neo-ozon',
    label: 'Ozon 蓝'
  },
  {
    value: 'neo-warm',
    label: '暖色'
  }
]

export const DEFAULT_THEME = 'neo-prime'
const STORAGE_KEY = 'ozon-theme'

const currentTheme = ref(DEFAULT_THEME)

function normalizeTheme(theme) {
  return THEMES.some((item) => item.value === theme) ? theme : DEFAULT_THEME
}

export function getTheme() {
  return currentTheme
}

export function applyTheme(theme) {
  const nextTheme = normalizeTheme(theme)
  const root = document.documentElement
  root.setAttribute('data-theme', nextTheme)
  localStorage.setItem(STORAGE_KEY, nextTheme)
  currentTheme.value = nextTheme
  window.dispatchEvent(new CustomEvent('ozon-theme-change', { detail: nextTheme }))
}

export function initTheme() {
  const saved = localStorage.getItem(STORAGE_KEY)
  applyTheme(saved || DEFAULT_THEME)
}

