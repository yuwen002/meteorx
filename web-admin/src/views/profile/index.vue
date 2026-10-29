<e></e><template>
  <div class="page">
    <el-row :gutter="20">
      <!-- 个人信息卡片 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>{{ t('profile.title') }}</span>
              <el-button type="primary" @click="openEditDialog">
                <el-icon><Edit /></el-icon>{{ t('profile.editProfile') }}
              </el-button>
            </div>
          </template>
          <div v-loading="loading" class="profile-info">
            <div class="info-item">
              <label>{{ t('profile.info.userId') }}</label>
              <span>{{ userInfo.id }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.username') }}</label>
              <span>{{ userInfo.username }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.nickname') }}</label>
              <span>{{ userInfo.nickname || '-' }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.email') }}</label>
              <span>{{ userInfo.email || '-' }}</span>
              <el-tag v-if="userInfo.email" :type="userInfo.email_verified ? 'success' : 'warning'" size="small" style="margin-left: 8px;">
                {{ userInfo.email_verified ? t('profile.info.verified') : t('profile.info.unverified') }}
              </el-tag>
              <el-button
                v-if="userInfo.email && !userInfo.email_verified"
                type="primary"
                link
                size="small"
                :loading="sendVerifyLoading"
                style="margin-left: 4px;"
                @click="handleSendEmailVerification"
              >
                {{ t('profile.info.sendVerifyEmail') }}
              </el-button>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.phone') }}</label>
              <span>{{ userInfo.phone || '-' }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.department') }}</label>
              <span>{{ userInfo.department || '-' }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.position') }}</label>
              <span>{{ userInfo.position || '-' }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.tenantId') }}</label>
              <el-tag v-if="userInfo.tenant_id" size="small" type="info">{{ userInfo.tenant_id }}</el-tag>
              <span v-else>-</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.status') }}</label>
              <el-tag :type="userInfo.status === 1 ? 'success' : 'info'">
                {{ userInfo.status === 1 ? t('common.status.enabled') : t('common.status.disabled') }}
              </el-tag>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.master') }}</label>
              <el-tag v-if="userInfo.is_master" type="danger" size="small">MASTER</el-tag>
              <span v-else style="color: #9ca3af">-</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.lastLogin') }}</label>
              <span>{{ userInfo.last_login_at || t('profile.info.neverLogin') }}</span>
            </div>
            <div class="info-item">
              <label>{{ t('profile.info.createdAt') }}</label>
              <span>{{ userInfo.created_at }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 修改密码卡片 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>{{ t('profile.changePassword') }}</span>
            </div>
          </template>
          <el-form
            ref="pwdFormRef"
            :model="pwdForm"
            :rules="pwdRules"
            label-width="100px"
            style="max-width: 400px;"
          >
            <el-form-item :label="t('auth.oldPassword')" prop="old_password">
              <el-input
                v-model="pwdForm.old_password"
                type="password"
                show-password
                :placeholder="t('profile.pwd.oldPlaceholder')"
              />
            </el-form-item>
            <el-form-item :label="t('auth.newPassword')" prop="new_password">
              <el-input
                v-model="pwdForm.new_password"
                type="password"
                show-password
                :placeholder="t('profile.pwd.newPlaceholder')"
              />
            </el-form-item>
            <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
              <el-input
                v-model="pwdForm.confirm_password"
                type="password"
                show-password
                :placeholder="t('profile.pwd.confirmPlaceholder')"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="pwdLoading" @click="submitPassword">
                {{ t('profile.changePassword') }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 我的角色卡片 -->
        <el-card shadow="never" style="margin-top: 20px;">
          <template #header>
            <div class="card-header">
              <span>{{ t('profile.myRoles') }}</span>
            </div>
          </template>
          <div v-loading="rolesLoading">
            <el-tag
              v-for="role in userRoles"
              :key="role.id"
              type="primary"
              style="margin-right: 8px; margin-bottom: 8px;"
            >
              {{ role.name }}
            </el-tag>
            <el-empty v-if="userRoles.length === 0" :description="t('profile.roles.empty')" :image-size="60" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- OAuth 第三方账号管理卡片 -->
    <el-card shadow="never" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>{{ t('profile.oauth.title') }}</span>
          <el-button type="primary" @click="bindOAuthDialogVisible = true">
            <el-icon><Link /></el-icon>{{ t('profile.oauth.bind') }}
          </el-button>
        </div>
      </template>
      <el-table v-loading="oauthLoading" :data="oauthAccounts" stripe style="width: 100%;">
        <el-table-column :label="t('profile.oauth.provider')" min-width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="email" :label="t('profile.oauth.email')" min-width="200" />
        <el-table-column prop="created_at" :label="t('profile.oauth.boundAt')" min-width="160" />
        <el-table-column :label="t('common.actions')" width="100" align="center">
          <template #default="{ row }">
            <el-button type="danger" link size="small" @click="handleUnbindOAuth(row)">{{ t('profile.oauth.unbind') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!oauthLoading && oauthAccounts.length === 0" :description="t('profile.oauth.empty')" :image-size="60" />
    </el-card>

    <!-- 绑定第三方账号弹窗 -->
    <el-dialog v-model="bindOAuthDialogVisible" :title="t('profile.oauth.bindDialogTitle')" width="400px">
      <p style="color: #606266; margin-bottom: 16px;">{{ t('profile.oauth.bindHint') }}</p>
      <div class="oauth-bind-buttons">
        <div
          v-for="p in oauthProviders"
          :key="p.id"
          class="oauth-bind-btn"
          :style="{ borderColor: providerColor(p.id), color: providerColor(p.id) }"
          @click="handleBindOAuth(p.id)"
        >
          <el-icon size="20"><Connection /></el-icon>
          <span>{{ p.label }}</span>
        </div>
      </div>
      <div v-if="bindOAuthLoading" style="text-align: center; margin-top: 12px;">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span style="margin-left: 4px; color: #909399;">{{ t('profile.oauth.redirecting') }}</span>
      </div>
    </el-dialog>

    <!-- API Token 管理卡片 -->
    <el-card shadow="never" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>{{ t('profile.token.title') }}</span>
          <el-button type="primary" @click="openCreateTokenDialog">
            <el-icon><Plus /></el-icon>{{ t('profile.token.create') }}
          </el-button>
        </div>
      </template>
      <el-table v-loading="tokensLoading" :data="apiTokens" stripe style="width: 100%">
        <el-table-column prop="name" :label="t('common.name')" min-width="120" />
        <el-table-column :label="t('profile.token.allowedPaths')" min-width="200">
          <template #default="{ row }">
            <template v-if="row.allowed_paths && row.allowed_paths.length > 0">
              <el-tag v-for="p in row.allowed_paths" :key="p" size="small" type="info" style="margin-right: 4px; margin-bottom: 2px;">{{ p }}</el-tag>
            </template>
            <span v-else style="color: #9ca3af;">{{ t('profile.token.unlimited') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('common.createdAt')" min-width="160" />
        <el-table-column :label="t('profile.token.expiresAt')" min-width="160">
          <template #default="{ row }">
            {{ row.expires_at || t('profile.token.neverExpire') }}
          </template>
        </el-table-column>
        <el-table-column :label="t('profile.token.lastUsed')" min-width="160">
          <template #default="{ row }">
            {{ row.last_used_at || t('profile.token.neverUsed') }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.status')" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.revoked" type="info" size="small">{{ t('profile.token.revoked') }}</el-tag>
            <el-tag v-else type="success" size="small">{{ t('profile.token.active') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="100" align="center">
          <template #default="{ row }">
            <el-button
              v-if="!row.revoked"
              type="danger"
              link
              size="small"
              @click="handleRevokeToken(row)"
            >
              {{ t('profile.token.revoke') }}
            </el-button>
            <span v-else style="color: #c0c4cc; font-size: 12px;">—</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!tokensLoading && apiTokens.length === 0" :description="t('profile.token.empty')" :image-size="60" />
    </el-card>

    <!-- 创建 API Token 弹窗 -->
    <el-dialog
      v-model="createTokenDialogVisible"
      :title="t('profile.token.createDialogTitle')"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-alert
        v-if="newTokenValue"
        :title="t('profile.token.copyWarning')"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 16px;"
      />
      <div v-if="newTokenValue" class="token-display">
        <div class="token-value-row">
          <el-input :model-value="newTokenValue" readonly type="textarea" :rows="2" class="token-input" />
          <el-button
            :type="tokenCopied ? 'success' : 'primary'"
            :icon="tokenCopied ? Check : CopyDocument"
            style="margin-top: 8px; min-width: 120px;"
            @click="copyToken"
          >
            {{ tokenCopied ? t('profile.token.copied') : t('profile.token.copy') }}
          </el-button>
        </div>
        <div class="token-hint">
          <el-icon><InfoFilled /></el-icon>
          <span>{{ t('profile.token.saveHint') }}</span>
        </div>
      </div>
      <el-form
        v-else
        ref="createTokenFormRef"
        :model="createTokenForm"
        :rules="createTokenRules"
        label-width="80px"
      >
        <el-form-item :label="t('common.name')" prop="name">
          <el-input v-model="createTokenForm.name" :placeholder="t('profile.token.namePlaceholder')" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('profile.token.expires')" prop="expires_in">
          <el-select v-model="createTokenForm.expires_in" :placeholder="t('profile.token.expiresPlaceholder')" clearable>
            <el-option :label="t('profile.token.days', 30)" value="720h" />
            <el-option :label="t('profile.token.days', 60)" value="1440h" />
            <el-option :label="t('profile.token.days', 90)" value="2160h" />
            <el-option :label="t('profile.token.days', 180)" value="4320h" />
            <el-option :label="t('profile.token.oneYear')" value="8760h" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('profile.token.allowedPaths')" prop="allowed_paths">
          <el-select
            v-model="createTokenForm.allowed_paths"
            multiple
            filterable
            allow-create
            default-first-option
            :placeholder="t('profile.token.pathsPlaceholder')"
            style="width: 100%;"
          >
            <el-option label="/users" value="/users" />
            <el-option label="/files" value="/files" />
            <el-option label="/wiki" value="/wiki" />
            <el-option label="/dashboard" value="/dashboard" />
            <el-option label="/audit" value="/audit" />
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">{{ t('profile.token.pathsHint') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeCreateTokenDialog">{{ t('common.close') }}</el-button>
        <el-button v-if="!newTokenValue" type="primary" :loading="createTokenLoading" @click="submitCreateToken">{{ t('profile.token.submit') }}</el-button>
      </template>
    </el-dialog>

    <!-- 双因素认证卡片 -->
    <el-card shadow="never" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>{{ t('profile.totp.title') }}</span>
          <div>
            <el-button v-if="userInfo.totp_enabled" type="primary" link @click="openRegenRecovery">{{ t('profile.totp.regenBtn') }}</el-button>
            <el-button v-if="!userInfo.totp_enabled" type="primary" @click="open2FASetup">{{ t('profile.totp.enable') }}</el-button>
            <el-button v-else type="danger" @click="open2FADisable">{{ t('profile.totp.disableBtn') }}</el-button>
          </div>
        </div>
      </template>
      <div>
        <el-tag :type="userInfo.totp_enabled ? 'success' : 'info'" size="small">
          {{ userInfo.totp_enabled ? t('profile.totp.enabled') : t('profile.totp.disabled') }}
        </el-tag>
        <p style="color:#6b7280;font-size:13px;margin:12px 0 0;">
          {{ t('profile.totp.desc') }}
        </p>
      </div>
    </el-card>

    <!-- 2FA 绑定弹窗 -->
    <el-dialog v-model="twoFASetupVisible" :title="t('profile.totp.setupTitle')" width="480px" :close-on-click-modal="false">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom:16px;">
        <template #title>{{ t('profile.totp.setupHint') }}</template>
      </el-alert>
      <div class="info-item">
        <label>{{ t('profile.totp.secret') }}</label>
        <span class="totp-secret">{{ twoFASecret }}</span>
        <el-button link type="primary" size="small" @click="copy2FASecret">{{ t('common.copy') }}</el-button>
      </div>
      <div class="info-item" style="margin-top:8px;">
        <label>{{ t('profile.totp.setupUrl') }}</label>
        <span style="word-break:break-all;color:#909399;font-size:12px;">{{ twoFAUrl }}</span>
      </div>
      <el-form style="margin-top:16px;">
        <el-form-item :label="t('profile.totp.code')">
          <el-input v-model="twoFACode" :placeholder="t('profile.totp.codePlaceholder')" maxlength="6" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="twoFASetupVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="twoFAEnableLoading" @click="submitEnable2FA">{{ t('profile.totp.confirmEnable') }}</el-button>
      </template>
    </el-dialog>

    <!-- 2FA 关闭弹窗 -->
    <el-dialog v-model="twoFADisableVisible" :title="t('profile.totp.disableTitle')" width="400px">
      <p style="color:#606266;">{{ t('profile.totp.disableHint') }}</p>
      <el-input v-model="twoFADisableCode" :placeholder="t('profile.totp.codePlaceholder')" maxlength="6" style="margin-top:12px;" />
      <template #footer>
        <el-button @click="twoFADisableVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="danger" :loading="twoFADisableLoading" @click="submitDisable2FA">{{ t('profile.totp.confirmDisable') }}</el-button>
      </template>
    </el-dialog>

    <!-- 备份恢复码展示弹窗（一次性明文） -->
    <el-dialog v-model="recoveryVisible" :title="t('profile.totp.recoveryTitle')" width="480px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom:16px;">
        <template #title>{{ t('profile.totp.recoveryOnceHint') }}</template>
      </el-alert>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <code v-for="c in recoveryCodes" :key="c" style="padding:6px 10px;background:#f5f7fa;border-radius:4px;text-align:center;letter-spacing:1px;">{{ c }}</code>
      </div>
      <template #footer>
        <el-button @click="copyAllRecoveryCodes">{{ t('profile.totp.recoveryCopyAll') }}</el-button>
        <el-button type="primary" @click="recoveryVisible = false">{{ t('profile.totp.recoveryClose') }}</el-button>
      </template>
    </el-dialog>

    <!-- 重新生成恢复码弹窗 -->
    <el-dialog v-model="regenDialogVisible" :title="t('profile.totp.regenTitle')" width="400px">
      <p style="color:#606266;">{{ t('profile.totp.recoveryRegenHint') }}</p>
      <el-input v-model="regenCode" :placeholder="t('profile.totp.codePlaceholder')" maxlength="6" style="margin-top:12px;" />
      <template #footer>
        <el-button @click="regenDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="regenLoading" @click="submitRegenRecovery">{{ t('profile.totp.regenConfirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 编辑资料弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="t('profile.edit.title')"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="80px"
      >
        <el-form-item :label="t('profile.edit.nickname')" prop="nickname">
          <el-input v-model="editForm.nickname" :placeholder="t('profile.edit.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('profile.edit.email')" prop="email">
          <el-input v-model="editForm.email" :placeholder="t('profile.edit.emailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('profile.edit.phone')" prop="phone">
          <el-input v-model="editForm.phone" :placeholder="t('profile.edit.phonePlaceholder')" maxlength="20" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Edit, Plus, Link, CopyDocument, Check, InfoFilled, Connection, Loading } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getProfile, updateProfile, changePassword } from '@/api/modules/user'
import { getUserRoles } from '@/api/modules/role'
import { createAPIToken, listAPITokens, revokeAPIToken, type APITokenItem, listOAuthAccounts, unbindOAuth, type OAuthAccountItem, getOAuthRedirectURL, sendEmailVerification, setup2FA, enable2FA, disable2FA, regenerateRecoveryCodes } from '@/api/auth'
import { listOAuthProviders, type ProviderMeta } from '@/api/modules/oauth'
import { providerColor } from '@/providers'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const userStore = useUserStore()

// 用户信息
const userInfo = reactive({
  id: '',
  username: '',
  nickname: '',
  email: '',
  email_verified: false,
  phone: '',
  avatar: '',
  department: '',
  position: '',
  tenant_id: '',
  status: 1,
  is_master: false,
  totp_enabled: false,
  last_login_at: '',
  created_at: ''
})
const loading = ref(false)

// 用户角色
const userRoles = ref<{ id: string; name: string; code: string }[]>([])
const rolesLoading = ref(false)

// 编辑资料
const editDialogVisible = ref(false)
const editFormRef = ref<FormInstance>()
const editLoading = ref(false)
const editForm = reactive({
  nickname: '',
  email: '',
  phone: ''
})
const editRules = computed<FormRules>(() => ({
  email: [{ type: 'email', message: t('profile.edit.emailInvalid'), trigger: 'blur' }]
}))

// 修改密码
const pwdFormRef = ref<FormInstance>()
const pwdLoading = ref(false)
const pwdForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (_rule: unknown, value: string, callback: (error?: Error) => void) => {
  if (value !== pwdForm.new_password) {
    callback(new Error(t('profile.pwd.mismatch')))
  } else {
    callback()
  }
}

const pwdRules = computed<FormRules>(() => ({
  old_password: [{ required: true, message: t('profile.pwd.oldRequired'), trigger: 'blur' }],
  new_password: [
    { required: true, message: t('profile.pwd.newRequired'), trigger: 'blur' },
    { min: 6, max: 32, message: t('profile.pwd.lengthRange'), trigger: 'blur' },
    { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: t('profile.pwd.pattern'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('profile.pwd.confirmRequired'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}))

// 加载用户信息
async function loadUserInfo() {
  loading.value = true
  try {
    const data = await getProfile()
    Object.assign(userInfo, data)
    // 同时更新 store
    userStore.updateUserInfo(data)
  } catch (e) {
    ElMessage.error(t('profile.getInfoFailed'))
  } finally {
    loading.value = false
  }
}

// 加载用户角色
async function loadUserRoles() {
  if (!userInfo.id) return
  rolesLoading.value = true
  try {
    const data = await getUserRoles(userInfo.id)
    userRoles.value = data || []
  } catch (e) {
    console.error('failed to load user roles', e)
  } finally {
    rolesLoading.value = false
  }
}

// 双因素认证 (2FA)
const twoFASetupVisible = ref(false)
const twoFASecret = ref('')
const twoFAUrl = ref('')
const twoFACode = ref('')
const twoFAEnableLoading = ref(false)
const twoFADisableVisible = ref(false)
const twoFADisableCode = ref('')
const twoFADisableLoading = ref(false)

// 备份恢复码：展示一次性明文码 + 重新生成
const recoveryVisible = ref(false)
const recoveryCodes = ref<string[]>([])
const regenDialogVisible = ref(false)
const regenCode = ref('')
const regenLoading = ref(false)

async function open2FASetup() {
  try {
    const data = await setup2FA()
    twoFASecret.value = data.secret
    twoFAUrl.value = data.otpauth_url
    twoFACode.value = ''
    twoFASetupVisible.value = true
  } catch (e) {
    ElMessage.error(t('profile.totp.initFailed'))
  }
}

async function copy2FASecret() {
  try {
    await navigator.clipboard.writeText(twoFASecret.value)
    ElMessage.success(t('profile.totp.secretCopied'))
  } catch {
    ElMessage.warning(t('profile.totp.copyManual'))
  }
}

async function submitEnable2FA() {
  if (!/^\d{6}$/.test(twoFACode.value)) {
    ElMessage.warning(t('profile.totp.codeInvalid'))
    return
  }
  twoFAEnableLoading.value = true
  try {
    const res = await enable2FA(twoFACode.value.trim())
    ElMessage.success(t('profile.totp.enableSuccess'))
    twoFASetupVisible.value = false
    userInfo.totp_enabled = true
    // 展示一次性下发的备份恢复码
    showRecoveryCodes(res.recovery_codes || [])
  } catch (e) {
    const err = e as { response?: { data?: { message?: string } } }
    ElMessage.error(err.response?.data?.message || t('profile.totp.enableFailed'))
  } finally {
    twoFAEnableLoading.value = false
  }
}

function open2FADisable() {
  twoFADisableCode.value = ''
  twoFADisableVisible.value = true
}

async function submitDisable2FA() {
  if (!/^\d{6}$/.test(twoFADisableCode.value)) {
    ElMessage.warning(t('profile.totp.codeInvalid'))
    return
  }
  twoFADisableLoading.value = true
  try {
    await disable2FA(twoFADisableCode.value.trim())
    ElMessage.success(t('profile.totp.disableSuccess'))
    twoFADisableVisible.value = false
    userInfo.totp_enabled = false
  } catch (e) {
    const err = e as { response?: { data?: { message?: string } } }
    ElMessage.error(err.response?.data?.message || t('profile.totp.disableFailed'))
  } finally {
    twoFADisableLoading.value = false
  }
}

// 展示一次性恢复码列表
function showRecoveryCodes(codes: string[]) {
  recoveryCodes.value = codes
  recoveryVisible.value = true
}

async function copyAllRecoveryCodes() {
  try {
    await navigator.clipboard.writeText(recoveryCodes.value.join('\n'))
    ElMessage.success(t('profile.totp.recoveryCopied'))
  } catch {
    ElMessage.warning(t('profile.totp.copyManual'))
  }
}

function openRegenRecovery() {
  regenCode.value = ''
  regenDialogVisible.value = true
}

async function submitRegenRecovery() {
  if (!/^\d{6}$/.test(regenCode.value)) {
    ElMessage.warning(t('profile.totp.codeInvalid'))
    return
  }
  regenLoading.value = true
  try {
    const res = await regenerateRecoveryCodes(regenCode.value.trim())
    ElMessage.success(t('profile.totp.regenSuccess'))
    regenDialogVisible.value = false
    showRecoveryCodes(res.recovery_codes || [])
  } catch (e) {
    const err = e as { response?: { data?: { message?: string } } }
    ElMessage.error(err.response?.data?.message || t('profile.totp.regenFailed'))
  } finally {
    regenLoading.value = false
  }
}

const sendVerifyLoading = ref(false)

async function handleSendEmailVerification() {
  if (!userInfo.email) return
  sendVerifyLoading.value = true
  try {
    await sendEmailVerification(userInfo.email)
    ElMessage.success(t('profile.info.verifySent'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('profile.info.verifySendFailed'))
  } finally {
    sendVerifyLoading.value = false
  }
}

// 打开编辑弹窗
function openEditDialog() {
  editForm.nickname = userInfo.nickname || ''
  editForm.email = userInfo.email || ''
  editForm.phone = userInfo.phone || ''
  editDialogVisible.value = true
}

// 提交编辑
async function submitEdit() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    editLoading.value = true
    try {
      await updateProfile(editForm)
      ElMessage.success(t('profile.edit.updateSuccess'))
      userInfo.nickname = editForm.nickname
      userInfo.email = editForm.email
      userInfo.phone = editForm.phone
      userStore.updateUserInfo({ nickname: editForm.nickname, email: editForm.email, phone: editForm.phone })
      editDialogVisible.value = false
    } catch (e) {
      ElMessage.error((e as Error).message || t('profile.edit.updateFailed'))
    } finally {
      editLoading.value = false
    }
  })
}

// 提交修改密码
async function submitPassword() {
  if (!pwdFormRef.value) return
  await pwdFormRef.value.validate(async (valid) => {
    if (!valid) return
    pwdLoading.value = true
    try {
      await changePassword({
        old_password: pwdForm.old_password,
        new_password: pwdForm.new_password
      })
      
      ElMessage.success(t('profile.pwd.updateSuccess'))
      // 清空表单
      pwdForm.old_password = ''
      pwdForm.new_password = ''
      pwdForm.confirm_password = ''
      // 退出登录
      setTimeout(() => {
        userStore.logout()
      }, 1500)
    } catch (e) {
      ElMessage.error((e as Error).message || t('profile.pwd.updateFailed'))
    } finally {
      pwdLoading.value = false
    }
  })
}

// OAuth 第三方账号管理
const oauthAccounts = ref<OAuthAccountItem[]>([])
const oauthLoading = ref(false)

// 已启用第三方登录提供商（动态渲染绑定按钮）
const oauthProviders = ref<ProviderMeta[]>([])

async function loadOAuthProviders() {
  try {
    const res = await listOAuthProviders()
    const data = (res as { data?: typeof res }).data || res
    oauthProviders.value = data?.providers || []
  } catch {
    oauthProviders.value = []
  }
}

async function loadOAuthAccounts() {
  oauthLoading.value = true
  try {
    const data = await listOAuthAccounts()
    oauthAccounts.value = data?.accounts || []
  } catch (e) {
    console.error('failed to load oauth accounts', e)
  } finally {
    oauthLoading.value = false
  }
}

const bindOAuthDialogVisible = ref(false)
const bindOAuthLoading = ref(false)

async function handleBindOAuth(provider: string) {
  bindOAuthLoading.value = true
  try {
    sessionStorage.setItem('oauth_bind_mode', 'true')
    const res = await getOAuthRedirectURL(provider)
    const data = (res as { data?: typeof res }).data || res
    if (data?.url) {
      window.location.href = data.url
    } else {
      ElMessage.error(t('profile.oauth.getUrlFailed'))
    }
  } catch (e) {
    ElMessage.error(t('profile.oauth.getUrlRetry'))
  } finally {
    bindOAuthLoading.value = false
  }
}

async function handleUnbindOAuth(row: OAuthAccountItem) {
  try {
    await ElMessageBox.confirm(t('profile.oauth.unbindConfirm', row.provider), t('profile.oauth.unbindTitle'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await unbindOAuth(row.provider)
    ElMessage.success(t('profile.oauth.unbindSuccess'))
    await loadOAuthAccounts()
  } catch {
    // 用户取消
  }
}

// API Token 管理
const apiTokens = ref<APITokenItem[]>([])
const tokensLoading = ref(false)
const createTokenDialogVisible = ref(false)
const createTokenFormRef = ref<FormInstance>()
const createTokenLoading = ref(false)
const newTokenValue = ref('')
const createTokenForm = reactive({
  name: '',
  expires_in: '',
  allowed_paths: [] as string[]
})
const createTokenRules = computed<FormRules>(() => ({
  name: [
    { required: true, message: t('profile.token.nameRequired'), trigger: 'blur' },
    { min: 1, max: 64, message: t('profile.token.nameLength'), trigger: 'blur' }
  ]
}))

async function loadAPITokens() {
  tokensLoading.value = true
  try {
    const data = await listAPITokens()
    apiTokens.value = data?.tokens || []
  } catch (e) {
    console.error('failed to load api tokens', e)
  } finally {
    tokensLoading.value = false
  }
}

function openCreateTokenDialog() {
  createTokenForm.name = ''
  createTokenForm.expires_in = ''
  createTokenForm.allowed_paths = []
  newTokenValue.value = ''
  createTokenDialogVisible.value = true
}

function closeCreateTokenDialog() {
  createTokenDialogVisible.value = false
  newTokenValue.value = ''
  tokenCopied.value = false
}

async function submitCreateToken() {
  if (!createTokenFormRef.value) return
  await createTokenFormRef.value.validate(async (valid) => {
    if (!valid) return
    createTokenLoading.value = true
    try {
      const data = await createAPIToken({
        name: createTokenForm.name,
        expires_in: createTokenForm.expires_in || undefined,
        allowed_paths: createTokenForm.allowed_paths.length > 0 ? createTokenForm.allowed_paths : undefined
      })
      newTokenValue.value = data.token
      ElMessage.success(t('profile.token.createSuccess'))
      loadAPITokens()
    } catch (e) {
      ElMessage.error((e as Error).message || t('profile.token.createFailed'))
    } finally {
      createTokenLoading.value = false
    }
  })
}

const tokenCopied = ref(false)

function copyToken() {
  navigator.clipboard.writeText(newTokenValue.value).then(() => {
    tokenCopied.value = true
    ElMessage.success(t('profile.token.copiedClipboard'))
    setTimeout(() => { tokenCopied.value = false }, 3000)
  }).catch(() => {
    ElMessage.error(t('profile.token.copyFailed'))
  })
}

async function handleRevokeToken(row: APITokenItem) {
  try {
    await ElMessageBox.confirm(t('profile.token.revokeConfirm', row.name), t('profile.token.revokeTitle'), {
      confirmButtonText: t('profile.token.revokeOk'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await revokeAPIToken({ id: row.id })
    ElMessage.success(t('profile.token.revokeSuccess'))
    loadAPITokens()
  } catch {
    // 用户取消
  }
}

onMounted(() => {
  loadUserInfo()
  loadUserRoles()
  loadOAuthAccounts()
  loadOAuthProviders()
  loadAPITokens()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.profile-info {
  padding: 10px 0;
}

.info-item {
  display: flex;
  padding: 12px 0;
  border-bottom: 1px solid #ebeef5;
}

.info-item:last-child {
  border-bottom: none;
}

.info-item label {
  width: 100px;
  color: #606266;
  font-weight: 500;
}

.info-item span {
  flex: 1;
  color: #303133;
}

.token-display {
  margin-top: 8px;

  .token-value-row {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .token-input {
    :deep(.el-textarea__inner) {
      font-family: 'Courier New', Courier, monospace;
      font-size: 13px;
      letter-spacing: 0.5px;
    }
  }

  .token-hint {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 8px;
    font-size: 12px;
    color: #909399;
  }
}

.oauth-bind-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.oauth-bind-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 160px;
  height: 44px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
  font-size: 15px;
  font-weight: 500;
  transition: all 0.2s;

  &:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  }
}
.totp-secret {
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  letter-spacing: 1px;
  color: #303133;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
}
</style>