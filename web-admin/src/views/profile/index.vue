<e></e><template>
  <div class="page">
    <el-row :gutter="20">
      <!-- 个人信息卡片 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>个人信息</span>
              <el-button type="primary" @click="openEditDialog">
                <el-icon><Edit /></el-icon>编辑资料
              </el-button>
            </div>
          </template>
          <div v-loading="loading" class="profile-info">
            <div class="info-item">
              <label>用户ID：</label>
              <span>{{ userInfo.id }}</span>
            </div>
            <div class="info-item">
              <label>用户名：</label>
              <span>{{ userInfo.username }}</span>
            </div>
            <div class="info-item">
              <label>昵称：</label>
              <span>{{ userInfo.nickname || '-' }}</span>
            </div>
            <div class="info-item">
              <label>邮箱：</label>
              <span>{{ userInfo.email || '-' }}</span>
              <el-tag v-if="userInfo.email" :type="userInfo.email_verified ? 'success' : 'warning'" size="small" style="margin-left: 8px;">
                {{ userInfo.email_verified ? '已验证' : '未验证' }}
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
                发送验证邮件
              </el-button>
            </div>
            <div class="info-item">
              <label>手机号：</label>
              <span>{{ userInfo.phone || '-' }}</span>
            </div>
            <div class="info-item">
              <label>租户ID：</label>
              <el-tag v-if="userInfo.tenant_id" size="small" type="info">{{ userInfo.tenant_id }}</el-tag>
              <span v-else>-</span>
            </div>
            <div class="info-item">
              <label>状态：</label>
              <el-tag :type="userInfo.status === 1 ? 'success' : 'info'">
                {{ userInfo.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </div>
            <div class="info-item">
              <label>主管理员：</label>
              <el-tag v-if="userInfo.is_master" type="danger" size="small">MASTER</el-tag>
              <span v-else style="color: #9ca3af">-</span>
            </div>
            <div class="info-item">
              <label>创建时间：</label>
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
              <span>修改密码</span>
            </div>
          </template>
          <el-form
            ref="pwdFormRef"
            :model="pwdForm"
            :rules="pwdRules"
            label-width="100px"
            style="max-width: 400px;"
          >
            <el-form-item label="原密码" prop="old_password">
              <el-input
                v-model="pwdForm.old_password"
                type="password"
                show-password
                placeholder="请输入原密码"
              />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input
                v-model="pwdForm.new_password"
                type="password"
                show-password
                placeholder="6-32位，包含字母和数字"
              />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirm_password">
              <el-input
                v-model="pwdForm.confirm_password"
                type="password"
                show-password
                placeholder="请再次输入新密码"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="pwdLoading" @click="submitPassword">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 我的角色卡片 -->
        <el-card shadow="never" style="margin-top: 20px;">
          <template #header>
            <div class="card-header">
              <span>我的角色</span>
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
            <el-empty v-if="userRoles.length === 0" description="暂无角色" :image-size="60" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- OAuth 第三方账号管理卡片 -->
    <el-card shadow="never" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>第三方账号绑定</span>
          <el-button type="primary" @click="bindOAuthDialogVisible = true">
            <el-icon><Link /></el-icon>绑定账号
          </el-button>
        </div>
      </template>
      <el-table v-loading="oauthLoading" :data="oauthAccounts" stripe style="width: 100%;">
        <el-table-column label="提供商" min-width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" min-width="200" />
        <el-table-column prop="created_at" label="绑定时间" min-width="160" />
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button type="danger" link size="small" @click="handleUnbindOAuth(row)">解绑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!oauthLoading && oauthAccounts.length === 0" description="暂无绑定的第三方账号" :image-size="60" />
    </el-card>

    <!-- 绑定第三方账号弹窗 -->
    <el-dialog v-model="bindOAuthDialogVisible" title="绑定第三方账号" width="400px">
      <p style="color: #606266; margin-bottom: 16px;">选择要绑定的第三方账号，将跳转至对应平台进行授权。</p>
      <div class="oauth-bind-buttons">
        <div class="oauth-bind-btn oauth-google" @click="handleBindOAuth('google')">
          <el-icon size="20"><Connection /></el-icon>
          <span>Google</span>
        </div>
        <div class="oauth-bind-btn oauth-github" @click="handleBindOAuth('github')">
          <el-icon size="20"><Connection /></el-icon>
          <span>GitHub</span>
        </div>
      </div>
      <div v-if="bindOAuthLoading" style="text-align: center; margin-top: 12px;">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span style="margin-left: 4px; color: #909399;">正在跳转...</span>
      </div>
    </el-dialog>

    <!-- API Token 管理卡片 -->
    <el-card shadow="never" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>API 令牌</span>
          <el-button type="primary" @click="openCreateTokenDialog">
            <el-icon><Plus /></el-icon>创建令牌
          </el-button>
        </div>
      </template>
      <el-table v-loading="tokensLoading" :data="apiTokens" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="可访问路径" min-width="200">
          <template #default="{ row }">
            <template v-if="row.allowed_paths && row.allowed_paths.length > 0">
              <el-tag v-for="p in row.allowed_paths" :key="p" size="small" type="info" style="margin-right: 4px; margin-bottom: 2px;">{{ p }}</el-tag>
            </template>
            <span v-else style="color: #9ca3af;">不限制</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="160" />
        <el-table-column label="过期时间" min-width="160">
          <template #default="{ row }">
            {{ row.expires_at || '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column label="最后使用" min-width="160">
          <template #default="{ row }">
            {{ row.last_used_at || '从未使用' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.revoked" type="info" size="small">已撤销</el-tag>
            <el-tag v-else type="success" size="small">有效</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button
              v-if="!row.revoked"
              type="danger"
              link
              size="small"
              @click="handleRevokeToken(row)"
            >
              撤销
            </el-button>
            <span v-else style="color: #c0c4cc; font-size: 12px;">—</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!tokensLoading && apiTokens.length === 0" description="暂无 API 令牌" :image-size="60" />
    </el-card>

    <!-- 创建 API Token 弹窗 -->
    <el-dialog
      v-model="createTokenDialogVisible"
      title="创建 API 令牌"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-alert
        v-if="newTokenValue"
        title="请立即复制令牌，关闭后将无法再次查看！"
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
            {{ tokenCopied ? '已复制' : '复制令牌' }}
          </el-button>
        </div>
        <div class="token-hint">
          <el-icon><InfoFilled /></el-icon>
          <span>建议将令牌保存到密码管理器中，如 1Password / Bitwarden 等</span>
        </div>
      </div>
      <el-form
        v-else
        ref="createTokenFormRef"
        :model="createTokenForm"
        :rules="createTokenRules"
        label-width="80px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="createTokenForm.name" placeholder="如：CI/CD 部署令牌" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="有效期" prop="expires_in">
          <el-select v-model="createTokenForm.expires_in" placeholder="选择有效期（可选）" clearable>
            <el-option label="30 天" value="720h" />
            <el-option label="60 天" value="1440h" />
            <el-option label="90 天" value="2160h" />
            <el-option label="180 天" value="4320h" />
            <el-option label="1 年" value="8760h" />
          </el-select>
        </el-form-item>
        <el-form-item label="可访问路径" prop="allowed_paths">
          <el-select
            v-model="createTokenForm.allowed_paths"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入路径前缀后回车添加，留空则不限制"
            style="width: 100%;"
          >
            <el-option label="/users" value="/users" />
            <el-option label="/files" value="/files" />
            <el-option label="/wiki" value="/wiki" />
            <el-option label="/dashboard" value="/dashboard" />
            <el-option label="/audit" value="/audit" />
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">路径前缀匹配，如 /files 可访问 /files/upload 等；不填则拥有默认权限</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeCreateTokenDialog">关闭</el-button>
        <el-button v-if="!newTokenValue" type="primary" :loading="createTokenLoading" @click="submitCreateToken">创建</el-button>
      </template>
    </el-dialog>

    <!-- 编辑资料弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑个人资料"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="80px"
      >
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="editForm.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="editForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="editForm.phone" placeholder="请输入手机号" maxlength="20" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Edit, Plus, Link, CopyDocument, Check, InfoFilled, Connection, Loading } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getProfile, updateProfile, changePassword } from '@/api/modules/user'
import { getUserRoles } from '@/api/modules/role'
import { createAPIToken, listAPITokens, revokeAPIToken, type APITokenItem, listOAuthAccounts, unbindOAuth, type OAuthAccountItem, getOAuthRedirectURL, sendEmailVerification } from '@/api/auth'

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
  tenant_id: '',
  status: 1,
  is_master: false,
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
const editRules: FormRules = {
  email: [{ type: 'email', message: '请输入正确的邮箱', trigger: 'blur' }]
}

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
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const pwdRules: FormRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' },
    { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: '密码必须包含字母和数字', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户信息
async function loadUserInfo() {
  loading.value = true
  try {
    const data = await getProfile()
    Object.assign(userInfo, data)
    // 同时更新 store
    userStore.updateUserInfo(data)
  } catch (e) {
    ElMessage.error('获取用户信息失败')
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
    console.error('加载角色失败', e)
  } finally {
    rolesLoading.value = false
  }
}

const sendVerifyLoading = ref(false)

async function handleSendEmailVerification() {
  if (!userInfo.email) return
  sendVerifyLoading.value = true
  try {
    await sendEmailVerification(userInfo.email)
    ElMessage.success('验证邮件已发送，请检查您的邮箱')
  } catch (e) {
    ElMessage.error((e as Error).message || '发送验证邮件失败')
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
      ElMessage.success('更新成功')
      userInfo.nickname = editForm.nickname
      userInfo.email = editForm.email
      userInfo.phone = editForm.phone
      userStore.updateUserInfo({ nickname: editForm.nickname, email: editForm.email, phone: editForm.phone })
      editDialogVisible.value = false
    } catch (e) {
      ElMessage.error((e as Error).message || '更新失败')
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
      
      ElMessage.success('密码修改成功，请重新登录')
      // 清空表单
      pwdForm.old_password = ''
      pwdForm.new_password = ''
      pwdForm.confirm_password = ''
      // 退出登录
      setTimeout(() => {
        userStore.logout()
      }, 1500)
    } catch (e) {
      ElMessage.error((e as Error).message || '密码修改失败')
    } finally {
      pwdLoading.value = false
    }
  })
}

// OAuth 第三方账号管理
const oauthAccounts = ref<OAuthAccountItem[]>([])
const oauthLoading = ref(false)

async function loadOAuthAccounts() {
  oauthLoading.value = true
  try {
    const data = await listOAuthAccounts()
    oauthAccounts.value = data?.accounts || []
  } catch (e) {
    console.error('加载第三方账号失败', e)
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
      ElMessage.error('获取授权链接失败')
    }
  } catch (e) {
    ElMessage.error('获取授权链接失败，请稍后重试')
  } finally {
    bindOAuthLoading.value = false
  }
}

async function handleUnbindOAuth(row: OAuthAccountItem) {
  try {
    await ElMessageBox.confirm(`确定要解绑 ${row.provider} 账号吗？`, '解绑确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await unbindOAuth(row.provider)
    ElMessage.success('解绑成功')
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
const createTokenRules: FormRules = {
  name: [
    { required: true, message: '请输入令牌名称', trigger: 'blur' },
    { min: 1, max: 64, message: '名称长度 1-64 位', trigger: 'blur' }
  ]
}

async function loadAPITokens() {
  tokensLoading.value = true
  try {
    const data = await listAPITokens()
    apiTokens.value = data?.tokens || []
  } catch (e) {
    console.error('加载 API 令牌失败', e)
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
      ElMessage.success('令牌创建成功')
      loadAPITokens()
    } catch (e) {
      ElMessage.error((e as Error).message || '创建令牌失败')
    } finally {
      createTokenLoading.value = false
    }
  })
}

const tokenCopied = ref(false)

function copyToken() {
  navigator.clipboard.writeText(newTokenValue.value).then(() => {
    tokenCopied.value = true
    ElMessage.success('已复制到剪贴板')
    setTimeout(() => { tokenCopied.value = false }, 3000)
  }).catch(() => {
    ElMessage.error('复制失败，请手动复制')
  })
}

async function handleRevokeToken(row: APITokenItem) {
  try {
    await ElMessageBox.confirm(`确定撤销令牌「${row.name}」？撤销后使用该令牌的请求将被拒绝。`, '撤销确认', {
      confirmButtonText: '确定撤销',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await revokeAPIToken({ id: row.id })
    ElMessage.success('令牌已撤销')
    loadAPITokens()
  } catch {
    // 用户取消
  }
}

onMounted(() => {
  loadUserInfo()
  loadUserRoles()
  loadOAuthAccounts()
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
  border-radius: 8px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 500;
  transition: all 0.2s;

  &.oauth-google {
    background: #fff;
    border: 1px solid #d9d9d9;
    color: #333;
    &:hover { border-color: #40a9ff; box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15); }
  }

  &.oauth-github {
    background: #24292e;
    border: 1px solid #24292e;
    color: #fff;
    &:hover { background: #2f363d; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3); }
  }
}
</style>