<template>
  <el-dialog
    v-model="visible"
    title="批量导入用户"
    width="820px"
    :close-on-click-modal="false"
    @closed="reset"
  >
    <!-- 第一步：选择文件与默认角色 -->
    <template v-if="!result">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 14px">
        <template #title>
          支持 CSV / XLSX，单次最多 {{ MAX_ROWS }} 行。表头必须包含用户名与昵称，可选邮箱、部门、职位、角色编码（多个用逗号分隔）。
          任一行校验不通过时整批都不会写入。
        </template>
      </el-alert>

      <el-upload
        drag
        :auto-upload="false"
        :limit="1"
        :file-list="fileList"
        accept=".csv,.xlsx"
        :on-change="onFileChange"
        :on-remove="onFileRemove"
        :on-exceed="onExceed"
      >
        <el-icon class="upload-icon"><UploadFilled /></el-icon>
        <div class="el-upload__text">把文件拖到这里，或<em>点击选择</em></div>
        <template #tip>
          <div class="el-upload__tip">仅支持 .csv / .xlsx，大小不超过 5MB</div>
        </template>
      </el-upload>

      <el-form label-width="90px" style="margin-top: 16px">
        <!-- 系统管理员需要显式选择目标租户：角色编码在各租户间通用，但角色ID 不通用 -->
        <el-form-item v-if="adminMode" label="目标租户" required>
          <el-select
            v-model="targetTenantId"
            filterable
            clearable
            :loading="tenantLoading"
            placeholder="选择导入到哪个租户"
            style="width: 100%"
          >
            <el-option v-for="t in tenantOptions" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-else label="默认角色">
          <el-select
            v-model="roleIds"
            multiple
            filterable
            clearable
            placeholder="文件 roles 列留空时使用（可选）"
            style="width: 100%"
          >
            <el-option v-for="r in roleOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>

      <el-alert v-if="adminMode" type="warning" :closable="false" show-icon style="margin-bottom: 8px">
        <template #title>
          跨租户导入不提供默认角色：请在文件 roles 列填写角色编码（如 member），服务端会按目标租户解析。
        </template>
      </el-alert>

      <div class="template-row">
        <el-button link type="primary" @click="downloadTemplate">
          <el-icon><Download /></el-icon>下载 CSV 模板
        </el-button>
        <span class="hint-text">初始密码由服务端随机生成，导入成功后仅展示一次</span>
      </div>
    </template>

    <!-- 第二步：导入结果 -->
    <template v-else>
      <el-result
        :icon="rowErrors.length ? 'warning' : 'success'"
        :title="rowErrors.length ? '导入未通过校验' : '导入成功'"
        :sub-title="summaryText"
      >
        <template #extra>
          <el-button v-if="rowErrors.length" @click="backToForm">返回修改</el-button>
          <el-button v-else type="primary" @click="close">完成</el-button>
        </template>
      </el-result>

      <el-alert
        v-for="(warn, idx) in warnings"
        :key="idx"
        :title="warn"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 8px"
      />

      <!-- 行级错误 -->
      <el-table v-if="rowErrors.length" :data="rowErrors" border stripe size="small" max-height="320">
        <el-table-column prop="row" label="行号" width="80" />
        <el-table-column label="列" width="120">
          <template #default="{ row }">{{ fieldLabel(row.field) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="问题" min-width="280" />
      </el-table>

      <!-- 成功明细（含初始密码） -->
      <template v-else>
        <div class="result-toolbar">
          <span class="hint-text">请复制初始密码后通过安全渠道下发给用户</span>
          <el-button size="small" type="primary" plain @click="copyPasswords">复制全部账号密码</el-button>
        </div>
        <el-table :data="result.users" border stripe size="small" max-height="320">
          <el-table-column prop="row" label="行号" width="70" />
          <el-table-column prop="username" label="用户名" min-width="120" />
          <el-table-column prop="nickname" label="昵称" min-width="100" />
          <el-table-column prop="email" label="邮箱" min-width="160" />
          <el-table-column label="角色" min-width="120">
            <template #default="{ row }">
              <el-tag v-for="code in row.role_codes" :key="code" size="small" style="margin-right: 4px">{{ code }}</el-tag>
              <span v-if="!row.role_codes?.length" class="hint-text">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="initial_password" label="初始密码" min-width="160" />
        </el-table>
      </template>
    </template>

    <template #footer>
      <template v-if="!result">
        <el-button @click="close">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">开始导入</el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { UploadFilled, Download } from '@element-plus/icons-vue'
import type { UploadFile, UploadUserFile } from 'element-plus'
import { importUsers, type ImportRowError, type ImportUsersResult } from '@/api/modules/user'
import { getRoleListForSelect, type RoleItem } from '@/api/modules/role'
import { getTenantList, type TenantItem } from '@/api/modules/tenant'

const props = defineProps<{
  modelValue: boolean
  // 传入租户ID 时走管理端端点（系统管理员为指定租户导入）
  tenantId?: string
  // 系统管理员模式：弹窗内自行选择目标租户
  adminMode?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  imported: []
}>()

// 与服务端 importer.MaxRows 保持一致，超限的文件在服务端也会被拒绝
const MAX_ROWS = 500
const MAX_FILE_SIZE = 5 * 1024 * 1024

