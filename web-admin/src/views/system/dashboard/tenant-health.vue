<template>
  <div class="page">
    <el-row :gutter="16" class="summary-row">
      <el-col :span="6">
        <el-card shadow="never" class="summary-card">
          <div class="summary-num">{{ total }}</div>
          <div class="summary-label">租户总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card healthy">
          <div class="summary-num">{{ summary.healthy }}</div>
          <div class="summary-label">健康</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card moderate">
          <div class="summary-num">{{ summary.moderate }}</div>
          <div class="summary-label">中等</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card risk">
          <div class="summary-num">{{ summary.atRisk }}</div>
          <div class="summary-label">风险</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>租户健康度看板</span>
        </div>
      </template>

      <div class="filter-bar">
        <el-input
          v-model="keyword"
          placeholder="按租户名称搜索"
          clearable
          style="width: 220px; margin-right: 12px;"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button @click="loadList">刷新</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;" @row-click="openDetail">
        <el-table-column prop="tenant_name" label="租户" min-width="140" show-overflow-tooltip />
        <el-table-column label="套餐" width="120">
          <template #default="{ row }">{{ row.plan_name || '-' }}</template>
        </el-table-column>
        <el-table-column label="订阅" width="150" align="center">
          <template #default="{ row }">
            <el-tag :type="subTagType(row.subscription_status)" size="small">
              {{ subLabel(row.subscription_status) }}
            </el-tag>
            <div v-if="row.subscription_status === 1" class="sub-days">
              <span v-if="row.days_remaining < 0">长期</span>
              <span v-else>剩 {{ row.days_remaining }} 天</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="用户" width="100" align="center">
          <template #default="{ row }">{{ row.active_user_count }}/{{ row.user_count }}</template>
        </el-table-column>
        <el-table-column label="任务(完成/总)" width="130" align="center">
          <template #default="{ row }">{{ row.task_completed_count }}/{{ row.task_count }}</template>
        </el-table-column>
        <el-table-column label="文档" width="80" align="center">
          <template #default="{ row }">{{ row.wiki_doc_count }}</template>
        </el-table-column>
        <el-table-column label="存储" width="110" align="center">
          <template #default="{ row }">{{ formatSize(row.storage_used) }}</template>
        </el-table-column>
        <el-table-column label="近7天活跃" width="110" align="center">
          <template #default="{ row }">{{ row.audit_count_7d }}</template>
        </el-table-column>
        <el-table-column label="健康度" width="150" align="center">
          <template #default="{ row }">
            <el-progress
              :percentage="row.health_score"
              :color="levelColor(row.health_level)"
              :stroke-width="12"
              text-inside
            />
          </template>
        </el-table-column>
        <el-table-column label="评级" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="levelTagType(row.health_level)" size="small">{{ levelLabel(row.health_level) }}</el-tag>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-drawer v-model="detailVisible" title="租户健康度详情" size="420px">
      <div v-if="detail" class="detail">
        <div class="detail-head">
          <span class="detail-name">{{ detail.tenant_name }}</span>
          <el-tag :type="levelTagType(detail.health_level)" size="small">{{ levelLabel(detail.health_level) }}</el-tag>
        </div>
        <el-progress :percentage="detail.health_score" :color="levelColor(detail.health_level)" :stroke-width="16" />

        <el-descriptions :column="1" border class="detail-desc">
          <el-descriptions-item label="租户状态">{{ detail.status === 1 ? '启用' : '禁用' }}</el-descriptions-item>
          <el-descriptions-item label="套餐">{{ detail.plan_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="订阅状态">{{ subLabel(detail.subscription_status) }}</el-descriptions-item>
          <el-descriptions-item label="到期时间">{{ formatTime(detail.subscription_expires_at) }}</el-descriptions-item>
          <el-descriptions-item label="剩余天数">
            {{ detail.subscription_status === 1 ? (detail.days_remaining < 0 ? '长期有效' : detail.days_remaining + ' 天') : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="用户（有效/总）">{{ detail.active_user_count }}/{{ detail.user_count }}</el-descriptions-item>
          <el-descriptions-item label="任务（完成/总）">{{ detail.task_completed_count }}/{{ detail.task_count }}</el-descriptions-item>
          <el-descriptions-item label="Wiki 文档">{{ detail.wiki_doc_count }}</el-descriptions-item>
          <el-descriptions-item label="存储占用">{{ formatSize(detail.storage_used) }}</el-descriptions-item>
          <el-descriptions-item label="近 7 天操作">{{ detail.audit_count_7d }}</el-descriptions-item>
          <el-descriptions-item label="最近活动">{{ formatTime(detail.last_activity_at) }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getTenantHealthList,
  getTenantHealthDetail,
  type TenantHealthItem
} from '@/api/modules/dashboard'

const loading = ref(false)
const list = ref<TenantHealthItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const summary = reactive({ healthy: 0, moderate: 0, atRisk: 0 })

async function loadList() {
  loading.value = true
  try {
    const res = await getTenantHealthList({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined
    })
    list.value = res.items || []
    total.value = res.total || 0
    summary.healthy = res.healthy || 0
    summary.moderate = res.moderate || 0
    summary.atRisk = res.at_risk || 0
  } catch {
    ElMessage.error('获取租户健康度失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadList()
}

const detailVisible = ref(false)
const detail = ref<TenantHealthItem | null>(null)

async function openDetail(row: TenantHealthItem) {
  detailVisible.value = true
  detail.value = row
  try {
    detail.value = await getTenantHealthDetail(row.tenant_id)
  } catch {
    // 详情拉取失败时保留列表行数据，已由拦截器提示
  }
}

function subLabel(status: number) {
  const map: Record<number, string> = { 0: '无订阅', 1: '生效', 2: '到期', 3: '取消' }
  return map[status] || '未知'
}

function subTagType(status: number) {
  const map: Record<number, string> = { 0: 'info', 1: 'success', 2: 'warning', 3: 'danger' }
  return map[status] || 'info'
}

function levelLabel(level: string) {
  const map: Record<string, string> = { healthy: '健康', moderate: '中等', 'at_risk': '风险' }
  return map[level] || level
}

function levelTagType(level: string) {
  const map: Record<string, string> = { healthy: 'success', moderate: 'warning', 'at_risk': 'danger' }
  return map[level] || 'info'
}

function levelColor(level: string) {
  const map: Record<string, string> = { healthy: '#67c23a', moderate: '#e6a23c', 'at_risk': '#f56c6c' }
  return map[level] || '#409eff'
}

function formatSize(bytes: number) {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatTime(t?: string | null) {
  if (!t) return '-'
  return t.replace('T', ' ').slice(0, 19)
}

onMounted(loadList)
</script>

<style scoped>
.summary-row {
  margin-bottom: 16px;
}
.summary-card {
  text-align: center;
}
.summary-num {
  font-size: 28px;
  font-weight: 600;
}
.summary-label {
  margin-top: 4px;
  color: #909399;
}
.summary-card.healthy .summary-num {
  color: #67c23a;
}
.summary-card.moderate .summary-num {
  color: #e6a23c;
}
.summary-card.risk .summary-num {
  color: #f56c6c;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.sub-days {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.detail-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.detail-name {
  font-size: 16px;
  font-weight: 600;
}
.detail-desc {
  margin-top: 16px;
}
</style>
