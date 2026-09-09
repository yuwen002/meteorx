<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total-icon"><el-icon><User /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">登录总次数</div>
              <div class="stat-value">{{ loginStats.total }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon success-icon"><el-icon><CircleCheck /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">登录成功</div>
              <div class="stat-value">{{ loginStats.success }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon fail-icon"><el-icon><CircleClose /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">登录失败</div>
              <div class="stat-value">{{ loginStats.failure }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon ip-icon"><el-icon><Monitor /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">独立IP数</div>
              <div class="stat-value">{{ loginStats.uniqueIPs }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="table-card">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input v-model="query.username" placeholder="用户名" clearable style="width: 160px" @keyup.enter="loadList" />
        <el-select v-model="query.result" placeholder="结果" clearable style="width: 120px">
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failure" />
        </el-select>
        <el-date-picker
          v-model="query.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 240px"
        />
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </div>

      <!-- 登录日志列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="50" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="action" label="类型" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.action === 'login' ? 'primary' : 'info'">
              {{ row.action === 'login' ? '登录' : '登出' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result" label="结果" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
              {{ row.result === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="getRiskLevelType(row.risk_level)">{{ getRiskLevelLabel(row.risk_level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="客户端IP" width="140" />
        <el-table-column prop="ip_location" label="地理位置" width="150" show-overflow-tooltip />
        <el-table-column prop="device_info" label="设备信息" width="160" show-overflow-tooltip />
        <el-table-column prop="user_agent" label="User-Agent" min-width="200" show-overflow-tooltip />
        <el-table-column prop="error_message" label="错误信息" width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span style="color: #f56c6c">{{ row.error_message || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180" />
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, User, CircleCheck, CircleClose, Monitor } from '@element-plus/icons-vue'
import { getAuditLogList } from '@/api/modules/audit'
import type { AuditLogItem } from '@/api/modules/audit'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<
  AuditLogItem,
  { username: string; result: string; dateRange: string[] }
>({
  fetchList: async (params) => {
    const res = await getAuditLogList({
      page: params.page,
      page_size: params.page_size,
      action: 'login,logout',
      username: params.username || undefined,
      result: params.result || undefined,
      start_time: params.dateRange.length === 2 ? params.dateRange[0] + ' 00:00:00' : undefined,
      end_time: params.dateRange.length === 2 ? params.dateRange[1] + ' 23:59:59' : undefined
    })
    return toPageResult(res)
  },
  initialQuery: {
    username: '',
    result: '',
    dateRange: []
  },
  defaultPageSize: 20
})

const loadList = reload

const loginStats = computed(() => {
  const items = list.value as AuditLogItem[]
  let total = 0, success = 0, failure = 0
  const ips = new Set<string>()
  items.forEach(item => {
    total++
    if (item.result === 'success') success++
    else failure++
    if (item.client_ip) ips.add(item.client_ip)
  })
  return { total, success, failure, uniqueIPs: ips.size }
})

onMounted(() => {
  loadList()
})

function resetSearch() {
  reset()
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
    &.success-icon { background: #f6ffed; color: #52c41a; }
    &.fail-icon { background: #fff2f0; color: #ff4d4f; }
    &.ip-icon { background: #fff7e6; color: #fa8c16; }
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
</style>