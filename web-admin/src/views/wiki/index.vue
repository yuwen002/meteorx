<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          placeholder="搜索空间名称"
          clearable
          style="width: 240px"
          @keyup.enter="search"
        />
        <el-button type="primary" @click="search">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button @click="goSearchPage">
          <el-icon><Search /></el-icon>Wiki 搜索
        </el-button>
        <el-button @click="goTrashPage">
          <el-icon><Delete /></el-icon>回收站
        </el-button>
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新建空间
        </el-button>
      </div>

      <el-row v-loading="loading" :gutter="16">
        <el-col v-for="item in list" :key="item.id" :span="8" style="margin-bottom: 16px">
          <el-card shadow="hover" class="space-card" @click="goDetail(item)">
            <div class="space-header">
              <el-icon :size="28"><Reading /></el-icon>
              <span class="space-name">{{ item.name }}</span>
              <el-tag :type="visibilityTagType(item.visibility)" size="small" style="margin-left: auto">
                {{ visibilityText(item.visibility) }}
              </el-tag>
            </div>
            <div v-if="item.description" class="space-desc">{{ item.description }}</div>
            <div v-else class="space-desc empty">暂无描述</div>
            <div class="space-footer">
              <span>成员 {{ item.member_count || 0 }}</span>
              <span>节点 {{ item.node_count || 0 }}</span>
              <span>{{ item.updated_at?.substring(0, 10) }}</span>
            </div>
            <div v-if="canManage(item)" class="space-actions" @click.stop>
              <el-button link type="primary" @click="openEditDialog(item)">编辑</el-button>
              <el-button link type="danger" @click="handleDelete(item)">删除</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-empty v-if="!loading && list.length === 0" description="暂无知识空间" />

      <div v-if="total > 0" class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[12, 24, 48]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑空间' : '新建空间'" width="480px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入空间名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入空间描述" />
        </el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="form.visibility">
            <el-radio :label="1">私有（仅成员可见）</el-radio>
            <el-radio :label="2">租户内可见</el-radio>
            <el-radio :label="3">公开（平台可见）</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus, Reading, Delete } from '@element-plus/icons-vue'
import {
  listSpaces,
  createSpace,
  updateSpace,
  deleteSpace,
  type WikiSpace
} from '@/api/modules/wiki'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const router = useRouter()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  search,
  reset,
  reload
} = useTableList<WikiSpace, { keyword: string }>({
  fetchList: async (params) =>
    toPageResult(
      await listSpaces({
        page: params.page,
        page_size: params.page_size,
        keyword: (params.keyword as string) || undefined
      })
    ),
  initialQuery: { keyword: '' },
  defaultPageSize: 12
})
const loadList = reload

const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ name: '', description: '', visibility: 1 })
const formRules: FormRules = {
  name: [{ required: true, message: '请输入空间名称', trigger: 'blur' }]
}

function resetSearch() {
  reset()
}

// 可见性：1=私有  2=租户内可见  3=公开
function visibilityText(v: number) {
  if (v === 3) return '公开'
  if (v === 2) return '租户可见'
  return '私有'
}
function visibilityTagType(v: number): 'success' | 'warning' | 'info' {
  if (v === 3) return 'success'
  if (v === 2) return 'warning'
  return 'info'
}

// 仅 owner/admin 可对空间做管理与删除
function canManage(item: WikiSpace) {
  return item.my_role === 'owner' || item.my_role === 'admin'
}

function goDetail(item: WikiSpace) {
  router.push(`/wiki/spaces/${item.id}`)
}
function goSearchPage() {
  router.push('/wiki/search')
}
function goTrashPage() {
  router.push('/wiki/trash')
}

function openCreateDialog() {
  editingId.value = null
  form.name = ''
  form.description = ''
  form.visibility = 1
  dialogVisible.value = true
}

function openEditDialog(item: WikiSpace) {
  editingId.value = item.id
  form.name = item.name
  form.description = item.description || ''
  form.visibility = item.visibility || 1
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateSpace(editingId.value, form)
        ElMessage.success('更新成功')
      } else {
        await createSpace(form)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      submitting.value = false
    }
  })
}

function handleDelete(item: WikiSpace) {
  ElMessageBox.confirm(`确定要删除空间 "${item.name}" 吗？`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      if (!item.id) return
      await deleteSpace(item.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
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
.space-card {
  cursor: pointer;
  transition: transform 0.2s;
  height: 100%;
}
.space-card:hover {
  transform: translateY(-2px);
}
.space-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.space-name {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 160px;
}
.space-desc {
  font-size: 13px;
  color: #6b7280;
  min-height: 36px;
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.space-desc.empty {
  color: #9ca3af;
  font-style: italic;
}
.space-footer {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #9ca3af;
  padding-top: 8px;
  border-top: 1px solid #f3f4f6;
}
.space-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}
</style>
