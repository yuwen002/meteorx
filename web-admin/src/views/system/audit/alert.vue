<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon rule-icon">
              <el-icon><Bell /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ t('alertPage.stat.rules') }}</div>
              <div class="stat-value">{{ rules.length }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon enabled-icon">
              <el-icon><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ t('alertPage.stat.enabled') }}</div>
              <div class="stat-value">{{ enabledRulesCount }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon alert-icon">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ t('alertPage.stat.todayAlerts') }}</div>
              <div class="stat-value">{{ alertStats.today_alerts }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon notified-icon">
              <el-icon><Message /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ t('alertPage.stat.notifiedPending') }}</div>
              <div class="stat-value">{{ alertStats.notified_count }} / {{ alertStats.pending_count }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 告警趋势 + 风险分布 -->
    <el-row :gutter="16" class="charts-row">
      <el-col :span="16">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span>{{ t('alertPage.trend.title', statsDays) }}</span>
              <el-radio-group v-model="statsDays" size="small" @change="loadAlertStats">
                <el-radio-button :label="7">{{ t('alertPage.trend.day7') }}</el-radio-button>
                <el-radio-button :label="14">{{ t('alertPage.trend.day14') }}</el-radio-button>
                <el-radio-button :label="30">{{ t('alertPage.trend.day30') }}</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div class="trend-chart">
            <div class="trend-y-axis">
              <span class="y-max">{{ trendMax }}</span>
              <span class="y-mid">{{ Math.round(trendMax / 2) }}</span>
              <span class="y-min">0</span>
            </div>
            <div class="trend-bars">
              <div v-for="point in alertStats.trend" :key="point.date" class="trend-bar-item">
                <div class="bar" :style="{ height: trendBarHeight(point.count) }"></div>
                <div class="label">{{ point.date.slice(5) }}</div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header><span>{{ t('alertPage.riskDist.title') }}</span></template>
          <div class="risk-dist">
            <div v-for="(count, level) in alertStats.risk_level_stats" :key="level" class="risk-item">
              <span class="risk-label">{{ getRiskLevelLabel(level) }}</span>
              <el-progress
                :percentage="riskPercentage(count)"
                :stroke-width="18"
                :color="getRiskColor(level)"
              />
              <span class="risk-count">{{ count }}</span>
            </div>
            <el-empty v-if="Object.keys(alertStats.risk_level_stats).length === 0" :description="t('alertPage.noData')" :image-size="40" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 告警规则管理 -->
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>{{ t('alertPage.ruleMgmt.title') }}</span>
          <el-button type="primary" @click="openCreateRule">
            <el-icon><Plus /></el-icon>{{ t('alertPage.ruleMgmt.create') }}
          </el-button>
        </div>
      </template>

      <el-table :data="rules" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="name" :label="t('alertPage.col.name')" width="180" />
        <el-table-column prop="description" :label="t('alertPage.col.description')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="trigger_type" :label="t('alertPage.col.triggerType')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="getTriggerTypeTag(row.trigger_type)">
              {{ getTriggerTypeLabel(row.trigger_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="trigger_value" :label="t('alertPage.col.triggerValue')" width="120" />
        <el-table-column prop="notify_channels" :label="t('alertPage.col.notifyChannels')" width="180">
          <template #default="{ row }">
            <el-tag
              v-for="channel in parseChannels(row.notify_channels)"
              :key="channel"
              size="small"
              style="margin-right: 4px"
            >
              {{ getChannelLabel(channel) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" :label="t('alertPage.col.status')" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? t('alertPage.status.enabled') : t('alertPage.status.disabled') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cooldown_minutes" :label="t('alertPage.col.cooldown')" width="100">
          <template #default="{ row }">
            {{ t('alertPage.cooldownMinutes', row.cooldown_minutes) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('alertPage.col.createdAt')" width="180" />
        <el-table-column :label="t('alertPage.col.action')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditRule(row)">{{ t('alertPage.action.edit') }}</el-button>
            <el-button link type="danger" @click="handleDeleteRule(row)">{{ t('alertPage.action.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 告警记录查询 -->
    <el-card shadow="never" class="table-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span>{{ t('alertPage.record.title') }}</span>
          <el-button @click="loadAlerts">
            <el-icon><Refresh /></el-icon>{{ t('alertPage.record.refresh') }}
          </el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-select v-model="query.rule_id" :placeholder="t('alertPage.filter.rulePlaceholder')" clearable style="width: 180px">
          <el-option
            v-for="rule in rules"
            :key="rule.id"
            :label="rule.name"
            :value="rule.id"
          />
        </el-select>
        <el-select v-model="query.risk_level" :placeholder="t('alertPage.filter.riskLevelPlaceholder')" clearable style="width: 140px">
          <el-option :label="t('alertPage.risk.low')" value="low" />
          <el-option :label="t('alertPage.risk.medium')" value="medium" />
          <el-option :label="t('alertPage.risk.high')" value="high" />
          <el-option :label="t('alertPage.risk.critical')" value="critical" />
        </el-select>
        <el-input v-model="query.user_id" :placeholder="t('alertPage.filter.userIdPlaceholder')" clearable style="width: 200px" />
        <el-button type="primary" @click="loadAlerts">
          <el-icon><Search /></el-icon>{{ t('common.query') }}
        </el-button>
        <el-button @click="resetAlertSearch">{{ t('common.reset') }}</el-button>
      </div>

      <el-table v-loading="alertLoading" :data="alerts" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="rule_name" :label="t('alertPage.col.name')" width="180" />
        <el-table-column prop="username" :label="t('alertPage.col.triggerUser')" width="120" />
        <el-table-column prop="action" :label="t('alertPage.col.actionType')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" :label="t('alertPage.col.riskLevel')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getRiskLevelType(row.risk_level)">
              {{ getRiskLevelLabel(row.risk_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" :label="t('alertPage.col.message')" min-width="250" show-overflow-tooltip />
        <el-table-column prop="notified" :label="t('alertPage.col.notifyStatus')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.notified ? 'success' : 'warning'">
              {{ row.notified ? t('alertPage.notifyStatus.notified') : t('alertPage.notifyStatus.unnotified') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="notify_time" :label="t('alertPage.col.notifyTime')" width="180" />
        <el-table-column prop="created_at" :label="t('alertPage.col.triggerTime')" width="180" />
        <el-table-column :label="t('alertPage.col.action')" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openAlertDetail(row.id)">{{ t('alertPage.action.detail') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="alertPage"
          v-model:page-size="alertPageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="alertTotal"
          layout="total, sizes, prev, pager, next"
          @size-change="loadAlerts"
          @current-change="loadAlerts"
        />
      </div>
    </el-card>

    <!-- 创建/编辑规则弹窗 -->
    <el-dialog
      v-model="ruleDialogVisible"
      :title="isEditRule ? t('alertPage.dialog.editTitle') : t('alertPage.dialog.createTitle')"
      width="700px"
      destroy-on-close
    >
      <el-form :model="ruleForm" label-width="120px">
        <el-form-item :label="t('alertPage.form.nameLabel')" required>
          <el-input v-model="ruleForm.name" :placeholder="t('alertPage.form.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.descLabel')">
          <el-input v-model="ruleForm.description" type="textarea" :rows="2" :placeholder="t('alertPage.form.descPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.enabledLabel')">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.triggerTypeLabel')" required>
          <el-select v-model="ruleForm.trigger_type" :placeholder="t('alertPage.form.triggerTypePlaceholder')" style="width: 100%">
            <el-option :label="t('alertPage.form.trigger.byRisk')" value="risk_level" />
            <el-option :label="t('alertPage.form.trigger.byAction')" value="action" />
            <el-option :label="t('alertPage.form.trigger.byUser')" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('alertPage.form.triggerValueLabel')" required>
          <el-input v-model="ruleForm.trigger_value" :placeholder="t('alertPage.form.triggerValuePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.notifyChannelsLabel')" required>
          <el-checkbox-group v-model="ruleForm.notify_channels">
            <el-checkbox label="email">{{ t('alertPage.channel.email') }}</el-checkbox>
            <el-checkbox label="dingtalk">{{ t('alertPage.channel.dingtalk') }}</el-checkbox>
            <el-checkbox label="wechat">{{ t('alertPage.channel.wechat') }}</el-checkbox>
            <el-checkbox label="webhook">{{ t('alertPage.channel.webhook') }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="t('alertPage.form.notifyTargetsLabel')" required>
          <el-input
            v-model="ruleForm.notify_targets_input"
            type="textarea"
            :rows="3"
            :placeholder="t('alertPage.form.notifyTargetsPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.notifyTemplateLabel')">
          <el-input
            v-model="ruleForm.notify_template"
            type="textarea"
            :rows="3"
            :placeholder="t('alertPage.form.notifyTemplatePlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('alertPage.form.cooldownLabel')">
          <el-input-number v-model="ruleForm.cooldown_minutes" :min="1" :max="1440" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmRule">{{ t('alertPage.form.submit') }}</el-button>
      </template>
    </el-dialog>
    <!-- 告警详情：从收件箱的告警站内信跳转时带 alert_id，进入页面直接打开这一条 -->
    <el-dialog v-model="alertDetailVisible" :title="t('alertPage.detail.title')" width="640px">
      <div v-loading="alertDetailLoading" class="alert-detail">
        <el-descriptions v-if="alertDetail" :column="2" border>
          <el-descriptions-item :label="t('alertPage.detail.ruleName')">{{ alertDetail.rule_name }}</el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.riskLevel')">
            <el-tag size="small" :type="getRiskLevelType(alertDetail.risk_level)">
              {{ getRiskLevelLabel(alertDetail.risk_level) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.triggerUser')">
            <el-button
              v-if="alertDetail.username || alertDetail.user_id"
              link
              type="primary"
              @click="openUserTimeline"
            >
              {{ alertDetail.username || alertDetail.user_id }}
            </el-button>
            <span v-else>-</span>
            <span v-if="alertDetail.username && alertDetail.user_id" class="detail-muted">
              （{{ alertDetail.user_id }}）
            </span>
          </el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.actionType')">
            <el-tag size="small">{{ alertDetail.action }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.message')" :span="2">{{ alertDetail.message }}</el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.notifyStatus')">
            <el-tag size="small" :type="alertDetail.notified ? 'success' : 'warning'">
              {{ alertDetail.notified ? t('alertPage.notifyStatus.notified') : t('alertPage.notifyStatus.unnotified') }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.notifyTime')">{{ alertDetail.notify_time || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.triggerTime')">{{ alertDetail.created_at }}</el-descriptions-item>
          <el-descriptions-item :label="t('alertPage.detail.auditLogId')">
            <el-button
              v-if="alertDetail.audit_log_id"
              link
              type="primary"
              @click="openAuditLog(alertDetail.audit_log_id)"
            >
              {{ alertDetail.audit_log_id }}
            </el-button>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
        <el-empty
          v-if="!alertDetailLoading && !alertDetail"
          :description="t('alertPage.detail.notFound')"
          :image-size="60"
        />
      </div>
      <template #footer>
        <el-button @click="alertDetailVisible = false">{{ t('alertPage.detail.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { Bell, CircleCheck, Warning, Message, Plus, Search, Refresh } from '@element-plus/icons-vue'
import { get, post, put, del } from '@/api/request'
import { getAlertStats, type AlertStatsResult } from '@/api/modules/audit'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

interface AlertRule {
  id: string
  name: string
  description: string
  enabled: boolean
  trigger_type: string
  trigger_value: string
  notify_channels: string
  notify_targets: string
  notify_template: string
  cooldown_minutes: number
  created_at: string
  updated_at: string
}

interface AlertRecord {
  id: string
  rule_id: string
  rule_name: string
  audit_log_id: string
  user_id: string
  username: string
  risk_level: string
  action: string
  message: string
  notified: boolean
  notify_time: string
  created_at: string
}

const rules = ref<AlertRule[]>([])
const route = useRoute()
const router = useRouter()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<AlertRecord, { rule_id: string; user_id: string; risk_level: string }>({
  fetchList: async (params) => {
    const res = await get('/audit/alerts', {
      page: params.page,
      page_size: params.page_size,
      rule_id: params.rule_id || undefined,
      user_id: params.user_id || undefined,
      risk_level: params.risk_level || undefined
    })
    return toPageResult(res)
  },
  initialQuery: { rule_id: '', user_id: '', risk_level: '' },
  defaultPageSize: 20,
  immediate: false
})
const alerts = list
const alertLoading = loading
const alertPage = page
const alertPageSize = pageSize
const alertTotal = total
const loadAlerts = reload

const ruleDialogVisible = ref(false)
const isEditRule = ref(false)
const editingRuleId = ref('')

const ruleForm = reactive({
  name: '',
  description: '',
  enabled: true,
  trigger_type: '',
  trigger_value: '',
  notify_channels: [] as string[],
  notify_targets_input: '',
  notify_template: '',
  cooldown_minutes: 30
})

const enabledRulesCount = computed(() => rules.value.filter(r => r.enabled).length)

const alertStats = reactive<AlertStatsResult>({
  total_alerts: 0,
  today_alerts: 0,
  notified_count: 0,
  pending_count: 0,
  risk_level_stats: {},
  rule_stats: {},
  trend: [],
  top_rules: []
})
const statsDays = ref(7)

const trendMax = computed(() => {
  const max = Math.max(...alertStats.trend.map(p => p.count), 0)
  return max || 1
})

function trendBarHeight(count: number): string {
  const pct = (count / trendMax.value) * 100
  return `${Math.max(pct, 0)}%`
}

function riskPercentage(count: number): number {
  const total = alertStats.total_alerts || 1
  return Math.round((count / total) * 100)
}

function getRiskColor(level: string): string {
  const map: Record<string, string> = {
    low: '#52c41a',
    medium: '#faad14',
    high: '#ff4d4f',
    critical: '#cf1322'
  }
  return map[level] || '#1890ff'
}

async function loadAlertStats() {
  try {
    const data = await getAlertStats(statsDays.value)
    Object.assign(alertStats, data)
  } catch (e) {
    console.error(t('alertPage.loadStatsFailed'), e)
  }
}

onMounted(() => {
  loadRules()
  loadAlerts()
  loadAlertStats()
})

async function loadRules() {
  try {
    // 拦截器已拆出 data 层：这里拿到的就是规则数组，再取 .data 会恒为空
    const res = await get<AlertRule[]>('/audit/alert-rules')
    rules.value = Array.isArray(res) ? res : []
  } catch (error) {
    console.error(t('alertPage.loadRulesFailed'), error)
  }
}

const alertDetailVisible = ref(false)
const alertDetailLoading = ref(false)
const alertDetail = ref<AlertRecord | null>(null)

/**
 * 打开告警详情。
 * 收件箱里的告警站内信链接形如 /system/audit/alert?alert_id=xxx，
 * 进入页面后据此直接定位到触发这条消息的那条告警，而不是让用户回到列表首页自己找。
 */
async function openAlertDetail(id: string) {
  if (!id) return
  alertDetailVisible.value = true
  alertDetailLoading.value = true
  alertDetail.value = null
  try {
    const res = await get<AlertRecord>(`/audit/alerts/${id}`)
    alertDetail.value = res ?? null
  } catch {
    // 失败提示由拦截器统一弹出
  } finally {
    alertDetailLoading.value = false
  }
}

// 已在告警页时再点另一条告警消息，只有 query 变化（不会重新 mount），因此 watch 而非 onMounted 读一次
watch(
  () => route.query.alert_id,
  id => {
    if (typeof id === 'string' && id) openAlertDetail(id)
  },
  { immediate: true }
)

/**
 * 跳到审计日志并直接打开这条详情。
 * 告警只存日志 ID，正文与请求参数都在审计日志里：留一个不可点的 ID 等于让用户
 * 退出告警页、自己按时间在审计列表里翻。
 */
function openAuditLog(id: string) {
  if (!id) return
  alertDetailVisible.value = false
  router.push({ path: '/system/audit', query: { log_id: id } })
}

/**
 * 跳到该用户的操作时间线。
 * 告警只带用户名与用户 ID，这个人还做过什么全在时间线里：留一段不可点的文本，
 * 等于让用户退出去、在时间线页把用户名重新敲一遍。
 * 优先传用户 ID（时间线按 user_id OR username 匹配），没有 ID 时退回用户名。
 */
function openUserTimeline() {
  const userID = alertDetail.value?.user_id || alertDetail.value?.username
  if (!userID) return
  alertDetailVisible.value = false
  router.push({ path: '/system/audit/timeline', query: { user_id: userID } })
}

function resetAlertSearch() {
  reset()
}

function openCreateRule() {
  isEditRule.value = false
  editingRuleId.value = ''
  Object.assign(ruleForm, {
    name: '',
    description: '',
    enabled: true,
    trigger_type: '',
    trigger_value: '',
    notify_channels: [],
    notify_targets_input: '',
    notify_template: '',
    cooldown_minutes: 30
  })
  ruleDialogVisible.value = true
}

function openEditRule(row: AlertRule) {
  isEditRule.value = true
  editingRuleId.value = row.id
  Object.assign(ruleForm, {
    name: row.name,
    description: row.description,
    enabled: row.enabled,
    trigger_type: row.trigger_type,
    trigger_value: row.trigger_value,
    notify_channels: parseChannels(row.notify_channels),
    notify_targets_input: parseTargets(row.notify_targets).join('\n'),
    notify_template: row.notify_template,
    cooldown_minutes: row.cooldown_minutes
  })
  ruleDialogVisible.value = true
}

async function confirmRule() {
  if (!ruleForm.name || !ruleForm.trigger_type || !ruleForm.trigger_value) {
    ElMessage.warning(t('alertPage.form.nameRequired'))
    return
  }
  if (ruleForm.notify_channels.length === 0) {
    ElMessage.warning(t('alertPage.form.channelsRequired'))
    return
  }

  try {
    const data = {
      name: ruleForm.name,
      description: ruleForm.description,
      enabled: ruleForm.enabled,
      trigger_type: ruleForm.trigger_type,
      trigger_value: ruleForm.trigger_value,
      notify_channels: ruleForm.notify_channels,
      notify_targets: ruleForm.notify_targets_input.split('\n').filter(t => t.trim()),
      notify_template: ruleForm.notify_template,
      cooldown_minutes: ruleForm.cooldown_minutes
    }

    if (isEditRule.value) {
      await put(`/audit/alert-rules/${editingRuleId.value}`, data)
      ElMessage.success(t('alertPage.rule.updateSuccess'))
    } else {
      await post('/audit/alert-rules', data)
      ElMessage.success(t('alertPage.rule.createSuccess'))
    }
    ruleDialogVisible.value = false
    loadRules()
  } catch (error) {
    ElMessage.error(isEditRule.value ? t('alertPage.rule.updateFailed') : t('alertPage.rule.createFailed'))
  }
}

async function handleDeleteRule(row: AlertRule) {
  try {
    await ElMessageBox.confirm(t('alertPage.rule.deleteConfirm', row.name), t('alertPage.rule.deleteWarning'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await del(`/audit/alert-rules/${row.id}`)
    ElMessage.success(t('alertPage.rule.deleteSuccess'))
    loadRules()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('alertPage.rule.deleteFailed'))
    }
  }
}

function parseChannels(channelsStr: string): string[] {
  try {
    return JSON.parse(channelsStr)
  } catch {
    return []
  }
}

function parseTargets(targetsStr: string): string[] {
  try {
    return JSON.parse(targetsStr)
  } catch {
    return []
  }
}

function getChannelLabel(channel: string): string {
  const map: Record<string, string> = {
    email: t('alertPage.channel.email'),
    dingtalk: t('alertPage.channel.dingtalk'),
    wechat: t('alertPage.channel.wechat'),
    webhook: 'Webhook'
  }
  return map[channel] || channel
}

function getTriggerTypeLabel(type: string): string {
  const map: Record<string, string> = {
    risk_level: t('alertPage.trigger.riskLevel'),
    action: t('alertPage.trigger.action'),
    user: t('alertPage.trigger.user')
  }
  return map[type] || type
}

function getTriggerTypeTag(type: string): string {
  const map: Record<string, string> = {
    risk_level: 'danger',
    action: 'warning',
    user: 'info'
  }
  return map[type] || ''
}

function getRiskLevelType(level: string): string {
  const map: Record<string, string> = {
    low: 'success',
    medium: 'warning',
    high: 'danger',
    critical: 'danger'
  }
  return map[level] || 'info'
}

function getRiskLevelLabel(level: string): string {
  const map: Record<string, string> = {
    low: t('alertPage.riskLevel.short.low'),
    medium: t('alertPage.riskLevel.short.medium'),
    high: t('alertPage.riskLevel.short.high'),
    critical: t('alertPage.riskLevel.short.critical')
  }
  return map[level] || level
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  .stat-content {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .stat-icon {
    width: 48px;
    height: 48px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;

    &.rule-icon {
      background: #e6f7ff;
      color: #1890ff;
    }

    &.enabled-icon {
      background: #f6ffed;
      color: #52c41a;
    }

    &.alert-icon {
      background: #fff2f0;
      color: #ff4d4f;
    }

    &.notified-icon {
      background: #f9f0ff;
      color: #722ed1;
    }
  }

  .stat-info {
    flex: 1;
  }

  .stat-label {
    font-size: 14px;
    color: #8c8c8c;
    margin-bottom: 4px;
  }

  .stat-value {
    font-size: 24px;
    font-weight: 600;
    color: #262626;
  }
}

.table-card {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.alert-detail {
  min-height: 80px;

  .detail-muted {
    color: #8c8c8c;
    font-size: 12px;
  }
}

.charts-row {
  margin-bottom: 16px;
}

.chart-card {
  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}

.trend-chart {
  display: flex;
  height: 200px;
  gap: 8px;

  .trend-y-axis {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    text-align: right;
    font-size: 12px;
    color: #8c8c8c;
    padding: 4px 0;
    min-width: 32px;
  }

  .trend-bars {
    flex: 1;
    display: flex;
    align-items: flex-end;
    gap: 4px;
    padding-bottom: 24px;
    position: relative;
  }

  .trend-bar-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
    justify-content: flex-end;

    .bar {
      width: 100%;
      max-width: 40px;
      background: linear-gradient(180deg, #ff4d4f, #ff7875);
      border-radius: 4px 4px 0 0;
      transition: height 0.3s;
    }

    .label {
      font-size: 11px;
      color: #8c8c8c;
      margin-top: 4px;
      white-space: nowrap;
    }
  }
}

.risk-dist {
  .risk-item {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;

    .risk-label {
      width: 36px;
      font-size: 13px;
      color: #595959;
    }

    .risk-count {
      font-size: 13px;
      color: #8c8c8c;
      min-width: 32px;
      text-align: right;
    }

    .el-progress {
      flex: 1;
    }
  }
}
</style>