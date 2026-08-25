<template>
  <div class="wiki-space">
    <div class="sidebar">
      <div class="sidebar-header">
        <el-button link @click="goBack" :icon="ArrowLeft" />
        <span class="space-title">{{ spaceInfo?.name || '加载中...' }}</span>
      </div>
      <div class="sidebar-actions">
        <el-button size="small" @click="createNode(null)">新建文件夹</el-button>
        <el-button size="small" type="primary" @click="createDocument(null)">新建文档</el-button>
      </div>
      <el-tree
        ref="treeRef"
        :data="treeData"
        node-key="id"
        :props="{ label: 'title', children: 'children' }"
        highlight-current
        default-expand-all
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <el-icon v-if="data.type === 'folder'"><Folder /></el-icon>
            <el-icon v-else><Document /></el-icon>
            <span class="node-title">{{ data.title }}</span>
            <span class="node-actions" @click.stop>
              <el-icon v-if="data.type === 'folder'" class="action-icon" @click="createNode(data.id)" title="新建子文件夹"><FolderAdd /></el-icon>
              <el-icon v-if="data.type === 'folder'" class="action-icon" @click="createDocument(data.id)" title="新建文档"><DocumentAdd /></el-icon>
              <el-icon class="action-icon" @click="editNode(data)" title="编辑"><Edit /></el-icon>
              <el-icon class="action-icon danger" @click="deleteNode(data)" title="删除"><Delete /></el-icon>
            </span>
          </span>
        </template>
      </el-tree>
    </div>

    <div class="content">
      <template v-if="currentNode">
        <div class="content-header">
          <h2>{{ currentNode.title }}</h2>
          <div class="header-actions">
            <el-button size="small" @click="showRevisions = !showRevisions">历史版本</el-button>
            <el-button type="primary" size="small" @click="saveDocument" :loading="saving">保存</el-button>
          </div>
        </div>
        <div class="content-meta">
          <span v-if="document?.view_count">浏览 {{ document.view_count }}</span>
          <span v-if="document?.updated_at">更新于 {{ document.updated_at?.substring(0, 19).replace('T', ' ') }}</span>
        </div>
        <div class="editor-area">
          <el-input
            v-model="editableContent"
            type="textarea"
            :rows="20"
            placeholder="开始编写文档内容..."
            resize="vertical"
            class="doc-editor"
          />
        </div>

        <el-table v-if="showRevisions && revisions.length > 0" :data="revisions" border size="small" style="margin-top: 16px">
          <el-table-column prop="version" label="版本" width="80" />
          <el-table-column prop="summary" label="修改说明" />
          <el-table-column prop="created_at" label="修改时间" width="170" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="restoreRevision(row)">恢复</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
      <div v-else class="empty">
        <el-icon :size="64" color="#d1d5db"><Document /></el-icon>
        <p>从左侧选择一个文档开始编辑</p>
        <p class="hint">或点击"新建文档"创建新文档</p>
      </div>
    </div>

    <el-dialog v-model="editDialogVisible" title="编辑节点" width="400px">
      <el-form ref="editFormRef" :model="editForm" :rules="editRules" label-width="80px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="editForm.title" />
        </el-form-item>
        <el-form-item label="图标">
          <el-select v-model="editForm.icon" placeholder="选择图标">
            <el-option label="文档" value="document" />
            <el-option label="文件夹" value="folder" />
            <el-option label="笔记" value="note" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitEditNode">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  ArrowLeft, FolderAdd, DocumentAdd, Folder, Document, Edit, Delete
} from '@element-plus/icons-vue'
import {
  getSpace,
  getNodeTree,
  createNode as createNodeApi,
  updateNode,
  deleteNode as deleteNodeApi,
  createDocument as createDocumentApi,
  getDocument,
  updateDocument,
  listRevisions,
  restoreRevision as restoreRevisionApi,
  type WikiSpace,
  type WikiNodeTree,
  type WikiDocument,
  type DocumentRevision
} from '@/api/modules/wiki'

const route = useRoute()
const router = useRouter()
const spaceId = computed(() => route.params.id as string)

const spaceInfo = ref<WikiSpace | null>(null)
const treeData = ref<WikiNodeTree[]>([])
const treeRef = ref()

const currentNode = ref<WikiNodeTree | null>(null)
const document = ref<WikiDocument | null>(null)
const editableContent = ref('')
const saving = ref(false)
const showRevisions = ref(false)
const revisions = ref<DocumentRevision[]>([])

const editDialogVisible = ref(false)
const editFormRef = ref<FormInstance>()
const editingNodeId = ref<string | null>(null)
const editForm = ref({ title: '', icon: '' })
const editRules: FormRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }]
}

function goBack() {
  router.push('/wiki')
}

async function loadSpaceInfo() {
  try {
    const res = await getSpace(spaceId.value)
    spaceInfo.value = res
  } catch (e) {
    ElMessage.error('加载空间信息失败')
  }
}

