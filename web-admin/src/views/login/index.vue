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
          <el-button type="primary" :loading="loading" @click="handleLogin" size="large" style="width: 100%">
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
import { ElMessage } from 'element-plus'
import { InfoFilled, Warning, Avatar, OfficeBuilding } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import type { LoginParams, LoginErrorData } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

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
</style>