export function getThemeChartTokens() {
  const styles = getComputedStyle(document.documentElement)

  return {
    color: [
      styles.getPropertyValue('--chart-color-1').trim() || '#1d4ed8',
      styles.getPropertyValue('--chart-color-2').trim() || '#dc2626',
      styles.getPropertyValue('--chart-color-3').trim() || '#f59e0b',
      styles.getPropertyValue('--chart-color-4').trim() || '#16a34a',
      styles.getPropertyValue('--chart-color-5').trim() || '#6b21a8'
    ],
    text: styles.getPropertyValue('--text-primary').trim() || '#171717',
    muted: styles.getPropertyValue('--text-muted').trim() || '#525252',
    border: styles.getPropertyValue('--neo-border-color').trim() || '#000000',
    surface: styles.getPropertyValue('--surface-bg').trim() || '#ffffff',
    primary: styles.getPropertyValue('--primary').trim() || '#1d4ed8'
  }
}

export function buildTheme() {
  const token = getThemeChartTokens()

  return {
    color: token.color,
    backgroundColor: 'transparent',
    textStyle: {
      fontFamily: "'Fira Sans', 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
      color: token.text
    },
    title: {
      textStyle: {
        color: token.text,
        fontWeight: 700,
        fontSize: 16
      },
      subtextStyle: {
        color: token.muted,
        fontSize: 12
      }
    },
    legend: {
      textStyle: {
        color: token.text
      }
    },
    tooltip: {
      backgroundColor: token.surface,
      borderColor: token.border,
      borderWidth: 2,
      borderRadius: 2,
      textStyle: {
        color: token.text
      },
      extraCssText: 'box-shadow: 4px 4px 0 #000;'
    },
    categoryAxis: {
      axisLine: {
        show: true,
        lineStyle: {
          color: token.border,
          width: 2
        }
      },
      axisTick: {
        show: false
      },
      axisLabel: {
        color: token.muted
      },
      splitLine: {
        show: false
      }
    },
    valueAxis: {
      axisLine: {
        show: false
      },
      axisTick: {
        show: false
      },
      axisLabel: {
        color: token.muted
      },
      splitLine: {
        lineStyle: {
          color: token.border,
          opacity: 0.2
        }
      }
    },
    line: {
      smooth: false,
      symbol: 'circle',
      symbolSize: 8,
      lineStyle: {
        width: 3
      },
      itemStyle: {
        borderWidth: 2,
        borderColor: token.border
      }
    },
    bar: {
      barMaxWidth: 40,
      itemStyle: {
        borderRadius: [0, 0, 0, 0],
        borderWidth: 2,
        borderColor: token.border
      }
    },
    pie: {
      itemStyle: {
        borderColor: token.border,
        borderWidth: 2
      },
      label: {
        color: token.text
      }
    }
  }
}

export function registerNeoTheme(echarts) {
  echarts.registerTheme('neo', buildTheme())
}

export function rebuildNeoTheme(echarts) {
  echarts.registerTheme('neo', buildTheme())
}

