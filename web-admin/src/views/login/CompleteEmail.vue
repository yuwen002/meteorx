<template>
  <div class="complete-email-page">
    <div class="complete-email-box">
      <h2>补全邮箱</h2>
      <p class="tip">当前第三方账号未提供邮箱，请填写邮箱以完成登录。</p>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @keyup.enter="handleSubmit"
      >
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" size="large" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" size="large" style="width: 100%" @click="handleSubmit">
            完成登录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useUserStore } from '@/stores/user'
import { completeOAuthEmail } from '@/api/modules/oauth'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

// pending 令牌仅从 sessionStorage 获取（由 OAuthCallback 写入）；
// 不再从 route.query.token 读取，避免令牌进入浏览器历史/Referer/访问日志
const token = sessionStorage.getItem('oauth_pending_token') || ''
const redirect = sanitizeRedirect(route.query.redirect as string)

// sanitizeRedirect 仅允许站内绝对路径，拒绝 //host 这类协议相对跳转，防止开放重定向
function sanitizeRedirect(target: string): string {
  if (target && target.startsWith('/') && !target.startsWith('//')) return target
  return '/'
}

const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive<{ email: string }>({ email: '' })

const rules = reactive<FormRules>({
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }
  ]
})

if (!token) {
  ElMessage.error('登录状态已失效，请重新登录')
  router.replace('/login')
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    if (!token) {
      ElMessage.error('登录状态已失效，请重新登录')
      router.replace('/login')
      return
    }

    submitting.value = true
    try {
      const res = await completeOAuthEmail(token, form.email.trim())
      const data = (res as { data?: typeof res }).data || res

      const pendingUser = sessionStorage.getItem('oauth_pending_user')
      const pendingPerms = sessionStorage.getItem('oauth_pending_permissions')
      sessionStorage.removeItem('oauth_pending_token')
      sessionStorage.removeItem('oauth_pending_user')
      sessionStorage.removeItem('oauth_pending_permissions')

      userStore.setOAuthUser({
        token: data.token || '',
        user: pendingUser ? JSON.parse(pendingUser) : null,
        permissions: pendingPerms ? JSON.parse(pendingPerms) : []
      })

      ElMessage.success('登录成功')
      router.replace(redirect)
    } catch (e) {
      ElMessage.error((e as Error)?.message || '补全邮箱失败')
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
.complete-email-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.complete-email-box {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}
.complete-email-box h2 {
  margin: 0 0 8px;
  font-size: 22px;
  color: #111827;
  text-align: center;
}
.tip {
  margin: 0 0 24px;
  color: #6b7280;
  font-size: 14px;
  text-align: center;
}
</style>
