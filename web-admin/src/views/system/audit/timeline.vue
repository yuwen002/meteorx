<template>
  <div class="page">
    <!-- 搜索区域 -->
    <el-card shadow="never" class="search-card">
      <div class="search-bar">
        <el-input
          v-model="searchUserID"
          placeholder="输入用户ID"
          clearable
          style="width: 200px"
          @keyup.enter="loadTimeline"
        />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 240px"
        />
        <el-button type="primary" @click="loadTimeline">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <span v-if="currentUser" class="user-info">
          当前用户：<strong>{{ currentUser }}</strong>
        </span>
      </div>
    </el-card>

    <!-- 统计概览 -->
    <el-row :gutter="16" class="stats-row" v-if="currentUser">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total-icon"><el-icon><Document /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">操作总数</div>
              <div class="stat-value">{{ totalStats.total }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon days-icon"><el-icon><Calendar /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">覆盖天数</div>
              <div class="stat-value">{{ timelineData.length }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon success-icon"><el-icon><CircleCheck /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">成功操作</div>
              <div class="stat-value">{{ totalStats.success }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon fail-icon"><el-icon><CircleClose /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">失败操作</div>
              <div class="stat-value">{{ totalStats.failure }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 时间线 -->
    <div v-if="currentUser" class="timeline-container">
      <div class="timeline-year" v-for="group in timelineData" :key="group.date">
        <div class="timeline-date-header">
          <span class="date-badge">{{ group.date }}</span>
          <span class="date-count">共 {{ group.count }} 条操作</span>
          <span class="date-success">成功 {{ group.success }}</span>
          <span class="date-failure">失败 {{ group.failure }}</span>
        </div>
        <div class="timeline-items">
          <div
            v-for="log in group.logs"
            :key="log.id"
            class="timeline-item"
            @click="openDetail(log)"
          >
            <div class="timeline-dot" :class="getDotClass(log)"></div>
            <div class="timeline-content">
              <div class="timeline-time">{{ formatTime(log.created_at) }}</div>
              <div class="timeline-action">
                <el-tag size="small" :type="getActionType(log.action)">{{ getActionLabel(log.action) }}</el-tag>
                <el-tag size="small" :type="getRiskLevelType(log.risk_level)" style="margin-left: 4px;">
                  {{ getRiskLevelLabel(log.risk_level) }}
                </el-tag>
                <span class="timeline-path">{{ log.method }} {{ log.path }}</span>
              </div>
              <div class="timeline-meta">
                <span>IP: {{ log.client_ip }}</span>
                <span v-if="log.ip_location"> | {{ log.ip_location }}</span>
                <span> | {{ log.duration }}ms</span>
                <el-tag size="small" :type="log.result === 'success' ? 'success' : 'danger'" style="margin-left: 4px;">
                  {{ log.result === 'success' ? '成功' : '失败' }}
                </el-tag>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTimeline"
          @current-change="loadTimeline"
        />
      </div>
    </div>

    <!-- 未选择用户提示 -->
    <el-empty v-else description="请输入用户ID查询操作时间线" />

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="日志详情" width="900px" destroy-on-close>
      <el-descriptions v-if="currentLog" :column="2" border>
        <el-descriptions-item label="日志ID">{{ currentLog.id }}</el-descriptions-item>
        <el-descriptions-item label="请求ID">{{ currentLog.request_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="操作用户">{{ currentLog.username }} ({{ currentLog.user_id }})</el-descriptions-item>
        <el-descriptions-item label="模块">{{ currentLog.module }}</el-descriptions-item>
        <el-descriptions-item label="操作类型">{{ getActionLabel(currentLog.action) }}</el-descriptions-item>
        <el-descriptions-item label="风险等级">
          <el-tag size="small" :type="getRiskLevelType(currentLog.risk_level)">{{ getRiskLevelLabel(currentLog.risk_level) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="HTTP方法">{{ currentLog.method }}</el-descriptions-item>
        <el-descriptions-item label="请求路径" :span="2">{{ currentLog.path }}</el-descriptions-item>
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
        <el-descriptions-item label="耗时">{{ currentLog.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ currentLog.created_at }}</el-descriptions-item>
        <el-descriptions-item v-if="currentLog.error_message" label="错误信息" :span="2">
          <span style="color: #f56c6c">{{ currentLog.error_message }}</span>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { Search, Document, Calendar, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import { getUserTimeline } from '@/api/modules/audit'
import type { TimelineItem, AuditLogItem } from '@/api/modules/audit'

const searchUserID = ref('')
const dateRange = ref<string[]>([])
const currentUser = ref('')
const timelineData = ref<TimelineItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const detailVisible = ref(false)
const currentLog = ref<AuditLogItem | null>(null)

const totalStats = computed(() => {
  let total = 0, success = 0, failure = 0
  timelineData.value.forEach(item => {
    total += item.count
    success += item.success
    failure += item.failure
  })
  return { total, success, failure }
})

async function loadTimeline() {
  if (!searchUserID.value.trim()) return
  currentUser.value = searchUserID.value.trim()
  loading.value = true
  try {
    const res = await getUserTimeline({
      user_id: currentUser.value,
      page: page.value,
      page_size: pageSize.value,
      start_time: dateRange.value.length === 2 ? dateRange.value[0] + ' 00:00:00' : undefined,
      end_time: dateRange.value.length === 2 ? dateRange.value[1] + ' 23:59:59' : undefined
    })
    timelineData.value = res.items || []
    total.value = res.total || 0
  } catch (error) {
    console.error('加载时间线失败', error)
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  searchUserID.value = ''
  dateRange.value = []
  currentUser.value = ''
  timelineData.value = []
  page.value = 1
  total.value = 0
}

function openDetail(log: AuditLogItem) {
  currentLog.value = log
  detailVisible.value = true
}

function formatTime(timeStr: string): string {
  try {
    return timeStr.split(' ')[1] || timeStr
  } catch {
    return timeStr
  }
}

function getDotClass(log: AuditLogItem): string {
  if (log.result === 'failure') return 'dot-failure'
  if (log.risk_level === 'high' || log.risk_level === 'critical') return 'dot-warning'
  return 'dot-success'
}

function getActionLabel(action: string): string {
  const map: Record<string, string> = { create: '创建', update: '更新', delete: '删除', query: '查询', login: '登录', logout: '登出', other: '其他' }
  return map[action] || action
}

function getActionType(action: string): string {
  const map: Record<string, string> = { create: 'success', update: 'warning', delete: 'danger', query: 'info', login: 'primary', logout: 'info' }
  return map[action] || ''
}

function getRiskLevelType(level: string): string {
  const map: Record<string, string> = { low: 'success', medium: 'warning', high: 'danger', critical: 'danger' }
  return map[level] || 'info'
}

function getRiskLevelLabel(level: string): string {
  const map: Record<string, string> = { low: '低', medium: '中', high: '高', critical: '严重' }
  return map[level] || level
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.search-card {
  margin-bottom: 16px;

  .search-bar {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .user-info {
    margin-left: 8px;
    font-size: 14px;
    color: #595959;
  }
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

    &.total-icon { background: #e6f7ff; color: #1890ff; }
    &.days-icon { background: #f6ffed; color: #52c41a; }
    &.success-icon { background: #f6ffed; color: #52c41a; }
    &.fail-icon { background: #fff2f0; color: #ff4d4f; }
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

.timeline-container {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
}

.timeline-year {
  margin-bottom: 24px;
}

.timeline-date-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #fafafa;
  border-radius: 6px;
  margin-bottom: 12px;

  .date-badge {
    font-weight: 600;
    font-size: 15px;
    color: #262626;
  }

  .date-count {
    font-size: 13px;
    color: #8c8c8c;
    margin-left: 8px;
  }

  .date-success {
    font-size: 13px;
    color: #52c41a;
    margin-left: 8px;
  }

  .date-failure {
    font-size: 13px;
    color: #f56c6c;
    margin-left: 8px;
  }
}

.timeline-items {
  padding-left: 20px;
  border-left: 2px solid #e8e8e8;
  margin-left: 8px;
}

.timeline-item {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;

  &:hover {
    background: #f5f5f5;
  }

  &::before {
    content: '';
    position: absolute;
    left: -22px;
    top: 20px;
    width: 10px;
    height: 2px;
    background: #e8e8e8;
  }
}

.timeline-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;

  &.dot-success {
    background: #52c41a;
    border: 2px solid #b7eb8f;
  }

  &.dot-failure {
    background: #f56c6c;
    border: 2px solid #ffa39e;
  }

  &.dot-warning {
    background: #faad14;
    border: 2px solid #ffe58f;
  }
}

.timeline-content {
  flex: 1;
  min-width: 0;
}

.timeline-time {
  font-size: 12px;
  color: #8c8c8c;
  margin-bottom: 4px;
}

.timeline-action {
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.timeline-path {
  font-size: 13px;
  color: #262626;
  font-family: monospace;
  margin-left: 4px;
}

.timeline-meta {
  font-size: 12px;
  color: #8c8c8c;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>