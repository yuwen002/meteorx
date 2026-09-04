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
              <div class="stat-label">告警规则</div>
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
              <div class="stat-label">已启用</div>
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
              <div class="stat-label">今日告警</div>
              <div class="stat-value">{{ todayAlertsCount }}</div>
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
              <div class="stat-label">已通知</div>
              <div class="stat-value">{{ notifiedAlertsCount }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 告警规则管理 -->
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>告警规则管理</span>
          <el-button type="primary" @click="openCreateRule">
            <el-icon><Plus /></el-icon>新建规则
          </el-button>
        </div>
      </template>

      <el-table :data="rules" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="name" label="规则名称" width="180" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="trigger_type" label="触发类型" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="getTriggerTypeTag(row.trigger_type)">
              {{ getTriggerTypeLabel(row.trigger_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="trigger_value" label="触发值" width="120" />
        <el-table-column prop="notify_channels" label="通知渠道" width="180">
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
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cooldown_minutes" label="冷却时间" width="100">
          <template #default="{ row }">
            {{ row.cooldown_minutes }}分钟
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditRule(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDeleteRule(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 告警记录查询 -->
    <el-card shadow="never" class="table-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span>告警记录</span>
          <el-button @click="loadAlerts">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-select v-model="alertSearch.rule_id" placeholder="告警规则" clearable style="width: 180px">
          <el-option
            v-for="rule in rules"
            :key="rule.id"
            :label="rule.name"
            :value="rule.id"
          />
        </el-select>
        <el-select v-model="alertSearch.risk_level" placeholder="风险等级" clearable style="width: 140px">
          <el-option label="低风险" value="low" />
          <el-option label="中风险" value="medium" />
          <el-option label="高风险" value="high" />
          <el-option label="严重风险" value="critical" />
        </el-select>
        <el-input v-model="alertSearch.user_id" placeholder="用户ID" clearable style="width: 200px" />
        <el-button type="primary" @click="loadAlerts">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetAlertSearch">重置</el-button>
      </div>

      <el-table v-loading="alertLoading" :data="alerts" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="rule_name" label="规则名称" width="180" />
        <el-table-column prop="username" label="触发用户" width="120" />
        <el-table-column prop="action" label="操作类型" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险等级" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getRiskLevelType(row.risk_level)">
              {{ getRiskLevelLabel(row.risk_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="告警消息" min-width="250" show-overflow-tooltip />
        <el-table-column prop="notified" label="通知状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.notified ? 'success' : 'warning'">
              {{ row.notified ? '已通知' : '未通知' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="notify_time" label="通知时间" width="180" />
        <el-table-column prop="created_at" label="触发时间" width="180" />
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
      :title="isEditRule ? '编辑告警规则' : '新建告警规则'"
      width="700px"
      destroy-on-close
    >
      <el-form :model="ruleForm" label-width="120px">
        <el-form-item label="规则名称" required>
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="规则描述">
          <el-input v-model="ruleForm.description" type="textarea" :rows="2" placeholder="请输入规则描述" />
        </el-form-item>
        <el-form-item label="是否启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
        <el-form-item label="触发类型" required>
          <el-select v-model="ruleForm.trigger_type" placeholder="请选择触发类型" style="width: 100%">
            <el-option label="按风险等级" value="risk_level" />
            <el-option label="按操作类型" value="action" />
            <el-option label="按特定用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item label="触发值" required>
          <el-input v-model="ruleForm.trigger_value" placeholder="如: high/critical/delete/user_id" />
        </el-form-item>
        <el-form-item label="通知渠道" required>
          <el-checkbox-group v-model="ruleForm.notify_channels">
            <el-checkbox label="email">邮件</el-checkbox>
            <el-checkbox label="dingtalk">钉钉</el-checkbox>
            <el-checkbox label="wechat">企业微信</el-checkbox>
            <el-checkbox label="webhook">自定义Webhook</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="通知目标" required>
          <el-input
            v-model="ruleForm.notify_targets_input"
            type="textarea"
            :rows="3"
            placeholder="每行一个，邮箱/Webhook地址"
          />
        </el-form-item>
        <el-form-item label="通知模板">
          <el-input
            v-model="ruleForm.notify_template"
            type="textarea"
            :rows="3"
            placeholder="支持Go模板语法，可用变量: {{.Username}}, {{.Action}}, {{.RiskLevel}}等"
          />
        </el-form-item>
        <el-form-item label="冷却时间(分钟)">
          <el-input-number v-model="ruleForm.cooldown_minutes" :min="1" :max="1440" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRule">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { Bell, CircleCheck, Warning, Message, Plus, Search, Refresh } from '@element-plus/icons-vue'
import { get, post, put, del } from '@/api/request'

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
const alerts = ref<AlertRecord[]>([])
const alertLoading = ref(false)
const alertPage = ref(1)
const alertPageSize = ref(20)
const alertTotal = ref(0)

const alertSearch = reactive({
  rule_id: '',
  user_id: '',
  risk_level: ''
})

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
const todayAlertsCount = computed(() => {
  const today = new Date().toISOString().slice(0, 10)
  return alerts.value.filter(a => a.created_at.startsWith(today)).length
})
const notifiedAlertsCount = computed(() => alerts.value.filter(a => a.notified).length)

onMounted(() => {
  loadRules()
  loadAlerts()
})

async function loadRules() {
  try {
    const res = await get('/audit/alert-rules')
    rules.value = res.data || []
  } catch (error) {
    console.error('加载告警规则失败', error)
  }
}

async function loadAlerts() {
  alertLoading.value = true
  try {
    const params: any = {
      page: alertPage.value,
      page_size: alertPageSize.value,
      rule_id: alertSearch.rule_id || undefined,
      user_id: alertSearch.user_id || undefined,
      risk_level: alertSearch.risk_level || undefined
    }
    const res = await get('/audit/alerts', params)
    alerts.value = res.items || []
    alertTotal.value = res.total || 0
  } catch (error) {
    console.error('加载告警记录失败', error)
  } finally {
    alertLoading.value = false
  }
}

function resetAlertSearch() {
  alertSearch.rule_id = ''
  alertSearch.user_id = ''
  alertSearch.risk_level = ''
  alertPage.value = 1
  loadAlerts()
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
    ElMessage.warning('请填写必填项')
    return
  }
  if (ruleForm.notify_channels.length === 0) {
    ElMessage.warning('请选择通知渠道')
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
      ElMessage.success('更新告警规则成功')
    } else {
      await post('/audit/alert-rules', data)
      ElMessage.success('创建告警规则成功')
    }
    ruleDialogVisible.value = false
    loadRules()
  } catch (error) {
    ElMessage.error(isEditRule.value ? '更新告警规则失败' : '创建告警规则失败')
  }
}

async function handleDeleteRule(row: AlertRule) {
  try {
    await ElMessageBox.confirm(`确定要删除告警规则 "${row.name}" 吗？`, '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await del(`/audit/alert-rules/${row.id}`)
    ElMessage.success('删除告警规则成功')
    loadRules()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除告警规则失败')
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
    email: '邮件',
    dingtalk: '钉钉',
    wechat: '企业微信',
    webhook: 'Webhook'
  }
  return map[channel] || channel
}

function getTriggerTypeLabel(type: string): string {
  const map: Record<string, string> = {
    risk_level: '风险等级',
    action: '操作类型',
    user: '特定用户'
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
    low: '低',
    medium: '中',
    high: '高',
    critical: '严重'
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
</style>