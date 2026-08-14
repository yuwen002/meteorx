<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 页面标题 -->
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button link @click="goBack">
              <el-icon><ArrowLeft /></el-icon>返回
            </el-button>
            <span class="title">租户回收站</span>
          </div>
          <el-tag type="info">已删除的租户可以恢复</el-tag>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="search.name"
          placeholder="搜索租户名称"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-alert
          :title="`已选择 ${selectedIds.length} 项`"
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <el-button link type="success" @click="handleBatchRestore">批量恢复</el-button>
          </template>
        </el-alert>
      </div>

      <!-- 列表 -->
      <el-table
        :data="list"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        row-key="id"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="租户ID" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 6px;">
              <span style="font-family: monospace; color: #6b7280; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ row.id }}</span>
              <el-button
                link
                type="primary"
                size="small"
                :icon="CopyDocument"
                @click="copyId(row.id)"
                title="复制"
              />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="租户名称" prop="name" min-width="150" />
        <el-table-column label="域名" prop="domain" min-width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.domain }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="联系邮箱" prop="contact_email" min-width="180" />
        <el-table-column label="删除时间" prop="deleted_at" min-width="160">
          <template #default="{ row }">
            {{ formatDate(row.deleted_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" @click="handleRestore(row)">
              <el-icon><RefreshLeft /></el-icon>恢复
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        class="pagination"
        @size-change="loadList"
        @current-change="loadList"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Search, RefreshLeft, CopyDocument } from '@element-plus/icons-vue'
import {
  getDeletedTenantList,
  restoreTenant,
  type TenantItem,
  type TenantListParams
} from '@/api/modules/tenant'

const router = useRouter()

const loading = ref(false)
const list = ref<TenantItem[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const selectedIds = ref<string[]>([])

const search = reactive({ name: '' })

function goBack() {
  router.push('/system/tenant')
}

async function copyId(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

async function loadList() {
  loading.value = true
  try {
    const params: TenantListParams = { page: page.value, page_size: pageSize.value }
    if (search.name) params.name = search.name
    const res: any = await getDeletedTenantList(params)
    list.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  search.name = ''
  page.value = 1
  loadList()
}

function handleSelectionChange(selection: TenantItem[]) {
  selectedIds.value = selection.map((item) => item.id)
}

// 恢复单个
function handleRestore(row: TenantItem) {
  ElMessageBox.confirm(`确定要恢复租户 "${row.name}" 吗？`, '确认恢复', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'success'
  })
    .then(async () => {
      await restoreTenant(row.id)
      ElMessage.success('恢复成功')
      loadList()
    })
    .catch(() => {})
}

// 批量恢复
function handleBatchRestore() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning('请选择要恢复的租户')
    return
  }
  ElMessageBox.confirm(
    `确定要恢复选中的 ${selectedIds.value.length} 个租户吗？`,
    '批量恢复',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'success'
    }
  )
    .then(async () => {
      for (const id of selectedIds.value) {
        await restoreTenant(id)
      }
      ElMessage.success('批量恢复成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

onMounted(() => {
  loadList()
})
</script>

<style scoped lang="scss">
.page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;

    .title {
      font-size: 16px;
      font-weight: 600;
    }
  }
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.batch-bar {
  margin-bottom: 16px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>