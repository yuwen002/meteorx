<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 页面标题 -->
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button link @click="goBack">
              <el-icon><ArrowLeft /></el-icon>{{ t('tenantPage.back') }}
            </el-button>
            <span class="title">{{ t('tenantPage.recycleTitle') }}</span>
          </div>
          <el-tag type="info">{{ t('tenantPage.recycleHint') }}</el-tag>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.name"
          :placeholder="t('tenantPage.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('tenantPage.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-alert
          :title="t('tenantPage.selectedCount', selectedIds.length)"
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <el-button link type="success" @click="handleBatchRestore">{{ t('tenantPage.batchRestore') }}</el-button>
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
        <el-table-column prop="id" :label="t('tenantPage.colTenantId')" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 6px;">
              <span style="font-family: monospace; color: #6b7280; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ row.id }}</span>
              <el-button
                link
                type="primary"
                size="small"
                :icon="CopyDocument"
                :title="t('tenantPage.copy')"
                @click="copyId(row.id)"
              />
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colName')" prop="name" min-width="150" />
        <el-table-column :label="t('tenantPage.colDomain')" prop="domain" min-width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.domain }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colContactEmail')" prop="contact_email" min-width="180" />
        <el-table-column :label="t('tenantPage.colDeletedAt')" prop="deleted_at" min-width="160">
          <template #default="{ row }">
            {{ formatDate(row.deleted_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colAction')" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" @click="handleRestore(row)">
              <el-icon><RefreshLeft /></el-icon>{{ t('tenantPage.restore') }}
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
import { ArrowLeft, Search, RefreshLeft, CopyDocument } from '@element-plus/icons-vue'
import {
  getDeletedTenantList,
  restoreTenant,
  type TenantItem,
  type TenantListParams
} from '@/api/modules/tenant'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const { t } = useI18n()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<TenantItem, { name: string }>({
  fetchList: async (params) => {
    const req: TenantListParams = { page: params.page, page_size: params.page_size }
    if (params.name) req.name = params.name
    return toPageResult(await getDeletedTenantList(req))
  },
  initialQuery: { name: '' }
})
const loadList = reload
const selectedIds = ref<string[]>([])

function goBack() {
  router.push('/system/tenant')
}

async function copyId(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('tenantPage.copied'))
  } catch {
    ElMessage.error(t('tenantPage.copyFailed'))
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function resetSearch() {
  reset()
}

function handleSelectionChange(selection: TenantItem[]) {
  selectedIds.value = selection.map((item) => item.id)
}

// 恢复单个
function handleRestore(row: TenantItem) {
  ElMessageBox.confirm(t('tenantPage.restoreConfirm', row.name), t('tenantPage.restoreTitle'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'success'
  })
    .then(async () => {
      await restoreTenant(row.id)
      ElMessage.success(t('tenantPage.restoreSuccess'))
      loadList()
    })
    .catch(() => {})
}

// 批量恢复
function handleBatchRestore() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning(t('tenantPage.selectRestoreFirst'))
    return
  }
  ElMessageBox.confirm(
    t('tenantPage.batchRestoreConfirm', selectedIds.value.length),
    t('tenantPage.batchRestore'),
    {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'success'
    }
  )
    .then(async () => {
      for (const id of selectedIds.value) {
        await restoreTenant(id)
      }
      ElMessage.success(t('tenantPage.batchRestoreSuccess'))
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