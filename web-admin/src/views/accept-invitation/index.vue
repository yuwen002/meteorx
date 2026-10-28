<template>
  <div class="accept-invitation-page">
    <div class="card">
      <h2>{{ t('acceptInvite.title') }}</h2>

      <div v-if="loading" class="loading-area">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p>{{ t('acceptInvite.loading') }}</p>
      </div>

      <div v-else-if="error" class="error-area">
        <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
        <p class="error-text">{{ error }}</p>
        <el-button type="primary" @click="$router.push('/login')">{{ t('auth.backToLogin') }}</el-button>
      </div>

      <div v-else-if="success" class="success-area">
        <el-icon :size="48" color="#67c23a"><CircleCheckFilled /></el-icon>
        <p class="success-text">{{ t('acceptInvite.success') }}</p>
        <el-button type="primary" @click="$router.push('/login')">{{ t('verifyEmail.goLogin') }}</el-button>
      </div>

      <el-form v-else ref="formRef" :model="form" :rules="rules" label-width="80px">
        <div class="invite-info">
          <p>{{ t('acceptInvite.invitedTo', inviteInfo?.tenant_name || inviteInfo?.tenant_id || '') }}</p>
        </div>
        <el-form-item :label="t('common.email')">
          <el-input :model-value="inviteInfo?.email" disabled />
        </el-form-item>
        <el-form-item :label="t('auth.username')" prop="username">
          <el-input v-model="form.username" :placeholder="t('acceptInvite.usernamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('register.nickname')" prop="nickname">
          <el-input v-model="form.nickname" :placeholder="t('acceptInvite.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('auth.password')" prop="password">
          <el-input v-model="form.password" type="password" show-password :placeholder="t('acceptInvite.passwordPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
          <el-input v-model="form.confirm_password" type="password" show-password :placeholder="t('register.confirmPlaceholder')" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitLoading" style="width: 100%;" @click="handleSubmit">{{ t('acceptInvite.submit') }}</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Loading, CircleCloseFilled, CircleCheckFilled } from '@element-plus/icons-vue'
import { getInvitationInfo, acceptInvitation, type InvitationItem } from '@/api/modules/invitation'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const route = useRoute()
const token = (route.query.token || route.params.token || '') as string

const loading = ref(true)
const error = ref('')
const success = ref(false)
const inviteInfo = ref<InvitationItem | null>(null)
const submitLoading = ref(false)

const formRef = ref<FormInstance>()
const form = reactive({
  username: '',
  nickname: '',
  password: '',
  confirm_password: ''
})

const validateConfirmPassword = (_rule: unknown, value: string, callback: (error?: Error) => void) => {
  if (value !== form.password) {
    callback(new Error(t('register.passwordMismatch')))
  } else {
    callback()
  }
}

const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('register.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 32, message: t('acceptInvite.usernameLength'), trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: t('acceptInvite.usernamePattern'), trigger: 'blur' }
  ],
  nickname: [{ required: true, message: t('register.nicknameRequired'), trigger: 'blur' }],
  password: [
    { required: true, message: t('register.passwordRequired'), trigger: 'blur' },
    { min: 6, max: 32, message: t('register.passwordLength'), trigger: 'blur' },
    { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: t('acceptInvite.passwordPattern'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('register.confirmRequired'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}))

async function loadInvitationInfo() {
  if (!token) {
    error.value = t('acceptInvite.missingToken')
    loading.value = false
    return
  }
  try {
    const data = await getInvitationInfo(token)
    inviteInfo.value = data
    if (data.status !== 'pending') {
      error.value = data.status === 'accepted' ? t('acceptInvite.statusAccepted') : data.status === 'cancelled' ? t('acceptInvite.statusCancelled') : t('acceptInvite.statusExpired')
    }
  } catch (e) {
    error.value = (e as Error).message || t('acceptInvite.loadFailed')
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      await acceptInvitation({
        token,
        username: form.username,
        password: form.password,
        nickname: form.nickname
      })
      success.value = true
      ElMessage.success(t('register.successMsg'))
    } catch (e) {
      ElMessage.error((e as Error).message || t('acceptInvite.registerFailed'))
    } finally {
      submitLoading.value = false
    }
  })
}

onMounted(() => {
  loadInvitationInfo()
})
</script>

<style scoped>
.accept-invitation-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f0f2f5;
}
.card {
  background: #fff;
  border-radius: 8px;
  padding: 32px 40px;
  width: 480px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
.card h2 {
  text-align: center;
  margin-bottom: 24px;
  color: #303133;
}
.loading-area, .error-area, .success-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px 0;
}
.error-text { color: #f56c6c; font-size: 14px; }
.success-text { color: #67c23a; font-size: 16px; font-weight: 500; }
.invite-info {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 20px;
  color: #606266;
  font-size: 14px;
}
</style>