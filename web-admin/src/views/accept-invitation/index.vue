<template>
  <div class="accept-invitation-page">
    <div class="card">
      <h2>接受邀请</h2>

      <div v-if="loading" class="loading-area">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p>正在加载邀请信息...</p>
      </div>

      <div v-else-if="error" class="error-area">
        <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
        <p class="error-text">{{ error }}</p>
        <el-button type="primary" @click="$router.push('/login')">返回登录</el-button>
      </div>

      <div v-else-if="success" class="success-area">
        <el-icon :size="48" color="#67c23a"><CircleCheckFilled /></el-icon>
        <p class="success-text">注册成功！</p>
        <el-button type="primary" @click="$router.push('/login')">去登录</el-button>
      </div>

      <el-form v-else ref="formRef" :model="form" :rules="rules" label-width="80px">
        <div class="invite-info">
          <p>您已被邀请加入租户 <strong>{{ inviteInfo?.tenant_name || inviteInfo?.tenant_id }}</strong></p>
        </div>
        <el-form-item label="邮箱">
          <el-input :model-value="inviteInfo?.email" disabled />
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="设置登录用户名" />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" placeholder="设置昵称" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="6-32位，包含字母和数字" />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input v-model="form.confirm_password" type="password" show-password placeholder="请再次输入密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitLoading" style="width: 100%;" @click="handleSubmit">接受邀请并注册</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Loading, CircleCloseFilled, CircleCheckFilled } from '@element-plus/icons-vue'
import { getInvitationInfo, acceptInvitation, type InvitationItem } from '@/api/modules/invitation'

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
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 32, message: '用户名长度 3-32 位', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线', trigger: 'blur' }
  ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' },
    { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: '密码必须包含字母和数字', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

async function loadInvitationInfo() {
  if (!token) {
    error.value = '缺少邀请令牌'
    loading.value = false
    return
  }
  try {
    const data = await getInvitationInfo(token)
    inviteInfo.value = data
    if (data.status !== 'pending') {
      error.value = data.status === 'accepted' ? '此邀请已被接受' : data.status === 'cancelled' ? '此邀请已被取消' : '此邀请已过期'
    }
  } catch (e) {
    error.value = (e as Error).message || '获取邀请信息失败'
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
      ElMessage.success('注册成功，请登录')
    } catch (e) {
      ElMessage.error((e as Error).message || '注册失败')
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