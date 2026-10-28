<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 页面标题 -->
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button link @click="goBack">
              <el-icon><ArrowLeft /></el-icon>{{ t('masterAdminRecycle.back') }}
            </el-button>
            <span class="title">{{ t('masterAdminRecycle.title') }}</span>
          </div>
          <el-tag type="info">{{ t('masterAdminRecycle.tip') }}</el-tag>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          :placeholder="t('masterAdminRecycle.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('common.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-alert
          :title="t('masterAdminRecycle.selected', selectedIds.length)"
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <el-button link type="success" @click="handleBatchRestore">{{ t('masterAdminRecycle.batchRestore') }}</el-button>
            <el-button link type="danger" @click="handleBatchPermanentDelete">{{ t('masterAdminRecycle.batchPermanentDelete') }}</el-button>
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
        <el-table-column :label="t('masterAdminRecycle.col.username')" prop="username" min-width="120" />
        <el-table-column :label="t('masterAdminRecycle.col.nickname')" prop="nickname" min-width="120" />
        <el-table-column :label="t('masterAdminRecycle.col.email')" prop="email" min-width="180" />
        <el-table-column :label="t('masterAdminRecycle.col.deletedAt')" prop="deleted_at" min-width="160">
          <template #default="{ row }">
            {{ formatDate(row.deleted_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('masterAdminRecycle.col.operation')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" @click="handleRestore(row)">
              <el-icon><RefreshLeft /></el-icon>{{ t('masterAdminRecycle.restore') }}
            </el-button>
            <el-button link type="danger" @click="handlePermanentDelete(row)">
              <el-icon><Delete /></el-icon>{{ t('masterAdminRecycle.permanentDelete') }}
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
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const { t, locale } = useI18n()

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
  return date.toLocaleString(locale.value)
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
    t('masterAdminRecycle.restoreConfirm', row.username),
    t('masterAdminRecycle.restoreTitle'),
    {
      confirmButtonText: t('masterAdminRecycle.confirmBtn'),
      cancelButtonText: t('masterAdminRecycle.cancelBtn'),
      type: 'success'
    }
  )
    .then(async () => {
      await restoreMasterAdmin(row.id)
      ElMessage.success(t('masterAdminRecycle.restored'))
      loadList()
    })
    .catch(() => {})
}

// 批量恢复
function handleBatchRestore() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning(t('masterAdminRecycle.selectToRestore'))
    return
  }
  ElMessageBox.confirm(
    t('masterAdminRecycle.batchRestoreConfirm', selectedIds.value.length),
    t('masterAdminRecycle.batchRestore'),
    {
      confirmButtonText: t('masterAdminRecycle.confirmBtn'),
      cancelButtonText: t('masterAdminRecycle.cancelBtn'),
      type: 'success'
    }
  )
    .then(async () => {
      // 逐个恢复
      for (const id of selectedIds.value) {
        await restoreMasterAdmin(id)
      }
      ElMessage.success(t('masterAdminRecycle.batchRestored'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

// 永久删除单个
function handlePermanentDelete(row: UserItem) {
  ElMessageBox.confirm(
    t('masterAdminRecycle.permanentDeleteConfirm', row.username),
    t('masterAdminRecycle.dangerTitle'),
    {
      confirmButtonText: t('masterAdminRecycle.confirmPermanentBtn'),
      cancelButtonText: t('masterAdminRecycle.cancelBtn'),
      type: 'error'
    }
  )
    .then(async () => {
      await permanentDeleteMasterAdmin(row.id)
      ElMessage.success(t('masterAdminRecycle.permanentDeleted'))
      loadList()
    })
    .catch(() => {})
}

// 批量永久删除
function handleBatchPermanentDelete() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning(t('masterAdminRecycle.selectToPermanentDelete'))
    return
  }
  ElMessageBox.confirm(
    t('masterAdminRecycle.batchPermanentDeleteConfirm', selectedIds.value.length),
    t('masterAdminRecycle.batchPermanentDelete'),
    {
      confirmButtonText: t('masterAdminRecycle.confirmPermanentBtn'),
      cancelButtonText: t('masterAdminRecycle.cancelBtn'),
      type: 'error'
    }
  )
    .then(async () => {
      // 逐个永久删除
      for (const id of selectedIds.value) {
        await permanentDeleteMasterAdmin(id)
      }
      ElMessage.success(t('masterAdminRecycle.batchPermanentDeleted'))
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