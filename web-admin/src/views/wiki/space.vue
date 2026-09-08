<template>
  <div class="space-page" v-loading="loadingSpace">
    <!-- 顶部导航 -->
    <div class="topbar">
      <el-button :icon="Back" circle @click="router.push('/wiki')" />
      <div class="space-info">
        <span class="space-title">{{ spaceInfo?.name || '知识空间' }}</span>
        <el-tag v-if="spaceInfo" :type="visibilityTagType(spaceInfo.visibility)" size="small">
          {{ visibilityText(spaceInfo.visibility) }}
        </el-tag>
        <span v-if="spaceInfo?.description" class="space-desc">{{ spaceInfo.description }}</span>
      </div>
      <div class="flex-1"></div>
      <!-- 通知中心 -->
      <NotificationCenter />
      <el-button
        v-if="canManage"
        :icon="User"
        :disabled="!spaceInfo"
        @click="membersDialogVisible = true"
      >
        成员
      </el-button>
      <el-button
        v-if="canManage"
        type="danger"
        plain
        :disabled="!spaceInfo"
        @click="handleDeleteSpace"
      >
        删除空间
      </el-button>
    </div>

    <div class="workspace">
      <!-- 左侧：目录树 -->
      <aside class="sidebar" v-loading="loadingTree">
        <div class="sidebar-head">
          <span class="title">目录</span>
          <div v-if="canEdit" class="tree-actions">
            <el-tooltip content="新建文档">
              <el-button circle size="small" :icon="Plus" @click="openCreateNode('document')" />
            </el-tooltip>
            <el-tooltip content="新建目录">
              <el-button circle size="small" :icon="FolderAdd" @click="openCreateNode('folder')" />
            </el-tooltip>
          </div>
        </div>
        <el-tree
          v-if="tree.length"
          ref="treeRef"
          :data="tree"
          node-key="id"
          :props="{ label: 'title', children: 'children' }"
          :current-node-key="selectedNodeId"
          :expand-on-click-node="false"
          :default-expanded-keys="expandedKeys"
          highlight-current
          @node-click="selectNode"
        >
          <template #default="{ data }">
            <div class="tree-node">
              <el-icon class="node-icon" :class="{ active: data.id === selectedNodeId }">
                <FolderOpened v-if="data.type === 'folder'" />
                <Document v-else />
              </el-icon>
              <span class="node-label">{{ data.title }}</span>
              <span v-if="canEdit" class="node-ops" @click.stop>
                <el-button
                  v-if="data.type === 'folder'"
                  link
                  size="small"
                  :icon="Plus"
                  title="在该目录下新建"
                  @click="openCreateNode(data.type === 'folder' ? 'folder' : 'document', data)"
                />
                <el-button link size="small" :icon="EditPen" title="重命名" @click="openRename(data)" />
                <el-button
                  link
                  size="small"
                  :icon="Lock"
                  title="节点权限"
                  @click="openNodePermission(data)"
                />
                <el-button link size="small" type="danger" :icon="Delete" title="删除" @click="removeNode(data)" />
              </span>
            </div>
          </template>
        </el-tree>
        <el-empty v-if="!loadingTree && tree.length === 0" description="空间是空的" :image-size="60" />
      </aside>

      <!-- 右侧：文档区 -->
      <main class="doc-area">
        <!-- 阅读态 -->
        <template v-if="!editing && currentDocument">
          <!-- 编辑锁指示器 -->
          <EditLockIndicator 
            v-if="canEdit"
            :document-id="currentDocument.id" 
            :current-user-id="myUserId"
            @locked="handleEditLocked"
            @unlocked="handleEditUnlocked"
          />
          
          <div class="doc-toolbar">
            <span class="doc-title">{{ currentNode?.title }}</span>
            <div class="flex-1"></div>
            <!-- 标签显示 -->
            <div v-if="documentTags.length > 0" class="doc-tags">
              <el-tag 
                v-for="tag in documentTags" 
                :key="tag.id" 
                :color="tag.tag?.color || '#409EFF'"
                size="small"
                style="margin-right: 4px"
              >
                {{ tag.tag?.name }}
              </el-tag>
            </div>
            <el-tag v-if="canEdit" type="warning" size="small" class="unsaved-tip">
              最后编辑 {{ currentDocument.last_edited_at?.replace('T', ' ').substring(0, 16) }}
            </el-tag>
            <el-button v-if="canEdit" :icon="PriceTag" @click="tagManagerRef?.open()">标签</el-button>
            <el-button v-if="canEdit" :icon="Share" @click="shareManagerRef?.open()">分享</el-button>
            <el-button :icon="DataAnalysis" @click="showStats = !showStats">统计</el-button>
            <el-button v-if="canEdit" type="primary" :icon="EditPen" @click="startEditing">编辑</el-button>
            <el-button v-if="canEdit" :icon="FolderAdd" @click="membersDialogVisible = true">成员</el-button>
          </div>
          <div class="doc-body article" v-html="currentDocument.content_html"></div>
          
          <!-- 评论区 -->
          <CommentSection v-if="canEdit" :document-id="currentDocument.id" />
          
          <!-- 统计面板 -->
          <StatsPanel v-if="showStats" :document-id="currentDocument.id" />
        </template>

        <!-- 编辑态 -->
        <template v-else-if="editing && currentDocument">
          <div class="editor-toolbar">
            <div class="mode-switch">
              <el-button
                size="small"
                :type="editorMode === 'editor' ? 'primary' : ''"
                :icon="Document"
                @click="editorMode = 'editor'"
              >
                仅编辑
              </el-button>
              <el-button
                size="small"
                :type="editorMode === 'split' ? 'primary' : ''"
                :icon="Operation"
                @click="editorMode = 'split'"
              >
                分栏
              </el-button>
              <el-button
                size="small"
                :type="editorMode === 'preview' ? 'primary' : ''"
                :icon="View"
                @click="editorMode = 'preview'"
              >
                仅预览
              </el-button>
            </div>
            <div class="md-tools">
              <el-tooltip content="使用模板"><el-button size="small" :icon="Files" @click="templateSelectorRef?.open()" /></el-tooltip>
              <el-tooltip content="标题"><el-button size="small" :icon="Menu" @click="mdHeading" /></el-tooltip>
              <el-tooltip content="加粗">
                <el-button size="small" @click="mdBold"><b>B</b></el-button>
              </el-tooltip>
              <el-tooltip content="斜体">
                <el-button size="small" @click="mdItalic"><i>I</i></el-button>
              </el-tooltip>
              <el-tooltip content="行内代码"><el-button size="small" :icon="Cpu" @click="mdCode" /></el-tooltip>
              <el-tooltip content="代码块"><el-button size="small" :icon="Box" @click="mdCodeBlock" /></el-tooltip>
              <el-tooltip content="链接"><el-button size="small" :icon="Link" @click="mdLink" /></el-tooltip>
              <el-tooltip content="列表"><el-button size="small" :icon="List" @click="mdList" /></el-tooltip>
              <el-tooltip content="引用"><el-button size="small" :icon="ChatDotRound" @click="mdQuote" /></el-tooltip>
              <el-tooltip content="插入图片">
                <el-button size="small" :icon="Picture" :loading="uploading" @click="imageInput?.click()" />
              </el-tooltip>
              <input
                ref="imageInput"
                type="file"
                accept="image/*"
                class="hidden-input"
                @change="handleImageUpload"
              />
            </div>
            <div class="flex-1"></div>
            <el-tooltip v-if="dirty" content="内容有未保存修改">
              <span class="dirty-dot">● 未保存</span>
            </el-tooltip>
            <el-button :icon="Close" @click="exitEditing">退出编辑</el-button>
            <el-button type="primary" :icon="Check" :loading="saving" @click="saveDoc">保存</el-button>
          </div>

          <div class="editor-body">
            <textarea
              v-if="editorMode !== 'preview'"
              ref="editorRef"
              v-model="editingContent"
              class="md-input"
              spellcheck="false"
              @input="schedulePreview"
            ></textarea>
            <div
              v-if="editorMode !== 'editor'"
              class="md-preview article"
              v-loading="previewLoading"
            >
              <div v-if="previewHtml" v-html="previewHtml"></div>
              <el-empty v-else description="输入内容后将自动渲染预览" :image-size="60" />
            </div>
          </div>
        </template>

        <!-- 附件 / 版本 / 扩展功能面板 -->
        <template v-else>
          <el-empty
            :description="currentNode ? '该目录下暂无打开文档' : '从左侧选择一个文档开始阅读'"
            :image-size="90"
          />
        </template>

        <!-- 附件、版本与扩展功能面板 -->
        <div v-if="currentDocument" class="sub-panels">
          <div class="panel-tabs">
            <span
              class="tab"
              :class="{ active: activePanel === 'attachments' }"
              @click="activePanel = 'attachments'"
            >
              附件 ({{ attachments.length }})
            </span>
            <span
              class="tab"
              :class="{ active: activePanel === 'history' }"
              @click="activePanel = 'history'"
            >
              历史版本 ({{ revisions.length }})
            </span>
            <span
              class="tab"
              :class="{ active: activePanel === 'tags' }"
              @click="activePanel = 'tags'"
            >
              标签 ({{ documentTags.length }})
            </span>
            <span
              class="tab"
              :class="{ active: activePanel === 'shares' }"
              @click="activePanel = 'shares'; shareManagerRef?.open()"
            >
              分享
            </span>
          </div>

          <div v-if="activePanel === 'attachments'" class="panel-body">
            <div v-if="canEdit" class="attach-ops">
              <el-upload
                :auto-upload="false"
                :show-file-list="false"
                multiple
                accept="*/*"
                :on-change="(f: UploadFile) => handleAttachmentAdd(f.raw as File)"
              >
                <el-button :icon="Upload" :loading="uploading">上传附件</el-button>
              </el-upload>
            </div>
            <div v-if="attachments.length" class="attach-grid">
              <div v-for="a in attachments" :key="a.id" class="attach-item">
                <el-icon class="attach-icon"><Document /></el-icon>
                <span class="attach-name" :title="a.file_name">{{ a.file_name }}</span>
                <span class="attach-size">{{ formatSize(a.file_size) }}</span>
                <div class="attach-ops">
                  <el-button link type="primary" size="small" @click="previewAttachment(a)">预览/下载</el-button>
                  <el-button
                    v-if="canEdit"
                    link
                    type="danger"
                    size="small"
                    @click="removeAttachment(a)"
                  >
                    删除
                  </el-button>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无附件" :image-size="50" />
          </div>

          <div v-else-if="activePanel === 'history'" class="panel-body">
            <div v-if="canEdit" class="history-ops" style="margin-bottom: 8px; display: flex; gap: 8px;">
              <el-button size="small" :icon="Download" @click="exportDocument">导出文档</el-button>
              <el-upload
                :auto-upload="false"
                :show-file-list="false"
                :on-change="handleImportFile"
                accept=".md,.markdown"
              >
                <el-button size="small" :icon="Upload">导入文档</el-button>
              </el-upload>
              <el-button 
                v-if="revisions.length >= 2"
                size="small" 
                :icon="Difference" 
                @click="openDiffDialog"
              >
                版本对比
              </el-button>
            </div>
            <el-table 
              v-loading="loadingRevisions" 
              :data="revisions" 
              size="small"
              @selection-change="handleRevisionSelection"
            >
              <el-table-column type="selection" width="40" />
              <el-table-column label="版本" width="70">
                <template #default="{ row }">v{{ row.version }}</template>
              </el-table-column>
              <el-table-column prop="summary" label="更新说明" min-width="140" show-overflow-tooltip />
              <el-table-column label="编辑时间" width="160">
                <template #default="{ row }">
                  {{ row.created_at?.replace('T', ' ').substring(0, 16) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="130" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" @click="viewRevision(row)">查看</el-button>
                  <el-button v-if="canEdit" link type="warning" @click="restoreVersion(row)">恢复</el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-if="!loadingRevisions && revisions.length === 0" description="暂无历史版本" :image-size="50" />
          </div>

          <!-- 标签面板 -->
          <div v-else-if="activePanel === 'tags'" class="panel-body">
            <div v-if="canEdit" class="tag-ops" style="margin-bottom: 12px">
              <el-button type="primary" size="small" @click="tagManagerRef?.open()">管理标签</el-button>
            </div>
            <div v-if="documentTags.length" class="tag-grid">
              <el-tag
                v-for="dt in documentTags"
                :key="dt.id"
                :color="dt.tag?.color || '#409EFF'"
                closable
                @close="handleRemoveTag(dt.tag?.id || '')"
                style="margin: 4px"
              >
                {{ dt.tag?.name }}
              </el-tag>
            </div>
            <el-empty v-else description="暂无标签，点击「管理标签」添加" :image-size="50" />
          </div>

          <!-- 分享面板 -->
          <div v-else-if="activePanel === 'shares'" class="panel-body">
            <ShareLinkManager v-if="currentDocument" :document-id="currentDocument.id" />
          </div>
        </div>
      </main>
    </div>

    <!-- 弹窗 -->
    <el-dialog v-model="createVisible" :title="createForm.type === 'document' ? '新建文档' : '新建目录'" width="420px">
      <el-form :model="createForm" label-width="60px">
        <el-form-item :label="createForm.type === 'document' ? '文档名' : '目录名'">
          <el-input v-model="createForm.title" placeholder="请输入名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreateNode">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="renameVisible" title="重命名" width="420px">
      <el-input v-model="renameTitle" placeholder="请输入新名称" />
      <template #footer>
        <el-button @click="renameVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitRename">确定</el-button>
      </template>
    </el-dialog>

    <!-- 历史版本预览 -->
    <el-dialog v-model="revisionDialogVisible" title="版本预览" width="760px" top="5vh">
      <div class="article" v-html="revisionPreviewHtml"></div>
    </el-dialog>

    <!-- 版本对比对话框 -->
    <el-dialog v-model="diffDialogVisible" title="版本对比" width="900px" top="5vh">
      <div class="diff-selector" style="margin-bottom: 16px; display: flex; gap: 16px; align-items: center;">
        <span>对比版本：</span>
        <el-select v-model="diffVersion1" placeholder="选择旧版本" style="width: 150px">
          <el-option
            v-for="rev in revisions"
            :key="rev.version"
            :label="`v${rev.version}`"
            :value="rev.version"
          />
        </el-select>
        <span>→</span>
        <el-select v-model="diffVersion2" placeholder="选择新版本" style="width: 150px">
          <el-option
            v-for="rev in revisions"
            :key="rev.version"
            :label="`v${rev.version}`"
            :value="rev.version"
          />
        </el-select>
        <el-button type="primary" :loading="loadingDiff" @click="loadDiff">对比</el-button>
      </div>
      <div v-if="diffHtml" class="diff-view" v-html="diffHtml"></div>
      <el-empty v-else description="选择两个版本后点击对比查看差异" :image-size="80" />
    </el-dialog>

    <!-- 附件图片预览 -->
    <el-dialog v-model="attachmentPreviewVisible" title="图片预览" width="640px">
      <div class="article preview-img-wrap">
        <img :src="attachmentPreviewUrl" alt="预览" />
      </div>
      <template #footer>
        <el-button type="primary" @click="downloadAttachment(currentAttachment!)">下载原图</el-button>
      </template>
    </el-dialog>

    <MembersDialog
      v-model="membersDialogVisible"
      :space-id="spaceId"
      :my-user-id="myUserId"
      :my-role="myRole"
      @changed="reloadTree"
    />
    <NodePermissionDialog
      v-model="nodePermVisible"
      :space-id="spaceId"
      :node-id="permNodeId"
      :title="permNodeTitle"
    />
    
    <!-- 扩展功能对话框 -->
    <TagManager ref="tagManagerRef" />
    <ShareLinkManager ref="shareManagerRef" :document-id="currentDocument?.id || ''" />
    <TemplateSelector ref="templateSelectorRef" @select="handleTemplateSelect" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { ElTree, UploadFile } from 'element-plus'
import {
  Back,
  Plus,
  EditPen,
  Delete,
  FolderAdd,
  FolderOpened,
  Document,
  User,
  Lock,
  Menu,
  Cpu,
  Box,
  Link,
  List,
  ChatDotRound,
  Picture,
  Upload,
  Check,
  Close,
  Operation,
  View,
  PriceTag,
  Share,
  DataAnalysis,
  Files,
  Download,
  Difference
} from '@element-plus/icons-vue'
import {
  getSpace,
  getNodeTree,
  createNode,
  updateNode,
  deleteNode,
  getDocument,
  updateDocument,
  deleteSpace,
  listRevisions,
  restoreRevision,
  listAttachments,
  createAttachment,
  deleteAttachment,
  previewMarkdown,
  compareRevisions,
  exportDocument as exportDocumentApi,
  importDocument as importDocumentApi,
  type WikiSpace,
  type WikiNode,
  type WikiNodeTree,
  type WikiDocument,
  type DocumentRevision,
  type WikiAttachment
} from '@/api/modules/wiki'
import { uploadFile, downloadFile } from '@/api/modules/file'
import { useUserStore } from '@/stores/user'
import MembersDialog from './components/MembersDialog.vue'
import NodePermissionDialog from './components/NodePermissionDialog.vue'
import TagManager from './components/TagManager.vue'
import ShareLinkManager from './components/ShareLinkManager.vue'
import CommentSection from './components/CommentSection.vue'
import StatsPanel from './components/StatsPanel.vue'
import TemplateSelector from './components/TemplateSelector.vue'
import NotificationCenter from './components/NotificationCenter.vue'
import EditLockIndicator from './components/EditLockIndicator.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const spaceId = route.params.id as string
const myUserId = computed(() => userStore.userInfo?.id || '')

// ============ 空间 & 权限 ============
const loadingSpace = ref(false)
const spaceInfo = ref<WikiSpace | null>(null)
const myRole = computed(() => spaceInfo.value?.my_role || '')
const canManage = computed(() => myRole.value === 'owner' || myRole.value === 'admin')
const canEdit = computed(() => myRole.value === 'owner' || myRole.value === 'admin' || myRole.value === 'editor')

// ============ 目录树 ============
const treeRef = ref<InstanceType<typeof ElTree>>()
const loadingTree = ref(false)
const tree = ref<WikiNodeTree[]>([])
const expandedKeys = ref<string[]>([])
const selectedNodeId = ref('')
const currentNode = ref<WikiNode | null>(null)

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

function collectAncestors(nodes: WikiNodeTree[], targetId: string, stack: string[] = []): string[] | null {
  for (const n of nodes) {
    const path = [...stack, n.id]
    if (n.id === targetId) return path
    if (n.children?.length) {
      const found = collectAncestors(n.children, targetId, path)
      if (found) return found
    }
  }
  return null
}

async function loadSpace() {
  loadingSpace.value = true
  try {
    spaceInfo.value = await getSpace(spaceId)
  } catch {
    router.replace('/wiki')
  } finally {
    loadingSpace.value = false
  }
}

async function reloadTree() {
  loadingTree.value = true
  try {
    const res = (await getNodeTree(spaceId)) ?? []
    tree.value = res
    const keep = selectedNodeId.value
    if (keep) {
      const path = collectAncestors(tree.value, keep)
      if (path) expandedKeys.value = path.slice(0, -1)
    }
  } catch {
    tree.value = []
  } finally {
    loadingTree.value = false
  }
}

function findNodeById(nodes: WikiNodeTree[], id: string): WikiNodeTree | null {
  for (const n of nodes) {
    if (n.id === id) return n
    if (n.children) {
      const hit = findNodeById(n.children, id)
      if (hit) return hit
    }
  }
  return null
}

async function selectNode(node: WikiNodeTree) {
  currentNode.value = node
  selectedNodeId.value = node.id
  activePanel.value = 'attachments'
  if (node.type === 'document') {
    await loadDocument(node.id)
  } else {
    currentDocument.value = null
    editing.value = false
  }
}

// ============ 文档 ============
const currentDocument = ref<WikiDocument | null>(null)
const editing = ref(false)
const editorMode = ref<'editor' | 'split' | 'preview'>('split')
const editingContent = ref('')
const previewHtml = ref('')
const previewTimer = ref<number>()
const previewLoading = ref(false)
const dirty = ref(false)
const saving = ref(false)
const editorRef = ref<HTMLTextAreaElement>()

async function loadDocument(nodeId: string) {
  try {
    currentDocument.value = await getDocument(nodeId)
    activePanel.value = 'attachments'
    await Promise.all([
      loadAttachments(),
      loadRevisions(),
      loadDocumentTags()
    ])
  } catch {
    currentDocument.value = null
    documentTags.value = []
  }
}

async function startEditing() {
  if (!currentDocument.value) return
  editing.value = true
  editingContent.value = currentDocument.value.content || ''
  previewHtml.value = ''
  dirty.value = false
  await nextTick()
  schedulePreview()
}

async function schedulePreview() {
  dirty.value = true
  window.clearTimeout(previewTimer.value)
  previewTimer.value = window.setTimeout(runPreview, 350)
}

async function runPreview() {
  if (!editingContent.value.trim()) {
    previewHtml.value = ''
    return
  }
  previewLoading.value = true
  try {
    const res = await previewMarkdown(editingContent.value)
    previewHtml.value = res?.content_html || ''
  } catch {
    previewHtml.value = ''
  } finally {
    previewLoading.value = false
  }
}

async function saveDoc() {
  if (!currentDocument.value || !canEdit.value) return
  saving.value = true
  try {
    currentDocument.value = await updateDocument(currentDocument.value.id, {
      content: editingContent.value,
      format: 'markdown'
    })
    dirty.value = false
    // 文档内容变化后刷新历史版本
    await loadRevisions()
    ElMessage.success('已保存')
  } finally {
    saving.value = false
  }
}

async function exitEditing() {
  if (dirty.value) {
    try {
      await ElMessageBox.confirm('有未保存的修改，退出编辑将丢弃，确定退出吗？', '提示', {
        type: 'warning'
      })
    } catch {
      return
    }
  }
  editing.value = false
  editorMode.value = 'split'
  dirty.value = false
  if (currentDocument.value) {
    // 重新拉取最新内容
    await loadDocument(currentDocument.value.node_id)
  }
}

onBeforeRouteLeave(() => {
  if (editing.value && dirty.value) {
    return window.confirm('有未保存的修改，确定离开吗？')
  }
  return true
})

// ============ Markdown 工具栏 ============
const imageInput = ref<HTMLInputElement>()
const uploading = ref(false)

function replaceSelection(prefix: string, suffix = '', placeholder = '') {
  const ta = editorRef.value
  if (!ta) return
  const start = ta.selectionStart
  const end = ta.selectionEnd
  const sel = editingContent.value.slice(start, end)
  const target = sel || placeholder
  const next = editingContent.value.slice(0, start) + prefix + target + suffix + editingContent.value.slice(end)
  editingContent.value = next
  void nextTick(() => {
    ta.focus()
    ta.setSelectionRange(start + prefix.length, start + prefix.length + target.length)
  })
  schedulePreview()
}

function blockPrefix(p: string) {
  const ta = editorRef.value
  if (!ta) return
  const start = ta.selectionStart
  const lineStart = editingContent.value.lastIndexOf('\n', start - 1) + 1
  const next = editingContent.value.slice(0, lineStart) + p + editingContent.value.slice(lineStart)
  editingContent.value = next
  void nextTick(() => {
    ta.focus()
    ta.setSelectionRange(start + p.length, start + p.length)
  })
  schedulePreview()
}

function mdHeading() {
  blockPrefix('## ')
}
function mdBold() {
  replaceSelection('**', '**', '加粗文字')
}
function mdItalic() {
  replaceSelection('*', '*', '斜体文字')
}
function mdCode() {
  replaceSelection('`', '`', 'code')
}
function mdCodeBlock() {
  blockPrefix('```\n')
  editingContent.value += '\n```'
  schedulePreview()
}
function mdLink() {
  replaceSelection('[', '](https://)', '链接文字')
}
function mdList() {
  blockPrefix('- ')
}
function mdQuote() {
  blockPrefix('> ')
}

function resolveUploadPath(url: string) {
  try {
    return new URL(url, window.location.origin).pathname
  } catch {
    return url
  }
}

async function handleImageUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try {
    const resp = await uploadFile(file)
    if (!resp?.id) {
      ElMessage.error('上传失败')
      return
    }
    // 图片同步登记为附件
    try {
      await createAttachment({
        document_id: currentDocument.value!.id,
        file_id: resp.id,
        file_name: resp.file_name || file.name,
        file_size: resp.file_size ?? file.size,
        mime_type: resp.mime_type || file.type,
        file_url: resp.url || ''
      })
      await loadAttachments()
    } catch {
      /* 附件登记失败不阻断插入 */
    }
    const path = resolveUploadPath(resp.url || '')
    editingContent.value += `\n![${file.name || 'image'}](${path})\n`
    schedulePreview()
    ElMessage.success('图片已插入')
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

// ============ 节点新建 / 重命名 / 删除 ============
const createVisible = ref(false)
const createForm = reactive({ type: 'document' as 'folder' | 'document', title: '' })
const createParent = ref<WikiNode | null>(null)
const submitting = ref(false)
const renameVisible = ref(false)
const renameTarget = ref<WikiNode | null>(null)
const renameTitle = ref('')

function openCreateNode(type: 'folder' | 'document', parent?: WikiNode) {
  createForm.type = type
  createForm.title = ''
  createParent.value = parent ?? null
  createVisible.value = true
}

async function submitCreateNode() {
  if (!createForm.title.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  submitting.value = true
  try {
    const node = await createNode(spaceId, {
      type: createForm.type,
      title: createForm.title.trim(),
      parent_id: createParent.value?.id || undefined
    })
    createVisible.value = false
    ElMessage.success(createForm.type === 'document' ? '文档已创建' : '目录已创建')
    await reloadTree()
    // 定位并打开新建节点
    const fresh = findNodeById(tree.value, node.id)
    if (fresh) {
      const path = collectAncestors(tree.value, node.id)
      if (path) expandedKeys.value = path.slice(0, -1)
      selectedNodeId.value = node.id
      await selectNode(fresh)
    }
  } finally {
    submitting.value = false
  }
}

function openRename(node: WikiNode) {
  renameTarget.value = node
  renameTitle.value = node.title
  renameVisible.value = true
}

async function submitRename() {
  if (!renameTarget.value || !renameTitle.value.trim()) return
  submitting.value = true
  try {
    await updateNode(spaceId, renameTarget.value.id, { title: renameTitle.value.trim() })
    renameVisible.value = false
    ElMessage.success('重命名成功')
    await reloadTree()
    if (renameTarget.value.id === selectedNodeId.value) {
      currentNode.value = findNodeById(tree.value, selectedNodeId.value)
      if (currentDocument.value) currentDocument.value.title = renameTitle.value.trim()
    }
  } finally {
    submitting.value = false
  }
}

async function removeNode(node: WikiNode) {
  try {
    await ElMessageBox.confirm(
      node.type === 'folder'
        ? `目录 "${node.title}" 及其下所有内容将移入回收站，确定删除吗？`
        : `文档 "${node.title}" 将移入回收站，确定删除吗？`,
      '删除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await deleteNode(spaceId, node.id)
    ElMessage.success('已移入回收站')
    if (selectedNodeId.value === node.id) {
      currentNode.value = null
      currentDocument.value = null
      editing.value = false
      selectedNodeId.value = ''
    }
    await reloadTree()
  } catch {
    /* 拦截器已提示 */
  }
}

async function handleDeleteSpace() {
  if (!spaceInfo.value) return
  try {
    await ElMessageBox.confirm(
      `知识库 "${spaceInfo.value.name}" 及其全部内容将移入回收站（保留 30 天），确定删除吗？`,
      '删除空间',
      { type: 'error' }
    )
  } catch {
    return
  }
  try {
    await deleteSpace(spaceId)
    ElMessage.success('已删除并移入回收站')
    router.replace('/wiki')
  } catch {
    /* 拦截器已提示 */
  }
}

// ============ 成员 / 节点权限 ============
const membersDialogVisible = ref(false)
const nodePermVisible = ref(false)
const permNodeId = ref('')
const permNodeTitle = ref('')

function openNodePermission(node: WikiNode) {
  permNodeId.value = node.id
  permNodeTitle.value = node.title
  nodePermVisible.value = true
}

// ============ 扩展功能 ============
const tagManagerRef = ref<InstanceType<typeof TagManager>>()
const shareManagerRef = ref<InstanceType<typeof ShareLinkManager>>()
const templateSelectorRef = ref<InstanceType<typeof TemplateSelector>>()
const showStats = ref(false)
const documentTags = ref<any[]>([])
const editLocked = ref(false)

// 加载文档标签
async function loadDocumentTags() {
  if (!currentDocument.value) return
  try {
    const { listDocumentTags } = await import('@/api/modules/wiki')
    documentTags.value = await listDocumentTags(currentDocument.value.id)
  } catch (error) {
    console.error('加载标签失败:', error)
  }
}

// 移除文档标签
async function handleRemoveTag(tagId: string) {
  if (!currentDocument.value) return
  try {
    const { removeDocumentTag } = await import('@/api/modules/wiki')
    await removeDocumentTag(currentDocument.value.id, tagId)
    ElMessage.success('标签已移除')
    await loadDocumentTags()
  } catch (error) {
    ElMessage.error('移除标签失败')
  }
}

// 处理模板选择
function handleTemplateSelect(template: any) {
  if (!editing.value) {
    startEditing()
  }
  editingContent.value = template.content
  ElMessage.success(`已使用模板: ${template.name}`)
}

// 编辑锁状态变化
function handleEditLocked() {
  editLocked.value = true
}

function handleEditUnlocked() {
  editLocked.value = false
}

// ============ 附件 ============
const activePanel = ref<'attachments' | 'history' | 'tags' | 'shares'>('attachments')
const attachments = ref<WikiAttachment[]>([])

async function loadAttachments() {
  if (!currentDocument.value) return
  attachments.value = (await listAttachments(currentDocument.value.id).catch(() => [])) ?? []
}

async function handleAttachmentAdd(file: File) {
  if (!currentDocument.value || !file) return
  uploading.value = true
  try {
    const resp = await uploadFile(file)
    await createAttachment({
      document_id: currentDocument.value.id,
      file_id: resp.id,
      file_name: resp.file_name || file.name,
      file_size: resp.file_size ?? file.size,
      mime_type: resp.mime_type || file.type,
      file_url: resp.url || ''
    })
    ElMessage.success('附件已上传')
    await loadAttachments()
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

async function removeAttachment(a: WikiAttachment) {
  try {
    await ElMessageBox.confirm(`确定删除附件 "${a.file_name}" 吗？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteAttachment(a.id)
    ElMessage.success('附件已删除')
    await loadAttachments()
  } catch {
    /* 拦截器已提示 */
  }
}

async function downloadAttachment(a: WikiAttachment) {
  if (!a.file_id) {
    // 无文件索引时退化为直接访问签名 URL
    window.open(a.file_url, '_blank')
    return
  }
  try {
    const blob = await downloadFile(a.file_id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = a.file_name
    link.click()
    URL.revokeObjectURL(url)
  } catch {
    /* 拦截器已提示 */
  }
}

// 图片附件在线预览
const attachmentPreviewVisible = ref(false)
const attachmentPreviewUrl = ref('')
const currentAttachment = ref<WikiAttachment | null>(null)

async function previewAttachment(a: WikiAttachment) {
  const isImage = a.mime_type?.startsWith('image/')
  if (isImage && a.file_id) {
    try {
      const blob = await downloadFile(a.file_id)
      if (attachmentPreviewUrl.value) URL.revokeObjectURL(attachmentPreviewUrl.value)
      attachmentPreviewUrl.value = URL.createObjectURL(blob)
      currentAttachment.value = a
      attachmentPreviewVisible.value = true
      return
    } catch {
      /* 失败退化为下载 */
    }
  }
  await downloadAttachment(a)
}

function formatSize(size?: number) {
  if (!size && size !== 0) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

// ============ 历史版本 ============
const revisions = ref<DocumentRevision[]>([])
const loadingRevisions = ref(false)
const revisionDialogVisible = ref(false)
const revisionPreviewHtml = ref('')

async function loadRevisions() {
  if (!currentDocument.value) return
  loadingRevisions.value = true
  try {
    revisions.value = (await listRevisions(currentDocument.value.id)) ?? []
  } catch {
    revisions.value = []
  } finally {
    loadingRevisions.value = false
  }
}

async function viewRevision(row: DocumentRevision) {
  revisionPreviewHtml.value = row.content_html || '<p class="empty-tip">（此版本无正文内容）</p>'
  revisionDialogVisible.value = true
}

async function restoreVersion(row: DocumentRevision) {
  try {
    await ElMessageBox.confirm(`确定将文档恢复到 v${row.version} 吗？将覆盖当前内容。`, '版本恢复', {
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await restoreRevision(currentDocument.value!.id, row.version)
    ElMessage.success('已恢复到该版本')
    await loadDocument(currentDocument.value!.node_id)
    await loadRevisions()
  } catch {
    /* 拦截器已提示 */
  }
}

// ============ 版本对比 ============
const diffDialogVisible = ref(false)
const diffVersion1 = ref<number>(0)
const diffVersion2 = ref<number>(0)
const diffHtml = ref('')
const loadingDiff = ref(false)
const selectedRevisions = ref<DocumentRevision[]>([])

function handleRevisionSelection(selection: DocumentRevision[]) {
  selectedRevisions.value = selection
}

function openDiffDialog() {
  if (revisions.value.length < 2) {
    ElMessage.warning('至少需要两个版本才能进行对比')
    return
  }
  diffVersion1.value = revisions.value[revisions.value.length - 1].version
  diffVersion2.value = revisions.value[0].version
  diffHtml.value = ''
  diffDialogVisible.value = true
}

async function loadDiff() {
  if (!currentDocument.value || !diffVersion1.value || !diffVersion2.value) {
    ElMessage.warning('请选择两个版本进行对比')
    return
  }
  loadingDiff.value = true
  try {
    const v1 = Math.min(diffVersion1.value, diffVersion2.value)
    const v2 = Math.max(diffVersion1.value, diffVersion2.value)
    const diff = await compareRevisions(currentDocument.value.id, v1, v2)
    diffHtml.value = renderDiffHtml(diff)
  } catch {
    ElMessage.error('获取版本对比失败')
    diffHtml.value = ''
  } finally {
    loadingDiff.value = false
  }
}

function renderDiffHtml(diff: any): string {
  if (!diff || !diff.diffs) return ''
  let html = '<div class="diff-container">'
  html += `<div class="diff-header">对比版本：v${diff.old_version} → v${diff.new_version}</div>`
  html += '<div class="diff-content">'
  
  for (const line of diff.diffs) {
    const lineClass = line.type === 'added' ? 'diff-added' : line.type === 'removed' ? 'diff-removed' : 'diff-unchanged'
    const lineNum = line.type === 'added' ? line.new_line : line.type === 'removed' ? line.old_line : line.line_num
    html += `<div class="${lineClass}">`
    html += `<span class="diff-line-num">${lineNum}</span>`
    html += `<span class="diff-line-content">${escapeHtml(line.content)}</span>`
    html += '</div>'
  }
  
  html += '</div></div>'
  return html
}

function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

// ============ 导入导出 ============
async function exportDocument() {
  if (!currentDocument.value) return
  try {
    const blob = await exportDocumentApi(currentDocument.value.id, 'markdown')
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${currentNode.value?.title || 'document'}.md`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    ElMessage.success('文档导出成功')
  } catch {
    ElMessage.error('文档导出失败')
  }
}

async function handleImportFile(file: any) {
  if (!currentDocument.value || !file.raw) return
  
  try {
    await ElMessageBox.confirm(
      `确定要导入文件 "${file.name}" 吗？这将覆盖当前文档内容。`,
      '导入文档',
      { type: 'warning' }
    )
  } catch {
    return
  }
  
  try {
    await importDocumentApi(currentDocument.value.id, file.raw)
    ElMessage.success('文档导入成功')
    await loadDocument(currentDocument.value.node_id)
  } catch {
    ElMessage.error('文档导入失败')
  }
}

// ============ 初始化 ============
onMounted(async () => {
  await loadSpace()
  await reloadTree()
  const initialNodeId = (route.query.node_id as string) || ''
  if (initialNodeId) {
    const node = findNodeById(tree.value, initialNodeId)
    if (node) {
      const path = collectAncestors(tree.value, initialNodeId)
      if (path) expandedKeys.value = path.slice(0, -1)
      await selectNode(node)
    }
  } else if (tree.value.length > 0) {
    // 默认展开第一级
    expandedKeys.value = tree.value.filter((n) => n.children?.length).map((n) => n.id)
  }
})
</script>

<style scoped>
.space-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 96px);
  gap: 10px;
}
.topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 4px;
}
.space-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.space-title {
  font-size: 17px;
  font-weight: 600;
}
.space-desc {
  color: #9ca3af;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 320px;
}
.flex-1 {
  flex: 1;
}
.workspace {
  display: flex;
  flex: 1;
  min-height: 0;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}
.sidebar {
  width: 260px;
  border-right: 1px solid #eef0f3;
  padding: 10px;
  overflow: auto;
  background: #fafbfc;
}
.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
}
.sidebar-head .title {
  font-weight: 600;
}
.tree-actions {
  display: flex;
  gap: 2px;
}
.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  overflow: hidden;
}
.node-icon {
  flex-shrink: 0;
  color: #c0a04c;
}
.node-icon.active {
  color: #409eff;
}
.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.node-ops {
  display: none;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
}
.tree-node:hover .node-ops {
  display: inline-flex;
}
.doc-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 16px 20px;
  overflow: hidden;
}
.doc-toolbar,
.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.doc-title {
  font-size: 20px;
  font-weight: 700;
}
.doc-tags {
  display: flex;
  align-items: center;
  gap: 4px;
}
.unsaved-tip {
  margin-right: auto;
}
.editor-toolbar {
  border-bottom: 1px solid #f0f1f3;
  padding-bottom: 10px;
}
.mode-switch {
  display: flex;
}
.md-tools {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}
.hidden-input {
  display: none;
}
.dirty-dot {
  color: #e6a23c;
  font-size: 12px;
}
.editor-body {
  display: flex;
  flex: 1;
  min-height: 0;
  gap: 10px;
}
.md-input {
  flex: 1;
  min-width: 0;
  resize: none;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px;
  font-family: 'JetBrains Mono', Consolas, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.7;
  outline: none;
}
.md-input:focus {
  border-color: #409eff;
}
.md-preview {
  flex: 1;
  min-width: 0;
  overflow: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 16px;
}
.doc-body {
  flex: 1;
  overflow: auto;
  border: 1px solid #f0f1f3;
  border-radius: 8px;
  padding: 12px 18px;
  background: #fff;
}
.sub-panels {
  border-top: 1px solid #f0f1f3;
  margin-top: 12px;
  padding-top: 10px;
  max-height: 280px;
  overflow: auto;
}
.panel-tabs {
  display: flex;
  gap: 16px;
  margin-bottom: 8px;
}
.panel-tabs .tab {
  cursor: pointer;
  color: #6b7280;
  font-size: 13px;
  padding-bottom: 4px;
}
.panel-tabs .tab.active {
  color: #409eff;
  font-weight: 600;
  border-bottom: 2px solid #409eff;
}
.panel-body {
  min-height: 60px;
}
.attach-ops {
  margin-bottom: 8px;
}
.attach-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
}
.attach-item {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid #f0f1f3;
  border-radius: 6px;
  padding: 6px 10px;
  background: #fafbfc;
}
.attach-icon {
  color: #909399;
}
.attach-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.attach-size {
  color: #9ca3af;
  font-size: 12px;
}
.preview-img-wrap {
  display: flex;
  justify-content: center;
}
.preview-img-wrap img {
  max-width: 100%;
  max-height: 70vh;
}

/* 版本对比样式 */
.diff-container {
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}
.diff-header {
  padding: 12px 16px;
  background: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
  font-weight: 500;
  color: #374151;
}
.diff-content {
  max-height: 60vh;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
}
.diff-added {
  background: #f0fff4;
  color: #22863a;
}
.diff-removed {
  background: #ffeef0;
  color: #cb2431;
}
.diff-unchanged {
  background: #fff;
  color: #586069;
}
.diff-line-num {
  display: inline-block;
  width: 50px;
  text-align: right;
  padding-right: 12px;
  color: #999;
  user-select: none;
  border-right: 1px solid #e5e7eb;
  margin-right: 12px;
}
.diff-line-content {
  white-space: pre-wrap;
  word-break: break-all;
}
</style>

<style>
/* 文章/预览通用排版（非 scoped，供 v-html 使用） */
.article {
  font-size: 14px;
  line-height: 1.8;
  color: #1f2937;
  word-break: break-word;
}
.article h1,
.article h2,
.article h3,
.article h4 {
  margin: 1em 0 0.5em;
  font-weight: 600;
  line-height: 1.4;
}
.article h1 {
  font-size: 22px;
}
.article h2 {
  font-size: 19px;
  padding-bottom: 6px;
  border-bottom: 1px solid #eef0f3;
}
.article h3 {
  font-size: 16px;
}
.article p {
  margin: 0.6em 0;
}
.article a {
  color: #409eff;
  text-decoration: none;
}
.article a:hover {
  text-decoration: underline;
}
.article blockquote {
  margin: 0.8em 0;
  padding: 4px 14px;
  color: #6b7280;
  border-left: 4px solid #e5e7eb;
  background: #f9fafb;
}
.article code {
  background: #f3f4f6;
  color: #d97706;
  padding: 2px 5px;
  border-radius: 4px;
  font-size: 13px;
  font-family: Consolas, 'Courier New', monospace;
}
.article pre {
  background: #1f2937;
  color: #e5e7eb;
  padding: 12px 14px;
  border-radius: 8px;
  overflow: auto;
}
.article pre code {
  background: transparent;
  color: inherit;
  padding: 0;
}
.article img {
  max-width: 100%;
  border-radius: 6px;
}
.article table {
  border-collapse: collapse;
  width: 100%;
}
.article th,
.article td {
  border: 1px solid #e5e7eb;
  padding: 6px 10px;
}
.article ul,
.article ol {
  padding-left: 22px;
}
.article li {
  margin: 4px 0;
}
.article hr {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 16px 0;
}
.article .empty-tip {
  color: #9ca3af;
}
</style>