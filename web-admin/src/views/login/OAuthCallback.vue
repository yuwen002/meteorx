<template>
  <div class="oauth-callback-page">
    <!-- 租户选择对话框 -->
    <el-dialog
      v-model="showTenantDialog"
      :title="t('oauthCallback.tenantTitle')"
      width="400px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <p class="dialog-tip">{{ t('oauthCallback.tenantTip') }}</p>
      <el-select
        v-model="selectedTenantId"
        :placeholder="t('oauthCallback.tenantPlaceholder')"
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
        <el-button type="primary" :loading="submitting" @click="handleTenantSelect">
          {{ t('oauthCallback.confirmLogin') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 加载状态 -->
    <div v-if="!showTenantDialog" class="loading-box">
      <el-icon class="loading-icon" :size="48" color="#409eff"><Loading /></el-icon>
      <h3>{{ statusMessage }}</h3>
      <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { Loading } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { oauthLogin, getOAuthTenants, bindOAuth } from '@/api/auth'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const statusMessage = ref(t('oauthCallback.processing'))
const errorMessage = ref('')
const showTenantDialog = ref(false)
const submitting = ref(false)
const tenantList = ref<Array<{ id: string; name: string }>>([])
const selectedTenantId = ref('')

// sanitizeRedirect 仅允许站内绝对路径，拒绝 //host 这类协议相对跳转，防止开放重定向
function sanitizeRedirect(target: string): string {
  if (target && target.startsWith('/') && !target.startsWith('//')) return target
  return '/'
}

// OAuth 回调参数
const oauthParams = ref({
  provider: '',
  code: '',
  state: ''
})

onMounted(async () => {
  const provider = route.params.provider as string
  const code = route.query.code as string
  const state = route.query.state as string

  if (!provider || !code) {
    statusMessage.value = t('oauthCallback.loginFailed')
    errorMessage.value = t('oauthCallback.missingParams')
    setTimeout(() => router.push('/login'), 3000)
    return
  }

  // 保存 OAuth 参数
  oauthParams.value = { provider, code, state }

  // 检查是否为绑定模式
  const isBindMode = sessionStorage.getItem('oauth_bind_mode') === 'true'
  if (isBindMode) {
    sessionStorage.removeItem('oauth_bind_mode')
    statusMessage.value = t('oauthCallback.binding')
    try {
      await bindOAuth(provider, code, state)
      ElMessage.success(t('oauthCallback.bindSuccess'))
      router.push('/profile')
    } catch (e) {
      ElMessage.error((e as Error)?.message || t('oauthCallback.bindFailed'))
      router.push('/profile')
    }
    return
  }

  // 从 sessionStorage 获取租户 ID
  const storedTenantId = sessionStorage.getItem('oauth_tenant_id')
  
  try {
    // 如果有存储的租户 ID，直接使用
    if (storedTenantId) {
      selectedTenantId.value = storedTenantId
      await handleTenantSelect()
      return
    }
    
    // 否则获取租户列表
    const res = await getOAuthTenants()
    const data = (res as { data?: typeof res }).data || res
    
    if (data?.tenants && data.tenants.length > 0) {
      tenantList.value = data.tenants
      // 如果只有一个租户，直接登录
      if (data.tenants.length === 1) {
        selectedTenantId.value = data.tenants[0].id
        await handleTenantSelect()
      } else {
        // 多个租户，显示选择对话框
        showTenantDialog.value = true
        statusMessage.value = t('oauthCallback.selectTenant')
      }
    } else {
      statusMessage.value = t('oauthCallback.loginFailed')
      errorMessage.value = t('oauthCallback.noTenant')
      setTimeout(() => router.push('/login'), 3000)
    }
  } catch (e) {
    statusMessage.value = t('oauthCallback.loginFailed')
    errorMessage.value = (e as Error)?.message || t('oauthCallback.fetchTenantsFailed')
    setTimeout(() => router.push('/login'), 3000)
  }
})

async function handleTenantSelect() {
  if (!selectedTenantId.value) {
    ElMessage.warning(t('oauthCallback.selectTenant'))
    return
  }

  submitting.value = true
  try {
    const res = await oauthLogin(
      oauthParams.value.provider,
      oauthParams.value.code,
      selectedTenantId.value,
      oauthParams.value.state
    )
    const data = (res as { data?: typeof res }).data || res
    
    // 清除 sessionStorage
    sessionStorage.removeItem('oauth_tenant_id')
    
    // 微信/QQ 等未返回邮箱的提供商，需先引导用户补全邮箱
    if (data.need_complete_email) {
      sessionStorage.setItem('oauth_pending_token', data.token || '')
      sessionStorage.setItem('oauth_pending_user', JSON.stringify(data.user || null))
      sessionStorage.setItem('oauth_pending_permissions', JSON.stringify(data.permissions || []))
      sessionStorage.removeItem('oauth_tenant_id')
      router.push({
        path: '/oauth/complete-email',
        query: { redirect: sanitizeRedirect(route.query.redirect as string) }
      })
      return
    }

    // 使用 OAuth 返回的 token 和用户信息登录
    userStore.setOAuthUser({
      token: data.token || '',
      user: data.user || null,
      permissions: data.permissions || []
    })

    ElMessage.success(t('completeEmail.success'))
    const redirect = sanitizeRedirect(route.query.redirect as string)
    router.push(redirect)
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('auth.loginFailed'))
    router.push('/login')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.oauth-callback-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.loading-box {
  text-align: center;
  padding: 48px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}
.loading-icon {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.loading-box h3 {
  margin: 16px 0 8px;
  color: #374151;
  font-size: 18px;
}
.error-message {
  color: #ef4444;
  font-size: 14px;
  margin: 8px 0 0;
}
.dialog-tip {
  color: #666;
  font-size: 14px;
  margin: 0 0 16px;
  text-align: center;
}
</style>