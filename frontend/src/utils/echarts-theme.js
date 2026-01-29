// ECharts Claude 风格主题配置
// 温暖米色/赭色配色方案

export const claudeTheme = {
  // 调色板
  color: [
    '#C4714E',  // 主色 - 赭色
    '#4A9668',  // 成功绿
    '#C4883A',  // 警告橙
    '#5A7BAF',  // 信息蓝
    '#D77757',  // 辅助色
    '#8B7355',  // 棕色
    '#6B8E7B',  // 灰绿
    '#A67B5B',  // 浅棕
  ],

  // 背景色
  backgroundColor: 'transparent',

  // 文字样式
  textStyle: {
    fontFamily: "'Inter', 'SF Pro Display', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
    color: '#5a5754'
  },

  // 标题
  title: {
    textStyle: {
      color: '#1a1a1a',
      fontWeight: 600,
      fontSize: 16
    },
    subtextStyle: {
      color: '#8a8780',
      fontSize: 12
    }
  },

  // 图例
  legend: {
    textStyle: {
      color: '#5a5754'
    },
    pageTextStyle: {
      color: '#8a8780'
    }
  },

  // 提示框
  tooltip: {
    backgroundColor: 'rgba(255, 255, 255, 0.96)',
    borderColor: 'rgba(0, 0, 0, 0.08)',
    borderWidth: 1,
    borderRadius: 8,
    textStyle: {
      color: '#1a1a1a'
    },
    extraCssText: 'box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);'
  },

  // 坐标轴
  categoryAxis: {
    axisLine: {
      show: true,
      lineStyle: {
        color: 'rgba(0, 0, 0, 0.08)'
      }
    },
    axisTick: {
      show: false
    },
    axisLabel: {
      color: '#8a8780'
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
      color: '#8a8780'
    },
    splitLine: {
      lineStyle: {
        color: 'rgba(0, 0, 0, 0.05)'
      }
    }
  },

  // 折线图
  line: {
    smooth: true,
    symbol: 'circle',
    symbolSize: 6,
    lineStyle: {
      width: 2
    },
    itemStyle: {
      borderWidth: 2
    },
    emphasis: {
      scale: true,
      focus: 'series'
    }
  },

  // 柱状图
  bar: {
    barMaxWidth: 40,
    itemStyle: {
      borderRadius: [4, 4, 0, 0]
    },
    emphasis: {
      focus: 'series'
    }
  },

  // 饼图
  pie: {
    itemStyle: {
      borderColor: '#ffffff',
      borderWidth: 2
    },
    label: {
      color: '#5a5754'
    },
    emphasis: {
      scale: true,
      scaleSize: 5
    }
  },

  // 数据区域缩放
  dataZoom: {
    backgroundColor: 'rgba(0, 0, 0, 0.02)',
    dataBackgroundColor: 'rgba(196, 113, 78, 0.1)',
    fillerColor: 'rgba(196, 113, 78, 0.15)',
    handleColor: '#C4714E',
    handleSize: '100%',
    textStyle: {
      color: '#8a8780'
    }
  }
}

// 注册主题的辅助函数
export function registerClaudeTheme(echarts) {
  echarts.registerTheme('claude', claudeTheme)
}

// 通用图表配置
export const chartConfig = {
  // 响应式配置
  responsive: {
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    }
  },

  // 动画配置
  animation: {
    animationDuration: 800,
    animationEasing: 'cubicOut'
  }
}
