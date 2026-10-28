<template>
  <div class="reset-password-page">
    <div class="reset-password-box">
      <div class="header">
        <h2>{{ t('auth.resetPassword') }}</h2>
        <p>{{ t('resetPassword.subtitle') }}</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        class="form"
        @keyup.enter="handleSubmit"
      >
        <el-form-item :label="t('auth.newPassword')" prop="new_password">
          <el-input
            v-model="form.new_password"
            type="password"
            :placeholder="t('resetPassword.newPasswordPlaceholder')"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>

        <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
          <el-input
            v-model="form.confirm_password"
            type="password"
            :placeholder="t('resetPassword.confirmPlaceholder')"
            prefix-icon="Lock"
            show-password
            size="large"
          />
        </el-form-item>

        <el-button
          type="primary"
          :loading="loading"
          size="large"
          style="width: 100%"
          @click="handleSubmit"
        >
          {{ t('resetPassword.submit') }}
        </el-button>

        <div class="footer-links">
          <router-link to="/login">{{ t('auth.backToLogin') }}</router-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { resetPassword } from '@/api/auth'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const router = useRouter()
const route = useRoute()
const formRef = ref<FormInstance>()
const loading = ref(false)
const token = ref('')

const form = reactive({
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (_rule: unknown, value: string, callback: (error?: string | Error) => void) => {
  if (value !== form.new_password) {
    callback(new Error(t('resetPassword.passwordMismatch')))
  } else {
    callback()
  }
}

const formRules = computed<FormRules>(() => ({
  new_password: [
    { required: true, message: t('resetPassword.newRequired'), trigger: 'blur' },
    { min: 8, max: 32, message: t('resetPassword.lengthRange'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('resetPassword.confirmRequired'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}))

onMounted(() => {
  const tokenParam = route.query.token as string
  if (!tokenParam) {
    ElMessage.error(t('resetPassword.invalidLink'))
    setTimeout(() => {
      router.push('/login')
    }, 2000)
    return
  }
  token.value = tokenParam
})

async function handleSubmit() {
  if (!formRef.value) return
  if (!token.value) {
    ElMessage.error(t('resetPassword.invalidLink'))
    router.push('/login')
    return
  }

  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      await resetPassword(token.value, form.new_password)
      ElMessage.success(t('resetPassword.successMsg'))
      setTimeout(() => {
        router.push('/login')
      }, 2000)
    } catch (e) {
      ElMessage.error((e as { response?: { data?: { message?: string } } })?.response?.data?.message || t('resetPassword.failed'))
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.reset-password-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.reset-password-box {
  width: 420px;
  padding: 40px 40px 24px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

.header {
  text-align: center;
  margin-bottom: 24px;
}

.header h2 {
  margin: 0 0 8px;
  font-size: 24px;
  color: #111827;
}

.header p {
  margin: 0;
  color: #6b7280;
  font-size: 14px;
}

.form {
  margin-top: 8px;
}

.footer-links {
  text-align: center;
  margin-top: 16px;
}

.footer-links a {
  color: #667eea;
  text-decoration: none;
  font-size: 14px;
}

.footer-links a:hover {
  text-decoration: underline;
}
</style>