<template>
  <div class="page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
      :title="t('oauthPage.alertTitle')"
      :description="t('oauthPage.alertDesc')"
    />

    <el-card shadow="never">
      <div class="toolbar">
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>{{ t('oauthPage.create') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="display_name" :label="t('oauthPage.col.provider')" min-width="130">
          <template #default="{ row }">
            <span class="provider-name" :style="{ color: providerColor(row.code) }">{{ row.display_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="code" :label="t('oauthPage.col.code')" min-width="120">
          <template #default="{ row }">
            <el-tag effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="kind" :label="t('oauthPage.col.kind')" width="150">
          <template #default="{ row }">
            {{ kindLabel(row.kind) }}
          </template>
        </el-table-column>
        <el-table-column prop="client_id" label="Client ID" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('oauthPage.col.clientSecret')" min-width="140">
          <template #default="{ row }">
            <span v-if="row.has_secret" class="secret-mask">{{ row.client_secret_mask }}</span>
            <el-tag v-else type="warning" size="small" effect="plain">{{ t('oauthPage.notSet') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="redirect_url" :label="t('oauthPage.col.redirectUrl')" min-width="220" show-overflow-tooltip />
        <el-table-column :label="t('oauthPage.col.status')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? t('oauthPage.status.enabled') : t('oauthPage.status.disabled') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" :label="t('oauthPage.col.updatedAt')" width="170" />
        <el-table-column :label="t('oauthPage.col.operation')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">{{ t('oauthPage.action.edit') }}</el-button>
            <el-button link :type="row.enabled ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.enabled ? t('oauthPage.action.disable') : t('oauthPage.action.enable') }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ t('oauthPage.action.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && list.length === 0" :description="t('oauthPage.empty')" />
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? t('oauthPage.dialog.createTitle') : t('oauthPage.dialog.editTitle', form.code)"
      width="620px"
      :close-on-click-modal="false"
    >
      <el-alert
        v-if="!hasSecretKey"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
        :title="t('oauthPage.noSecretTitle')"
        :description="t('oauthPage.noSecretDesc')"
      />
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item :label="t('oauthPage.form.kindLabel')" prop="kind">
          <el-select v-model="form.kind" :disabled="dialogMode === 'edit'" style="width: 100%" @change="onKindChange">
            <el-option v-for="opt in providerKindOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('oauthPage.form.codeLabel')" prop="code">
          <el-input v-model="form.code" :disabled="dialogMode === 'edit'" :placeholder="t('oauthPage.form.codePlaceholder')" />
          <div class="field-tip" v-html="t('oauthPage.form.codeTip')"></div>
        </el-form-item>
        <el-form-item :label="t('oauthPage.form.displayNameLabel')" prop="display_name">
          <el-input v-model="form.display_name" :placeholder="t('oauthPage.form.displayNamePlaceholder')" />
        </el-form-item>
        <el-form-item label="Client ID" prop="client_id">
          <el-input v-model="form.client_id" :placeholder="t('oauthPage.form.clientIdPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('oauthPage.col.clientSecret')" prop="client_secret">
          <el-input
            v-model="form.client_secret"
            type="password"
            show-password
            :placeholder="dialogMode === 'edit' ? t('oauthPage.form.secretEditPlaceholder') : t('oauthPage.form.secretCreatePlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('oauthPage.form.redirectUrlLabel')" prop="redirect_url">
          <el-input v-model="form.redirect_url" :placeholder="t('oauthPage.form.redirectUrlPlaceholder')" />
        </el-form-item>

        <template v-if="form.kind === 'microsoft'">
          <el-form-item :label="t('oauthPage.form.tenantLabel')" prop="tenant">
            <el-input v-model="form.tenant" :placeholder="t('oauthPage.form.tenantPlaceholder')" />
            <div class="field-tip" v-html="t('oauthPage.form.tenantTip')"></div>
          </el-form-item>
        </template>

        <template v-if="form.kind === 'oidc'">
          <el-form-item :label="t('oauthPage.form.authUrlLabel')" prop="auth_url">
            <el-input v-model="form.auth_url" placeholder="https://idp.example.com/oauth2/authorize" />
          </el-form-item>
          <el-form-item :label="t('oauthPage.form.tokenUrlLabel')" prop="token_url">
            <el-input v-model="form.token_url" placeholder="https://idp.example.com/oauth2/token" />
          </el-form-item>
          <el-form-item :label="t('oauthPage.form.userinfoUrlLabel')" prop="userinfo_url">
            <el-input v-model="form.userinfo_url" placeholder="https://idp.example.com/oauth2/userinfo" />
          </el-form-item>
          <el-form-item :label="t('oauthPage.form.scopesLabel')" prop="scopes">
            <el-input v-model="scopesText" :placeholder="t('oauthPage.form.scopesPlaceholder')" />
          </el-form-item>
        </template>

        <el-form-item :label="t('oauthPage.form.enabledLabel')" prop="enabled">
          <el-switch v-model="form.enabled" :active-text="t('oauthPage.status.enabled')" :inactive-text="t('oauthPage.status.disabled')" />
          <div class="field-tip">{{ t('oauthPage.form.enabledTip') }}</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useI18n } from '@/composables/useI18n'
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

const { t } = useI18n()

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

const rules = computed<FormRules>(() => ({
  kind: [{ required: true, message: t('oauthPage.form.kindRequired'), trigger: 'change' }],
  code: [{ required: true, message: t('oauthPage.form.codeRequired'), trigger: 'blur' }],
  client_id: [{ required: true, message: t('oauthPage.form.clientIdRequired'), trigger: 'blur' }],
  redirect_url: [{ required: true, message: t('oauthPage.form.redirectUrlRequired'), trigger: 'blur' }],
  auth_url: [{ required: true, message: t('oauthPage.form.authUrlRequired'), trigger: 'blur' }],
  token_url: [{ required: true, message: t('oauthPage.form.tokenUrlRequired'), trigger: 'blur' }],
  userinfo_url: [{ required: true, message: t('oauthPage.form.userinfoUrlRequired'), trigger: 'blur' }]
}))

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
        ElMessage.success(t('oauthPage.createdSuccess'))
      } else {
        await updateOAuthProviderConfig(form.code, payload)
        ElMessage.success(t('oauthPage.savedSuccess'))
      }
      dialogVisible.value = false
      await loadList()
    } catch (e) {
      const err = e as { response?: { status?: number }; message?: string }
      if (err.response?.status === 422) {
        hasSecretKey.value = false
        ElMessage.error(t('oauthPage.noSecretError'))
      } else {
        ElMessage.error(err.message || t('oauthPage.saveFailed'))
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
    ElMessage.success(next ? t('oauthPage.status.enabled') : t('oauthPage.status.disabled'))
    await loadList()
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('oauthPage.opFailed'))
  }
}

async function handleDelete(row: OAuthProviderItem) {
  try {
    await ElMessageBox.confirm(t('oauthPage.deleteConfirm', row.display_name), t('oauthPage.deleteTitle'), {
      type: 'warning',
      confirmButtonText: t('common.delete'),
      confirmButtonClass: 'el-button--danger'
    })
  } catch {
    return
  }

  try {
    await deleteOAuthProviderConfig(row.code)
    ElMessage.success(t('oauthPage.deleted'))
    await loadList()
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('oauthPage.deleteFailed'))
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
