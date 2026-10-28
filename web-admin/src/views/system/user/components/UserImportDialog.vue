<template>
  <el-dialog
    v-model="visible"
    :title="t('userImport.title')"
    width="820px"
    :close-on-click-modal="false"
    @closed="reset"
  >
    <!-- 第一步：选择文件与默认角色 -->
    <template v-if="!result">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 14px">
        <template #title>
          {{ t('userImport.alert', MAX_ROWS) }}
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
        <div class="el-upload__text">{{ t('userImport.dragText') }}<em>{{ t('userImport.clickSelect') }}</em></div>
        <template #tip>
          <div class="el-upload__tip">{{ t('userImport.uploadTip') }}</div>
        </template>
      </el-upload>

      <el-form label-width="90px" style="margin-top: 16px">
        <!-- 系统管理员需要显式选择目标租户：角色编码在各租户间通用，但角色ID 不通用 -->
        <el-form-item v-if="adminMode" :label="t('userPage.targetTenant')" required>
          <el-select
            v-model="targetTenantId"
            filterable
            clearable
            :loading="tenantLoading"
            :placeholder="t('userImport.tenantPlaceholder')"
            style="width: 100%"
          >
            <el-option v-for="opt in tenantOptions" :key="opt.id" :label="opt.name" :value="opt.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-else :label="t('userImport.defaultRole')">
          <el-select
            v-model="roleIds"
            multiple
            filterable
            clearable
            :placeholder="t('userImport.rolePlaceholder')"
            style="width: 100%"
          >
            <el-option v-for="r in roleOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>

      <el-alert v-if="adminMode" type="warning" :closable="false" show-icon style="margin-bottom: 8px">
        <template #title>
          {{ t('userImport.crossTenantWarn') }}
        </template>
      </el-alert>

      <div class="template-row">
        <el-button link type="primary" @click="downloadTemplate">
          <el-icon><Download /></el-icon>{{ t('userImport.downloadTemplate') }}
        </el-button>
        <span class="hint-text">{{ t('userImport.passwordHint') }}</span>
      </div>
    </template>

    <!-- 第二步：导入结果 -->
    <template v-else>
      <el-result
        :icon="rowErrors.length ? 'warning' : 'success'"
        :title="rowErrors.length ? t('userImport.resultFailed') : t('userImport.resultSuccess')"
        :sub-title="summaryText"
      >
        <template #extra>
          <el-button v-if="rowErrors.length" @click="backToForm">{{ t('userImport.backToEdit') }}</el-button>
          <el-button v-else type="primary" @click="close">{{ t('userImport.done') }}</el-button>
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
        <el-table-column prop="row" :label="t('userImport.colRow')" width="80" />
        <el-table-column :label="t('userImport.colField')" width="120">
          <template #default="{ row }">{{ fieldLabel(row.field) }}</template>
        </el-table-column>
        <el-table-column prop="message" :label="t('userImport.colIssue')" min-width="280" />
      </el-table>

      <!-- 成功明细（含初始密码） -->
      <template v-else>
        <div class="result-toolbar">
          <span class="hint-text">{{ t('userImport.copyHint') }}</span>
          <el-button size="small" type="primary" plain @click="copyPasswords">{{ t('userImport.copyAll') }}</el-button>
        </div>
        <el-table :data="result.users" border stripe size="small" max-height="320">
          <el-table-column prop="row" :label="t('userImport.colRow')" width="70" />
          <el-table-column prop="username" :label="t('auth.username')" min-width="120" />
          <el-table-column prop="nickname" :label="t('register.nickname')" min-width="100" />
          <el-table-column prop="email" :label="t('common.email')" min-width="160" />
          <el-table-column :label="t('userPage.colRole')" min-width="120">
            <template #default="{ row }">
              <el-tag v-for="code in row.role_codes" :key="code" size="small" style="margin-right: 4px">{{ code }}</el-tag>
              <span v-if="!row.role_codes?.length" class="hint-text">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="initial_password" :label="t('userImport.colInitPwd')" min-width="160" />
        </el-table>
      </template>
    </template>

    <template #footer>
      <template v-if="!result">
        <el-button @click="close">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">{{ t('userImport.start') }}</el-button>
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
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

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
const FIELD_LABELS = computed<Record<string, string>>(() => ({
  username: t('auth.username'),
  nickname: t('register.nickname'),
  email: t('common.email'),
  department: t('userPage.colDepartment'),
  position: t('userImport.fieldPosition'),
  roles: t('userPage.colRole')
}))

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
    return errorMessage.value || t('userImport.summaryErrors', rowErrors.value.length)
  }
  const created = result.value?.created ?? 0
  return t('userImport.summaryCreated', created)
})

const fieldLabel = (field: string) => FIELD_LABELS.value[field] || field

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
    ElMessage.warning(t('userImport.typeUnsupported'))
    fileList.value = []
    pickedFile.value = null
    return
  }
  if (file.size && file.size > MAX_FILE_SIZE) {
    ElMessage.warning(t('userImport.sizeExceeded'))
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
  // Prepend a UTF-8 BOM so Excel opens the header row without garbled characters
  const csv = '\uFEFF' + 'username,nickname,email,department,position,roles\n' + 'alice,Alice Li,alice@example.com,Product,Product Manager,member\n'
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = t('userImport.templateFile') + '.csv'
  link.click()
  URL.revokeObjectURL(url)
}

async function submit() {
  if (!pickedFile.value) {
    ElMessage.warning(t('userImport.pickFileFirst'))
    return
  }
  if (props.adminMode && !targetTenantId.value) {
    ElMessage.warning(t('userPage.selectTargetTenantFirst'))
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
      ElMessage.error(data?.message || t('userImport.failed'))
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
    .then(() => ElMessage.success(t('userImport.copied')))
    .catch(() => ElMessage.warning(t('userImport.copyFailed')))
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
