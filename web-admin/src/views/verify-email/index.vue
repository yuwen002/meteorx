<template>
  <div class="verify-email-page">
    <div class="card">
      <h2>邮箱验证</h2>

      <div v-if="loading" class="status-area">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p>正在验证...</p>
      </div>

      <div v-else-if="verified" class="status-area">
        <el-icon :size="48" color="#67c23a"><CircleCheckFilled /></el-icon>
        <p class="success-text">邮箱验证成功！</p>
        <el-button type="primary" @click="$router.push('/login')">去登录</el-button>
      </div>

      <div v-else class="status-area">
        <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
        <p class="error-text">{{ errorMsg }}</p>
        <el-button type="primary" @click="$router.push('/login')">返回登录</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Loading, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import { verifyEmail } from '@/api/auth'

const route = useRoute()
const token = (route.query.token || '') as string

const loading = ref(true)
const verified = ref(false)
const errorMsg = ref('验证失败，请重新发送验证邮件')

onMounted(async () => {
  if (!token) {
    errorMsg.value = '缺少验证令牌'
    loading.value = false
    return
  }
  try {
    await verifyEmail(token)
    verified.value = true
  } catch (e) {
    errorMsg.value = (e as Error).message || '验证失败，请重新发送验证邮件'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.verify-email-page {
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
  width: 400px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
.card h2 {
  text-align: center;
  margin-bottom: 24px;
  color: #303133;
}
.status-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px 0;
}
.success-text { color: #67c23a; font-size: 16px; font-weight: 500; }
.error-text { color: #f56c6c; font-size: 14px; }
</style>