<template>
  <div class="page">
    <el-card shadow="never" class="search-card">
      <div class="search-bar">
        <el-input v-model="search.user_id" placeholder="用户ID" clearable style="width: 200px" />
        <el-button type="primary" @click="loadSessions">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>会话分析</span>
          <el-button @click="loadSessions">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="sessions" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="session_id" label="会话ID" width="280" show-overflow-tooltip />
        <el-table-column prop="user_id" label="用户ID" width="150" />
        <el-table-column prop="username" label="用户名" width="120" />
        <el-table-column prop="total_requests" label="请求次数" width="100" />
        <el-table-column prop="success_count" label="成功次数" width="100">
          <template #default="{ row }">
            <span style="color: #52c41a">{{ row.success_count }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="failure_count" label="失败次数" width="100">
          <template #default="{ row }">
            <span style="color: #f56c6c">{{ row.failure_count }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="avg_duration" label="平均耗时(ms)" width="120">
          <template #default="{ row }">
            <span :class="getDurationClass(row.avg_duration)">{{ row.avg_duration }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="first_request" label="首次请求" width="180" />
        <el-table-column prop="last_request" label="最后请求" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openSessionDetail(row.session_id)">
              查看详情
            </el-button>
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
          @size-change="loadSessions"
          @current-change="loadSessions"
        />
      </div>
    </el-card>

    <el-dialog v-model="detailVisible" title="会话详情" width="1200px" destroy-on-close>
      <div v-if="sessionDetail" class="session-detail">
        <el-descriptions :column="3" border class="session-summary">
          <el-descriptions-item label="会话ID">{{ sessionDetail.session_id }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ sessionDetail.username }} ({{ sessionDetail.user_id }})</el-descriptions-item>
          <el-descriptions-item label="总请求数">{{ sessionDetail.total_requests }}</el-descriptions-item>
          <el-descriptions-item label="成功次数">
            <span style="color: #52c41a">{{ sessionDetail.success_count }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="失败次数">
            <span style="color: #f56c6c">{{ sessionDetail.failure_count }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="平均耗时">{{ sessionDetail.avg_duration }}ms</el-descriptions-item>
          <el-descriptions-item label="首次请求">{{ sessionDetail.first_request }}</el-descriptions-item>
          <el-descriptions-item label="最后请求">{{ sessionDetail.last_request }}</el-descriptions-item>
          <el-descriptions-item label="会话时长">{{ sessionDetail.duration_minutes }}分钟</el-descriptions-item>
        </el-descriptions>

        <h3 style="margin: 20px 0 12px">操作时间线</h3>
        <el-timeline>
          <el-timeline-item
            v-for="log in sessionDetail.logs"
            :key="log.id"
            :timestamp="log.created_at"
            placement="top"
            :color="log.result === 'success' ? '#52c41a' : '#f56c6c'"
          >
            <el-card>
              <div class="timeline-item">
                <div class="timeline-header">
                  <el-tag size="small" :type="getActionType(log.action)">{{ getActionLabel(log.action) }}</el-tag>
                  <el-tag size="small" type="info">{{ log.module }}</el-tag>
                  <el-tag size="small" :type="getMethodType(log.method)">{{ log.method }}</el-tag>
                  <span class="timeline-path">{{ log.path }}</span>
                  <el-tag size="small" :type="log.status_code >= 400 ? 'danger' : 'success'">
                    {{ log.status_code }}
                  </el-tag>
                </div>
                <div class="timeline-body">
                  <div class="timeline-info">
                    <span>IP: {{ log.client_ip }}</span>
                    <span v-if="log.ip_location">位置: {{ log.ip_location }}</span>
                    <span>耗时: {{ log.duration }}ms</span>
                  </div>
                  <div v-if="log.error_message" class="timeline-result">
                    <span style="color: #f56c6c">错误: {{ log.error_message }}</span>
                  </div>
                </div>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'
import { get } from '@/api/request'

interface SessionSummary {
  session_id: string
  user_id: string
  username: string
  total_requests: number
  success_count: number
  failure_count: number
  avg_duration: number
  first_request: string
  last_request: string
  duration_minutes: number
}

interface SessionDetail extends SessionSummary {
  logs: any[]
}

const loading = ref(false)
const sessions = ref<SessionSummary[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const search = reactive({
  user_id: ''
})

const detailVisible = ref(false)
const sessionDetail = ref<SessionDetail | null>(null)

onMounted(() => {
  loadSessions()
})

async function loadSessions() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
      user_id: search.user_id || undefined
    }
    const res = await get('/audit/sessions', params)
    sessions.value = res.items || []
    total.value = res.total || 0
  } catch (error) {
    console.error('加载会话列表失败', error)
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  search.user_id = ''
  page.value = 1
  loadSessions()
}

async function openSessionDetail(sessionId: string) {
  try {
    const res = await get(`/audit/sessions/${sessionId}/logs`)
    sessionDetail.value = res
    detailVisible.value = true
  } catch (error) {
    console.error('加载会话详情失败', error)
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
}

.table-card {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.session-summary {
  margin-bottom: 20px;
}

.timeline-item {
  .timeline-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .timeline-path {
      flex: 1;
      font-family: monospace;
      font-size: 13px;
      color: #595959;
    }
  }

  .timeline-body {
    .timeline-info {
      display: flex;
      gap: 16px;
      font-size: 13px;
      color: #8c8c8c;
      margin-bottom: 4px;
    }

    .timeline-result {
      font-size: 13px;
    }
  }
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
</style>