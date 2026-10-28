<template>
  <div class="page">
    <el-row :gutter="16" class="summary-row">
      <el-col :span="6">
        <el-card shadow="never" class="summary-card">
          <div class="summary-num">{{ total }}</div>
          <div class="summary-label">{{ t('healthPage.totalTenants') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card healthy">
          <div class="summary-num">{{ summary.healthy }}</div>
          <div class="summary-label">{{ t('healthPage.level.healthy') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card moderate">
          <div class="summary-num">{{ summary.moderate }}</div>
          <div class="summary-label">{{ t('healthPage.level.moderate') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card risk">
          <div class="summary-num">{{ summary.atRisk }}</div>
          <div class="summary-label">{{ t('healthPage.level.atRisk') }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('healthPage.title') }}</span>
        </div>
      </template>

      <div class="filter-bar">
        <el-input
          v-model="keyword"
          :placeholder="t('healthPage.searchPlaceholder')"
          clearable
          style="width: 220px; margin-right: 12px;"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-button type="primary" @click="handleSearch">{{ t('common.query') }}</el-button>
        <el-button @click="loadList">{{ t('common.refresh') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;" @row-click="openDetail">
        <el-table-column prop="tenant_name" :label="t('healthPage.colTenant')" min-width="140" show-overflow-tooltip />
        <el-table-column :label="t('healthPage.colPlan')" width="120">
          <template #default="{ row }">{{ row.plan_name || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colSubscription')" width="150" align="center">
          <template #default="{ row }">
            <el-tag :type="subTagType(row.subscription_status)" size="small">
              {{ subLabel(row.subscription_status) }}
            </el-tag>
            <div v-if="row.subscription_status === 1" class="sub-days">
              <span v-if="row.days_remaining < 0">{{ t('healthPage.longTerm') }}</span>
              <span v-else>{{ t('healthPage.daysLeft', row.days_remaining) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colUsers')" width="100" align="center">
          <template #default="{ row }">{{ row.active_user_count }}/{{ row.user_count }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colTasks')" width="130" align="center">
          <template #default="{ row }">{{ row.task_completed_count }}/{{ row.task_count }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colDocs')" width="80" align="center">
          <template #default="{ row }">{{ row.wiki_doc_count }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colStorage')" width="110" align="center">
          <template #default="{ row }">{{ formatSize(row.storage_used) }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colActive7d')" width="110" align="center">
          <template #default="{ row }">{{ row.audit_count_7d }}</template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colHealth')" width="150" align="center">
          <template #default="{ row }">
            <el-progress
              :percentage="row.health_score"
              :color="levelColor(row.health_level)"
              :stroke-width="12"
              text-inside
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('healthPage.colLevel')" width="90" align="center">
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

    <el-drawer v-model="detailVisible" :title="t('healthPage.drawerTitle')" size="420px">
      <div v-if="detail" class="detail">
        <div class="detail-head">
          <span class="detail-name">{{ detail.tenant_name }}</span>
          <el-tag :type="levelTagType(detail.health_level)" size="small">{{ levelLabel(detail.health_level) }}</el-tag>
        </div>
        <el-progress :percentage="detail.health_score" :color="levelColor(detail.health_level)" :stroke-width="16" />

        <el-descriptions :column="1" border class="detail-desc">
          <el-descriptions-item :label="t('healthPage.dStatus')">{{ detail.status === 1 ? t('common.status.enabled') : t('common.status.disabled') }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.colPlan')">{{ detail.plan_name || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dSubStatus')">{{ subLabel(detail.subscription_status) }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dExpiresAt')">{{ formatTime(detail.subscription_expires_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dRemainDays')">
            {{ detail.subscription_status === 1 ? (detail.days_remaining < 0 ? t('healthPage.forever') : t('healthPage.daysN', detail.days_remaining)) : '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dUsers')">{{ detail.active_user_count }}/{{ detail.user_count }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dTasks')">{{ detail.task_completed_count }}/{{ detail.task_count }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dDocs')">{{ detail.wiki_doc_count }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dStorage')">{{ formatSize(detail.storage_used) }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dOps7d')">{{ detail.audit_count_7d }}</el-descriptions-item>
          <el-descriptions-item :label="t('healthPage.dLastActivity')">{{ formatTime(detail.last_activity_at) }}</el-descriptions-item>
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
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

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
    ElMessage.error(t('healthPage.loadFailed'))
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
  const map: Record<number, string> = {
    0: t('healthPage.sub.none'),
    1: t('healthPage.sub.active'),
    2: t('healthPage.sub.expired'),
    3: t('healthPage.sub.cancelled')
  }
  return map[status] || t('healthPage.sub.unknown')
}

function subTagType(status: number) {
  const map: Record<number, string> = { 0: 'info', 1: 'success', 2: 'warning', 3: 'danger' }
  return map[status] || 'info'
}

function levelLabel(level: string) {
  const map: Record<string, string> = {
    healthy: t('healthPage.level.healthy'),
    moderate: t('healthPage.level.moderate'),
    'at_risk': t('healthPage.level.atRisk')
  }
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
