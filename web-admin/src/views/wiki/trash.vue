<template>
  <div class="page">
    <el-card shadow="never">
      <div class="head">
        <div>
          <h3>{{ t('wiki.trash.title') }}</h3>
          <p class="tip">{{ t('wiki.trash.tip') }}</p>
        </div>
        <div class="head-actions">
          <el-button v-if="isSuperAdmin" type="warning" :loading="expiring" @click="handleExpire">
            <el-icon><Delete /></el-icon>{{ t('wiki.trash.expire') }}
          </el-button>
          <el-button @click="load">
            <el-icon><Refresh /></el-icon>{{ t('common.refresh') }}
          </el-button>
        </div>
      </div>

      <el-tabs v-model="activeType" @tab-change="onTypeChange">
        <el-tab-pane :label="t('common.all')" name="" />
        <el-tab-pane :label="t('wiki.trash.tabSpace')" name="space" />
        <el-tab-pane :label="t('wiki.trash.tabNode')" name="node" />
        <el-tab-pane :label="t('wiki.trash.tabDocument')" name="document" />
      </el-tabs>

      <el-table v-loading="loading" :data="list" size="default">
        <el-table-column :label="t('common.type')" width="100">
          <template #default="{ row }">
            <el-tag :type="tagType(row.item_type)" size="small">{{ typeText(row.item_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" :label="t('wiki.trash.colName')" min-width="220" show-overflow-tooltip />
        <el-table-column :label="t('wiki.trash.colSpace')" min-width="160">
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
        <el-table-column :label="t('wiki.trash.colDeletedAt')" width="170">
          <template #default="{ row }">
            {{ row.deleted_at?.replace('T', ' ').substring(0, 19) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('wiki.trash.colExpiresAt')" width="170">
          <template #default="{ row }">
            {{ row.expires_at?.replace('T', ' ').substring(0, 19) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="170" align="center">
          <template #default="{ row }">
            <el-button link type="success" :loading="operating" @click="restore(row)">{{ t('wiki.trash.restore') }}</el-button>
            <el-button link type="danger" :loading="operating" @click="permanentDelete(row)">
              {{ t('wiki.trash.permanentDelete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && list.length === 0" :description="t('wiki.trash.empty')" />

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
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { Refresh, Delete } from '@element-plus/icons-vue'
import {
  listTrash,
  restoreTrashItem,
  permanentDeleteTrashItem,
  expireTrash,
  type WikiTrashItem
} from '@/api/modules/wiki'
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const userStore = useUserStore()
const { t } = useI18n()
const isSuperAdmin = computed(() => userStore.isAdmin)

const list = ref<WikiTrashItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const operating = ref(false)
const expiring = ref(false)
const activeType = ref('')

function tagType(type: string): 'success' | 'danger' | 'warning' | 'info' {
  if (type === 'document') return 'success'
  if (type === 'space') return 'danger'
  return 'warning'
}
function typeText(type: string) {
  if (type === 'document') return t('wiki.trash.tabDocument')
  if (type === 'space') return t('wiki.trash.tabSpace')
  return t('wiki.trash.tabNode')
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
    ElMessage.success(t('wiki.trash.restored'))
    await load()
  } finally {
    operating.value = false
  }
}

async function permanentDelete(row: WikiTrashItem) {
  try {
    await ElMessageBox.confirm(
      t('wiki.trash.permanentConfirm', row.title),
      t('wiki.trash.warning'),
      { type: 'error' }
    )
  } catch {
    return
  }
  operating.value = true
  try {
    await permanentDeleteTrashItem(row.id)
    ElMessage.success(t('wiki.trash.permanentlyDeleted'))
    await load()
  } finally {
    operating.value = false
  }
}

async function handleExpire() {
  try {
    await ElMessageBox.confirm(
      t('wiki.trash.expireConfirm'),
      t('wiki.trash.expireTitle'),
      { type: 'warning' }
    )
  } catch {
    return
  }
  expiring.value = true
  try {
    const res = await expireTrash()
    const purged = res?.purged ?? 0
    if (purged > 0) {
      ElMessage.success(t('wiki.trash.expireDone', purged))
    } else {
      ElMessage.info(t('wiki.trash.expireNone'))
    }
    await load()
  } catch {
    // errors are surfaced by the shared response interceptor
  } finally {
    expiring.value = false
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
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
