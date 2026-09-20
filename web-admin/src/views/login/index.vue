<template>
  <div class="login-page">
    <div class="login-box">
      <div class="login-title">
        <h2>MeteorX 管理后台</h2>
        <p>{{ loginMode === 'admin' ? '系统管理员登录' : '租户用户登录' }}</p>
      </div>

      <div class="login-mode-switch">
        <div
          class="mode-tab"
          :class="{ active: loginMode === 'admin' }"
          @click="switchMode('admin')"
        >
          <el-icon><Avatar /></el-icon>
          <span>管理员</span>
        </div>
        <div
          class="mode-tab"
          :class="{ active: loginMode === 'tenant' }"
          @click="switchMode('tenant')"
        >
          <el-icon><OfficeBuilding /></el-icon>
          <span>租户</span>
        </div>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        class="login-form"
        @keyup.enter="handleLogin"
      >
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" prefix-icon="User" clearable size="large" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>
        <el-form-item v-if="loginMode === 'tenant'" label="租户 ID" prop="tenant_id">
          <el-input
            v-model="form.tenant_id"
            placeholder="请输入您的租户 ID"
            prefix-icon="OfficeBuilding"
            clearable
            size="large"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" size="large" style="width: 100%" @click="handleLogin">
            登 录
          </el-button>
        </el-form-item>
        <div v-if="loginError.show" class="login-error" :class="{ locked: loginError.locked }">
          <el-icon><Warning /></el-icon>
          <span>{{ loginError.message }}</span>
          <div v-if="!loginError.locked && loginError.remainingAttempts > 0" class="attempts-warning">
            剩余尝试次数：{{ loginError.remainingAttempts }} 次
          </div>
          <div v-if="loginError.locked" class="lockout-info">
            账号已锁定，请 {{ Math.ceil(loginError.lockoutDuration / 60) }} 分钟后重试
          </div>
        </div>
        <div class="form-footer">
          <el-link type="primary" :underline="false" @click="goForgotPassword">
            忘记密码？
          </el-link>
        </div>

        <div class="register-link">
          <span>还没有账号？</span>
          <el-dropdown trigger="click" @command="handleRegisterCommand">
            <el-link type="primary" :underline="false">
              立即注册
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-link>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="user">用户注册（加入已有租户）</el-dropdown-item>
                <el-dropdown-item command="tenant">租户注册（创建新租户）</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <!-- OAuth2 第三方登录 -->
        <div class="oauth-divider">
          <span>或</span>
        </div>
        <div class="oauth-buttons">
          <div class="oauth-btn oauth-google" @click="handleOAuthLogin('google')">
            <el-icon><Link /></el-icon>
            <span>Google 登录</span>
          </div>
          <div class="oauth-btn oauth-github" @click="handleOAuthLogin('github')">
            <el-icon><Connection /></el-icon>
            <span>GitHub 登录</span>
          </div>
        </div>

        <!-- 租户选择对话框 -->
        <el-dialog
          v-model="showTenantDialog"
          title="选择租户"
          width="400px"
          :close-on-click-modal="false"
          :close-on-press-escape="false"
        >
          <p class="dialog-tip">请选择您要登录的租户</p>
          <el-select
            v-model="selectedTenantId"
            placeholder="请选择租户"
            size="large"
            style="width: 100%"
            filterable
          >
            <el-option
              v-for="tenant in tenantList"
              :key="tenant.id"
              :label="tenant.name"
              :value="tenant.id"
            />
          </el-select>
          <template #footer>
            <el-button @click="showTenantDialog = false">取消</el-button>
            <el-button type="primary" :loading="oauthSubmitting" @click="confirmOAuthLogin">
              确认登录
            </el-button>
          </template>
        </el-dialog>

        <div class="tips">
          <el-icon><InfoFilled /></el-icon>
          {{ loginMode === 'admin' ? '默认管理员账号：admin / 123456' : '请输入租户 ID 和租户账号' }}
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { InfoFilled, Warning, Avatar, OfficeBuilding, Link, Connection, ArrowDown } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getOAuthRedirectURL, oauthLogin, getOAuthTenants, type LoginParams, type LoginErrorData } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

// OAuth 租户选择相关
const showTenantDialog = ref(false)
const oauthSubmitting = ref(false)
const tenantList = ref<Array<{ id: string; name: string }>>([])
const selectedTenantId = ref('')
const currentOAuthProvider = ref('')

type LoginMode = 'admin' | 'tenant'
const loginMode = ref<LoginMode>('admin')

const loginError = reactive<{
  show: boolean
  message: string
  remainingAttempts: number
  locked: boolean
  lockoutDuration: number
}>({
  show: false,
  message: '',
  remainingAttempts: 5,
  locked: false,
  lockoutDuration: 0
})

const form = reactive<LoginParams>({
  username: '',
  password: '',
  tenant_id: ''
})

