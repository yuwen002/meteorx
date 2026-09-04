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
            <span class="title">系统管理员回收站</span>
          </div>
          <el-tag type="info">已删除的管理员可以恢复或永久删除</el-tag>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          placeholder="搜索用户名/昵称"
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
            <el-button link type="danger" @click="handleBatchPermanentDelete">批量永久删除</el-button>
          </template>
        </el-alert>
      </div>

      <!-- 列表 -->
      <el-table
        v-loading="loading"
        :data="list"
        row-key="id"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column label="用户名" prop="username" min-width="120" />
        <el-table-column label="昵称" prop="nickname" min-width="120" />
        <el-table-column label="邮箱" prop="email" min-width="180" />
        <el-table-column label="删除时间" prop="deleted_at" min-width="160">
          <template #default="{ row }">
            {{ formatDate(row.deleted_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" @click="handleRestore(row)">
              <el-icon><RefreshLeft /></el-icon>恢复
            </el-button>
            <el-button link type="danger" @click="handlePermanentDelete(row)">
              <el-icon><Delete /></el-icon>永久删除
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
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { ArrowLeft, Search, RefreshLeft, Delete } from '@element-plus/icons-vue'
import {
  getDeletedMasterAdminList,
  restoreMasterAdmin,
  permanentDeleteMasterAdmin,
  type UserItem,
  type UserListParams
} from '@/api/modules/user'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const router = useRouter()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<UserItem, { keyword: string }>({
  fetchList: async (params) => {
    const req: UserListParams = { page: params.page, page_size: params.page_size }
    if (params.keyword) req.keyword = params.keyword
    return toPageResult(await getDeletedMasterAdminList(req))
  },
  initialQuery: { keyword: '' }
})
const loadList = reload
const selectedIds = ref<string[]>([])

function goBack() {
  router.push('/system/master-admin')
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function resetSearch() {
  reset()
}

function handleSelectionChange(selection: UserItem[]) {
  selectedIds.value = selection.map(item => item.id)
}

// 恢复单个
function handleRestore(row: UserItem) {
  ElMessageBox.confirm(
    `确定要恢复系统管理员 "${row.username}" 吗？`,
    '确认恢复',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'success'
    }
  )
    .then(async () => {
      await restoreMasterAdmin(row.id)
      ElMessage.success('恢复成功')
      loadList()
    })
    .catch(() => {})
}

// 批量恢复
function handleBatchRestore() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning('请选择要恢复的管理员')
    return
  }
  ElMessageBox.confirm(
    `确定要恢复选中的 ${selectedIds.value.length} 个系统管理员吗？`,
    '批量恢复',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'success'
    }
  )
    .then(async () => {
      // 逐个恢复
      for (const id of selectedIds.value) {
        await restoreMasterAdmin(id)
      }
      ElMessage.success('批量恢复成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

// 永久删除单个
function handlePermanentDelete(row: UserItem) {
  ElMessageBox.confirm(
    `确定要永久删除系统管理员 "${row.username}" 吗？此操作不可恢复！`,
    '危险操作',
    {
      confirmButtonText: '确定永久删除',
      cancelButtonText: '取消',
      type: 'error'
    }
  )
    .then(async () => {
      await permanentDeleteMasterAdmin(row.id)
      ElMessage.success('永久删除成功')
      loadList()
    })
    .catch(() => {})
}

// 批量永久删除
function handleBatchPermanentDelete() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning('请选择要永久删除的管理员')
    return
  }
  ElMessageBox.confirm(
    `确定要永久删除选中的 ${selectedIds.value.length} 个系统管理员吗？此操作不可恢复！`,
    '危险操作',
    {
      confirmButtonText: '确定永久删除',
      cancelButtonText: '取消',
      type: 'error'
    }
  )
    .then(async () => {
      // 逐个永久删除
      for (const id of selectedIds.value) {
        await permanentDeleteMasterAdmin(id)
      }
      ElMessage.success('批量永久删除成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

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
  }

  .title {
    font-size: 16px;
    font-weight: 600;
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
  justify-content: flex-end;
}
</style>