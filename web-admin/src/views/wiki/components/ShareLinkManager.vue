<template>
  <el-dialog v-model="visible" title="分享链接管理" width="800px">
    <div class="share-manager">
      <el-button type="primary" @click="openCreateDialog">创建分享链接</el-button>

      <el-table :data="shareLinks" style="width: 100%; margin-top: 16px">
        <el-table-column prop="token" label="链接Token" min-width="120">
          <template #default="{ row }">
            <el-link type="primary" @click="copyLink(row.token)">
              {{ row.token }}
            </el-link>
          </template>
        </el-table-column>
        <el-table-column label="密码" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.password" type="warning">有</el-tag>
            <el-tag v-else type="info">无</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="浏览次数" width="100">
          <template #default="{ row }">
            {{ row.view_count }}{{ row.max_views ? ` / ${row.max_views}` : '' }}
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="180">
          <template #default="{ row }">
            {{ row.expires_at ? formatTime(row.expires_at) : '永久' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'danger'">
              {{ row.is_active ? '有效' : '已失效' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建分享链接弹窗 -->
    <el-dialog v-model="createDialogVisible" title="创建分享链接" width="480px" append-to-body>
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="访问密码">
          <el-input
            v-model="createForm.password"
            placeholder="留空则无需密码"
            clearable
          />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="createForm.expires_at"
            type="datetime"
            placeholder="选择过期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item label="最大浏览次数">
          <el-input-number
            v-model="createForm.max_views"
            :min="1"
            :max="10000"
            placeholder="留空则无限制"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">确定</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listShareLinks,
  createShareLink,
  deleteShareLink,
  type ShareLink
} from '@/api/modules/wiki'

const props = defineProps<{
  documentId: string
}>()

const visible = ref(false)
const shareLinks = ref<ShareLink[]>([])
const createDialogVisible = ref(false)
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
    ElMessage.error('加载分享链接失败')
  }
}

function openCreateDialog() {
  createForm.value = {
    password: '',
    expires_at: '',
    max_views: undefined
  }
  createDialogVisible.value = true
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
    ElMessage.success('创建成功')
    createDialogVisible.value = false
    await loadShareLinks()
  } catch (error) {
    ElMessage.error('创建失败')
  } finally {
    submitting.value = false
  }
}

function handleDelete(id: string) {
  ElMessageBox.confirm('确定要删除此分享链接吗？', '提示', {
    type: 'warning'
  })
    .then(async () => {
      await deleteShareLink(id)
      ElMessage.success('删除成功')
      await loadShareLinks()
    })
    .catch(() => {})
}

function copyLink(token: string) {
  const link = `${window.location.origin}/wiki/share/${token}`
  navigator.clipboard.writeText(link)
  ElMessage.success('链接已复制到剪贴板')
}

function formatTime(time: string) {
  const date = new Date(time)
  return date.toLocaleString('zh-CN')
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
</style>