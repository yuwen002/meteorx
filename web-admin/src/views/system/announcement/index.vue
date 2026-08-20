<template>
  <div class="page">
    <el-card shadow="never" class="table-card">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-input
          v-model="search.keyword"
          placeholder="搜索公告标题"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="search.status" placeholder="状态" clearable style="width: 140px">
          <el-option label="草稿" :value="0" />
          <el-option label="已发布" :value="1" />
          <el-option label="已下架" :value="2" />
        </el-select>
        <el-select v-model="search.scope" placeholder="范围" clearable style="width: 140px">
          <el-option label="全平台" value="all" />
          <el-option label="指定租户" value="tenant" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>新建公告
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table :data="list" border stripe v-loading="loading" style="width: 100%">
        <el-table-column type="index" label="#" width="55" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="title-text">{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="scope" label="范围" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.scope === 'all' ? 'primary' : 'warning'">
              {{ row.scope === 'all' ? '全平台' : '指定租户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_tenant_id" label="目标租户" width="110">
          <template #default="{ row }">
            {{ row.target_tenant_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="getStatusType(row.status)">{{ row.status_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="publish_at" label="发布时间" width="160">
          <template #default="{ row }">{{ row.publish_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="expire_at" label="过期时间" width="160">
          <template #default="{ row }">{{ row.expire_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button link type="warning" @click="openEdit(row)">编辑</el-button>
            <el-button
              v-if="row.status !== 1"
              link
              type="success"
              @click="handleStatus(row, 1)"
            >发布</el-button>
            <el-button
              v-else
              link
              type="warning"
              @click="handleStatus(row, 2)"
            >下架</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑公告' : '新建公告'" width="680px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="请输入公告标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="6"
            placeholder="请输入公告内容"
            show-word-limit
            maxlength="5000"
          />
        </el-form-item>
        <el-form-item label="范围" prop="scope">
          <el-radio-group v-model="form.scope">
            <el-radio-button value="all">全平台</el-radio-button>
            <el-radio-button value="tenant">指定租户</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.scope === 'tenant'" label="目标租户" prop="target_tenant_id">
          <el-input v-model="form.target_tenant_id" placeholder="请输入目标租户ID" />
        </el-form-item>
        <el-form-item label="发布时间">
          <el-date-picker
            v-model="form.publish_at"
            type="datetime"
            placeholder="选择发布时间（可选）"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="form.expire_at"
            type="datetime"
            placeholder="选择过期时间（可选）"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio-button :value="0">草稿</el-radio-button>
            <el-radio-button :value="1">直接发布</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="公告详情" width="680px" destroy-on-close>
      <el-descriptions :column="2" border v-if="current">
        <el-descriptions-item label="标题" :span="2">{{ current.title }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag size="small" :type="getStatusType(current.status)">{{ current.status_text }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="范围">
          {{ current.scope === 'all' ? '全平台' : '指定租户' }}
        </el-descriptions-item>
        <el-descriptions-item label="目标租户" :span="2">{{ current.target_tenant_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="发布时间">{{ current.publish_at || '-' }}</el-descriptions-item>
        <el-descriptions-item label="过期时间">{{ current.expire_at || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间" :span="2">{{ current.created_at }}</el-descriptions-item>
        <el-descriptions-item label="公告内容" :span="2">
          <div class="content-box">{{ current.content }}</div>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import {
  getAnnouncementList,
  createAnnouncement,
  updateAnnouncement,
  updateAnnouncementStatus,
  deleteAnnouncement,
  getAnnouncementDetail
} from '@/api/modules/announcement'
import type { AnnouncementItem } from '@/api/modules/announcement'

const loading = ref(false)
const list = ref<AnnouncementItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const search = reactive({
  keyword: '',
  status: undefined as number | undefined,
  scope: ''
})

const dialogVisible = ref(false)
const detailVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const currentId = ref('')
const current = ref<AnnouncementItem | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({
  title: '',
  content: '',
  scope: 'all',
  target_tenant_id: '',
  status: 0,
  publish_at: '',
  expire_at: ''
})

const rules: FormRules = {
  title: [{ required: true, message: '请输入公告标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入公告内容', trigger: 'blur' }],
  scope: [{ required: true, message: '请选择范围', trigger: 'change' }],
  target_tenant_id: [
    {
      validator: (_: any, value: string, callback: any) => {
        if (form.scope === 'tenant' && !value) {
          callback(new Error('指定租户时必须填写目标租户ID'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

onMounted(() => {
  loadList()
})

async function loadList() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
      keyword: search.keyword || undefined
    }
    if (search.status !== undefined) params.status = search.status
    if (search.scope) params.scope = search.scope
    const res = await getAnnouncementList(params)
    list.value = res.data
    total.value = res.pagination?.total ?? 0
  } catch (error) {
    console.error('加载公告列表失败', error)
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  search.keyword = ''
  search.status = undefined
  search.scope = ''
  page.value = 1
  loadList()
}

function openCreate() {
  isEdit.value = false
  currentId.value = ''
  form.title = ''
  form.content = ''
  form.scope = 'all'
  form.target_tenant_id = ''
  form.status = 0
  form.publish_at = ''
  form.expire_at = ''
  dialogVisible.value = true
}

async function openEdit(row: AnnouncementItem) {
  isEdit.value = true
  currentId.value = row.id
  try {
    const detail = await getAnnouncementDetail(row.id)
    form.title = detail.title
    form.content = detail.content
    form.scope = detail.scope
    form.target_tenant_id = detail.target_tenant_id
    form.status = detail.status
    form.publish_at = detail.publish_at || ''
    form.expire_at = detail.expire_at || ''
  } catch {
    // 回退到行数据
    form.title = row.title
    form.content = row.content
    form.scope = row.scope
    form.target_tenant_id = row.target_tenant_id
    form.status = row.status
    form.publish_at = row.publish_at || ''
    form.expire_at = row.expire_at || ''
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    const payload = {
      title: form.title,
      content: form.content,
      scope: form.scope,
      target_tenant_id: form.scope === 'tenant' ? form.target_tenant_id : '',
      status: form.status,
      publish_at: form.publish_at || undefined,
      expire_at: form.expire_at || undefined
    }
    if (isEdit.value) {
      await updateAnnouncement(currentId.value, payload)
      ElMessage.success('公告更新成功')
    } else {
      await createAnnouncement(payload)
      ElMessage.success('公告创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch (error) {
    console.error('保存公告失败', error)
  } finally {
    submitting.value = false
  }
}

async function handleStatus(row: AnnouncementItem, status: number) {
  const text = status === 1 ? '发布' : '下架'
  try {
    await ElMessageBox.confirm(`确定要${text}公告「${row.title}」吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await updateAnnouncementStatus(row.id, status)
    ElMessage.success(`${text}成功`)
    loadList()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`${text}失败`)
    }
  }
}

async function handleDelete(row: AnnouncementItem) {
  try {
    await ElMessageBox.confirm(`确定要删除公告「${row.title}」吗？`, '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteAnnouncement(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function openDetail(row: AnnouncementItem) {
  current.value = row
  detailVisible.value = true
}

function getStatusType(status: number): string {
  const map: Record<number, string> = { 0: 'info', 1: 'success', 2: 'warning' }
  return map[status] || 'info'
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.table-card {
  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }
}

.title-text {
  font-weight: 500;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.flex-1 {
  flex: 1;
}

.content-box {
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow: auto;
  background: #f8fafc;
  padding: 12px;
  border-radius: 6px;
  font-size: 14px;
  line-height: 1.6;
}
</style>
