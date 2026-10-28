<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          :placeholder="t('filePage.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select
          v-model="query.file_type"
          :placeholder="t('filePage.typePlaceholder')"
          clearable
          style="width: 120px"
        >
          <el-option :label="t('filePage.type.image')" value="image" />
          <el-option :label="t('filePage.type.document')" value="document" />
          <el-option :label="t('filePage.type.video')" value="video" />
          <el-option :label="t('filePage.type.audio')" value="audio" />
          <el-option :label="t('filePage.type.other')" value="other" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('common.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleRecycleBin">
          <el-icon><Delete /></el-icon>{{ isRecycleBin ? t('filePage.backToList') : t('filePage.recycleBin') }}
        </el-button>
        <el-button v-if="userStore.hasPermission('file:upload') && !isRecycleBin" type="success" @click="openUploadDialog">
          <el-icon><Upload /></el-icon>{{ t('filePage.upload') }}
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="original_name" :label="t('filePage.colName')" min-width="180" show-overflow-tooltip />
        <el-table-column :label="t('filePage.colType')" width="100">
          <template #default="{ row }">
            <el-tag :type="getFileTypeColor(row.file_type)" size="small">
              {{ getFileTypeLabel(row.file_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" :label="t('filePage.colSize')" width="100">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mime_type" :label="t('filePage.colMime')" min-width="150" show-overflow-tooltip />
        <el-table-column prop="created_at" :label="t('filePage.colCreatedAt')" width="180" />
        <el-table-column v-if="isRecycleBin" prop="deleted_at" :label="t('filePage.colDeletedAt')" width="180" />
        <el-table-column :label="t('filePage.colAction')" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isRecycleBin">
              <el-button link type="success" @click="handleRestore(row)">{{ t('filePage.restore') }}</el-button>
              <el-button link type="danger" @click="handlePermanentDelete(row)">{{ t('filePage.permanentDelete') }}</el-button>
            </template>
            <template v-else>
              <el-button link type="primary" @click="handleDownload(row)">{{ t('filePage.download') }}</el-button>
              <el-button v-if="userStore.hasPermission('file:update')" link type="warning" @click="openEditDialog(row)">{{ t('filePage.rename') }}</el-button>
              <el-button v-if="userStore.hasPermission('file:delete')" link type="danger" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 上传文件弹窗 -->
    <el-dialog
      v-model="uploadDialogVisible"
      :title="t('filePage.uploadTitle')"
      width="520px"
      :close-on-click-modal="false"
      @close="handleUploadDialogClose"
    >
      <el-upload
        ref="uploadRef"
        v-model:file-list="fileList"
        :auto-upload="false"
        :on-change="handleFileChange"
        :on-exceed="handleExceed"
        :limit="MAX_UPLOAD_COUNT"
        :on-remove="handleFileRemove"
        drag
        multiple
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          {{ t('filePage.dragText') }}<em>{{ t('filePage.clickUpload') }}</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            {{ t('filePage.uploadTip', formatFileSize(MAX_FILE_SIZE), MAX_UPLOAD_COUNT) }}
          </div>
        </template>
      </el-upload>

      <!-- 上传进度条 -->
      <div v-if="uploading" class="upload-progress">
        <el-progress :percentage="uploadProgress" :status="uploadStatus" :stroke-width="14" />
        <div class="upload-progress__text">{{ uploadProgressText }}</div>
      </div>

      <template #footer>
        <el-button :disabled="uploading" @click="uploadDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="uploading" :disabled="fileList.length === 0" @click="submitUpload">
          {{ uploading ? t('filePage.uploading') : t('filePage.confirmUpload') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重命名弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="t('filePage.renameTitle')"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="80px"
      >
        <el-form-item :label="t('filePage.fileNameLabel')" prop="file_name">
          <el-input v-model="editForm.file_name" :placeholder="t('filePage.newNamePlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="editing" @click="submitEdit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules, UploadInstance, UploadUserFile } from 'element-plus'
import { Search, Delete, Upload, UploadFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import {
  getFileList,
  uploadFile,
  updateFile,
  deleteFile,
  permanentDeleteFile,
  restoreFile,
  downloadFile,
  getDeletedFileList,
  type FileItem
} from '@/api/modules/file'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const userStore = useUserStore()

// 常量
const MAX_FILE_SIZE = 10 * 1024 * 1024 // aligned with backend config.yaml
const MAX_UPLOAD_COUNT = 10

const isRecycleBin = ref(false)

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<FileItem, { keyword: string; file_type: string }>({
  fetchList: async (params) => {
    const listReq = {
      page: params.page,
      page_size: params.page_size,
      keyword: params.keyword || undefined,
      file_type: params.file_type || undefined
    }
    const res = isRecycleBin.value ? await getDeletedFileList(listReq) : await getFileList(listReq)
    return toPageResult(res)
  },
  initialQuery: { keyword: '', file_type: '' }
})
const loadList = reload

// 上传相关
const uploadDialogVisible = ref(false)
// Element Plus 的 UploadInstance 类型未暴露内部 uploadFiles 列表，
// 通过交叉类型扩展以便移除超限文件。
type UploadInstanceWithFiles = UploadInstance & { uploadFiles: UploadUserFile[] }
const uploadRef = ref<UploadInstanceWithFiles>()
const uploading = ref(false)
const fileList = ref<UploadUserFile[]>([])
const uploadProgress = ref(0)
const uploadStatus = ref<'' | 'success' | 'error'>('')
const uploadProgressText = ref('')

// 编辑相关
const editDialogVisible = ref(false)
const editFormRef = ref<FormInstance>()
const editing = ref(false)
const editingId = ref<string | null>(null)
const editForm = reactive({ file_name: '' })
const editRules = computed<FormRules>(() => ({
  file_name: [{ required: true, message: t('filePage.fileNameRequired'), trigger: 'blur' }]
}))

function toggleRecycleBin() {
  isRecycleBin.value = !isRecycleBin.value
  page.value = 1
  void reload()
}

function resetSearch() {
  reset()
}

function openUploadDialog() {
  fileList.value = []
  uploadProgress.value = 0
  uploadStatus.value = ''
  uploadProgressText.value = ''
  uploadDialogVisible.value = true
}

function handleUploadDialogClose() {
  if (uploading.value) return
  fileList.value = []
  uploadProgress.value = 0
  uploadStatus.value = ''
}

function handleFileChange(file: UploadUserFile) {
  if (file.size && file.size > MAX_FILE_SIZE) {
    ElMessage.error(t('filePage.fileTooLarge', file.name, formatFileSize(MAX_FILE_SIZE)))
    fileList.value = fileList.value.filter(f => f.uid !== file.uid)
    return false
  }
}

function handleFileRemove() {
  // v-model:file-list 会自动同步，无需额外操作
}

function handleExceed() {
  ElMessage.warning(t('filePage.maxFiles', MAX_UPLOAD_COUNT))
}

async function submitUpload() {
  const validFiles = fileList.value.filter(f => f.raw && f.size && f.size <= MAX_FILE_SIZE)
  if (validFiles.length === 0) {
    ElMessage.warning(t('filePage.pickValid'))
    return
  }

  uploading.value = true
  uploadProgress.value = 0
  uploadStatus.value = ''
  uploadProgressText.value = t('filePage.uploadingProgress', 0, validFiles.length)

  let successCount = 0
  const failedNames: string[] = []
  const total = validFiles.length
  let completed = 0

  try {
    await Promise.all(
      validFiles.map(async (file) => {
        try {
          await uploadFile(file.raw!)
          successCount++
        } catch {
          failedNames.push(file.name)
        } finally {
          completed++
          uploadProgress.value = Math.round((completed / total) * 100)
          uploadProgressText.value = t('filePage.uploadingProgress', completed, total)
        }
      })
    )

    if (failedNames.length === 0) {
      uploadStatus.value = 'success'
      uploadProgressText.value = t('filePage.uploadDone', successCount)
      ElMessage.success(t('filePage.uploadSuccess', successCount))
    } else {
      uploadStatus.value = 'error'
      uploadProgressText.value = t('filePage.uploadPartial', successCount, failedNames.length)
      ElMessage.warning(t('filePage.uploadPartialDetail', successCount, failedNames.length, failedNames.join('、')))
    }

    loadList()
  } finally {
    uploading.value = false
    // 2 秒后自动关闭
    setTimeout(() => {
      if (uploadStatus.value === 'success') {
        uploadDialogVisible.value = false
      }
    }, 2000)
  }
}

function openEditDialog(row: FileItem) {
  editingId.value = row.id
  editForm.file_name = row.file_name
  editDialogVisible.value = true
}

async function submitEdit() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    if (!editingId.value) return

    editing.value = true
    try {
      await updateFile(editingId.value, { file_name: editForm.file_name })
      ElMessage.success(t('filePage.renameSuccess'))
      editDialogVisible.value = false
      loadList()
    } finally {
      editing.value = false
    }
  })
}

function handleDownload(row: FileItem) {
  downloadFile(row.id).then(blob => {
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = row.original_name
    link.click()
    window.URL.revokeObjectURL(url)
  }).catch(() => {
    ElMessage.error(t('filePage.downloadFailed'))
  })
}

function handleDelete(row: FileItem) {
  ElMessageBox.confirm(t('filePage.deleteConfirm', row.original_name), t('common.tip'), {
    type: 'warning'
  })
    .then(async () => {
      if (!row.id) return
      await deleteFile(row.id)
      ElMessage.success(t('filePage.deleteSuccess'))
      loadList()
    })
    .catch(() => {})
}

async function handleRestore(row: FileItem) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm(t('filePage.restoreConfirm', row.original_name), t('common.tip'), {
      type: 'warning'
    })
    await restoreFile(row.id)
    ElMessage.success(t('filePage.restoreSuccess'))
    loadList()
  } catch (e) {
    // 用户取消
  }
}

function handlePermanentDelete(row: FileItem) {
  ElMessageBox.confirm(t('filePage.permanentDeleteConfirm', row.original_name), t('filePage.dangerTitle'), {
    type: 'error',
    confirmButtonText: t('filePage.confirmPermanentDelete'),
    cancelButtonText: t('common.cancel')
  })
    .then(async () => {
      if (!row.id) return
      await permanentDeleteFile(row.id)
      ElMessage.success(t('filePage.permanentDeleted'))
      loadList()
    })
    .catch(() => {})
}

function getFileTypeLabel(type: string) {
  const map: Record<string, string> = {
    image: t('filePage.type.image'),
    document: t('filePage.type.document'),
    video: t('filePage.type.video'),
    audio: t('filePage.type.audio'),
    other: t('filePage.type.other')
  }
  return map[type] || t('filePage.type.other')
}

function getFileTypeColor(type: string) {
  const map: Record<string, string> = {
    image: 'success',
    document: 'primary',
    video: 'warning',
    audio: 'info',
    other: 'info'
  }
  return map[type] || 'info'
}

function formatFileSize(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.search-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}
.flex-1 {
  flex: 1;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
.upload-progress {
  margin-top: 16px;
  padding: 0 10px;
}
.upload-progress__text {
  margin-top: 6px;
  font-size: 13px;
  color: #606266;
  text-align: center;
}
</style>