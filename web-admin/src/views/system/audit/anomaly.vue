<template>
  <div class="page">
    <!-- 检测参数 -->
    <el-card shadow="never" class="search-card">
      <div class="search-bar">
        <span class="label">检测窗口：</span>
        <el-select v-model="windowMinutes" style="width: 160px">
          <el-option label="10 分钟" :value="10" />
          <el-option label="30 分钟" :value="30" />
          <el-option label="1 小时" :value="60" />
          <el-option label="2 小时" :value="120" />
        </el-select>
        <span class="label" style="margin-left: 16px;">失败阈值：</span>
        <el-input-number v-model="threshold" :min="3" :max="100" style="width: 120px" />
        <el-button type="primary" @click="detectAnomalies" :loading="loading">
          <el-icon><Search /></el-icon>开始检测
        </el-button>
      </div>
    </el-card>

    <!-- 统计 -->
    <el-row :gutter="16" class="stats-row" v-if="anomalies.length > 0">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon alert-icon"><el-icon><Warning /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">异常总数</div>
              <div class="stat-value">{{ anomalies.length }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon high-icon"><el-icon><CircleClose /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">高风险</div>
              <div class="stat-value">{{ highRiskCount }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon medium-icon"><el-icon><Warning /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">中风险</div>
              <div class="stat-value">{{ mediumRiskCount }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon user-icon"><el-icon><User /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">涉及用户</div>
              <div class="stat-value">{{ uniqueUserCount }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 异常列表 -->
    <el-card shadow="never" class="table-card" v-if="anomalies.length > 0">
      <template #header>
        <div class="card-header">
          <span>异常检测结果</span>
          <el-tag type="warning" size="small">窗口 {{ windowMinutes }} 分钟 | 阈值 {{ threshold }} 次</el-tag>
        </div>
      </template>
      <el-table :data="anomalies" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="anomaly_label" label="异常类型" width="140">
          <template #default="{ row }">
            <el-tag :type="getAnomalyTagType(row.anomaly_type)" size="small">
              {{ row.anomaly_label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险等级" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getRiskLevelType(row.risk_level)">
              {{ getRiskLevelLabel(row.risk_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="total_count" label="总操作" width="80" />
        <el-table-column prop="failure_count" label="失败次数" width="100">
          <template #default="{ row }">
            <span style="color: #f56c6c; font-weight: bold;">{{ row.failure_count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="失败率" width="100">
          <template #default="{ row }">
            <span :style="{ color: row.total_count > 0 && row.failure_count / row.total_count > 0.5 ? '#f56c6c' : '#faad14' }">
              {{ row.total_count > 0 ? (row.failure_count / row.total_count * 100).toFixed(1) + '%' : '0%' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="window_minutes" label="检测窗口" width="100">
          <template #default="{ row }">{{ row.window_minutes }} 分钟</template>
        </el-table-column>
        <el-table-column prop="first_seen" label="首次出现" width="170" />
        <el-table-column prop="last_seen" label="最后出现" width="170" />
        <el-table-column prop="details" label="详情" min-width="200" show-overflow-tooltip />
      </el-table>
    </el-card>

    <!-- 空状态 -->
    <el-empty v-if="!loading && anomalies.length === 0 && hasDetected" description="未检测到异常操作" />
    <el-empty v-if="!hasDetected" description="设置检测参数后点击"开始检测"" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, Warning, CircleClose, User } from '@element-plus/icons-vue'
import { getAnomalyLogs } from '@/api/modules/audit'
import type { AnomalyLogItem } from '@/api/modules/audit'

const anomalies = ref<AnomalyLogItem[]>([])
const loading = ref(false)
const hasDetected = ref(false)
const threshold = ref(5)
const windowMinutes = ref(30)

const highRiskCount = computed(() => anomalies.value.filter(a => a.risk_level === 'high' || a.risk_level === 'critical').length)
const mediumRiskCount = computed(() => anomalies.value.filter(a => a.risk_level === 'medium').length)
const uniqueUserCount = computed(() => {
  const users = new Set(anomalies.value.map(a => a.user_id))
  return users.size
})

async function detectAnomalies() {
  loading.value = true
  hasDetected.value = true
  try {
    const res = await getAnomalyLogs({
      threshold: threshold.value,
      window_minutes: windowMinutes.value
    })
    anomalies.value = res || []
  } catch (error) {
    console.error('异常检测失败', error)
  } finally {
    loading.value = false
  }
}

function getAnomalyTagType(type: string): string {
  const map: Record<string, string> = {
    high_failure_rate: 'danger',
    geo_anomaly: 'warning',
    brute_force: 'danger'
  }
  return map[type] || 'info'
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
    flex-wrap: wrap;
    gap: 0;

    .label {
      font-size: 14px;
      color: #595959;
      white-space: nowrap;
    }
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

    &.alert-icon { background: #fff2f0; color: #ff4d4f; }
    &.high-icon { background: #fff2f0; color: #f56c6c; }
    &.medium-icon { background: #fff7e6; color: #faad14; }
    &.user-icon { background: #e6f7ff; color: #1890ff; }
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
  margin-top: 16px;

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}
</style>