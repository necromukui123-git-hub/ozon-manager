import request from '@/utils/request'

// 获取任务状态
export function getJobDetail(jobId, shopId) {
    return request.get(`/automation/jobs/${jobId}`, {
        params: { shop_id: shopId }
    })
}
