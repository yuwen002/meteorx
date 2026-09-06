<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-head">
        <el-input
          v-model="keyword"
          placeholder="搜索 Wiki 空间、目录与文档标题/正文…"
          clearable
          size="large"
          @keyup.enter="doSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" size="large" :loading="loading" @click="doSearch">搜索</el-button>
      </div>

      <div v-if="searched && !loading && list.length === 0" class="empty">
        <el-empty description="未找到与关键字匹配的内容" />
      </div>

      <div v-loading="loading" class="results">
        <div v-for="item in list" :key="item.id" class="result-item" @click="openItem(item)">
          <div class="result-head">
            <el-tag :type="tagType(item.type)" size="small">{{ typeText(item.type) }}</el-tag>
            <span class="result-title">{{ item.title }}</span>
            <span class="result-time">{{ item.updated_at?.replace('T', ' ').substring(0, 16) }}</span>
          </div>
          <div class="result-snippet" v-html="snippetHtml(item)"></div>
        </div>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          background
          @current-change="load"
          @size-change="reload"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import { searchWiki, type WikiSearchItem } from '@/api/modules/wiki'

const router = useRouter()

const keyword = ref('')
const list = ref<WikiSearchItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const searched = ref(false)

function tagType(type: string): 'success' | 'primary' | 'info' {
  if (type === 'document') return 'success'
  if (type === 'space') return 'primary'
  return 'info'
}
function typeText(type: string) {
  if (type === 'document') return '文档'
  if (type === 'space') return '空间'
  return '节点'
}

function snippetHtml(item: WikiSearchItem) {
  return item.highlight || item.snippet || '（无摘要）'
}

async function doSearch() {
  page.value = 1
  await load()
}

async function load() {
  if (!keyword.value.trim()) {
    list.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const res = await searchWiki({
      q: keyword.value.trim(),
      page: page.value,
      page_size: pageSize.value
    })
    list.value = res?.results ?? []
    total.value = res?.total ?? 0
    searched.value = true
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  void load()
}

function openItem(item: WikiSearchItem) {
  if (!item.space_id) return
  router.push({
    path: `/wiki/spaces/${item.space_id}`,
    query: item.type === 'document' ? { node_id: item.node_id } : {}
  })
}
</script>

<style scoped>
.search-head {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}
.results {
  min-height: 120px;
}
.result-item {
  padding: 14px 6px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
}
.result-item:hover {
  background: #f9fafb;
}
.result-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.result-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2937;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.result-time {
  margin-left: auto;
  font-size: 12px;
  color: #9ca3af;
  white-space: nowrap;
}
.result-snippet {
  margin-top: 6px;
  font-size: 13px;
  color: #6b7280;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.result-snippet mark {
  color: #d97706;
  background: #fef3c7;
  padding: 0 1px;
  border-radius: 2px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.empty {
  padding: 40px 0;
}
</style>
