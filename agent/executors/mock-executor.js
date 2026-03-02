function createMockExecutor() {
  return {
    name: 'mock-executor',

    async executeJob(job) {
      if (job.job_type === 'sync_shop_actions') {
        const results = [{
          source_sku: '__sync_shop_actions__',
          overall_status: 'success',
          step_exit_status: 'success',
          step_reprice_status: 'success',
          step_readd_status: 'success',
          step_exit_error: '',
          step_reprice_error: '',
          step_readd_error: '',
        }]
        results.__meta = {
          actions: [
            {
              source_action_id: 'shop-demo-1',
              title: '店铺促销(模拟) #1',
              action_type: 'SHOP_PRIVATE_PROMO',
              participating_products_count: 2,
              potential_products_count: 5,
            },
          ],
        }
        return results
      }

      if (job.job_type === 'sync_action_products') {
        const results = [{
          source_sku: '__sync_action_products__',
          overall_status: 'success',
          step_exit_status: 'success',
          step_reprice_status: 'success',
          step_readd_status: 'success',
          step_exit_error: '',
          step_reprice_error: '',
          step_readd_error: '',
        }]
        results.__meta = {
          items: [
            {
              source_sku: 'demo-sku-001',
              ozon_product_id: 10001,
              name: '店铺活动商品(模拟) 1',
              price: 99.9,
              action_price: 89.9,
              stock: 12,
              status: 'active',
            },
            {
              source_sku: 'demo-sku-002',
              ozon_product_id: 10002,
              name: '店铺活动商品(模拟) 2',
              price: 119.9,
              action_price: 109.9,
              stock: 8,
              status: 'active',
            },
          ],
        }
        return results
      }

      if (job.job_type === 'shop_action_declare' || job.job_type === 'promo_unified_enroll') {
        return (job.items || []).map((item) => ({
          source_sku: item.source_sku,
          overall_status: 'success',
          step_exit_status: 'skipped',
          step_reprice_status: 'skipped',
          step_readd_status: 'success',
          step_exit_error: '',
          step_reprice_error: '',
          step_readd_error: '',
        }))
      }

      if (job.job_type === 'shop_action_remove' || job.job_type === 'promo_unified_remove') {
        return (job.items || []).map((item) => ({
          source_sku: item.source_sku,
          overall_status: 'success',
          step_exit_status: 'success',
          step_reprice_status: 'skipped',
          step_readd_status: 'skipped',
          step_exit_error: '',
          step_reprice_error: '',
          step_readd_error: '',
        }))
      }

      return (job.items || []).map((item) => ({
        source_sku: item.source_sku,
        overall_status: 'success',
        step_exit_status: 'success',
        step_reprice_status: 'success',
        step_readd_status: 'success',
        step_exit_error: '',
        step_reprice_error: '',
        step_readd_error: '',
      }))
    },

    async close() {},
  }
}

module.exports = {
  createMockExecutor,
}
