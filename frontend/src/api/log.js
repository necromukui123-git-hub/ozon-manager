import request from '@/utils/request'

// 获取操作日志
export function getOperationLogs(params) {
  return request.get('/operation-logs', { params })
}
