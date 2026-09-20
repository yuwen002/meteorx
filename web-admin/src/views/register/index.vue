<template>
  <div class="register-page">
    <div class="register-box">
      <div class="register-title">
        <h2>注册新租户</h2>
        <p>创建您的专属工作空间</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        class="register-form"
      >
        <el-divider content-position="left">租户信息</el-divider>

        <el-form-item label="租户名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="请输入租户名称（企业/团队名称）"
            prefix-icon="OfficeBuilding"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item label="域名标识" prop="domain">
          <el-input
            v-model="form.domain"
            placeholder="请输入域名标识（英文、数字、连字符）"
            prefix-icon="Link"
            clearable
            size="large"
          >
            <template #append>.meteorx.com</template>
          </el-input>
          <div class="form-tip">用于生成您的专属访问域名，如：your-domain.meteorx.com</div>
        </el-form-item>

        <el-form-item label="联系邮箱" prop="contact_email">
          <el-input
            v-model="form.contact_email"
            placeholder="请输入联系邮箱（选填）"
            prefix-icon="Message"
            clearable
            size="large"
          />
        </el-form-item>

        <el-divider content-position="left">管理员信息</el-divider>

        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="form.username"
            placeholder="请输入用户名（4-20个字符）"
            prefix-icon="User"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item label="昵称" prop="nickname">
          <el-input
            v-model="form.nickname"
            placeholder="请输入昵称"
            prefix-icon="Avatar"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="form.email"
            placeholder="请输入邮箱"
            prefix-icon="Message"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码（至少6个字符）"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>

        <el-form-item label="确认密码" prop="confirm_password">
          <el-input
            v-model="form.confirm_password"
            type="password"
            placeholder="请再次输入密码"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            size="large"
            style="width: 100%"
            @click="handleRegister"
          >
            立即注册
          </el-button>
        </el-form-item>

        <div class="form-footer">
          <span>已有账号？</span>
          <el-link type="primary" :underline="false" @click="goLogin">
            立即登录
          </el-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { register } from '@/api/auth'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  name: '',
  domain: '',
  description: '',
  contact_email: '',
  username: '',
  nickname: '',
  email: '',
  password: '',
  confirm_password: '',
})

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const validateDomain = (_rule: any, value: string, callback: any) => {
  const pattern = /^[a-z0-9-]+$/
  if (!pattern.test(value)) {
    callback(new Error('只能包含小写字母、数字和连字符'))
  } else {
    callback()
  }
}

const formRules: FormRules = {
  name: [
    { required: true, message: '请输入租户名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' },
  ],
  domain: [
    { required: true, message: '请输入域名标识', trigger: 'blur' },
    { min: 2, max: 30, message: '长度在 2 到 30 个字符', trigger: 'blur' },
    { validator: validateDomain, trigger: 'blur' },
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 4, max: 20, message: '长度在 4 到 20 个字符', trigger: 'blur' },
  ],
  nickname: [
    { required: true, message: '请输入昵称', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 32, message: '长度在 6 到 32 个字符', trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' },
  ],
}

const handleRegister = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    loading.value = true

    const registerData = {
      name: form.name,
      domain: form.domain,
      description: '',
      contact_email: form.contact_email,
      admin_user: {
        username: form.username,
        password: form.password,
        nickname: form.nickname,
        email: form.email,
      },
    }
    await register(registerData)

    ElMessage.success('注册成功，请登录')
    router.push('/login')
  } catch (error: any) {
    if (error.message) {
      ElMessage.error(error.message)
    }
  } finally {
    loading.value = false
  }
}

const goLogin = () => {
  router.push('/login')
}
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.register-box {
  width: 100%;
  max-width: 500px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  padding: 40px;
}

.register-title {
  text-align: center;
  margin-bottom: 30px;
}

.register-title h2 {
  font-size: 28px;
  color: #333;
  margin: 0 0 8px;
}

.register-title p {
  font-size: 14px;
  color: #999;
  margin: 0;
}

.register-form {
  margin-top: 20px;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
  line-height: 1.5;
}

.form-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
  font-size: 14px;
  color: #666;
}

:deep(.el-divider__text) {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}
</style>