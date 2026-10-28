<template>
  <div class="page">
    <el-row :gutter="16" class="summary-row">
      <el-col :span="6">
        <el-card shadow="never" class="summary-card">
          <div class="summary-num">{{ stats.total_queries }}</div>
          <div class="summary-label">{{ t('slowQuery.sumTotal') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card risk">
          <div class="summary-num">{{ stats.slow_queries }}</div>
          <div class="summary-label">{{ t('slowQuery.sumSlow') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card moderate">
          <div class="summary-num">{{ stats.max_duration_ms }}<span class="unit">ms</span></div>
          <div class="summary-label">{{ t('slowQuery.sumMax') }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="summary-card healthy">
          <div class="summary-num">{{ avgSlowDisplay }}<span class="unit">ms</span></div>
          <div class="summary-label">{{ t('slowQuery.sumAvg') }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('slowQuery.title') }}</span>
          <span class="threshold-tip">{{ t('slowQuery.thresholdTip', stats.threshold_ms, stats.capacity) }}</span>
        </div>
      </template>

      <div class="filter-bar">
        <el-button type="primary" @click="load">{{ t('common.refresh') }}</el-button>
        <el-button @click="toggleAuto">{{ auto ? t('slowQuery.stopAuto') : t('slowQuery.startAuto') }}</el-button>
        <el-button type="danger" plain @click="handleReset">{{ t('slowQuery.resetStats') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="entries" stripe style="width: 100%;">
        <el-table-column :label="t('slowQuery.colTime')" width="180">
          <template #default="{ row }">{{ formatTime(row.time) }}</template>
        </el-table-column>
        <el-table-column :label="t('slowQuery.colDuration')" width="110" align="right" sortable :sort-by="(r: SlowQueryEntry) => r.duration_ms">
          <template #default="{ row }">
            <span :class="durationClass(row.duration_ms)">{{ row.duration_ms }} ms</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('slowQuery.colRows')" width="90" align="right">
          <template #default="{ row }">{{ row.rows }}</template>
        </el-table-column>
        <el-table-column prop="sql" label="SQL" min-width="320" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="sql-text">{{ row.sql }}</code>
          </template>
        </el-table-column>
        <el-table-column :label="t('slowQuery.colStatus')" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.error" type="danger" size="small">{{ t('slowQuery.tagError') }}</el-tag>
            <el-tag v-else type="warning" size="small">{{ t('slowQuery.tagSlow') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Trace ID" width="120" show-overflow-tooltip>
          <template #default="{ row }">{{ row.trace_id || '-' }}</template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && entries.length === 0" :description="t('slowQuery.empty')" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getSlowQueries,
  resetSlowQueries,
  type SlowQueryEntry,
  type SlowQueryStats
} from '@/api/modules/dashboard'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const loading = ref(false)
const entries = ref<SlowQueryEntry[]>([])
const stats = reactive<SlowQueryStats>({
  threshold_ms: 0,
  total_queries: 0,
  slow_queries: 0,
  max_duration_ms: 0,
  avg_slow_ms: 0,
  capacity: 0
})

const avgSlowDisplay = computed(() => Math.round(stats.avg_slow_ms * 10) / 10)

async function load() {
  loading.value = true
  try {
    const res = await getSlowQueries()
    entries.value = res.entries || []
    Object.assign(stats, res.stats || {})
  } catch {
    ElMessage.error(t('slowQuery.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleReset() {
  try {
    await ElMessageBox.confirm(t('slowQuery.resetConfirm'), t('slowQuery.resetTitle'), {
      type: 'warning',
      confirmButtonText: t('slowQuery.resetBtn'),
      cancelButtonText: t('common.cancel')
    })
  } catch {
    return
  }
  try {
    await resetSlowQueries()
    ElMessage.success(t('slowQuery.resetDone'))
    load()
  } catch {
    // 错误已由拦截器提示
  }
}

function durationClass(ms: number) {
  if (ms >= stats.threshold_ms * 4) return 'dur-critical'
  if (ms >= stats.threshold_ms * 2) return 'dur-high'
  return 'dur-normal'
}

function formatTime(t?: string | null) {
  if (!t) return '-'
  return t.replace('T', ' ').slice(0, 19)
}

const auto = ref(false)
let timer: ReturnType<typeof setInterval> | null = null
function toggleAuto() {
  auto.value = !auto.value
  if (auto.value) {
    timer = setInterval(load, 5000)
  } else if (timer) {
    clearInterval(timer)
    timer = null
  }
}

onMounted(load)
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
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
.summary-num .unit {
  font-size: 14px;
  color: #909399;
  margin-left: 2px;
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
.threshold-tip {
  font-size: 12px;
  color: #909399;
}
.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.sql-text {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  word-break: break-all;
}
.dur-critical {
  color: #f56c6c;
  font-weight: 600;
}
.dur-high {
  color: #e6a23c;
  font-weight: 600;
}
.dur-normal {
  color: #606266;
}
</style>
