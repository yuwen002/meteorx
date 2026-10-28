<template>
  <div class="register-page">
    <div class="register-box">
      <div class="register-title">
        <h2>{{ t('register.tenantTitle') }}</h2>
        <p>{{ t('register.tenantSubtitle') }}</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        class="register-form"
      >
        <el-divider content-position="left">{{ t('register.section.tenant') }}</el-divider>

        <el-form-item :label="t('register.tenantName')" prop="name">
          <el-input
            v-model="form.name"
            :placeholder="t('register.tenantNamePlaceholder')"
            prefix-icon="OfficeBuilding"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('register.domain')" prop="domain">
          <el-input
            v-model="form.domain"
            :placeholder="t('register.domainPlaceholder')"
            prefix-icon="Link"
            clearable
            size="large"
          >
            <template #append>.meteorx.com</template>
          </el-input>
          <div class="form-tip">{{ t('register.domainTip') }}</div>
        </el-form-item>

        <el-form-item :label="t('register.contactEmail')" prop="contact_email">
          <el-input
            v-model="form.contact_email"
            :placeholder="t('register.contactEmailPlaceholder')"
            prefix-icon="Message"
            clearable
            size="large"
          />
        </el-form-item>

        <el-divider content-position="left">{{ t('register.section.admin') }}</el-divider>

        <el-form-item :label="t('auth.username')" prop="username">
          <el-input
            v-model="form.username"
            :placeholder="t('register.usernamePlaceholder')"
            prefix-icon="User"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('register.nickname')" prop="nickname">
          <el-input
            v-model="form.nickname"
            :placeholder="t('register.nicknamePlaceholder')"
            prefix-icon="Avatar"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('common.email')" prop="email">
          <el-input
            v-model="form.email"
            :placeholder="t('register.emailPlaceholder')"
            prefix-icon="Message"
            clearable
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('auth.password')" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('register.passwordPlaceholder')"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
          <el-input
            v-model="form.confirm_password"
            type="password"
            :placeholder="t('register.confirmPlaceholder')"
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
            {{ t('register.submit') }}
          </el-button>
        </el-form-item>

        <div class="form-footer">
          <span>{{ t('register.haveAccount') }}</span>
          <el-link type="primary" :underline="false" @click="goLogin">
            {{ t('register.goLogin') }}
          </el-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { register } from '@/api/auth'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

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

const validateConfirmPassword = (_rule: unknown, value: string, callback: (error?: string | Error) => void) => {
  if (value !== form.password) {
    callback(new Error(t('register.passwordMismatch')))
  } else {
    callback()
  }
}

const validateDomain = (_rule: unknown, value: string, callback: (error?: string | Error) => void) => {
  const pattern = /^[a-z0-9-]+$/
  if (!pattern.test(value)) {
    callback(new Error(t('register.domainPattern')))
  } else {
    callback()
  }
}

const formRules = computed<FormRules>(() => ({
  name: [
    { required: true, message: t('register.tenantNameRequired'), trigger: 'blur' },
    { min: 2, max: 50, message: t('register.tenantNameLength'), trigger: 'blur' },
  ],
  domain: [
    { required: true, message: t('register.domainRequired'), trigger: 'blur' },
    { min: 2, max: 30, message: t('register.domainLength'), trigger: 'blur' },
    { validator: validateDomain, trigger: 'blur' },
  ],
  username: [
    { required: true, message: t('register.usernameRequired'), trigger: 'blur' },
    { min: 4, max: 20, message: t('register.usernameLength'), trigger: 'blur' },
  ],
  nickname: [
    { required: true, message: t('register.nicknameRequired'), trigger: 'blur' },
  ],
  email: [
    { required: true, message: t('register.emailRequired'), trigger: 'blur' },
    { type: 'email', message: t('register.emailInvalid'), trigger: 'blur' },
  ],
  password: [
    { required: true, message: t('register.passwordRequired'), trigger: 'blur' },
    { min: 6, max: 32, message: t('register.passwordLength'), trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: t('register.confirmRequired'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' },
  ],
}))

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

    ElMessage.success(t('register.successMsg'))
    router.push('/login')
  } catch (error) {
    const err = error as { message?: string }
    if (err.message) {
      ElMessage.error(err.message)
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