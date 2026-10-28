<template>
  <div class="edit-lock-indicator">
    <el-alert
      v-if="isLocked && !isMyLock"
      :title="t('wiki.lock.lockedBy', lockInfo.user_name || t('wiki.lock.otherUser'))"
      type="warning"
      :closable="false"
      show-icon
    >
      <template #default>
        <div class="lock-details">
          <p>{{ t('wiki.lock.lockedAt') }}: {{ formatTime(lockInfo.locked_at) }}</p>
          <p>{{ t('wiki.lock.expiresAt') }}: {{ formatTime(lockInfo.expires_at) }}</p>
          <el-button size="small" type="primary" @click="handleForceLock">
            {{ t('wiki.lock.force') }}
          </el-button>
        </div>
      </template>
    </el-alert>

    <el-alert
      v-else-if="isLocked && isMyLock"
      :title="t('wiki.lock.myLock')"
      type="success"
      :closable="false"
      show-icon
    >
      <template #default>
        <div class="lock-details">
          <p>{{ t('wiki.lock.lockedAt') }}: {{ formatTime(lockInfo.locked_at) }}</p>
          <p>{{ t('wiki.lock.expiresAt') }}: {{ formatTime(lockInfo.expires_at) }}</p>
          <el-button size="small" type="danger" @click="handleRelease">
            {{ t('wiki.lock.release') }}
          </el-button>
          <el-button size="small" @click="handleRefresh">
            {{ t('wiki.lock.refresh') }}
          </el-button>
        </div>
      </template>
    </el-alert>

    <el-button
      v-else
      type="primary"
      size="small"
      @click="handleAcquire"
    >
      <el-icon><Edit /></el-icon>
      {{ t('wiki.lock.start') }}
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Edit } from '@element-plus/icons-vue'
import {
  acquireEditLock,
  releaseEditLock,
  refreshEditLock,
  getEditLock,
  type EditLock
} from '@/api/modules/wiki'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from '@/composables/useI18n'

const { t, locale } = useI18n()

// Backend rejects with a Chinese message; match by code points to keep this source ASCII
const LOCKED_KEYWORD = /\u9501\u5b9a\u6587\u6863|\u5df2\u88ab\u9501\u5b9a/

const props = defineProps<{
  documentId: string
  currentUserId: string
}>()

const emit = defineEmits<{
  locked: []
  unlocked: []
}>()

const lockInfo = ref<EditLock>({
  document_id: '',
  user_id: '',
  locked_at: '',
  expires_at: '',
  can_edit: true
})

// user_id 非空表示有有效锁占用（后端空闲响应 user_id 为空、can_edit=true）
const isLocked = computed(() => !!lockInfo.value.user_id)
let refreshTimer: number | null = null

const isMyLock = computed(() => lockInfo.value.user_id === props.currentUserId)

onMounted(async () => {
  await checkLockStatus()
  // 每5分钟刷新一次锁定状态
  refreshTimer = window.setInterval(checkLockStatus, 5 * 60 * 1000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})

async function checkLockStatus() {
  try {
    lockInfo.value = await getEditLock(props.documentId)
  } catch (error) {
    // 查询失败视为空闲
    lockInfo.value = {
      document_id: props.documentId,
      user_id: '',
      locked_at: '',
      expires_at: '',
      can_edit: true
    }
  }
}

async function handleAcquire() {
  try {
    lockInfo.value = await acquireEditLock(props.documentId)
    ElMessage.success(t('wiki.lock.acquireSuccess'))
    emit('locked')
  } catch (error) {
    const msg = (error as { response?: { data?: { message?: string } } }).response?.data?.message
    if (msg && LOCKED_KEYWORD.test(msg)) {
      ElMessage.warning(t('wiki.lock.lockedByOther'))
    } else {
      ElMessage.error(t('wiki.lock.acquireFailed'))
    }
  }
}

async function handleRelease() {
  try {
    await releaseEditLock(props.documentId)
    lockInfo.value = {
      document_id: props.documentId,
      user_id: '',
      locked_at: '',
      expires_at: '',
      can_edit: true
    }
    ElMessage.success(t('wiki.lock.releaseSuccess'))
    emit('unlocked')
  } catch (error) {
    ElMessage.error(t('wiki.lock.releaseFailed'))
  }
}

async function handleRefresh() {
  try {
    await refreshEditLock(props.documentId)
    await checkLockStatus()
    ElMessage.success(t('wiki.lock.refreshSuccess'))
  } catch (error) {
    ElMessage.error(t('wiki.lock.refreshFailed'))
  }
}

async function handleForceLock() {
  try {
    await ElMessageBox.confirm(
      t('wiki.lock.forceConfirm'),
      t('wiki.lock.forceTitle'),
      {
        type: 'warning'
      }
    )
    lockInfo.value = await acquireEditLock(props.documentId)
    ElMessage.success(t('wiki.lock.forceSuccess'))
    emit('locked')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('wiki.lock.forceFailed'))
    }
  }
}

function formatTime(time: string) {
  const date = new Date(time)
  return date.toLocaleString(locale.value)
}
</script>

<style scoped>
.edit-lock-indicator {
  margin-bottom: 16px;
}

.lock-details {
  margin-top: 8px;
}

.lock-details p {
  margin: 4px 0;
  font-size: 13px;
  color: #4b5563;
}

.lock-details .el-button {
  margin-top: 8px;
  margin-right: 8px;
}
</style>