import request from '@/utils/request'


// 获取操作日志
export function getOperationLogs(params) {
  return request.get('/operation-logs', { params })
}

/**
 * 提交前端系统/错误日志
 */
export function createSystemLog(data) {
  return request({
    url: '/system/logs',
    method: 'post',
    data,
    silent: true // 增加标识，如果在 request.js 拦截器里报错不循环抛出
  }).catch(() => { })
}

