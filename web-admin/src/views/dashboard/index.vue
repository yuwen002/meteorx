<template>
  <div class="dashboard">
    <!-- ======== 平台超级管理员：运营数据看板 ======== -->
    <template v-if="isAdmin">
      <el-row :gutter="16">
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon user-icon">
                <el-icon><OfficeBuilding /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.stat.tenantTotal') }}</div>
                <div class="stat-value">{{ overview.tenant_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon role-icon">
                <el-icon><UserFilled /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.stat.userTotal') }}</div>
                <div class="stat-value">{{ overview.user_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon perm-icon">
                <el-icon><Goods /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.stat.activeSub') }}</div>
                <div class="stat-value">{{ overview.subscription_stats?.active ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon info-icon">
                <el-icon><Document /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.stat.auditLog') }}</div>
                <div class="stat-value">{{ overview.audit_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第二行：今日 / 本周 / 本月新增 -->
      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><TrendCharts /></el-icon> {{ t('dashboardPage.growth.title') }}</span></template>
            <el-table :data="growthRows" size="small" border>
              <el-table-column prop="label" :label="t('dashboardPage.growth.col.metric')" width="140" />
              <el-table-column prop="today" :label="t('dashboardPage.growth.col.today')" />
              <el-table-column prop="week" :label="t('dashboardPage.growth.col.week')" />
              <el-table-column prop="month" :label="t('dashboardPage.growth.col.month')" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Coin /></el-icon> {{ t('dashboardPage.subDist.title') }}</span></template>
            <div class="sub-stats">
              <div class="sub-item">
                <div class="sub-num">{{ overview.subscription_stats?.total ?? 0 }}</div>
                <div class="sub-label">{{ t('dashboardPage.sub.total') }}</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#10b981">{{ overview.subscription_stats?.active ?? 0 }}</div>
                <div class="sub-label">{{ t('dashboardPage.sub.active') }}</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#f59e0b">{{ overview.subscription_stats?.expired ?? 0 }}</div>
                <div class="sub-label">{{ t('dashboardPage.sub.expired') }}</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#ef4444">{{ overview.subscription_stats?.cancelled ?? 0 }}</div>
                <div class="sub-label">{{ t('dashboardPage.sub.cancelled') }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第三行：审计操作分布 -->
      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Odometer /></el-icon> {{ t('dashboardPage.action.title') }}</span></template>
            <el-empty v-if="!actionRows.length" :description="t('dashboardPage.noData')" :image-size="60" />
            <el-table v-else :data="actionRows" size="small" border>
              <el-table-column prop="key" :label="t('dashboardPage.action.col.type')" />
              <el-table-column prop="value" :label="t('dashboardPage.action.col.count')" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><DataAnalysis /></el-icon> {{ t('dashboardPage.module.title') }}</span></template>
            <el-empty v-if="!moduleRows.length" :description="t('dashboardPage.noData')" :image-size="60" />
            <el-table v-else :data="moduleRows" size="small" border>
              <el-table-column prop="key" :label="t('dashboardPage.module.col.module')" />
              <el-table-column prop="value" :label="t('dashboardPage.module.col.count')" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第四行：增长趋势（零依赖内联 SVG 柱状图） -->
      <el-row :gutter="16">
        <el-col :span="24">
          <el-card shadow="hover" class="section-card trend-card">
            <template #header>
              <div class="trend-header">
                <span><el-icon><TrendCharts /></el-icon> {{ t('dashboardPage.trend.title') }}</span>
                <div class="trend-controls">
                  <el-radio-group v-model="trendMetric" size="small">
                    <el-radio-button value="new_users">{{ t('dashboardPage.trend.metric.users') }}</el-radio-button>
                    <el-radio-button value="new_tenants">{{ t('dashboardPage.trend.metric.tenants') }}</el-radio-button>
                    <el-radio-button value="audit_logs">{{ t('dashboardPage.trend.metric.audit') }}</el-radio-button>
                  </el-radio-group>
                  <el-select v-model="trendPeriod" size="small" style="width: 100px" @change="loadTrend">
                    <el-option :value="7" :label="t('dashboardPage.trend.period7')" />
                    <el-option :value="30" :label="t('dashboardPage.trend.period30')" />
                    <el-option :value="90" :label="t('dashboardPage.trend.period90')" />
                  </el-select>
                </div>
              </div>
            </template>
            <el-empty v-if="!trend.list.length" :description="t('dashboardPage.noData')" :image-size="60" />
            <div v-else class="trend-scroll">
              <svg
                class="trend-svg"
                :viewBox="`0 0 ${trendChart.width} ${trendChart.height}`"
                :width="trendChart.width"
                :height="trendChart.height"
                role="img"
                :aria-label="t('dashboardPage.trend.aria')"
              >
                <line
                  :x1="trendChart.padLeft"
                  :y1="trendChart.baseY"
                  :x2="trendChart.width - trendChart.padRight"
                  :y2="trendChart.baseY"
                  stroke="#e5e7eb"
                  stroke-width="1"
                />
                <g v-for="b in trendChart.bars" :key="b.date">
                  <rect
                    :x="b.x"
                    :y="b.y"
                    :width="b.w"
                    :height="b.h"
                    :fill="b.fill"
                    rx="2"
                  >
                    <title>{{ b.date }}：{{ b.value }}</title>
                  </rect>
                  <text v-if="b.showLabel" :x="b.cx" :y="trendChart.baseY + 14" class="bar-label" text-anchor="middle">
                    {{ b.label }}
                  </text>
                </g>
              </svg>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- ======== 普通租户管理员：工作台 ======== -->
    <template v-else>
      <el-row :gutter="16">
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon user-icon">
                <el-icon><List /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.ws.taskTitle') }}</div>
                <div class="stat-value">{{ ws.task.total }}</div>
                <div class="stat-sub">
                  {{ t('dashboardPage.ws.taskPending') }} {{ ws.task.pending }} ·
                  {{ t('dashboardPage.ws.taskDoing') }} {{ ws.task.in_progress }} ·
                  <span :class="{ 'overdue-red': ws.task.overdue > 0 }">
                    {{ t('dashboardPage.ws.taskOverdue') }} {{ ws.task.overdue }}
                  </span>
                </div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon role-icon">
                <el-icon><Reading /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.ws.wikiTitle') }}</div>
                <div class="stat-value">{{ ws.wiki.total_documents }}</div>
                <div class="stat-sub">
                  {{ t('dashboardPage.ws.wikiSpaces') }} {{ ws.wiki.total_spaces }} ·
                  {{ t('dashboardPage.ws.wikiViews') }} {{ ws.wiki.total_views }}
                </div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon perm-icon">
                <el-icon><UserFilled /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.ws.memberTitle') }}</div>
                <div class="stat-value">{{ ws.memberCount }}</div>
                <div class="stat-sub">{{ t('dashboardPage.ws.storage') }} {{ storageText }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon info-icon">
                <el-icon><Aim /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">{{ t('dashboardPage.ws.completion') }}</div>
                <el-progress
                  type="dashboard"
                  :width="72"
                  :percentage="completionPercent"
                  :stroke-width="6"
                />
                <div class="stat-sub">{{ t('dashboardPage.ws.avgHandle') }} {{ avgHandleText }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第二行：待我处理 + 快捷入口 -->
      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Bell /></el-icon> {{ t('dashboardPage.ws.todoTitle') }}</span></template>
            <div class="todo-list">
              <router-link to="/messages" class="todo-item">
                <span class="todo-label"><el-icon><Message /></el-icon> {{ t('dashboardPage.ws.todoMessages') }}</span>
                <el-badge :value="ws.unreadMessages" :hidden="!ws.unreadMessages" :max="99" />
              </router-link>
              <router-link to="/wiki" class="todo-item">
                <span class="todo-label"><el-icon><Reading /></el-icon> {{ t('dashboardPage.ws.todoNotifications') }}</span>
                <el-badge :value="ws.unreadNotifications" :hidden="!ws.unreadNotifications" :max="99" />
              </router-link>
              <router-link to="/chat" class="todo-item">
                <span class="todo-label"><el-icon><ChatDotRound /></el-icon> {{ t('dashboardPage.ws.todoChats') }}</span>
                <span class="todo-right">
                  <el-tag v-if="ws.mentionChats" size="small" type="danger" effect="light">
                    {{ t('dashboardPage.ws.mentionBadge', ws.mentionChats) }}
                  </el-tag>
                  <el-badge :value="ws.unreadChats" :hidden="!ws.unreadChats" :max="99" />
                </span>
              </router-link>
            </div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Promotion /></el-icon> {{ t('dashboardPage.ws.quickTitle') }}</span></template>
            <div class="quick-grid">
              <el-button v-for="q in quickEntries" :key="q.path" text class="quick-btn" @click="go(q.path)">
                <el-icon><component :is="q.icon" /></el-icon>&nbsp;{{ q.label }}
              </el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <el-card v-if="isAdmin" class="welcome-card" shadow="never">
      <template #header>
        <span><el-icon><TrendCharts /></el-icon>{{ isAdmin ? t('dashboardPage.welcome.overview') : t('dashboardPage.welcome.workspace') }}</span>
      </template>
      <div class="welcome">
        <h3>{{ t('dashboardPage.welcome.greeting', userStore.userInfo?.nickname || userStore.userInfo?.username || t('dashboardPage.welcome.userFallback')) }}</h3>
        <p v-if="isAdmin">{{ t('dashboardPage.welcome.adminDesc') }}</p>
        <p v-else>{{ t('dashboardPage.welcome.tenantDesc') }}</p>
        <el-divider />
        <div class="feature-list">
          <template v-if="isAdmin">
            <div>{{ t('dashboardPage.feature.admin1') }}</div>
            <div>{{ t('dashboardPage.feature.admin2') }}</div>
            <div>{{ t('dashboardPage.feature.admin3') }}</div>
            <div>{{ t('dashboardPage.feature.admin4') }}</div>
            <div>{{ t('dashboardPage.feature.admin5') }}</div>
            <div>{{ t('dashboardPage.feature.admin6') }}</div>
          </template>
          <template v-else>
            <div>{{ t('dashboardPage.feature.tenant1') }}</div>
            <div>{{ t('dashboardPage.feature.tenant2') }}</div>
            <div>{{ t('dashboardPage.feature.tenant3') }}</div>
            <div>{{ t('dashboardPage.feature.tenant4') }}</div>
          </template>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  OfficeBuilding,
  UserFilled,
  Goods,
  Document,
  TrendCharts,
  Coin,
  Odometer,
  DataAnalysis,
  List,
  Reading,
  Aim,
  Bell,
  Message,
  ChatDotRound,
  Promotion,
  Checked,
  Plus
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useUserStore } from '@/stores/user'
import { getRBACStats } from '@/api/modules/role'
import { getDashboardOverview, getGrowthTrend, type DashboardOverview, type GrowthTrend } from '@/api/modules/dashboard'
import { useWorkspaceStats } from '@/composables/useWorkspaceStats'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const userStore = useUserStore()
const router = useRouter()
const overview = ref<DashboardOverview>({} as DashboardOverview)
// 租户工作台统计（仅非超管分支使用）
const ws = useWorkspaceStats()

const isAdmin = computed(() => !!userStore.userInfo?.is_master)

// 新增概览行数据
const growthRows = computed(() => [
  { label: t('dashboardPage.growth.newTenant'), today: overview.value.tenant_stats?.today_new ?? 0, week: overview.value.tenant_stats?.week_new ?? 0, month: overview.value.tenant_stats?.month_new ?? 0 },
  { label: t('dashboardPage.growth.newUser'), today: overview.value.user_stats?.today_new ?? 0, week: overview.value.user_stats?.week_new ?? 0, month: overview.value.user_stats?.month_new ?? 0 }
])

// 操作类型 / 模块统计行
const actionRows = computed(() => Object.entries(overview.value.audit_stats?.action_stats ?? {}).map(([key, value]) => ({ key, value })))
const moduleRows = computed(() => Object.entries(overview.value.audit_stats?.module_stats ?? {}).map(([key, value]) => ({ key, value })))

// ---------- 增长趋势（零依赖内联 SVG 柱状图）----------
type TrendMetric = 'new_users' | 'new_tenants' | 'audit_logs'
const trendMetric = ref<TrendMetric>('new_users')
const trendPeriod = ref<number>(30)
const trend = ref<GrowthTrend>({ days: 0, list: [] })
const METRIC_COLORS: Record<TrendMetric, string> = { new_users: '#10b981', new_tenants: '#3b82f6', audit_logs: '#8b5cf6' }

async function loadTrend() {
  try {
    trend.value = await getGrowthTrend(trendPeriod.value)
  } catch {
    ElMessage.warning(t('dashboardPage.msg.trendFailed'))
  }
}

// 将时间序列渲染为等宽柱状；缺失日期后端已以零值补齐，此处直接逐日绘制。
const trendChart = computed(() => {
  const list = trend.value.list ?? []
  const metric = trendMetric.value
  const height = 200
  const padLeft = 32
  const padRight = 12
  const padTop = 16
  const baseY = height - 28
  const innerW = Math.max(list.length * 22, 1)
  const width = padLeft + innerW + padRight
  const slot = innerW / Math.max(list.length, 1)
  const barW = Math.min(14, slot - 4)
  const maxVal = list.reduce((m, d) => Math.max(m, d[metric]), 0)
  const plotH = baseY - padTop
  const usable = maxVal > 0 ? maxVal : 1
  const labelStep = Math.max(1, Math.ceil(list.length / 8))
  const color = METRIC_COLORS[metric]
  const bars = list.map((d, i) => {
    const value = d[metric]
    const h = maxVal > 0 ? (value / usable) * plotH : 0
    const x = padLeft + i * slot + (slot - barW) / 2
    const y = baseY - h
    return {
      date: d.date,
      x: +x.toFixed(1),
      y: +y.toFixed(1),
      w: +barW.toFixed(1),
      h: +h.toFixed(1),
      cx: +(x + barW / 2).toFixed(1),
      value,
      fill: color,
      label: d.date.slice(5),
      showLabel: i % labelStep === 0
    }
  })
  return { width, height, padLeft, padRight, baseY, bars }
})

// ---------- 租户工作台（非超管） ----------
// 完成率百分比化（后端返 0~1，el-progress 需 0~100 整数）
const completionPercent = computed(() => Math.round((ws.task.completion_rate || 0) * 100))

/** 平均处理时长可读化：按量级切换 分钟/小时/天，保留整数 */
const avgHandleText = computed(() => {
  const s = ws.task.avg_handle_seconds || 0
  if (!s) return '-'
  if (s < 3600) return Math.max(1, Math.round(s / 60)) + ' ' + t('dashboardPage.ws.unitMinute')
  if (s < 86400) return Math.round(s / 3600) + ' ' + t('dashboardPage.ws.unitHour')
  return Math.round(s / 86400) + ' ' + t('dashboardPage.ws.unitDay')
})

/** 存储占用可读化：按 1024 进制换算 B/KB/MB/GB/TB */
function formatBytes(n: number): string {
  if (!n || n < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1)
  const val = n / Math.pow(1024, i)
  return `${i === 0 ? val : val.toFixed(1)} ${units[i]}`
}
const storageText = computed(() => formatBytes(ws.storageUsed))

// 快捷入口：替代原静态欢迎语文案，直达各业务页
const quickEntries = computed(() => [
  { path: '/task', label: t('dashboardPage.ws.quickTask'), icon: Checked },
  { path: '/chat', label: t('dashboardPage.ws.quickChat'), icon: ChatDotRound },
  { path: '/wiki', label: t('dashboardPage.ws.quickWiki'), icon: Reading },
  { path: '/export', label: t('dashboardPage.ws.quickExport'), icon: Goods },
  { path: '/profile', label: t('dashboardPage.ws.quickProfile'), icon: UserFilled },
  { path: '/invitation', label: t('dashboardPage.ws.quickInvite'), icon: Plus }
])

function go(path: string) {
  void router.push(path)
}

onMounted(async () => {
  try {
    if (isAdmin.value) {
      // 平台超级管理员：加载运营数据看板
      try {
        overview.value = await getDashboardOverview()
      } catch {
        ElMessage.warning(t('dashboardPage.msg.overviewFailed'))
      }
      loadTrend()
      // 兼容旧统计接口，作为兑底
      const rbacStats = await getRBACStats()
      if (!rbacStats) ElMessage.warning(t('dashboardPage.msg.statsPartialFailed'))
    } else {
      // 租户工作台：五路统计并发拉取，单路失败仅该卡降级为零值
      await ws.load()
    }
  } catch (e) {
    ElMessage.warning(t('dashboardPage.msg.statsPartialFailed'))
  }
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.stat-card {
  height: 120px;
}
.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 100%;
}
.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
}
.user-icon { background: #3b82f6; }
.role-icon { background: #10b981; }
.perm-icon { background: #f59e0b; }
.info-icon { background: #8b5cf6; }
.stat-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 26px;
  font-weight: 600;
  color: #111827;
}
.stat-sub {
  margin-top: 4px;
  font-size: 12px;
  color: #6b7280;
}
.overdue-red {
  color: #ef4444;
  font-weight: 600;
}
.todo-list {
  display: flex;
  flex-direction: column;
}
.todo-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 4px;
  border-bottom: 1px solid #f1f5f9;
  text-decoration: none;
  color: inherit;
}
.todo-item:last-child {
  border-bottom: none;
}
.todo-item:hover {
  background: #f8fafc;
}
.todo-label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #374151;
}
.todo-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.quick-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.quick-btn {
  height: 56px;
  font-size: 14px;
}
.section-card {
  min-height: 180px;
}
.sub-stats {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.sub-item {
  flex: 1;
  text-align: center;
  padding: 12px 0;
  background: #f8fafc;
  border-radius: 8px;
}
.sub-num {
  font-size: 22px;
  font-weight: 600;
  color: #111827;
}
.sub-label {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
.welcome-card {
  margin-top: 0;
}
.welcome h3 {
  margin: 0 0 6px;
  color: #111827;
}
.welcome p {
  margin: 0;
  color: #4b5563;
  font-size: 14px;
}
.feature-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 24px;
  color: #374151;
  font-size: 14px;
}
.trend-card :deep(.el-card__header) {
  padding: 12px 16px;
}
.trend-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.trend-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}
.trend-scroll {
  width: 100%;
  overflow-x: auto;
}
.trend-svg {
  display: block;
}
.bar-label {
  font-size: 10px;
  fill: #6b7280;
}
</style>
