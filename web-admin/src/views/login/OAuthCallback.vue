<template>
  <div class="oauth-callback-page">
    <div class="loading-box">
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
import { oauthLogin } from '@/api/auth'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const statusMessage = ref('正在处理第三方登录...')
const errorMessage = ref('')

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

  try {
    const res = await oauthLogin(provider, code, state)
    const data = (res as any)?.data || res
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
    statusMessage.value = '登录失败'
    errorMessage.value = e?.message || '第三方登录验证失败，请稍后重试'
    setTimeout(() => router.push('/login'), 3000)
  }
})
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
</style>