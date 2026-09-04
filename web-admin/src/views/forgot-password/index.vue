<template>
  <div class="forgot-password-page">
    <div class="forgot-password-box">
      <div class="header">
        <h2>忘记密码</h2>
        <p>输入您的邮箱，我们将发送重置链接</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        class="form"
        @keyup.enter="handleSubmit"
      >
        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="form.email"
            placeholder="请输入您的注册邮箱"
            prefix-icon="Message"
            clearable
            size="large"
            type="email"
          />
        </el-form-item>

        <el-button
          type="primary"
          :loading="loading"
          size="large"
          style="width: 100%"
          @click="handleSubmit"
        >
          发送重置链接
        </el-button>

        <div v-if="success" class="success-tip">
          <el-icon><SuccessFilled /></el-icon>
          <span>重置链接已发送至您的邮箱，请查收邮件</span>
        </div>

        <div class="footer-links">
          <router-link to="/login">返回登录</router-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { SuccessFilled } from '@element-plus/icons-vue'
import { forgotPassword } from '@/api/auth'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)
const success = ref(false)

const form = reactive({
  email: ''
})

const formRules: FormRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ]
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    success.value = false
    try {
      await forgotPassword(form.email.trim())
      success.value = true
      ElMessage.success('重置链接已发送至您的邮箱')
      setTimeout(() => {
        router.push('/login')
      }, 3000)
    } catch (e: any) {
      ElMessage.error(e?.response?.data?.message || '发送失败，请稍后重试')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped>
.forgot-password-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.forgot-password-box {
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

.success-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 6px;
  color: #16a34a;
  font-size: 14px;
  margin: 16px 0;
}

.success-tip .el-icon {
  font-size: 18px;
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