const formRules = computed<FormRules>(() => ({
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  tenant_id: loginMode.value === 'tenant'
    ? [{ required: true, message: '请输入租户 ID', trigger: 'blur' }]
    : []
}))

function switchMode(mode: LoginMode) {
  loginMode.value = mode
  loginError.show = false
  if (mode === 'admin') {
    form.tenant_id = ''
  }
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    loginError.show = false
    try {
      const params: LoginParams = {
        username: form.username.trim(),
        password: form.password
      }
      if (loginMode.value === 'tenant' && form.tenant_id) {
        params.tenant_id = form.tenant_id.trim()
      }
      await userStore.doLogin(params)
      ElMessage.success('登录成功')
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    } catch (e: any) {
      if (e.response?.status === 401 && e.response?.data?.data) {
        const errorData: LoginErrorData = e.response.data.data
        loginError.show = true
        loginError.message = errorData.message
        loginError.remainingAttempts = errorData.remaining_attempts
        loginError.locked = errorData.locked
        loginError.lockoutDuration = errorData.lockout_duration
      }
    } finally {
      loading.value = false
    }
  })
}

async function handleOAuthLogin(provider: string) {
  try {
    // 先获取租户列表
    const res = await getOAuthTenants()
    const data = (res as any)?.data || res
    
    if (data?.tenants && data.tenants.length > 0) {
      tenantList.value = data.tenants
      currentOAuthProvider.value = provider
      
      // 如果只有一个租户，直接跳转 OAuth 授权
      if (data.tenants.length === 1) {
        selectedTenantId.value = data.tenants[0].id
        await confirmOAuthLogin()
      } else {
        // 多个租户，显示选择对话框
        selectedTenantId.value = ''
        showTenantDialog.value = true
      }
    } else {
      ElMessage.error('没有可用的租户，请联系管理员')
    }
  } catch (e) {
    ElMessage.error('获取租户列表失败，请稍后重试')
  }
}

async function confirmOAuthLogin() {
  if (!selectedTenantId.value) {
    ElMessage.warning('请选择租户')
    return
  }
  
  oauthSubmitting.value = true
  try {
    // 保存租户 ID 到 sessionStorage，OAuth 回调时会用到
    sessionStorage.setItem('oauth_tenant_id', selectedTenantId.value)
    
    // 获取 OAuth 跳转链接
    const res = await getOAuthRedirectURL(currentOAuthProvider.value)
    const data = (res as any)?.data || res
    if (data?.url) {
      window.location.href = data.url
      showTenantDialog.value = false
    }
  } catch (e) {
    ElMessage.error('OAuth 登录失败，请稍后重试')
  } finally {
    oauthSubmitting.value = false
  }
}

function goForgotPassword() {
  router.push('/forgot-password')
}

function handleRegisterCommand(command: string) {
  if (command === 'user') {
    router.push('/register-user')
  } else {
    router.push('/register')
  }
}
</script>

<style scoped>
.login-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-box {
  width: 400px;
  padding: 40px 40px 24px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}
.login-title {
  text-align: center;
  margin-bottom: 24px;
}
.login-title h2 {
  margin: 0 0 8px;
  font-size: 24px;
  color: #111827;
}
.login-title p {
  margin: 0;
  color: #6b7280;
  font-size: 14px;
}

.login-mode-switch {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  background: #f3f4f6;
  padding: 4px;
  border-radius: 8px;
}
.mode-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  color: #6b7280;
  transition: all 0.2s;
  user-select: none;
}
.mode-tab:hover {
  color: #374151;
}
.mode-tab.active {
  background: #fff;
  color: #667eea;
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.login-form {
  margin-top: 0;
}
.form-footer {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}
.register-link {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  color: #666;
}
.tips {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  color: #9ca3af;
  margin-top: 8px;
}

.login-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  color: #dc2626;
  font-size: 14px;
  margin-bottom: 16px;
}

.login-error.locked {
  background: #fef3c7;
  border-color: #fcd34d;
  color: #d97706;
}

.login-error .el-icon {
  font-size: 16px;
}

.attempts-warning {
  font-size: 12px;
  color: #ef4444;
}

.lockout-info {
  font-size: 12px;
  color: #d97706;
}

/* OAuth2 第三方登录 */
.oauth-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0;
  color: #9ca3af;
  font-size: 13px;
}
.oauth-divider::before,
.oauth-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: #e5e7eb;
}
.oauth-buttons {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.oauth-btn {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
  user-select: none;
}
.oauth-btn .el-icon {
  font-size: 18px;
  flex-shrink: 0;
}
.oauth-google {
  border-color: #ea4335;
  color: #ea4335;
}
.oauth-google:hover {
  background: #fef2f2;
}
.oauth-github {
  border-color: #24292f;
  color: #24292f;
}
.oauth-github:hover {
  background: #f6f8fa;
}

.dialog-tip {
  color: #666;
  font-size: 14px;
  margin: 0 0 16px;
  text-align: center;
}
</style>