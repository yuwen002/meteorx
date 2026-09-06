<template>
  <div class="edit-lock-indicator">
    <el-alert
      v-if="isLocked && !isMyLock"
      :title="`文档正在被 ${lockInfo.user_name || '其他用户'} 编辑`"
      type="warning"
      :closable="false"
      show-icon
    >
      <template #default>
        <div class="lock-details">
          <p>锁定时间: {{ formatTime(lockInfo.locked_at) }}</p>
          <p>过期时间: {{ formatTime(lockInfo.expires_at) }}</p>
          <el-button size="small" type="primary" @click="handleForceLock">
            强制获取编辑权
          </el-button>
        </div>
      </template>
    </el-alert>

    <el-alert
      v-else-if="isLocked && isMyLock"
      title="您正在编辑此文档"
      type="success"
      :closable="false"
      show-icon
    >
      <template #default>
        <div class="lock-details">
          <p>锁定时间: {{ formatTime(lockInfo.locked_at) }}</p>
          <p>过期时间: {{ formatTime(lockInfo.expires_at) }}</p>
          <el-button size="small" type="danger" @click="handleRelease">
            释放编辑权
          </el-button>
          <el-button size="small" @click="handleRefresh">
            刷新锁定时间
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
      开始编辑
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
  can_edit: false
})

const isLocked = ref(false)
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
    const lock = await getEditLock(props.documentId)
    lockInfo.value = lock
    isLocked.value = true
  } catch (error) {
    // 404 表示未锁定
    isLocked.value = false
  }
}

async function handleAcquire() {
  try {
    lockInfo.value = await acquireEditLock(props.documentId)
    isLocked.value = true
    ElMessage.success('已获取编辑权')
    emit('locked')
  } catch (error: any) {
    if (error.response?.data?.message?.includes('已被锁定')) {
      ElMessage.warning('文档已被其他用户锁定')
    } else {
      ElMessage.error('获取编辑权失败')
    }
  }
}

async function handleRelease() {
  try {
    await releaseEditLock(props.documentId)
    isLocked.value = false
    ElMessage.success('已释放编辑权')
    emit('unlocked')
  } catch (error) {
    ElMessage.error('释放编辑权失败')
  }
}

async function handleRefresh() {
  try {
    await refreshEditLock(props.documentId)
    await checkLockStatus()
    ElMessage.success('已刷新锁定时间')
  } catch (error) {
    ElMessage.error('刷新锁定时间失败')
  }
}

async function handleForceLock() {
  try {
    await ElMessageBox.confirm(
      '强制获取编辑权将中断其他用户的编辑，确定要继续吗？',
      '强制编辑',
      {
        type: 'warning'
      }
    )
    lockInfo.value = await acquireEditLock(props.documentId)
    isLocked.value = true
    ElMessage.success('已强制获取编辑权')
    emit('locked')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('强制获取编辑权失败')
    }
  }
}

function formatTime(time: string) {
  const date = new Date(time)
  return date.toLocaleString('zh-CN')
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