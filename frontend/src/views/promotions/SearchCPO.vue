<template>
  <div class="search-cpo-page">
    <div class="hero-shell">
      <div class="hero-copy">
        <h2 class="gradient">CPO 商品批量报名与状态迁移</h2>
        <p class="hero-subtitle">
          同一页面保留两条链路：人工按当前筛选结果批量报名，以及按 state1 到 state2 规则迁移到 Morkovsk。
        </p>
        <div class="hero-metrics">
          <div class="metric-pill">
            <span class="metric-label">缓存商品</span>
            <span class="metric-value">{{ products.length }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">当前筛选</span>
            <span class="metric-value">{{ filteredItems.length }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">默认活动</span>
            <span class="metric-value">{{ totalDefaultActions }}</span>
          </div>
          <div class="metric-pill">
            <span class="metric-label">自动化</span>
            <span class="metric-value metric-time">{{ automationStatusText }}</span>
          </div>
        </div>
      </div>
      <div class="page-actions">
        <el-button :loading="refreshing" @click="handleRefreshProducts">刷新 CPO 商品</el-button>
        <el-button :loading="savingConfig" @click="handleSaveConfig">保存默认活动</el-button>
        <el-button type="primary" :loading="running" :disabled="filteredItems.length === 0" @click="handleRunNow">
          报名当前筛选结果
        </el-button>
      </div>
    </div>

    <div class="bento-grid--2col">
      <BentoCard title="执行说明" :icon="InfoFilled" size="1x1">
        <div class="hint-list">
          <div>1. “报名当前筛选结果”只处理当前页面筛选出的商品，用于人工批量报名。</div>
          <div>2. “状态迁移自动化”会先刷新 CPO 商品，再同步 availability 后推进规则。</div>
          <div>3. State 1：加入已保存的默认活动，不在这一阶段执行 `enable`。</div>
          <div>4. State 2：先同步全部活动并退出已命中的官方/店铺活动，再执行 `enable` 和 `carrots/batch_enable`。</div>
          <div>5. State 3 兜底：用于收敛历史漏跑后已变成 enabled 的商品，继续走同一条迁移链路。</div>
        </div>
      </BentoCard>

      <BentoCard title="状态迁移自动化" :icon="Clock" size="1x1">
        <div class="automation-card">
          <div class="automation-target">
            <span class="metric-label">固定目标</span>
            <strong>Morkovsk</strong>
          </div>
          <el-form label-width="90px" class="automation-form">
            <el-form-item label="自动触发">
              <div class="inline-row">
                <el-switch v-model="config.auto_enabled" />
                <span class="inline-tip">{{ config.auto_enabled ? '已启用定时调度' : '当前仅支持手动触发' }}</span>
              </div>
            </el-form-item>
            <el-form-item label="触发时间">
              <div class="inline-row">
                <el-time-select
                  v-model="config.schedule_time"
                  start="00:00"
                  step="00:05"
                  end="23:55"
                  style="width: 160px"
                />
                <span class="inline-tip">按服务器分钟粒度执行</span>
              </div>
            </el-form-item>
          </el-form>
          <div class="automation-actions">
            <el-button type="primary" plain :loading="automationRunning" @click="handleStartAutomationRun">
              手动执行一次
            </el-button>
            <span class="inline-tip">自动触发和按钮手动触发共用同一条状态迁移链路。</span>
          </div>
        </div>
      </BentoCard>
    </div>

    <div class="bento-grid--2col actions-grid">
      <BentoCard title="官方活动" :icon="Flag" size="1x1">
        <el-empty v-if="!actionsLoading && officialActions.length === 0" description="暂无官方活动" />
        <el-checkbox-group v-else v-model="config.official_action_ids" class="action-group">
          <el-checkbox
            v-for="action in officialActions"
            :key="action.id"
            :value="action.id"
            class="action-checkbox"
          >
            <div class="action-item">
              <div class="action-title">{{ action.display_name || action.title || `活动 #${action.action_id}` }}</div>
              <div class="action-meta">官方 ID: {{ action.action_id }}</div>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </BentoCard>

      <BentoCard title="店铺活动" :icon="Discount" size="1x1">
        <el-empty v-if="!actionsLoading && shopActions.length === 0" description="暂无店铺活动" />
        <el-checkbox-group v-else v-model="config.shop_action_ids" class="action-group">
          <el-checkbox
            v-for="action in shopActions"
            :key="action.id"
            :value="action.id"
            class="action-checkbox"
          >
            <div class="action-item">
              <div class="action-title">{{ action.display_name || action.title || `活动 #${action.source_action_id}` }}</div>
              <div class="action-meta">店铺 ID: {{ action.source_action_id }}</div>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </BentoCard>
    </div>

    <BentoCard title="本地筛选" :icon="Filter" size="4x1">
      <el-form :inline="true" class="filter-form">
        <el-form-item label="关键词">
          <el-input
            v-model="filters.keyword"
            placeholder="标题 / source_sku / sku / 类目"
            clearable
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item label="推广状态">
          <el-select v-model="filters.promoStatus" style="width: 210px">
            <el-option label="全部" value="all" />
            <el-option label="已开启" value="SEARCH_PROMO_STATUS_ENABLED" />
            <el-option label="已关闭" value="SEARCH_PROMO_STATUS_DISABLED" />
          </el-select>
        </el-form-item>
        <el-form-item label="规则状态">
          <el-select v-model="filters.ruleState" style="width: 180px">
            <el-option label="全部" value="all" />
            <el-option label="State 1" value="state1" />
            <el-option label="State 2（迁移触发）" value="state2" />
            <el-option label="State 3 兜底" value="state3_trigger" />
            <el-option label="已加入 Morkovsk" value="morkovsk_joined" />
            <el-option label="其它 / 未识别" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="库存">
          <el-select v-model="filters.stockStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="有库存" value="in_stock" />
            <el-option label="无库存" value="out_of_stock" />
          </el-select>
        </el-form-item>
        <el-form-item label="收藏">
          <el-select v-model="filters.favoriteStatus" style="width: 140px">
            <el-option label="全部" value="all" />
            <el-option label="已收藏" value="favorite" />
            <el-option label="未收藏" value="not_favorite" />
          </el-select>
        </el-form-item>
      </el-form>
    </BentoCard>

    <BentoCard title="商品列表" :icon="Goods" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="pagedItems" v-loading="productsLoading">
          <el-table-column label="图片" width="88" align="center">
            <template #default="{ row }">
              <el-image
                v-if="row.image_url"
                :src="row.image_url"
                class="thumb"
                fit="cover"
                :preview-src-list="[row.image_url]"
                preview-teleported
              />
              <span v-else class="no-data">-</span>
            </template>
          </el-table-column>

          <el-table-column label="商品信息" min-width="320">
            <template #default="{ row }">
              <div class="name">{{ row.title || '-' }}</div>
              <div class="meta">source_sku: {{ row.source_sku || '-' }}</div>
              <div class="meta">sku: {{ row.sku || '-' }}</div>
              <div class="meta">类目: {{ row.category_name || '-' }}</div>
            </template>
          </el-table-column>

          <el-table-column label="价格 / 库存" min-width="160" align="center">
            <template #default="{ row }">
              <div class="meta">价格: {{ formatMoney(row.price) }}</div>
              <div class="meta">库存总量: {{ row.stock_total ?? 0 }}</div>
              <el-tag size="small" :type="row.is_in_stock ? 'success' : 'warning'">
                {{ row.is_in_stock ? '有库存' : '无库存' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="CPO 状态" width="190" align="center">
            <template #default="{ row }">
              <div class="status-stack">
                <el-tag :type="searchPromoStatusTagType(row.search_promo_status)">
                  {{ searchPromoStatusLabel(row.search_promo_status) }}
                </el-tag>
                <el-tag size="small" :type="carrotsStatusTagType(row.carrots_status)">
                  {{ carrotsStatusLabel(row.carrots_status) }}
                </el-tag>
                <el-tag size="small" :type="availabilityTagType(row.availability_promo)">
                  {{ availabilityLabel(row.availability_promo) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="规则状态" min-width="190" align="center">
            <template #default="{ row }">
              <div class="meta-block">
                <el-tag :type="ruleStateTagType(row.rule_state)">{{ ruleStateLabel(row.rule_state) }}</el-tag>
                <div class="meta" v-if="row.availability_checked_at">检测: {{ row.availability_checked_at }}</div>
                <div class="meta" v-if="row.state2_detected_at">State2: {{ row.state2_detected_at }}</div>
                <div class="meta" v-if="row.morkovsk_joined_at">Morkovsk: {{ row.morkovsk_joined_at }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="核心指标" min-width="180" align="center">
            <template #default="{ row }">
              <div class="meta">订单: {{ row.orders ?? 0 }}</div>
              <div class="meta">点击: {{ row.clicks ?? 0 }}</div>
              <div class="meta">花费: {{ formatMoney(row.spent) }}</div>
              <div class="meta">CTR: {{ formatPercent(row.ctr_percent) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <div class="pager-line">
          <span>共 {{ filteredItems.length }} 条，当前页 {{ pager.page }}</span>
          <el-pagination
            v-model:current-page="pager.page"
            v-model:page-size="pager.pageSize"
            :total="filteredItems.length"
            layout="prev, pager, next, sizes"
            :page-sizes="[20, 50, 100]"
            small
          />
        </div>
      </template>
    </BentoCard>

    <BentoCard title="手动报名历史" :icon="List" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="runs" v-loading="runsLoading">
          <el-table-column prop="id" label="任务ID" width="90" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="统计" min-width="260">
            <template #default="{ row }">
              <div class="meta">缓存 {{ row.total_fetched }} / 筛选 {{ row.total_selected }} / 处理 {{ row.total_processed }}</div>
              <div class="meta">成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column prop="error_message" label="错误摘要" min-width="250">
            <template #default="{ row }">
              <span class="error-text">{{ row.error_message || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button text type="primary" @click="openRunDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </BentoCard>

    <BentoCard title="状态迁移历史" :icon="Clock" size="4x1" no-padding>
      <div class="table-shell">
        <el-table :data="automationRuns" v-loading="automationRunsLoading">
          <el-table-column prop="id" label="任务ID" width="90" />
          <el-table-column label="触发方式" width="110">
            <template #default="{ row }">
              <el-tag size="small" :type="row.trigger_mode === 'scheduled' ? 'warning' : 'primary'">
                {{ triggerModeLabel(row.trigger_mode) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态统计" min-width="280">
            <template #default="{ row }">
              <div class="meta">抓取 {{ row.total_fetched }} / State1 {{ row.total_state1 }} / State2 迁移 {{ row.total_state2 }} / State3 兜底 {{ row.total_state3_trigger }}</div>
              <div class="meta">处理 {{ row.total_processed }} / 成功 {{ row.success_items }} / 失败 {{ row.failed_items }} / 跳过 {{ row.skipped_items }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column prop="error_message" label="错误摘要" min-width="250">
            <template #default="{ row }">
              <span class="error-text">{{ row.error_message || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button text type="primary" @click="openAutomationRunDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </BentoCard>

    <el-dialog v-model="detailVisible" title="手动报名详情" width="1100px">
      <div v-if="detail" class="detail-summary">
        <el-tag :type="statusTagType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
        <span>筛选商品 {{ detail.source_skus?.length || 0 }} 件</span>
        <span>成功 {{ detail.success_items }} / 失败 {{ detail.failed_items }} / 跳过 {{ detail.skipped_items }}</span>
      </div>

      <el-table v-if="detail" :data="detail.items || []" max-height="520" v-loading="detailLoading">
        <el-table-column prop="source_sku" label="source_sku" width="160" />
        <el-table-column prop="sku" label="sku" width="130" />
        <el-table-column prop="title" label="商品" min-width="220" />
        <el-table-column label="总状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="官方结果" min-width="220">
          <template #default="{ row }">
            <div class="result-lines">
              <div v-for="item in row.official_results" :key="`official-${row.id}-${item.promotion_action_id}`">
                {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.official_results || row.official_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="店铺结果" min-width="240">
          <template #default="{ row }">
            <div class="result-lines">
              <div v-for="item in row.shop_results" :key="`shop-${row.id}-${item.promotion_action_id}`">
                {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.shop_results || row.shop_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="automationDetailVisible" title="状态迁移详情" width="1320px">
      <div v-if="automationDetail" class="detail-summary detail-summary--wrap">
        <el-tag :type="statusTagType(automationDetail.status)">{{ statusLabel(automationDetail.status) }}</el-tag>
        <span>触发方式 {{ triggerModeLabel(automationDetail.trigger_mode) }}</span>
        <span>State1 {{ automationDetail.total_state1 }} / State2 迁移 {{ automationDetail.total_state2 }} / State3 兜底 {{ automationDetail.total_state3_trigger }}</span>
        <span>成功 {{ automationDetail.success_items }} / 失败 {{ automationDetail.failed_items }} / 跳过 {{ automationDetail.skipped_items }}</span>
      </div>

      <el-table v-if="automationDetail" :data="automationDetail.items || []" max-height="560" v-loading="automationDetailLoading">
        <el-table-column type="expand" width="48">
          <template #default="{ row }">
            <div class="availability-debug-panel">
              <div class="availability-debug-title">可推广状态诊断</div>
              <div class="availability-debug-grid">
                <div>
                  <span class="availability-debug-label">可推广状态</span>
                  <el-tag size="small" :type="availabilityTagType(row.availability_promo)">
                    {{ availabilityLabel(row.availability_promo) }}
                  </el-tag>
                </div>
                <div>
                  <span class="availability-debug-label">检查时间</span>
                  <span>{{ row.availability_checked_at || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">requested_sku</span>
                  <span>{{ row.availability_diagnostics?.requested_sku || row.sku || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">parser_revision</span>
                  <span>{{ row.availability_diagnostics?.parser_revision || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">build_revision</span>
                  <span>{{ row.availability_diagnostics?.build_revision || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">脚本结果类型</span>
                  <span>{{ row.availability_diagnostics?.script_result_type || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">响应类型</span>
                  <span>{{ row.availability_diagnostics?.response_kind || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">HTTP</span>
                  <span>{{ formatAvailabilityHTTP(row.availability_diagnostics) }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">Content-Type</span>
                  <span>{{ row.availability_diagnostics?.response_content_type || '-' }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">availability keys</span>
                  <span>{{ row.availability_diagnostics?.availability_map_key_count ?? 0 }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">reason keys</span>
                  <span>{{ row.availability_diagnostics?.reason_map_key_count ?? 0 }}</span>
                </div>
                <div>
                  <span class="availability-debug-label">业务原因</span>
                  <span>{{ row.availability_diagnostics?.unavailable_reason || '-' }}</span>
                </div>
              </div>
              <div class="availability-debug-block">
                <div class="availability-debug-label">root keys</div>
                <div>{{ joinAvailabilityDiagnosticKeys(row.availability_diagnostics?.response_root_keys) }}</div>
              </div>
              <div class="availability-debug-block">
                <div class="availability-debug-label">sample keys</div>
                <div>{{ joinAvailabilityDiagnosticKeys(row.availability_diagnostics?.sample_response_keys) }}</div>
              </div>
              <div v-if="row.availability_diagnostics?.response_parse_error" class="availability-debug-block">
                <div class="availability-debug-label">解析错误</div>
                <div class="error-text">{{ row.availability_diagnostics.response_parse_error }}</div>
              </div>
              <div v-if="row.availability_diagnostics?.response_excerpt" class="availability-debug-block">
                <div class="availability-debug-label">响应摘要</div>
                <pre class="availability-debug-pre">{{ row.availability_diagnostics.response_excerpt }}</pre>
              </div>
              <div v-if="!row.availability_diagnostics && !row.availability_checked_at" class="result-line-muted">
                当前没有可展示的 availability 诊断。
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="source_sku" label="source_sku" width="160" />
        <el-table-column prop="sku" label="sku" width="130" />
        <el-table-column prop="title" label="商品" min-width="220" />
        <el-table-column label="规则" width="170">
          <template #default="{ row }">
            <div class="result-lines">
              <div>进入前: {{ ruleStateLabel(row.rule_state_before) }}</div>
              <div>执行后: {{ ruleStateLabel(row.rule_state_after) }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="总状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.overall_status)">{{ statusLabel(row.overall_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="初始活动" min-width="240">
          <template #default="{ row }">
            <div class="result-lines">
              <div class="result-line-muted">步骤状态: {{ statusLabel(row.initial_status) }}</div>
              <div v-for="item in row.initial_results" :key="`init-${row.id}-${item.promotion_action_id}`">
                {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.initial_results || row.initial_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="退出其它活动" min-width="240">
          <template #default="{ row }">
            <div class="result-lines">
              <div class="result-line-muted">步骤状态: {{ statusLabel(row.exit_status) }}</div>
              <div v-for="item in row.exit_results" :key="`exit-${row.id}-${item.promotion_action_id}`">
                {{ actionResultLabel(item) }}: {{ statusLabel(item.status) }}<span v-if="item.error"> / {{ item.error }}</span>
              </div>
              <div v-if="!row.exit_results || row.exit_results.length === 0">-</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Enable" min-width="220">
          <template #default="{ row }">
            <div class="result-lines">
              <div>{{ statusLabel(row.enable_status) }}</div>
              <div v-if="row.enable_result?.message">{{ row.enable_result.message }}</div>
              <div v-if="row.enable_result?.error" class="error-text">{{ row.enable_result.error }}</div>
              <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
                requested_sku: {{ row.enable_result.diagnostics.requested_sku || '-' }} / parser_revision: {{ row.enable_result.diagnostics.parser_revision || '-' }}
              </div>
              <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
                HTTP: {{ formatAvailabilityHTTP(row.enable_result.diagnostics) }} / Content-Type: {{ row.enable_result.diagnostics.response_content_type || '-' }}
              </div>
              <div v-if="row.enable_result?.diagnostics" class="result-line-muted">
                root: {{ joinAvailabilityDiagnosticKeys(row.enable_result.diagnostics.response_root_keys) }} / sample: {{ joinAvailabilityDiagnosticKeys(row.enable_result.diagnostics.sample_response_keys) }}
              </div>
              <div v-if="row.enable_result?.diagnostics?.response_parse_error" class="error-text">解析错误: {{ row.enable_result.diagnostics.response_parse_error }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Morkovsk" min-width="220">
          <template #default="{ row }">
            <div class="result-lines">
              <div>{{ statusLabel(row.morkovsk_status) }}</div>
              <div v-if="row.morkovsk_result?.message">{{ row.morkovsk_result.message }}</div>
              <div v-if="row.morkovsk_result?.error" class="error-text">{{ row.morkovsk_result.error }}</div>
              <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
                requested_sku: {{ row.morkovsk_result.diagnostics.requested_sku || '-' }} / parser_revision: {{ row.morkovsk_result.diagnostics.parser_revision || '-' }}
              </div>
              <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
                HTTP: {{ formatAvailabilityHTTP(row.morkovsk_result.diagnostics) }} / Content-Type: {{ row.morkovsk_result.diagnostics.response_content_type || '-' }}
              </div>
              <div v-if="row.morkovsk_result?.diagnostics" class="result-line-muted">
                root: {{ joinAvailabilityDiagnosticKeys(row.morkovsk_result.diagnostics.response_root_keys) }} / sample: {{ joinAvailabilityDiagnosticKeys(row.morkovsk_result.diagnostics.sample_response_keys) }}
              </div>
              <div v-if="row.morkovsk_result?.diagnostics?.response_parse_error" class="error-text">解析错误: {{ row.morkovsk_result.diagnostics.response_parse_error }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="备注" min-width="180" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import {
  getActions,
  getSearchCPOAutomationRunDetail,
  getSearchCPOConfig,
  getSearchCPORunDetail,
  listSearchCPOAutomationRuns,
  listSearchCPOProducts,
  listSearchCPORuns,
  refreshSearchCPOProducts,
  startSearchCPOAutomationRun,
  startSearchCPORun,
  updateSearchCPOConfig
} from '@/api/promotion'
import { BentoCard } from '@/components/bento'
import { Clock, Discount, Filter, Flag, Goods, InfoFilled, List } from '@element-plus/icons-vue'

const userStore = useUserStore()

const actions = ref([])
const products = ref([])
const runs = ref([])
const automationRuns = ref([])
const detail = ref(null)
const automationDetail = ref(null)
const detailVisible = ref(false)
const automationDetailVisible = ref(false)

const actionsLoading = ref(false)
const productsLoading = ref(false)
const runsLoading = ref(false)
const automationRunsLoading = ref(false)
const detailLoading = ref(false)
const automationDetailLoading = ref(false)
const savingConfig = ref(false)
const refreshing = ref(false)
const running = ref(false)
const automationRunning = ref(false)

const config = reactive({
  official_action_ids: [],
  shop_action_ids: [],
  auto_enabled: false,
  schedule_time: '09:05'
})

const filters = reactive({
  keyword: '',
  promoStatus: 'all',
  ruleState: 'all',
  stockStatus: 'all',
  favoriteStatus: 'all'
})

const pager = reactive({
  page: 1,
  pageSize: 20
})

const lastSynced = ref('')
let runPollTimer = null

const officialActions = computed(() => actions.value.filter(action => action.source === 'official'))
const shopActions = computed(() => actions.value.filter(action => action.source === 'shop'))
const totalDefaultActions = computed(() => config.official_action_ids.length + config.shop_action_ids.length)
const automationStatusText = computed(() => {
  if (!config.auto_enabled) {
    return `未开启 / ${config.schedule_time || '09:05'}`
  }
  return `已开启 / ${config.schedule_time || '09:05'}`
})

const filteredItems = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  return products.value.filter(item => {
    if (filters.promoStatus !== 'all' && item.search_promo_status !== filters.promoStatus) return false
    if (filters.ruleState !== 'all' && (item.rule_state || 'other') !== filters.ruleState) return false
    if (filters.stockStatus === 'in_stock' && !item.is_in_stock) return false
    if (filters.stockStatus === 'out_of_stock' && item.is_in_stock) return false
    if (filters.favoriteStatus === 'favorite' && !item.is_favorite) return false
    if (filters.favoriteStatus === 'not_favorite' && item.is_favorite) return false
    if (!keyword) return true
    const blob = `${item.title || ''} ${item.source_sku || ''} ${item.sku || ''} ${item.category_name || ''}`.toLowerCase()
    return blob.includes(keyword)
  })
})

const pagedItems = computed(() => {
  const start = (pager.page - 1) * pager.pageSize
  return filteredItems.value.slice(start, start + pager.pageSize)
})

watch(filteredItems, () => {
  pager.page = 1
})

watch(
  () => userStore.currentShopId,
  () => {
    resetState()
    loadPageData()
  }
)

onMounted(() => {
  loadPageData()
})

onUnmounted(() => {
  stopRunPolling()
})

function resetState() {
  products.value = []
  runs.value = []
  automationRuns.value = []
  config.official_action_ids = []
  config.shop_action_ids = []
  config.auto_enabled = false
  config.schedule_time = '09:05'
  detail.value = null
  automationDetail.value = null
  detailVisible.value = false
  automationDetailVisible.value = false
  lastSynced.value = ''
  stopRunPolling()
}

async function loadPageData() {
  const shopId = userStore.currentShopId
  if (!shopId) return
  try {
    await Promise.all([
      loadActions(),
      loadConfig(),
      loadProducts(),
      loadRuns(),
      loadAutomationRuns()
    ])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '加载 CPO 页面失败')
  }
}

async function loadActions() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  actionsLoading.value = true
  try {
    const res = await getActions(shopId)
    actions.value = res.data || []
  } finally {
    actionsLoading.value = false
  }
}

async function loadConfig() {
  const shopId = userStore.currentShopId
  if (!shopId) return

  const res = await getSearchCPOConfig(shopId)
  const data = res.data || {}
  config.official_action_ids = Array.isArray(data.official_action_ids) ? data.official_action_ids : []
  config.shop_action_ids = Array.isArray(data.shop_action_ids) ? data.shop_action_ids : []
  config.auto_enabled = Boolean(data.auto_enabled)
  config.schedule_time = data.schedule_time || '09:05'
}

async function loadProducts(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) productsLoading.value = true
  try {
    const res = await listSearchCPOProducts(shopId)
    const data = res.data || {}
    products.value = data.items || []
    lastSynced.value = data.last_synced || ''
  } finally {
    if (!silent) productsLoading.value = false
  }
}

async function loadRuns(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) runsLoading.value = true
  try {
    const res = await listSearchCPORuns({ shop_id: shopId, page: 1, page_size: 20 })
    runs.value = res.data?.items || []
    updateRunPollingState()
  } finally {
    if (!silent) runsLoading.value = false
  }
}

async function loadAutomationRuns(silent = false) {
  const shopId = userStore.currentShopId
  if (!shopId) return

  if (!silent) automationRunsLoading.value = true
  try {
    const res = await listSearchCPOAutomationRuns({ shop_id: shopId, page: 1, page_size: 20 })
    automationRuns.value = res.data?.items || []
    updateRunPollingState()
  } finally {
    if (!silent) automationRunsLoading.value = false
  }
}

async function handleSaveConfig() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  savingConfig.value = true
  try {
    await updateSearchCPOConfig({
      shop_id: shopId,
      official_action_ids: config.official_action_ids,
      shop_action_ids: config.shop_action_ids,
      auto_enabled: config.auto_enabled,
      schedule_time: config.schedule_time || '09:05'
    })
    ElMessage.success('CPO 配置已保存')
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存 CPO 配置失败')
  } finally {
    savingConfig.value = false
  }
}

async function handleRefreshProducts() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }

  refreshing.value = true
  try {
    await refreshSearchCPOProducts({ shop_id: shopId })
    ElMessage.success('刷新完成')
    await loadProducts()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '刷新失败')
  } finally {
    refreshing.value = false
  }
}

async function handleRunNow() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }
  if (filteredItems.value.length === 0) {
    ElMessage.warning('当前筛选结果为空')
    return
  }
  if (totalDefaultActions.value === 0) {
    ElMessage.warning('请先选择活动并保存默认活动')
    return
  }

  try {
    await ElMessageBox.confirm(
      `将对当前筛选结果 ${filteredItems.value.length} 个商品执行报名，是否继续？`,
      '确认执行',
      {
        confirmButtonText: '开始执行',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  running.value = true
  try {
    await startSearchCPORun({
      shop_id: shopId,
      source_skus: filteredItems.value.map(item => item.source_sku).filter(Boolean),
      official_action_ids: config.official_action_ids,
      shop_action_ids: config.shop_action_ids
    })
    ElMessage.success('已创建 CPO 报名任务')
    await loadRuns()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '创建任务失败')
  } finally {
    running.value = false
  }
}

async function handleStartAutomationRun() {
  const shopId = userStore.currentShopId
  if (!shopId) {
    ElMessage.warning('请先选择店铺')
    return
  }
  if (totalDefaultActions.value === 0) {
    ElMessage.warning('请先选择默认活动并保存配置')
    return
  }

  try {
    await ElMessageBox.confirm(
      '自动化执行会刷新 CPO 商品、同步 availability，并按 state1 / state2 / state3 兜底规则推进迁移，是否继续？',
      '确认执行自动化',
      {
        confirmButtonText: '开始执行',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  automationRunning.value = true
  try {
    await startSearchCPOAutomationRun({ shop_id: shopId })
    ElMessage.success('已创建 CPO 自动化任务')
    await Promise.all([loadAutomationRuns(), loadProducts()])
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '创建自动化任务失败')
  } finally {
    automationRunning.value = false
  }
}

function updateRunPollingState() {
  const hasRunning = [...runs.value, ...automationRuns.value].some(row => ['pending', 'running'].includes(row.status))
  if (hasRunning) {
    startRunPolling()
  } else {
    stopRunPolling()
  }
}

function startRunPolling() {
  if (runPollTimer) return
  runPollTimer = setInterval(async () => {
    await Promise.allSettled([
      loadRuns(true),
      loadAutomationRuns(true),
      loadProducts(true)
    ])
  }, 2500)
}

function stopRunPolling() {
  if (runPollTimer) {
    clearInterval(runPollTimer)
    runPollTimer = null
  }
}

async function openRunDetail(row) {
  const shopId = userStore.currentShopId
  if (!shopId || !row?.id) return

  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res = await getSearchCPORunDetail(row.id, shopId)
    detail.value = res.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '获取运行详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function openAutomationRunDetail(row) {
  const shopId = userStore.currentShopId
  if (!shopId || !row?.id) return

  automationDetailVisible.value = true
  automationDetailLoading.value = true
  automationDetail.value = null
  try {
    const res = await getSearchCPOAutomationRunDetail(row.id, shopId)
    automationDetail.value = res.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '获取自动化详情失败')
  } finally {
    automationDetailLoading.value = false
  }
}

function formatMoney(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '-'
  return amount.toFixed(2)
}

function formatPercent(value) {
  const amount = Number(value || 0)
  if (!Number.isFinite(amount)) return '0%'
  return `${amount.toFixed(2)}%`
}

function statusLabel(status) {
  switch (status) {
    case 'success':
      return '成功'
    case 'partial_success':
      return '部分成功'
    case 'failed':
      return '失败'
    case 'running':
      return '执行中'
    case 'pending':
      return '等待中'
    case 'skipped':
      return '跳过'
    default:
      return status || '-'
  }
}

function statusTagType(status) {
  switch (status) {
    case 'success':
      return 'success'
    case 'partial_success':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'running':
      return 'primary'
    case 'pending':
      return 'info'
    case 'skipped':
      return ''
    default:
      return 'info'
  }
}

function actionResultLabel(item) {
  const title = String(item?.title || '').trim() || '未命名活动'
  const source = String(item?.source || '').trim()
  const sourceActionID = String(item?.source_action_id || '').trim()
  if (source === 'shop' && sourceActionID) {
    return `${title} (source_action_id=${sourceActionID})`
  }
  return title
}

function availabilityLabel(value) {
  if (value === true) return '可进入下一阶段'
  if (value === false) return '暂不可推进'
  return '待检测'
}

function availabilityTagType(value) {
  if (value === true) return 'success'
  if (value === false) return 'info'
  return 'warning'
}

function joinAvailabilityDiagnosticKeys(keys) {
  return Array.isArray(keys) && keys.length > 0 ? keys.join(' | ') : '-'
}

function formatAvailabilityHTTP(diagnostics) {
  const status = Number(diagnostics?.response_http_status || 0)
  const statusText = String(diagnostics?.response_http_status_text || '').trim()
  if (status > 0 && statusText) return `${status} ${statusText}`
  if (status > 0) return String(status)
  return '-'
}

function searchPromoStatusLabel(status) {
  if (status === 'SEARCH_PROMO_STATUS_ENABLED') return '已开启'
  if (status === 'SEARCH_PROMO_STATUS_DISABLED') return '已关闭'
  return '状态未知'
}

function searchPromoStatusTagType(status) {
  if (status === 'SEARCH_PROMO_STATUS_ENABLED') return 'success'
  if (status === 'SEARCH_PROMO_STATUS_DISABLED') return 'info'
  return 'warning'
}

function carrotsStatusLabel(status) {
  if (status === 'CARROTS_STATUS_ENABLED') return 'Carrots 已开启'
  if (status === 'CARROTS_STATUS_DISABLED') return 'Carrots 已关闭'
  return status || 'Carrots 未知'
}

function carrotsStatusTagType(status) {
  if (status === 'CARROTS_STATUS_ENABLED') return 'success'
  if (status === 'CARROTS_STATUS_DISABLED') return 'info'
  return 'warning'
}

function ruleStateLabel(state) {
  switch (state) {
    case 'state1':
      return 'State 1'
    case 'state2':
      return 'State 2（迁移触发）'
    case 'state3_trigger':
      return 'State 3 兜底'
    case 'morkovsk_joined':
      return '已加入 Morkovsk'
    case 'other':
      return '其它 / 未识别'
    default:
      return state || '-'
  }
}

function ruleStateTagType(state) {
  switch (state) {
    case 'state1':
      return 'warning'
    case 'state2':
      return 'primary'
    case 'state3_trigger':
      return 'danger'
    case 'morkovsk_joined':
      return 'success'
    default:
      return 'info'
  }
}

function triggerModeLabel(mode) {
  switch (mode) {
    case 'scheduled':
      return '自动触发'
    case 'manual':
      return '手动触发'
    default:
      return mode || '-'
  }
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Nunito+Sans:wght@400;500;600;700&family=Rubik:wght@500;600;700&display=swap');

.search-cpo-page {
  min-height: 100%;
  --cpo-primary: #059669;
  --cpo-secondary: #10b981;
  --cpo-cta: #f97316;
  --cpo-bg-soft: #ecfdf5;
  --cpo-text: #064e3b;
  --cpo-border: #c7f0dd;
  font-family: 'Nunito Sans', sans-serif;
  background:
    radial-gradient(1200px 440px at -8% -20%, rgba(16, 185, 129, 0.16), transparent 62%),
    radial-gradient(980px 440px at 108% -18%, rgba(249, 115, 22, 0.12), transparent 64%);
}

.hero-shell {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  padding: 18px 20px;
  border-radius: 16px;
  border: 1px solid var(--cpo-border);
  background: linear-gradient(120deg, rgba(5, 150, 105, 0.08), rgba(249, 115, 22, 0.08));
}

.hero-copy {
  flex: 1;
  min-width: 320px;
}

.hero-subtitle {
  margin: 8px 0 12px;
  line-height: 1.55;
  color: var(--cpo-text);
  opacity: 0.9;
}

.hero-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.metric-pill {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 62px;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(5, 150, 105, 0.16);
  transition: border-color 0.24s ease, transform 0.24s ease;
}

.metric-pill:hover {
  border-color: rgba(5, 150, 105, 0.36);
  transform: translateY(-1px);
}

.metric-label {
  color: #0f766e;
  font-size: 12px;
}

.metric-value {
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 15px;
  color: #064e3b;
}

.metric-time {
  font-size: 12px;
  line-height: 1.4;
}

.availability-debug-panel {
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(5, 150, 105, 0.05);
  border: 1px solid rgba(5, 150, 105, 0.12);
}

.availability-debug-title {
  margin-bottom: 10px;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  font-size: 14px;
  color: #065f46;
}

.availability-debug-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 18px;
  margin-bottom: 12px;
}

.availability-debug-grid > div,
.availability-debug-block {
  font-size: 13px;
  line-height: 1.6;
  color: #134e4a;
}

.availability-debug-label {
  display: inline-block;
  min-width: 112px;
  margin-right: 8px;
  color: #0f766e;
}

.availability-debug-block + .availability-debug-block {
  margin-top: 8px;
}

.availability-debug-pre {
  margin: 8px 0 0;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(15, 118, 110, 0.12);
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.5;
  color: #1f2937;
}

.hero-shell h2 {
  margin: 0;
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  max-width: 340px;
  justify-content: flex-end;
}

.page-actions :deep(.el-button) {
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.page-actions :deep(.el-button:hover) {
  transform: translateY(-1px);
}

.page-actions :deep(.el-button--primary) {
  box-shadow: 0 8px 18px rgba(249, 115, 22, 0.18);
}

.hint-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.automation-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.automation-target {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.76);
  border: 1px solid rgba(5, 150, 105, 0.14);
}

.automation-target strong {
  font-family: 'Rubik', 'Nunito Sans', sans-serif;
  color: #064e3b;
}

.automation-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.inline-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.inline-tip {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.automation-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.bento-grid--2col {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.actions-grid {
  margin-top: 6px;
}

@media (max-width: 992px) {
  .bento-grid--2col {
    grid-template-columns: 1fr;
  }
}

.action-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-checkbox {
  width: 100%;
  margin-right: 0;
}

.action-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.action-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-meta {
  color: var(--text-muted);
  font-size: 12px;
}

.filter-form {
  row-gap: 8px;
}

.table-shell {
  overflow-x: auto;
}

.thumb {
  width: 56px;
  height: 56px;
  border-radius: 6px;
}

.name {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.meta {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
}

.meta-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.status-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.no-data {
  color: var(--text-muted);
}

.pager-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.error-text {
  color: #d14343;
}

.detail-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.detail-summary--wrap {
  flex-wrap: wrap;
}

.result-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.result-line-muted {
  color: var(--text-muted);
}

@media (max-width: 1280px) {
  .hero-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 920px) {
  .hero-shell {
    flex-direction: column;
  }

  .page-actions {
    max-width: none;
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 600px) {
  .hero-metrics {
    grid-template-columns: 1fr;
  }

  .pager-line {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>

