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
              <div class="stat-value">{{ stats.total_count }}</div>
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
              <div class="stat-value">{{ stats.today_count }}</div>
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
              <div class="stat-value">{{ stats.result_stats?.success || 0 }}</div>
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
              <div class="stat-value">{{ stats.result_stats?.failure || 0 }}</div>
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
    <el-dialog v-model="detailVisible" title="日志详情" width="700px" destroy-on-close>
      <el-descriptions :column="2" border v-if="currentLog">
        <el-descriptions-item label="日志ID">{{ currentLog.id }}</el-descriptions-item>
        <el-descriptions-item label="操作用户">{{ currentLog.username }} ({{ currentLog.user_id }})</el-descriptions-item>
        <el-descriptions-item label="租户ID">{{ currentLog.tenant_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="模块">{{ currentLog.module }}</el-descriptions-item>
        <el-descriptions-item label="操作类型">{{ getActionLabel(currentLog.action) }}</el-descriptions-item>
        <el-descriptions-item label="HTTP方法">{{ currentLog.method }}</el-descriptions-item>
        <el-descriptions-item label="请求路径" :span="2">{{ currentLog.path }}</el-descriptions-item>
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
        <el-descriptions-item label="用户代理" :span="2">{{ currentLog.user_agent }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ currentLog.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ currentLog.created_at }}</el-descriptions-item>
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Calendar, CircleCheck, CircleClose, Search, Delete, Download, User, SwitchButton } from '@element-plus/icons-vue'
import { getAuditLogList, getAuditStats, cleanupAuditLogs, exportAuditLogs } from '@/api/modules/audit'
import type { AuditLogItem, AuditLogStats } from '@/api/modules/audit'

const loading = ref(false)
const list = ref<AuditLogItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const search = reactive({
  keyword: '',
  module: '',
  action: '',
  result: '',
  dateRange: [] as string[]
})

const stats = ref<AuditLogStats>({
  total_count: 0,
  today_count: 0,
  action_stats: {},
  module_stats: {},
  result_stats: {}
})

const detailVisible = ref(false)
const currentLog = ref<AuditLogItem | null>(null)

const cleanupVisible = ref(false)
const cleanupForm = reactive({ days: 30 })

onMounted(() => {
  loadList()
  loadStats()
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
      result: search.result || undefined
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

async function loadStats() {
  try {
    const res = await getAuditStats()
    stats.value = res
  } catch (error) {
    console.error('加载统计失败', error)
  }
}

function resetSearch() {
  search.keyword = ''
  search.module = ''
  search.action = ''
  search.result = ''
  search.dateRange = []
  page.value = 1
  loadList()
}

// 快捷筛选
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
    loadStats()
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

function formatJson(jsonStr: string): string {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
}

// 导出审计日志
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
  
  // 创建临时链接下载
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