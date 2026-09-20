<template>
  <div class="oauth-callback-page">
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
        <el-button type="primary" :loading="submitting" @click="handleTenantSelect">
          确认登录
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
import { oauthLogin, getOAuthTenants } from '@/api/auth'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const statusMessage = ref('正在处理第三方登录...')
const errorMessage = ref('')
const showTenantDialog = ref(false)
const submitting = ref(false)
const tenantList = ref<Array<{ id: string; name: string }>>([])
const selectedTenantId = ref('')

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
    statusMessage.value = '登录失败'
    errorMessage.value = '缺少必要的认证参数'
    setTimeout(() => router.push('/login'), 3000)
    return
  }

  // 保存 OAuth 参数
  oauthParams.value = { provider, code, state }

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
    const data = (res as any)?.data || res
    
    if (data?.tenants && data.tenants.length > 0) {
      tenantList.value = data.tenants
      // 如果只有一个租户，直接登录
      if (data.tenants.length === 1) {
        selectedTenantId.value = data.tenants[0].id
        await handleTenantSelect()
      } else {
        // 多个租户，显示选择对话框
        showTenantDialog.value = true
        statusMessage.value = '请选择租户'
      }
    } else {
      statusMessage.value = '登录失败'
      errorMessage.value = '没有可用的租户，请联系管理员'
      setTimeout(() => router.push('/login'), 3000)
    }
  } catch (e: any) {
    statusMessage.value = '登录失败'
    errorMessage.value = e?.message || '获取租户列表失败'
    setTimeout(() => router.push('/login'), 3000)
  }
})

async function handleTenantSelect() {
  if (!selectedTenantId.value) {
    ElMessage.warning('请选择租户')
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
    const data = (res as any)?.data || res
    
    // 清除 sessionStorage
    sessionStorage.removeItem('oauth_tenant_id')
    
    // 使用 OAuth 返回的 token 和用户信息登录
    userStore.setOAuthUser({
      token: data.token || '',
      user: data.user || null,
      permissions: data.permissions || []
    })

    ElMessage.success('登录成功')
    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
  } catch (e: any) {
    ElMessage.error(e?.message || '登录失败')
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