// 服务端回传的字段名 → 表格列名，便于操作者对应到文件中的列
const FIELD_LABELS: Record<string, string> = {
  username: '用户名',
  nickname: '昵称',
  email: '邮箱',
  department: '部门',
  position: '职位',
  roles: '角色'
}

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const fileList = ref<UploadUserFile[]>([])
const pickedFile = ref<File | null>(null)
const roleIds = ref<string[]>([])
const roleOptions = ref<RoleItem[]>([])
const roleLoaded = ref(false)
const targetTenantId = ref('')
const tenantOptions = ref<TenantItem[]>([])
const tenantLoaded = ref(false)
const tenantLoading = ref(false)

const submitting = ref(false)
const result = ref<ImportUsersResult | null>(null)
const rowErrors = ref<ImportRowError[]>([])
const errorMessage = ref('')

const warnings = computed(() => result.value?.warnings ?? [])

const summaryText = computed(() => {
  if (rowErrors.value.length) {
    return errorMessage.value || `共 ${rowErrors.value.length} 处问题，未写入任何用户`
  }
  const created = result.value?.created ?? 0
  return `已创建 ${created} 个用户`
})

const fieldLabel = (field: string) => FIELD_LABELS[field] || field

async function loadRoleOptions() {
  if (roleLoaded.value) return
  roleLoaded.value = true
  try {
    roleOptions.value = (await getRoleListForSelect('tenant')) ?? []
  } catch {
    // 角色列表拉取失败不阻塞导入：文件里已写 roles 列时仍然可用
    roleLoaded.value = false
  }
}

async function loadTenantOptions() {
  if (tenantLoaded.value || !props.adminMode) return
  tenantLoaded.value = true
  tenantLoading.value = true
  try {
    const res = await getTenantList({ page: 1, page_size: 200 })
    // 只保留启用中的租户，停用租户无法登录也没有导入意义
    tenantOptions.value = (res?.data ?? []).filter((t) => t.status === 1)
  } catch {
    tenantLoaded.value = false
  } finally {
    tenantLoading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    if (props.adminMode) loadTenantOptions()
    else loadRoleOptions()
  }
)

function onFileChange(file: UploadFile) {
  if (!file.raw) return
  if (!/\.(csv|xlsx)$/i.test(file.name)) {
    ElMessage.warning('仅支持 .csv 或 .xlsx 文件')
    fileList.value = []
    pickedFile.value = null
    return
  }
  if (file.size && file.size > MAX_FILE_SIZE) {
    ElMessage.warning('文件大小不能超过 5MB')
    fileList.value = []
    pickedFile.value = null
    return
  }
  pickedFile.value = file.raw
  fileList.value = [file]
}

function onFileRemove() {
  pickedFile.value = null
  fileList.value = []
}

function onExceed(files: File[]) {
  // 只允许一个文件：再次选择时直接替换，而不是报错
  const next = files[0]
  if (!next) return
  onFileChange({ name: next.name, size: next.size, raw: next } as UploadFile)
}

function downloadTemplate() {
  // 前置 UTF-8 BOM，避免 Excel 直接打开中文表头出现乱码
  const csv = '\uFEFF' + 'username,nickname,email,department,position,roles\n' + 'alice,张三,alice@example.com,产品部,产品经理,member\n'
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = '用户导入模板.csv'
  link.click()
  URL.revokeObjectURL(url)
}

async function submit() {
  if (!pickedFile.value) {
    ElMessage.warning('请先选择要导入的文件')
    return
  }
  if (props.adminMode && !targetTenantId.value) {
    ElMessage.warning('请先选择目标租户')
    return
  }
  // 跨租户导入不使用默认角色（角色ID 不跨租户通用，故按角色编码在目标租户内解析）
  const tenantId = props.adminMode ? targetTenantId.value : props.tenantId
  submitting.value = true
  rowErrors.value = []
  errorMessage.value = ''
  try {
    const res = await importUsers({
      file: pickedFile.value,
      roleIds: props.adminMode ? [] : roleIds.value,
      tenantId
    })
    result.value = res
    emit('imported')
  } catch (err: any) {
    // 后端把逐行明细放在 details 里；没有 details 时退回通用错误文案
    const data = err?.response?.data
    if (Array.isArray(data?.details) && data.details.length) {
      rowErrors.value = data.details as ImportRowError[]
      errorMessage.value = data.message || ''
      result.value = { total: 0, created: 0, users: [] }
    } else {
      ElMessage.error(data?.message || '导入失败')
    }
  } finally {
    submitting.value = false
  }
}

function copyPasswords() {
  const users = result.value?.users ?? []
  if (!users.length) return
  const text = users.map((u) => `${u.username}\t${u.initial_password}`).join('\n')
  navigator.clipboard
    ?.writeText(text)
    .then(() => ElMessage.success('已复制到剪贴板'))
    .catch(() => ElMessage.warning('复制失败，请手动选择表格内容'))
}

function backToForm() {
  result.value = null
  rowErrors.value = []
  errorMessage.value = ''
}

function close() {
  visible.value = false
}

function reset() {
  fileList.value = []
  pickedFile.value = null
  roleIds.value = []
  targetTenantId.value = ''
  result.value = null
  rowErrors.value = []
  errorMessage.value = ''
  submitting.value = false
}
</script>

<style scoped>
.upload-icon {
  font-size: 48px;
  color: #c0c4cc;
  margin-bottom: 8px;
}

.template-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.result-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.hint-text {
  color: #909399;
  font-size: 12px;
}
</style>