async function loadTree() {
  try {
    const res = await getNodeTree(spaceId.value)
    treeData.value = res || []
  } catch (e) {
    treeData.value = []
  }
}

function handleNodeClick(node: WikiNodeTree) {
  if (node.type !== 'document') {
    currentNode.value = null
    document.value = null
    editableContent.value = ''
    showRevisions.value = false
    return
  }
  currentNode.value = node
  showRevisions.value = false
  loadDocument(node.id)
}

async function loadDocument(nodeId: string) {
  try {
    const res = await getDocument(nodeId)
    document.value = res
    editableContent.value = res?.content || ''
  } catch (e) {
    ElMessage.error('加载文档失败')
  }
}

async function saveDocument() {
  if (!document.value) return
  saving.value = true
  try {
    await updateDocument(document.value.id, {
      content: editableContent.value,
      summary: '编辑更新'
    })
    ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function createNode(parentId: string | null) {
  try {
    const res = await createNodeApi(spaceId.value, {
      parent_id: parentId || undefined,
      type: 'folder',
      title: '新建文件夹',
      icon: 'folder'
    })
    ElMessage.success('创建成功')
    await loadTree()
  } catch (e) {
    ElMessage.error('创建失败')
  }
}

async function createDocument(parentId: string | null) {
  try {
    const nodeRes = await createNodeApi(spaceId.value, {
      parent_id: parentId || undefined,
      type: 'document',
      title: '新建文档',
      icon: 'document'
    })
    await createDocumentApi(nodeRes.id, { node_id: nodeRes.id })
    ElMessage.success('创建成功')
    await loadTree()
    currentNode.value = nodeRes
    editableContent.value = ''
  } catch (e) {
    ElMessage.error('创建失败')
  }
}

function editNode(node: WikiNodeTree) {
  editingNodeId.value = node.id
  editForm.value = { title: node.title, icon: node.icon }
  editDialogVisible.value = true
}

async function submitEditNode() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      if (editingNodeId.value) {
        await updateNode(editingNodeId.value, editForm.value)
        ElMessage.success('更新成功')
        editDialogVisible.value = false
        await loadTree()
        if (currentNode.value?.id === editingNodeId.value) {
          currentNode.value = { ...currentNode.value!, title: editForm.value.title, icon: editForm.value.icon }
        }
      }
    } catch (e) {
      ElMessage.error('更新失败')
    }
  })
}

function deleteNode(node: WikiNodeTree) {
  ElMessageBox.confirm(`确定要删除 "${node.title}" 吗？此操作不可恢复！`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      await deleteNodeApi(node.id)
      ElMessage.success('删除成功')
      if (currentNode.value?.id === node.id) {
        currentNode.value = null
        document.value = null
      }
      await loadTree()
    })
    .catch(() => {})
}

async function loadRevisions() {
  if (!document.value) return
  try {
    const res = await listRevisions(document.value.id)
    revisions.value = res || []
  } catch (e) {
    revisions.value = []
  }
}

async function restoreRevision(rev: DocumentRevision) {
  if (!document.value) return
  try {
    await ElMessageBox.confirm(`确定要恢复到版本 v${rev.version} 吗？`, '提示', { type: 'warning' })
    await restoreRevisionApi(document.value.id, rev.version)
    ElMessage.success('恢复成功')
    loadDocument(currentNode.value!.id)
    loadRevisions()
  } catch (e) {
    // user cancelled
  }
}

watch(showRevisions, (val) => {
  if (val && document.value) {
    loadRevisions()
  }
})

onMounted(async () => {
  await Promise.all([loadSpaceInfo(), loadTree()])
})
</script>

<style scoped>
.wiki-space {
  display: flex;
  height: calc(100vh - 56px - 32px);
  gap: 16px;
}
.sidebar {
  width: 280px;
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.sidebar-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.space-title {
  font-weight: 600;
  font-size: 15px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sidebar-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.sidebar-actions .el-button {
  flex: 1;
}
.sidebar :deep(.el-tree) {
  flex: 1;
  overflow-y: auto;
  border-right: none;
}
.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  font-size: 13px;
}
.node-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-actions {
  display: none;
  gap: 2px;
}
.el-tree-node:hover .node-actions {
  display: flex;
}
.action-icon {
  font-size: 14px;
  cursor: pointer;
  color: #9ca3af;
}
.action-icon:hover {
  color: #3b82f6;
}
.action-icon.danger:hover {
  color: #ef4444;
}
.content {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}
.content-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.content-header h2 {
  margin: 0;
  font-size: 18px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.content-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f3f4f6;
}
.editor-area {
  flex: 1;
}
.doc-editor :deep(.el-textarea__inner) {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 14px;
  line-height: 1.6;
  min-height: 400px !important;
}
.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
}
.empty p {
  margin: 8px 0 0;
}
.empty .hint {
  font-size: 13px;
}
</style>