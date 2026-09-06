<template>
  <div class="stats-panel">
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #409eff">
            <el-icon :size="24"><View /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_views || 0 }}</div>
            <div class="stat-label">总浏览量</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #67c23a">
            <el-icon :size="24"><Edit /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_edits || 0 }}</div>
            <div class="stat-label">总编辑次数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #e6a23c">
            <el-icon :size="24"><Download /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_downloads || 0 }}</div>
            <div class="stat-label">总下载次数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #f56c6c">
            <el-icon :size="24"><Share /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_shares || 0 }}</div>
            <div class="stat-label">总分享次数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>独立访客</span>
              <el-tag type="success">{{ stats.unique_viewers || 0 }}</el-tag>
            </div>
          </template>
          <el-empty v-if="!stats.unique_viewers" description="暂无数据" />
          <div v-else class="stat-detail">
            <p>最近浏览: {{ stats.last_viewed_at ? formatTime(stats.last_viewed_at) : '无' }}</p>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>访问日志</span>
              <el-button link type="primary" @click="showLogs = !showLogs">
                {{ showLogs ? '收起' : '展开' }}
              </el-button>
            </div>
          </template>
          <div v-if="showLogs">
            <el-table :data="accessLogs" style="width: 100%" max-height="300">
              <el-table-column prop="user_name" label="用户" width="120" />
              <el-table-column prop="action" label="操作" width="100">
                <template #default="{ row }">
                  <el-tag :type="getActionType(row.action)" size="small">
                    {{ getActionLabel(row.action) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="ip_address" label="IP地址" width="140" />
              <el-table-column prop="created_at" label="时间" width="180">
                <template #default="{ row }">
                  {{ formatTime(row.created_at) }}
                </template>
              </el-table-column>
            </el-table>
            <el-pagination
              v-if="totalLogs > 0"
              v-model:current-page="logPage"
              :page-size="logPageSize"
              :total="totalLogs"
              layout="prev, pager, next"
              style="margin-top: 12px; justify-content: center"
              @current-change="loadAccessLogs"
            />
          </div>
          <el-empty v-else description="点击展开查看访问日志" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { View, Edit, Download, Share } from '@element-plus/icons-vue'
import {
  getDocumentStats,
  listAccessLogs,
  type DocumentStats,
  type DocumentAccessLog
} from '@/api/modules/wiki'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  documentId: string
}>()

const stats = ref<DocumentStats>({
  document_id: '',
  total_views: 0,
  total_edits: 0,
  total_downloads: 0,
  total_shares: 0,
  unique_viewers: 0,
  last_viewed_at: ''
})

const showLogs = ref(false)
const accessLogs = ref<DocumentAccessLog[]>([])
const logPage = ref(1)
const logPageSize = 10
const totalLogs = ref(0)

watch(
  () => props.documentId,
  async () => {
    await loadStats()
  },
  { immediate: true }
)

watch(showLogs, async (val) => {
  if (val) {
    await loadAccessLogs()
  }
})

async function loadStats() {
  try {
    stats.value = await getDocumentStats(props.documentId)
  } catch (error) {
    ElMessage.error('加载统计数据失败')
  }
}

async function loadAccessLogs() {
  try {
    const res = await listAccessLogs(props.documentId, logPage.value, logPageSize)
    accessLogs.value = res.data ?? []
    totalLogs.value = res.pagination?.total ?? 0
  } catch (error) {
    ElMessage.error('加载访问日志失败')
  }
}

function formatTime(time: string) {
  const date = new Date(time)
  return date.toLocaleString('zh-CN')
}

function getActionType(action: string) {
  const types: Record<string, any> = {
    view: 'info',
    edit: 'warning',
    download: 'success',
    share: 'danger'
  }
  return types[action] || 'info'
}

function getActionLabel(action: string) {
  const labels: Record<string, string> = {
    view: '浏览',
    edit: '编辑',
    download: '下载',
    share: '分享'
  }
  return labels[action] || action
}
</script>

<style scoped>
.stats-panel {
  padding: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px;
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 8px;
  color: #fff;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.stat-label {
  font-size: 13px;
  color: #6b7280;
  margin-top: 4px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-detail {
  padding: 8px 0;
}

.stat-detail p {
  margin: 8px 0;
  color: #4b5563;
}
</style>