<template>
  <div class="announcement-list">
    <div class="page-header">
      <h2>平台公告</h2>
      <p class="subtitle">查看平台发布的最新消息和通知</p>
    </div>

    <el-table v-loading="loading" :data="list" stripe style="width: 100%">
      <el-table-column prop="title" label="公告标题" min-width="200">
        <template #default="{ row }">
          <span class="title">{{ row.title }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="content" label="内容" min-width="300">
        <template #default="{ row }">
          <span class="content-preview">{{ stripHtml(row.content) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="发布时间" width="180">
        <template #default="{ row }">
          {{ formatDate(row.publish_at || row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="handleView(row)">查看</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        @size-change="loadList"
        @current-change="loadList"
      />
    </div>

    <el-dialog v-model="detailVisible" :title="currentItem?.title" width="600px">
      <div class="announcement-content" v-html="currentItem?.content"></div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-empty v-if="!loading && list.length === 0" description="暂无公告" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listTenantAnnouncements, type Announcement } from '@/api/modules/announcement'

const loading = ref(false)
const list = ref<Announcement[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const detailVisible = ref(false)
const currentItem = ref<Announcement | null>(null)

function stripHtml(html: string) {
  return html.replace(/<[^>]*>/g, '').substring(0, 100) + (html.length > 100 ? '...' : '')
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

async function loadList() {
  loading.value = true
  try {
    const res = await listTenantAnnouncements({ page: page.value, page_size: pageSize.value })
    list.value = res.items || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
    ElMessage.error('加载公告列表失败')
  } finally {
    loading.value = false
  }
}

function handleView(row: Announcement) {
  currentItem.value = row
  detailVisible.value = true
}

onMounted(loadList)
</script>

<style scoped>
.announcement-list {
  padding: 16px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0 0 4px;
  font-size: 20px;
  font-weight: 600;
}

.subtitle {
  margin: 0;
  color: #9ca3af;
  font-size: 13px;
}

.title {
  font-weight: 500;
}

.content-preview {
  color: #6b7280;
  font-size: 13px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.announcement-content {
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
}

.announcement-content :deep(img) {
  max-width: 100%;
  height: auto;
}
</style>