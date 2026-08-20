<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="search.keyword"
          placeholder="搜索文件名"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select
          v-model="search.file_type"
          placeholder="文件类型"
          clearable
          style="width: 120px"
        >
          <el-option label="图片" value="image" />
          <el-option label="文档" value="document" />
          <el-option label="视频" value="video" />
          <el-option label="音频" value="audio" />
          <el-option label="其他" value="other" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleRecycleBin">
          <el-icon><Delete /></el-icon>{{ isRecycleBin ? '返回列表' : '回收站' }}
        </el-button>
        <el-button type="success" v-if="userStore.hasPermission('file:upload') && !isRecycleBin" @click="openUploadDialog">
          <el-icon><Upload /></el-icon>上传文件
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table :data="list" border stripe v-loading="loading" style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="original_name" label="文件名" min-width="180" show-overflow-tooltip />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getFileTypeColor(row.file_type)" size="small">
              {{ getFileTypeLabel(row.file_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="大小" width="100">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mime_type" label="MIME类型" min-width="150" show-overflow-tooltip />
        <el-table-column prop="created_at" label="上传时间" width="180" />
        <el-table-column v-if="isRecycleBin" prop="deleted_at" label="删除时间" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isRecycleBin">
              <el-button link type="success" @click="handleRestore(row)">恢复</el-button>
              <el-button link type="danger" @click="handlePermanentDelete(row)">永久删除</el-button>
            </template>
            <template v-else>
              <el-button link type="primary" @click="handleDownload(row)">下载</el-button>
              <el-button link type="warning" v-if="userStore.hasPermission('file:update')" @click="openEditDialog(row)">重命名</el-button>
              <el-button link type="danger" v-if="userStore.hasPermission('file:delete')" @click="handleDelete(row)">删除</el-button>
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
      title="上传文件"
      width="520px"
      :close-on-click-modal="false"
      @close="handleUploadDialogClose"
    >
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :on-change="handleFileChange"
        :on-exceed="handleExceed"
        :limit="MAX_UPLOAD_COUNT"
        :file-list="fileList"
        :on-remove="handleFileRemove"
        drag
        multiple
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          将文件拖到此处，或<em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持图片、PDF、Office 文档等格式，单个文件不超过 {{ formatFileSize(MAX_FILE_SIZE) }}，最多 {{ MAX_UPLOAD_COUNT }} 个
          </div>
        </template>
      </el-upload>

      <!-- 上传进度条 -->
      <div v-if="uploading" class="upload-progress">
        <el-progress :percentage="uploadProgress" :status="uploadStatus" :stroke-width="14" />
        <div class="upload-progress__text">{{ uploadProgressText }}</div>
      </div>

      <template #footer>
        <el-button @click="uploadDialogVisible = false" :disabled="uploading">取消</el-button>
        <el-button type="primary" :loading="uploading" @click="submitUpload" :disabled="fileList.length === 0">
          {{ uploading ? '上传中...' : '确定上传' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重命名弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      title="重命名文件"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="80px"
      >
        <el-form-item label="文件名" prop="file_name">
          <el-input v-model="editForm.file_name" placeholder="请输入新文件名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editing" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules, type UploadInstance, type UploadUserFile } from 'element-plus'
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

const userStore = useUserStore()

// 常量
const MAX_FILE_SIZE = 10 * 1024 * 1024 // 与后端 config.yaml 对齐
const MAX_UPLOAD_COUNT = 10

const list = ref<FileItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const isRecycleBin = ref(false)

const search = reactive({ keyword: '', file_type: '' })

// 上传相关
const uploadDialogVisible = ref(false)
const uploadRef = ref<UploadInstance>()
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
const editRules: FormRules = {
  file_name: [{ required: true, message: '请输入文件名', trigger: 'blur' }]
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.keyword) params.keyword = search.keyword
    if (search.file_type) params.file_type = search.file_type
    
    const res = isRecycleBin.value 
      ? await getDeletedFileList(params)
      : await getFileList(params)
    
    list.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function toggleRecycleBin() {
  isRecycleBin.value = !isRecycleBin.value
  page.value = 1
  loadList()
}

function resetSearch() {
  search.keyword = ''
  search.file_type = ''
  loadList()
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
    ElMessage.error(`文件 "${file.name}" 超过 ${formatFileSize(MAX_FILE_SIZE)}`)
    // 从列表中移除超限文件
    if (uploadRef.value) {
      uploadRef.value.uploadFiles = uploadRef.value.uploadFiles.filter(f => f.uid !== file.uid)
    }
    return false
  }
}

function handleFileRemove() {
  // 文件移除时的回调（可用于校验）
}

function handleExceed() {
  ElMessage.warning(`最多只能上传 ${MAX_UPLOAD_COUNT} 个文件`)
}

async function submitUpload() {
  const validFiles = fileList.value.filter(f => f.raw && f.size && f.size <= MAX_FILE_SIZE)
  if (validFiles.length === 0) {
    ElMessage.warning('请选择有效的文件')
    return
  }

  uploading.value = true
  uploadProgress.value = 0
  uploadStatus.value = ''
  uploadProgressText.value = `正在上传 0 / ${validFiles.length}`

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
        } catch (e: any) {
          failedNames.push(file.name)
        } finally {
          completed++
          uploadProgress.value = Math.round((completed / total) * 100)
          uploadProgressText.value = `正在上传 ${completed} / ${total}`
        }
      })
    )

    if (failedNames.length === 0) {
      uploadStatus.value = 'success'
      uploadProgressText.value = `上传完成：${successCount} 个文件`
      ElMessage.success(`上传成功（${successCount} 个文件）`)
    } else {
      uploadStatus.value = 'error'
      uploadProgressText.value = `成功 ${successCount} 个，失败 ${failedNames.length} 个`
      ElMessage.warning(`成功 ${successCount} 个，失败 ${failedNames.length} 个：${failedNames.join('、')}`)
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
      ElMessage.success('重命名成功')
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
    ElMessage.error('下载失败')
  })
}

function handleDelete(row: FileItem) {
  ElMessageBox.confirm(`确定要删除文件 "${row.original_name}" 吗？`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      if (!row.id) return
      await deleteFile(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

async function handleRestore(row: FileItem) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm(`确定要恢复文件 "${row.original_name}" 吗？`, '提示', {
      type: 'warning'
    })
    await restoreFile(row.id)
    ElMessage.success('恢复成功')
    loadList()
  } catch (e) {
    // 用户取消
  }
}

function handlePermanentDelete(row: FileItem) {
  ElMessageBox.confirm(`确定要永久删除文件 "${row.original_name}" 吗？此操作不可恢复！`, '危险操作', {
    type: 'error',
    confirmButtonText: '确定永久删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      if (!row.id) return
      await permanentDeleteFile(row.id)
      ElMessage.success('已永久删除')
      loadList()
    })
    .catch(() => {})
}

function getFileTypeLabel(type: string) {
  const map: Record<string, string> = {
    image: '图片',
    document: '文档',
    video: '视频',
    audio: '音频',
    other: '其他'
  }
  return map[type] || '其他'
}

function getFileTypeColor(type: string) {
  const map: Record<string, any> = {
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

onMounted(loadList)
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