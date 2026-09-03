<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total-icon">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总日志数</div>
              <div class="stat-value">{{ dashboard.total_count }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon today-icon">
              <el-icon><Calendar /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">今日日志</div>
              <div class="stat-value">{{ dashboard.today_count }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon success-icon">
              <el-icon><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">成功操作</div>
              <div class="stat-value">{{ dashboard.result_stats?.success || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon fail-icon">
              <el-icon><CircleClose /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">失败操作</div>
              <div class="stat-value">{{ dashboard.result_stats?.failure || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区域 -->
    <el-row :gutter="16" class="charts-row">
      <!-- 趋势图 -->
      <el-col :span="16">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span>操作趋势（近{{ trendDays }}天）</span>
              <el-radio-group v-model="trendDays" size="small" @change="loadDashboard">
                <el-radio-button :label="7">7天</el-radio-button>
                <el-radio-button :label="14">14天</el-radio-button>
                <el-radio-button :label="30">30天</el-radio-button>
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
              <div
                v-for="point in dashboard.trend"
                :key="point.date"
                class="trend-bar-item"
              >
                <div class="bar-wrapper">
                  <div
                    class="bar bar-failure"
                    :style="{ height: trendMax > 0 ? (point.failure / trendMax * 100) + '%' : '0%' }"
                    :title="`失败: ${point.failure}`"
                  ></div>
                  <div
                    class="bar bar-success"
                    :style="{ height: trendMax > 0 ? (point.success / trendMax * 100) + '%' : '0%' }"
                    :title="`成功: ${point.success}`"
                  ></div>
                </div>
                <div class="bar-label">{{ formatDateLabel(point.date) }}</div>
              </div>
            </div>
            <div class="trend-legend">
              <span class="legend-item"><i class="dot success"></i>成功</span>
              <span class="legend-item"><i class="dot failure"></i>失败</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 模块分布 -->
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span>模块分布</span>
            </div>
          </template>
          <div class="pie-chart-container">
            <div
              class="pie-chart"
              :style="{ background: pieConicGradient }"
            ></div>
            <div class="pie-center">
              <div class="pie-total">{{ totalModuleCount }}</div>
              <div class="pie-label">总操作</div>
            </div>
          </div>
          <div class="pie-legend">
            <div
              v-for="(item, index) in dashboard.top_modules"
              :key="item.module"
              class="legend-row"
            >
              <span class="legend-color" :style="{ background: pieColors[index % pieColors.length] }"></span>
              <span class="legend-name">{{ getModuleLabel(item.module) }}</span>
              <span class="legend-value">{{ item.count }}</span>
              <span class="legend-percent">{{ getModulePercent(item.count) }}%</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 操作类型统计 -->
    <el-row :gutter="16" class="charts-row">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span>操作类型统计</span>
            </div>
          </template>
          <div class="action-stats">
            <div
              v-for="(count, action) in dashboard.action_stats"
              :key="action"
              class="action-stat-item"
            >
              <div class="action-info">
                <span class="action-name">{{ getActionLabel(action) }}</span>
                <span class="action-count">{{ count }}</span>
              </div>
              <div class="action-bar-bg">
                <div
                  class="action-bar-fill"
                  :style="{ width: actionStatsMax > 0 ? (count / actionStatsMax * 100) + '%' : '0%', background: getActionColor(action) }"
                ></div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span>结果分布</span>
            </div>
          </template>
          <div class="result-stats">
            <div class="result-ring">
              <svg viewBox="0 0 120 120" class="ring-svg">
                <circle cx="60" cy="60" r="50" class="ring-bg" />
                <circle
                  cx="60"
                  cy="60"
                  r="50"
                  class="ring-success"
                  :style="{ strokeDasharray: successRingArray, strokeDashoffset: 0 }"
                />
                <circle
                  cx="60"
                  cy="60"
                  r="50"
                  class="ring-failure"
                  :style="{ strokeDasharray: failureRingArray, strokeDashoffset: -successRingLength }"
                />
              </svg>
              <div class="ring-center-text">
                <span class="ring-percent">{{ successRate }}%</span>
                <span class="ring-label">成功率</span>
              </div>
            </div>
            <div class="result-details">
              <div class="result-item success">
                <span class="result-dot"></span>
                <span class="result-name">成功</span>
                <span class="result-count">{{ dashboard.result_stats?.success || 0 }}</span>
              </div>
              <div class="result-item failure">
                <span class="result-dot"></span>
                <span class="result-name">失败</span>
                <span class="result-count">{{ dashboard.result_stats?.failure || 0 }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="table-card">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="search.keyword"
          placeholder="搜索路径/用户名"
          clearable
          style="width: 200px"
          @keyup.enter="loadList"
        />
        <el-select v-model="search.module" placeholder="模块" clearable style="width: 140px">
          <el-option label="认证" value="auth" />
          <el-option label="用户" value="user" />
          <el-option label="租户" value="tenant" />
          <el-option label="权限" value="rbac" />
          <el-option label="审计" value="audit" />
        </el-select>
        <el-select v-model="search.action" placeholder="操作类型" clearable style="width: 140px">
          <el-option label="创建" value="create" />
          <el-option label="更新" value="update" />
          <el-option label="删除" value="delete" />
          <el-option label="查询" value="query" />
          <el-option label="登录" value="login" />
          <el-option label="登出" value="logout" />
          <el-option label="其他" value="other" />
        </el-select>
        <el-select v-model="search.result" placeholder="结果" clearable style="width: 120px">
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failure" />
        </el-select>
        <el-select v-model="search.risk_level" placeholder="风险等级" clearable style="width: 140px">
          <el-option label="低风险" value="low" />
          <el-option label="中风险" value="medium" />
          <el-option label="高风险" value="high" />
          <el-option label="严重风险" value="critical" />
        </el-select>
        <el-date-picker
          v-model="search.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 240px"
        />
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <el-button type="success" @click="handleExport">
          <el-icon><Download /></el-icon>导出
        </el-button>
        <div class="flex-1"></div>
        <el-button-group>
          <el-button :type="search.action === 'login' ? 'primary' : 'default'" @click="quickFilter('login')">
            <el-icon><User /></el-icon>登录日志
          </el-button>
          <el-button :type="search.action === 'logout' ? 'primary' : 'default'" @click="quickFilter('logout')">
            <el-icon><SwitchButton /></el-icon>登出日志
          </el-button>
          <el-button :type="search.action === '' ? 'primary' : 'default'" @click="quickFilter('')">
            <el-icon><Document /></el-icon>全部
          </el-button>
        </el-button-group>
        <el-button type="danger" @click="handleCleanup" style="margin-left: 10px;">
          <el-icon><Delete /></el-icon>清理日志
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table :data="list" border stripe v-loading="loading" style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" label="操作用户" width="120" />
        <el-table-column prop="module" label="模块" width="100">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.module }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="操作" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getActionType(row.action)">{{ getActionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险等级" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getRiskLevelType(row.risk_level)">{{ getRiskLevelLabel(row.risk_level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="method" label="方法" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="getMethodType(row.method)">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status_code" label="状态码" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status_code >= 400 ? 'danger' : 'success'">{{ row.status_code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result" label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
              {{ row.result === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="客户端IP" width="140" />
        <el-table-column prop="device_info" label="设备" width="140" show-overflow-tooltip />
        <el-table-column prop="duration" label="耗时(ms)" width="100">
          <template #default="{ row }">
            <span :class="getDurationClass(row.duration)">{{ row.duration }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="日志详情" width="900px" destroy-on-close>
      <el-descriptions :column="2" border v-if="currentLog">
        <el-descriptions-item label="日志ID">{{ currentLog.id }}</el-descriptions-item>
        <el-descriptions-item label="请求ID">{{ currentLog.request_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="操作用户">{{ currentLog.username }} ({{ currentLog.user_id }})</el-descriptions-item>
        <el-descriptions-item label="租户ID">{{ currentLog.tenant_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="会话ID">{{ currentLog.session_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="链路ID">{{ currentLog.trace_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="模块">{{ currentLog.module }}</el-descriptions-item>
        <el-descriptions-item label="操作类型">{{ getActionLabel(currentLog.action) }}</el-descriptions-item>
        <el-descriptions-item label="风险等级">
          <el-tag size="small" :type="getRiskLevelType(currentLog.risk_level)">{{ getRiskLevelLabel(currentLog.risk_level) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="HTTP方法">{{ currentLog.method }}</el-descriptions-item>
        <el-descriptions-item label="请求路径" :span="2">{{ currentLog.path }}</el-descriptions-item>
        <el-descriptions-item label="来源页面" :span="2">{{ currentLog.referer || '-' }}</el-descriptions-item>
        <el-descriptions-item label="资源">{{ currentLog.resource || '-' }}</el-descriptions-item>
        <el-descriptions-item label="资源ID">{{ currentLog.resource_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态码">
          <el-tag size="small" :type="currentLog.status_code >= 400 ? 'danger' : 'success'">{{ currentLog.status_code }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="结果">
          <el-tag size="small" :type="currentLog.result === 'success' ? 'success' : 'danger'">
            {{ currentLog.result === 'success' ? '成功' : '失败' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="客户端IP">{{ currentLog.client_ip }}</el-descriptions-item>
        <el-descriptions-item label="IP位置">{{ currentLog.ip_location || '未知' }}</el-descriptions-item>
        <el-descriptions-item label="用户代理" :span="2">{{ currentLog.user_agent }}</el-descriptions-item>
        <el-descriptions-item label="设备信息" :span="2">{{ currentLog.device_info || '-' }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ currentLog.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ currentLog.created_at }}</el-descriptions-item>
        <el-descriptions-item label="标签" :span="2" v-if="currentLog.tags">
          <el-tag v-for="tag in parseTags(currentLog.tags)" :key="tag" size="small" style="margin-right: 4px;">{{ tag }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="错误信息" :span="2" v-if="currentLog.error_message">
          <span style="color: #f56c6c">{{ currentLog.error_message }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="请求参数" :span="2" v-if="currentLog.request_body">
          <pre class="json-pre">{{ formatJson(currentLog.request_body) }}</pre>
        </el-descriptions-item>
        <el-descriptions-item label="响应结果" :span="2" v-if="currentLog.response_body">
          <pre class="json-pre">{{ formatJson(currentLog.response_body) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 清理弹窗 -->
    <el-dialog v-model="cleanupVisible" title="清理审计日志" width="400px">
      <el-form :model="cleanupForm" label-width="120px">
        <el-form-item label="保留天数">
          <el-input-number v-model="cleanupForm.days" :min="1" :max="365" />
        </el-form-item>
        <p style="color: #f56c6c; font-size: 13px; margin-left: 120px;">
          将永久删除 {{ cleanupForm.days }} 天前的所有审计日志，此操作不可恢复！
        </p>
      </el-form>
      <template #footer>
        <el-button @click="cleanupVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmCleanup">确认清理</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Calendar, CircleCheck, CircleClose, Search, Delete, Download, User, SwitchButton } from '@element-plus/icons-vue'
import { getAuditLogList, getAuditDashboard, cleanupAuditLogs, exportAuditLogs } from '@/api/modules/audit'
import type { AuditLogItem, AuditDashboardData } from '@/api/modules/audit'

const loading = ref(false)
const list = ref<AuditLogItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const trendDays = ref(7)

const search = reactive({
  keyword: '',
  module: '',
  action: '',
  result: '',
  risk_level: '',
  dateRange: [] as string[]
})

const dashboard = ref<AuditDashboardData>({
  total_count: 0,
  today_count: 0,
  action_stats: {},
  module_stats: {},
  result_stats: {},
  trend: [],
  top_modules: []
})

const detailVisible = ref(false)
const currentLog = ref<AuditLogItem | null>(null)
const cleanupVisible = ref(false)
const cleanupForm = reactive({ days: 30 })

const pieColors = ['#667eea', '#52c41a', '#faad14', '#f56c6c', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16']

const trendMax = computed(() => {
  let max = 0
  dashboard.value.trend.forEach(p => {
    max = Math.max(max, p.count, p.success, p.failure)
  })
  return max > 0 ? Math.ceil(max * 1.1) : 100
})

const actionStatsMax = computed(() => {
  let max = 0
  Object.values(dashboard.value.action_stats).forEach(v => {
    max = Math.max(max, v)
  })
  return max
})

const totalModuleCount = computed(() => {
  return dashboard.value.top_modules.reduce((sum, m) => sum + m.count, 0)
})

const pieConicGradient = computed(() => {
  if (totalModuleCount.value === 0) return 'conic-gradient(#f0f0f0 0deg 360deg)'
  let result = ''
  let currentAngle = 0
  dashboard.value.top_modules.forEach((m, index) => {
    const angle = (m.count / totalModuleCount.value) * 360
    const color = pieColors[index % pieColors.length]
    result += `${color} ${currentAngle}deg ${currentAngle + angle}deg, `
    currentAngle += angle
  })
  return `conic-gradient(${result.slice(0, -2)})`
})

const successRingLength = computed(() => {
  const total = (dashboard.value.result_stats?.success || 0) + (dashboard.value.result_stats?.failure || 0)
  if (total === 0) return 0
  return (dashboard.value.result_stats?.success || 0) / total * 314
})

const failureRingLength = computed(() => {
  const total = (dashboard.value.result_stats?.success || 0) + (dashboard.value.result_stats?.failure || 0)
  if (total === 0) return 0
  return (dashboard.value.result_stats?.failure || 0) / total * 314
})

const successRingArray = computed(() => `${successRingLength.value} 314`)
const failureRingArray = computed(() => `${failureRingLength.value} 314`)

const successRate = computed(() => {
  const total = (dashboard.value.result_stats?.success || 0) + (dashboard.value.result_stats?.failure || 0)
  if (total === 0) return 0
  return Math.round((dashboard.value.result_stats?.success || 0) / total * 100)
})

onMounted(() => {
  loadList()
  loadDashboard()
})

async function loadList() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
      keyword: search.keyword || undefined,
      module: search.module || undefined,
      action: search.action || undefined,
      result: search.result || undefined,
      risk_level: search.risk_level || undefined
    }
    if (search.dateRange && search.dateRange.length === 2) {
      params.start_time = search.dateRange[0] + ' 00:00:00'
      params.end_time = search.dateRange[1] + ' 23:59:59'
    }
    const res = await getAuditLogList(params)
    list.value = res.items
    total.value = res.total
  } catch (error) {
    console.error('加载审计日志失败', error)
  } finally {
    loading.value = false
  }
}

async function loadDashboard() {
  try {
    const res = await getAuditDashboard(trendDays.value)
    dashboard.value = res
  } catch (error) {
    console.error('加载看板数据失败', error)
  }
}

function resetSearch() {
  search.keyword = ''
  search.module = ''
  search.action = ''
  search.result = ''
  search.risk_level = ''
  search.dateRange = []
  page.value = 1
  loadList()
}

function quickFilter(action: string) {
  search.action = action
  page.value = 1
  loadList()
}

function openDetail(row: AuditLogItem) {
  currentLog.value = row
  detailVisible.value = true
}

function handleCleanup() {
  cleanupForm.days = 30
  cleanupVisible.value = true
}

async function confirmCleanup() {
  try {
    await ElMessageBox.confirm(
      `确定要清理 ${cleanupForm.days} 天前的审计日志吗？此操作不可恢复！`,
      '警告',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
    const res = await cleanupAuditLogs(cleanupForm.days)
    ElMessage.success(`已清理 ${res.deleted_count} 条日志`)
    cleanupVisible.value = false
    loadList()
    loadDashboard()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('清理失败')
    }
  }
}

function getActionLabel(action: string): string {
  const map: Record<string, string> = {
    create: '创建',
    update: '更新',
    delete: '删除',
    query: '查询',
    login: '登录',
    logout: '登出',
    other: '其他'
  }
  return map[action] || action
}

function getActionType(action: string): string {
  const map: Record<string, string> = {
    create: 'success',
    update: 'warning',
    delete: 'danger',
    query: 'info',
    login: 'primary',
    logout: 'info',
    other: ''
  }
  return map[action] || ''
}

function getActionColor(action: string): string {
  const map: Record<string, string> = {
    create: '#52c41a',
    update: '#faad14',
    delete: '#f56c6c',
    query: '#667eea',
    login: '#722ed1',
    logout: '#13c2c2',
    other: '#8c8c8c'
  }
  return map[action] || '#8c8c8c'
}

function getMethodType(method: string): string {
  const map: Record<string, string> = {
    GET: 'info',
    POST: 'success',
    PUT: 'warning',
    PATCH: 'warning',
    DELETE: 'danger'
  }
  return map[method] || ''
}

function getDurationClass(duration: number): string {
  if (duration < 100) return 'duration-fast'
  if (duration < 500) return 'duration-normal'
  return 'duration-slow'
}

function getModuleLabel(module: string): string {
  const map: Record<string, string> = {
    auth: '认证',
    user: '用户',
    tenant: '租户',
    rbac: '权限',
    audit: '审计',
    system: '系统'
  }
  return map[module] || module
}

function getModulePercent(count: number): string {
  if (totalModuleCount.value === 0) return '0'
  return ((count / totalModuleCount.value) * 100).toFixed(1)
}

function formatDateLabel(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    return `${date.getMonth() + 1}/${date.getDate()}`
  } catch {
    return dateStr.slice(5)
  }
}

function formatJson(jsonStr: string): string {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
}

function parseTags(tagsStr: string): string[] {
  try {
    const tags = JSON.parse(tagsStr)
    return Array.isArray(tags) ? tags : []
  } catch {
    return []
  }
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

function handleExport() {
  const params: any = {
    format: 'csv',
    module: search.module || undefined,
    action: search.action || undefined,
    result: search.result || undefined,
    keyword: search.keyword || undefined
  }
  if (search.dateRange && search.dateRange.length === 2) {
    params.start_time = search.dateRange[0] + ' 00:00:00'
    params.end_time = search.dateRange[1] + ' 23:59:59'
  }
  
  const exportUrl = exportAuditLogs(params)
  const link = document.createElement('a')
  link.href = exportUrl
  link.download = `audit_logs_${new Date().toISOString().slice(0, 10)}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  ElMessage.success('开始导出审计日志')
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.stats-row {
  margin-bottom: 16px;
}

.charts-row {
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

    &.total-icon {
      background: #e6f7ff;
      color: #1890ff;
    }

    &.today-icon {
      background: #f6ffed;
      color: #52c41a;
    }

    &.success-icon {
      background: #f6ffed;
      color: #52c41a;
    }

    &.fail-icon {
      background: #fff2f0;
      color: #ff4d4f;
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

.chart-card {
  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 500;
  }
}

.trend-chart {
  height: 240px;
  display: flex;
  padding: 16px 0;
  position: relative;
}

.trend-y-axis {
  width: 40px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: flex-end;
  padding-right: 8px;
  font-size: 12px;
  color: #8c8c8c;

  span {
    line-height: 1;
  }
}

.trend-bars {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 4px;
  border-bottom: 1px solid #f0f0f0;
  border-left: 1px solid #f0f0f0;
  padding-left: 8px;
}

.trend-bar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.bar-wrapper {
  flex: 1;
  width: 100%;
  display: flex;
  flex-direction: column-reverse;
  gap: 2px;
  justify-content: flex-start;
}

.bar {
  width: 100%;
  max-width: 20px;
  margin: 0 auto;
  border-radius: 2px 2px 0 0;
  transition: height 0.3s;

  &.bar-success {
    background: linear-gradient(180deg, #52c41a 0%, #389e0d 100%);
  }

  &.bar-failure {
    background: linear-gradient(180deg, #f56c6c 0%, #cf1322 100%);
  }
}

.bar-label {
  font-size: 11px;
  color: #8c8c8c;
  margin-top: 4px;
  white-space: nowrap;
}

.trend-legend {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #595959;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;

  &.success {
    background: #52c41a;
  }

  &.failure {
    background: #f56c6c;
  }
}

.pie-chart-container {
  position: relative;
  width: 180px;
  height: 180px;
  margin: 0 auto 16px;
}

.pie-chart {
  width: 100%;
  height: 100%;
  border-radius: 50%;
}

.pie-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  background: #fff;
  border-radius: 50%;
  width: 90px;
  height: 90px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.pie-total {
  font-size: 24px;
  font-weight: 600;
  color: #262626;
}

.pie-label {
  font-size: 12px;
  color: #8c8c8c;
}

.pie-legend {
  padding: 0 8px;
}

.legend-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 13px;

  .legend-color {
    width: 12px;
    height: 12px;
    border-radius: 2px;
  }

  .legend-name {
    flex: 1;
    color: #595959;
  }

  .legend-value {
    color: #262626;
    font-weight: 500;
    min-width: 30px;
    text-align: right;
  }

  .legend-percent {
    color: #8c8c8c;
    min-width: 45px;
    text-align: right;
  }
}

.action-stats {
  padding: 8px 0;
}

.action-stat-item {
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }
}

.action-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
  font-size: 13px;

  .action-name {
    color: #595959;
  }

  .action-count {
    color: #262626;
    font-weight: 500;
  }
}

.action-bar-bg {
  height: 8px;
  background: #f5f5f5;
  border-radius: 4px;
  overflow: hidden;
}

.action-bar-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s;
}

.result-stats {
  display: flex;
  align-items: center;
  gap: 32px;
  padding: 8px 0;
}

.result-ring {
  position: relative;
  width: 160px;
  height: 160px;
}

.ring-svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring-bg {
  fill: none;
  stroke: #f5f5f5;
  stroke-width: 12;
}

.ring-success {
  fill: none;
  stroke: #52c41a;
  stroke-width: 12;
  stroke-linecap: round;
}

.ring-failure {
  fill: none;
  stroke: #f56c6c;
  stroke-width: 12;
  stroke-linecap: round;
}

.ring-center-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;

  .ring-percent {
    display: block;
    font-size: 28px;
    font-weight: 600;
    color: #262626;
  }

  .ring-label {
    display: block;
    font-size: 12px;
    color: #8c8c8c;
  }
}

.result-details {
  flex: 1;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;

  &:last-child {
    border-bottom: none;
  }

  .result-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  &.success .result-dot {
    background: #52c41a;
  }

  &.failure .result-dot {
    background: #f56c6c;
  }

  .result-name {
    flex: 1;
    color: #595959;
  }

  .result-count {
    color: #262626;
    font-weight: 500;
  }
}

.table-card {
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

.flex-1 {
  flex: 1;
}

.duration-fast {
  color: #52c41a;
}

.duration-normal {
  color: #faad14;
}

.duration-slow {
  color: #ff4d4f;
  font-weight: bold;
}

.risk-low {
  color: #52c41a;
}

.risk-medium {
  color: #faad14;
}

.risk-high {
  color: #f56c6c;
}

.risk-critical {
  color: #ff4d4f;
  font-weight: bold;
}

.json-pre {
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  max-height: 300px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>