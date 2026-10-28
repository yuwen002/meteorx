<template>
  <el-dialog v-model="visible" :title="t('wiki.share.title')" width="800px">
    <div class="share-manager">
      <el-button type="primary" @click="toggleCreate">
        {{ showCreateForm ? t('wiki.share.cancelCreate') : t('wiki.share.create') }}
      </el-button>

      <div v-if="showCreateForm" class="create-form">
        <el-divider>{{ t('wiki.share.newTitle') }}</el-divider>
        <el-form :model="createForm" label-width="120px" inline>
          <el-form-item :label="t('wiki.share.password')">
            <el-input v-model="createForm.password" :placeholder="t('wiki.share.passwordPlaceholder')" clearable style="width: 180px" />
          </el-form-item>
          <el-form-item :label="t('wiki.share.expiresAt')">
            <el-date-picker
              v-model="createForm.expires_at"
              type="datetime"
              :placeholder="t('wiki.share.expiresPlaceholder')"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
            />
          </el-form-item>
          <el-form-item :label="t('wiki.share.maxViews')">
            <el-input-number v-model="createForm.max_views" :min="1" :max="10000" :placeholder="t('wiki.share.unlimited')" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="submitting" @click="submitCreate">{{ t('wiki.share.submitCreate') }}</el-button>
            <el-button @click="toggleCreate">{{ t('common.cancel') }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <el-table :data="shareLinks" style="width: 100%; margin-top: 16px">
        <el-table-column prop="token" :label="t('wiki.share.colToken')" min-width="120">
          <template #default="{ row }">
            <el-link type="primary" @click="copyLink(row.token)">{{ row.token }}</el-link>
          </template>
        </el-table-column>
        <el-table-column :label="t('wiki.share.colPassword')" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.password" type="warning">{{ t('wiki.share.hasYes') }}</el-tag>
            <el-tag v-else type="info">{{ t('wiki.share.hasNo') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('wiki.share.colViews')" width="100">
          <template #default="{ row }">
            {{ row.view_count }}{{ row.max_views ? ` / ${row.max_views}` : '' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('wiki.share.expiresAt')" width="180">
          <template #default="{ row }">
            {{ row.expire_at ? formatTime(row.expire_at) : t('wiki.share.permanent') }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.status')" width="90">
          <template #default="{ row }">
            <el-tag :type="isShareActive(row) ? 'success' : 'info'">
              {{ isShareActive(row) ? t('wiki.share.active') : t('wiki.share.expired') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="100">
          <template #default="{ row }">
            <el-button link type="danger" @click="handleDelete(row.id)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ t('common.close') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listShareLinks, createShareLink, deleteShareLink, type ShareLink } from '@/api/modules/wiki'
import { useI18n } from '@/composables/useI18n'

const { t, locale } = useI18n()

const props = defineProps<{
  documentId: string
}>()

const visible = ref(false)
const shareLinks = ref<ShareLink[]>([])
const showCreateForm = ref(false)
const submitting = ref(false)

const createForm = ref({
  password: '',
  expires_at: '',
  max_views: undefined as number | undefined
})

watch(visible, async (val) => {
  if (val) {
    await loadShareLinks()
  }
})

async function loadShareLinks() {
  try {
    shareLinks.value = await listShareLinks(props.documentId)
  } catch (error) {
    ElMessage.error(t('wiki.share.loadFailed'))
  }
}

function toggleCreate() {
  if (showCreateForm.value) {
    showCreateForm.value = false
  } else {
    createForm.value = { password: '', expires_at: '', max_views: undefined }
    showCreateForm.value = true
  }
}

async function submitCreate() {
  submitting.value = true
  try {
    await createShareLink({
      document_id: props.documentId,
      password: createForm.value.password || undefined,
      expires_at: createForm.value.expires_at || undefined,
      max_views: createForm.value.max_views
    })
    ElMessage.success(t('wiki.share.createSuccess'))
    showCreateForm.value = false
    await loadShareLinks()
  } catch (error) {
    ElMessage.error(t('wiki.share.createFailed'))
  } finally {
    submitting.value = false
  }
}

function handleDelete(id: string) {
  ElMessageBox.confirm(t('wiki.share.deleteConfirm'), t('common.tip'), {
    type: 'warning'
  })
    .then(async () => {
      await deleteShareLink(id)
      ElMessage.success(t('wiki.share.deleteSuccess'))
      await loadShareLinks()
    })
    .catch(() => {})
}

function copyLink(token: string) {
  const base = window.location.href.split('#')[0]
  const link = `${base}#/wiki/share/${token}`
  navigator.clipboard.writeText(link)
  ElMessage.success(t('wiki.share.copied'))
}

function isShareActive(row: ShareLink): boolean {
  if (row.max_views && row.view_count >= row.max_views) return false
  if (row.expire_at) {
    const expireTime = new Date(row.expire_at).getTime()
    if (!Number.isNaN(expireTime) && expireTime < Date.now()) return false
  }
  return true
}

function formatTime(time: string) {
  const date = new Date(time)
  return date.toLocaleString(locale.value)
}

defineExpose({
  open() {
    visible.value = true
  }
})
</script>

<style scoped>
.share-manager {
  padding: 8px 0;
}
.create-form {
  padding: 8px 0;
}
</style>