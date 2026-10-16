<template>
  <div class="page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
      title="配置即时生效，无需重启服务"
      description="在此新增或修改第三方登录提供商后，登录页按钮会立即更新（最多 30 秒内同步至全部副本）。client_secret 以 AES-256-GCM 加密后存储，页面仅展示脱敏值。"
    />

    <el-card shadow="never">
      <div class="toolbar">
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>新增登录方式</el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="display_name" label="登录方式" min-width="130">
          <template #default="{ row }">
            <span class="provider-name" :style="{ color: providerColor(row.code) }">{{ row.display_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="标识" min-width="120">
          <template #default="{ row }">
            <el-tag effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="kind" label="类型" width="150">
          <template #default="{ row }">
            {{ kindLabel(row.kind) }}
          </template>
        </el-table-column>
        <el-table-column prop="client_id" label="Client ID" min-width="160" show-overflow-tooltip />
        <el-table-column label="Client Secret" min-width="140">
          <template #default="{ row }">
            <span v-if="row.has_secret" class="secret-mask">{{ row.client_secret_mask }}</span>
            <el-tag v-else type="warning" size="small" effect="plain">未设置</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="redirect_url" label="回调地址" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '已启用' : '已停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link :type="row.enabled ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.enabled ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && list.length === 0" description="尚未配置任何第三方登录方式，点击右上角「新增登录方式」开始配置" />
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增第三方登录方式' : `编辑 ${form.code}`"
      width="620px"
      :close-on-click-modal="false"
    >
      <el-alert
        v-if="!hasSecretKey"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
        title="服务端未配置加密主密钥（oauth.secret_key）"
        description="此时无法保存 client_secret。请先在服务端配置 METEORX_OAUTH_SECRET_KEY（openssl rand -base64 32）后重启，再回来配置。"
      />
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="类型" prop="kind">
          <el-select v-model="form.kind" :disabled="dialogMode === 'edit'" style="width: 100%" @change="onKindChange">
            <el-option v-for="opt in providerKindOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="标识" prop="code">
          <el-input v-model="form.code" :disabled="dialogMode === 'edit'" placeholder="如 wechat / oidc_acme" />
          <div class="field-tip">
            内置类型的标识固定不可改；通用 OIDC 需形如 <code>oidc_&lt;企业名&gt;</code>，最终作为登录回调路径的一部分。
          </div>
        </el-form-item>
        <el-form-item label="展示名称" prop="display_name">
          <el-input v-model="form.display_name" placeholder="留空则使用内置名称（如 微信）" />
        </el-form-item>
        <el-form-item label="Client ID" prop="client_id">
          <el-input v-model="form.client_id" placeholder="开放平台应用的 AppID / Client ID" />
        </el-form-item>
        <el-form-item label="Client Secret" prop="client_secret">
          <el-input
            v-model="form.client_secret"
            type="password"
            show-password
            :placeholder="dialogMode === 'edit' ? '留空表示保持原值不变' : '开放平台应用的 AppSecret'"
          />
        </el-form-item>
        <el-form-item label="回调地址" prop="redirect_url">
          <el-input v-model="form.redirect_url" placeholder="如 http://localhost:5173/auth/wechat/callback" />
        </el-form-item>

        <template v-if="form.kind === 'microsoft'">
          <el-form-item label="目录租户" prop="tenant">
            <el-input v-model="form.tenant" placeholder="common / organizations / 目录(租户) ID" />
            <div class="field-tip">多租户应用填 <code>common</code>，仅本组织填 <code>organizations</code> 或具体租户 ID。</div>
          </el-form-item>
        </template>

        <template v-if="form.kind === 'oidc'">
          <el-form-item label="授权端点" prop="auth_url">
            <el-input v-model="form.auth_url" placeholder="https://idp.example.com/oauth2/authorize" />
          </el-form-item>
          <el-form-item label="令牌端点" prop="token_url">
            <el-input v-model="form.token_url" placeholder="https://idp.example.com/oauth2/token" />
          </el-form-item>
          <el-form-item label="用户信息端点" prop="userinfo_url">
            <el-input v-model="form.userinfo_url" placeholder="https://idp.example.com/oauth2/userinfo" />
          </el-form-item>
          <el-form-item label="授权范围" prop="scopes">
            <el-input v-model="scopesText" placeholder="openid email profile（空格或逗号分隔）" />
          </el-form-item>
        </template>

        <el-form-item label="启用" prop="enabled">
          <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          <div class="field-tip">停用后登录页不再展示该按钮，已绑定用户仍可正常登录。</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  listOAuthProviderConfigs,
  createOAuthProviderConfig,
  updateOAuthProviderConfig,
  deleteOAuthProviderConfig,
  setOAuthProviderStatus,
  providerKindOptions,
  type OAuthProviderItem,
  type OAuthProviderKind,
  type OAuthProviderSaveParams
} from '@/api/modules/oauth-provider'
import { providerColor } from '@/providers'

const loading = ref(false)
const list = ref<OAuthProviderItem[]>([])
// 服务端是否配置了加密主密钥：通过一次空保存探测不可行，改为读配置页提示由用户确认。
// 后端在未配置主密钥时会返回 422，前端据此提示。
const hasSecretKey = ref(true)

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const submitting = ref(false)
const formRef = ref<FormInstance>()
const scopesText = ref('')

const emptyForm = (): OAuthProviderSaveParams => ({
  kind: 'wechat',
  code: '',
  display_name: '',
  enabled: true,
  client_id: '',
  client_secret: '',
  redirect_url: '',
  tenant: '',
  auth_url: '',
  token_url: '',
  userinfo_url: '',
  scopes: []
})

const form = reactive<OAuthProviderSaveParams>(emptyForm())

const rules = reactive<FormRules>({
  kind: [{ required: true, message: '请选择类型', trigger: 'change' }],
  code: [{ required: true, message: '请输入标识', trigger: 'blur' }],
  client_id: [{ required: true, message: '请输入 Client ID', trigger: 'blur' }],
  redirect_url: [{ required: true, message: '请输入回调地址', trigger: 'blur' }],
  auth_url: [{ required: true, message: '请输入授权端点', trigger: 'blur' }],
  token_url: [{ required: true, message: '请输入令牌端点', trigger: 'blur' }],
  userinfo_url: [{ required: true, message: '请输入用户信息端点', trigger: 'blur' }]
})

function kindLabel(kind: string) {
  return providerKindOptions.find((o) => o.value === kind)?.label ?? kind
}

function resetForm(next: Partial<OAuthProviderSaveParams> = {}) {
  Object.assign(form, emptyForm(), next)
  scopesText.value = (next.scopes ?? []).join(' ')
}

function onKindChange(kind: OAuthProviderKind) {
  // 内置类型自动填充固定标识与默认回调地址，减少手工输入
  if (kind !== 'oidc') {
    form.code = kind
    if (!form.redirect_url) {
      form.redirect_url = `${window.location.origin}/auth/${kind}/callback`
    }
  } else if (!form.code) {
    form.code = ''
  }
}

async function loadList() {
  loading.value = true
  try {
    const res = await listOAuthProviderConfigs()
    const data = (res as { data?: typeof res }).data || res
    list.value = data?.providers || []
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  dialogMode.value = 'create'
  hasSecretKey.value = true
  resetForm()
  dialogVisible.value = true
}

function openEditDialog(row: OAuthProviderItem) {
  dialogMode.value = 'edit'
  hasSecretKey.value = true
  resetForm({
    kind: row.kind,
    code: row.code,
    display_name: row.display_name,
    enabled: row.enabled,
    client_id: row.client_id,
    client_secret: '',
    redirect_url: row.redirect_url,
    tenant: row.tenant ?? '',
    auth_url: row.auth_url ?? '',
    token_url: row.token_url ?? '',
    userinfo_url: row.userinfo_url ?? '',
    scopes: row.scopes ?? []
  })
  dialogVisible.value = true
}

function parseScopes(): string[] {
  return scopesText.value
    .split(/[\s,;]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      const payload: OAuthProviderSaveParams = {
        ...form,
        scopes: form.kind === 'oidc' ? parseScopes() : undefined
      }
      if (dialogMode.value === 'create') {
        await createOAuthProviderConfig(payload)
        ElMessage.success('新增成功，配置已即时生效')
      } else {
        await updateOAuthProviderConfig(form.code, payload)
        ElMessage.success('保存成功，配置已即时生效')
      }
      dialogVisible.value = false
      await loadList()
    } catch (e) {
      const err = e as { response?: { status?: number }; message?: string }
      if (err.response?.status === 422) {
        hasSecretKey.value = false
        ElMessage.error('服务端未配置加密主密钥，无法保存 client_secret')
      } else {
        ElMessage.error(err.message || '保存失败')
      }
    } finally {
      submitting.value = false
    }
  })
}

async function handleToggleStatus(row: OAuthProviderItem) {
  const next = !row.enabled
  try {
    await setOAuthProviderStatus(row.code, next)
    ElMessage.success(next ? '已启用' : '已停用')
    await loadList()
  } catch (e) {
    ElMessage.error((e as Error)?.message || '操作失败')
  }
}

async function handleDelete(row: OAuthProviderItem) {
  try {
    await ElMessageBox.confirm(`确定删除「${row.display_name}」的第三方登录配置吗？已绑定该账号的用户将无法再通过它登录。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      confirmButtonClass: 'el-button--danger'
    })
  } catch {
    return
  }

  try {
    await deleteOAuthProviderConfig(row.code)
    ElMessage.success('已删除')
    await loadList()
  } catch (e) {
    ElMessage.error((e as Error)?.message || '删除失败')
  }
}

onMounted(loadList)
</script>

<style scoped>
.page {
  padding: 0;
}
.toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.flex-1 {
  flex: 1;
}
.provider-name {
  font-weight: 600;
}
.secret-mask {
  font-family: 'Courier New', Courier, monospace;
  color: #909399;
}
.field-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}
.field-tip code {
  background: #f3f4f6;
  padding: 1px 4px;
  border-radius: 3px;
}
</style>
