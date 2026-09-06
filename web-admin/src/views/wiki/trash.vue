<template>
  <div class="page">
    <el-card shadow="never">
      <div class="head">
        <div>
          <h3>回收站</h3>
          <p class="tip">被删除的空间、目录与文档会在此保留 30 天，过期后自动清除。</p>
        </div>
        <el-button @click="load">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
      </div>

      <el-tabs v-model="activeType" @tab-change="onTypeChange">
        <el-tab-pane label="全部" name="" />
        <el-tab-pane label="空间" name="space" />
        <el-tab-pane label="目录" name="node" />
        <el-tab-pane label="文档" name="document" />
      </el-tabs>

      <el-table v-loading="loading" :data="list" size="default">
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="tagType(row.item_type)" size="small">{{ typeText(row.item_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="名称" min-width="220" show-overflow-tooltip />
        <el-table-column label="空间" min-width="160">
          <template #default="{ row }">
            <el-link
              v-if="row.item_type === 'document' || row.item_type === 'node'"
              type="primary"
              :underline="false"
              @click="goSpace(row)"
            >
              {{ row.space_id }}
            </el-link>
            <span v-else>{{ row.space_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="删除时间" width="170">
          <template #default="{ row }">
            {{ row.deleted_at?.replace('T', ' ').substring(0, 19) }}
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="170">
          <template #default="{ row }">
            {{ row.expires_at?.replace('T', ' ').substring(0, 19) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" align="center">
          <template #default="{ row }">
            <el-button link type="success" :loading="operating" @click="restore(row)">恢复</el-button>
            <el-button link type="danger" :loading="operating" @click="permanentDelete(row)">
              彻底删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && list.length === 0" description="回收站是空的" />

      <div v-if="total > 0" class="pagination">
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
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { Refresh } from '@element-plus/icons-vue'
import {
  listTrash,
  restoreTrashItem,
  permanentDeleteTrashItem,
  type WikiTrashItem
} from '@/api/modules/wiki'

const router = useRouter()

const list = ref<WikiTrashItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const operating = ref(false)
const activeType = ref('')

function tagType(type: string): 'success' | 'danger' | 'warning' | 'info' {
  if (type === 'document') return 'success'
  if (type === 'space') return 'danger'
  return 'warning'
}
function typeText(type: string) {
  if (type === 'document') return '文档'
  if (type === 'space') return '空间'
  return '目录'
}

function onTypeChange() {
  page.value = 1
  void load()
}

async function load() {
  loading.value = true
  try {
    const res = await listTrash({
      page: page.value,
      page_size: pageSize.value,
      item_type: activeType.value || undefined
    })
    list.value = res?.items ?? []
    total.value = res?.total ?? 0
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

async function restore(row: WikiTrashItem) {
  operating.value = true
  try {
    await restoreTrashItem(row.id)
    ElMessage.success('已恢复')
    await load()
  } finally {
    operating.value = false
  }
}

async function permanentDelete(row: WikiTrashItem) {
  try {
    await ElMessageBox.confirm(
      `彻底删除 "${row.title}" 后将无法恢复，确定继续吗？`,
      '警告',
      { type: 'error' }
    )
  } catch {
    return
  }
  operating.value = true
  try {
    await permanentDeleteTrashItem(row.id)
    ElMessage.success('已彻底删除')
    await load()
  } finally {
    operating.value = false
  }
}

function goSpace(row: WikiTrashItem) {
  router.push(`/wiki/spaces/${row.space_id}`)
}
</script>

<style scoped>
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
h3 {
  margin: 0 0 4px;
  font-size: 16px;
}
.tip {
  margin: 0;
  color: #9ca3af;
  font-size: 12px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
