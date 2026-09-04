<e></e><template>
  <div class="page">
    <el-row :gutter="20">
      <!-- 个人信息卡片 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>个人信息</span>
              <el-button type="primary" @click="openEditDialog">
                <el-icon><Edit /></el-icon>编辑资料
              </el-button>
            </div>
          </template>
          <div v-loading="loading" class="profile-info">
            <div class="info-item">
              <label>用户ID：</label>
              <span>{{ userInfo.id }}</span>
            </div>
            <div class="info-item">
              <label>用户名：</label>
              <span>{{ userInfo.username }}</span>
            </div>
            <div class="info-item">
              <label>昵称：</label>
              <span>{{ userInfo.nickname || '-' }}</span>
            </div>
            <div class="info-item">
              <label>邮箱：</label>
              <span>{{ userInfo.email || '-' }}</span>
            </div>
            <div class="info-item">
              <label>租户ID：</label>
              <el-tag v-if="userInfo.tenant_id" size="small" type="info">{{ userInfo.tenant_id }}</el-tag>
              <span v-else>-</span>
            </div>
            <div class="info-item">
              <label>状态：</label>
              <el-tag :type="userInfo.status === 1 ? 'success' : 'info'">
                {{ userInfo.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </div>
            <div class="info-item">
              <label>主管理员：</label>
              <el-tag v-if="userInfo.is_master" type="danger" size="small">MASTER</el-tag>
              <span v-else style="color: #9ca3af">-</span>
            </div>
            <div class="info-item">
              <label>创建时间：</label>
              <span>{{ userInfo.created_at }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 修改密码卡片 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>修改密码</span>
            </div>
          </template>
          <el-form
            ref="pwdFormRef"
            :model="pwdForm"
            :rules="pwdRules"
            label-width="100px"
            style="max-width: 400px;"
          >
            <el-form-item label="原密码" prop="old_password">
              <el-input
                v-model="pwdForm.old_password"
                type="password"
                show-password
                placeholder="请输入原密码"
              />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input
                v-model="pwdForm.new_password"
                type="password"
                show-password
                placeholder="6-32位，包含字母和数字"
              />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirm_password">
              <el-input
                v-model="pwdForm.confirm_password"
                type="password"
                show-password
                placeholder="请再次输入新密码"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="pwdLoading" @click="submitPassword">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 我的角色卡片 -->
        <el-card shadow="never" style="margin-top: 20px;">
          <template #header>
            <div class="card-header">
              <span>我的角色</span>
            </div>
          </template>
          <div v-loading="rolesLoading">
            <el-tag
              v-for="role in userRoles"
              :key="role.id"
              type="primary"
              style="margin-right: 8px; margin-bottom: 8px;"
            >
              {{ role.name }}
            </el-tag>
            <el-empty v-if="userRoles.length === 0" description="暂无角色" :image-size="60" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 编辑资料弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑个人资料"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="80px"
      >
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="editForm.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="editForm.email" placeholder="请输入邮箱" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Edit } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { getProfile, updateProfile, changePassword } from '@/api/modules/user'
import { getUserRoles } from '@/api/modules/role'

const userStore = useUserStore()

// 用户信息
const userInfo = reactive({
  id: '',
  username: '',
  nickname: '',
  email: '',
  tenant_id: '',
  status: 1,
  is_master: false,
  created_at: ''
})
const loading = ref(false)

// 用户角色
const userRoles = ref<{ id: string; name: string; code: string }[]>([])
const rolesLoading = ref(false)

// 编辑资料
const editDialogVisible = ref(false)
const editFormRef = ref<FormInstance>()
const editLoading = ref(false)
const editForm = reactive({
  nickname: '',
  email: ''
})
const editRules: FormRules = {
  email: [{ type: 'email', message: '请输入正确的邮箱', trigger: 'blur' }]
}

// 修改密码
const pwdFormRef = ref<FormInstance>()
const pwdLoading = ref(false)
const pwdForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (_rule: unknown, value: string, callback: (error?: Error) => void) => {
  if (value !== pwdForm.new_password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const pwdRules: FormRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' },
    { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: '密码必须包含字母和数字', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户信息
async function loadUserInfo() {
  loading.value = true
  try {
    const data = await getProfile()
    Object.assign(userInfo, data)
    // 同时更新 store
    userStore.updateUserInfo(data)
  } catch (e) {
    ElMessage.error('获取用户信息失败')
  } finally {
    loading.value = false
  }
}

// 加载用户角色
async function loadUserRoles() {
  if (!userInfo.id) return
  rolesLoading.value = true
  try {
    const data = await getUserRoles(userInfo.id)
    userRoles.value = data || []
  } catch (e) {
    console.error('加载角色失败', e)
  } finally {
    rolesLoading.value = false
  }
}

// 打开编辑弹窗
function openEditDialog() {
  editForm.nickname = userInfo.nickname || ''
  editForm.email = userInfo.email || ''
  editDialogVisible.value = true
}

// 提交编辑
async function submitEdit() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    editLoading.value = true
    try {
      await updateProfile(editForm)
      ElMessage.success('更新成功')
      // 更新本地信息
      userInfo.nickname = editForm.nickname
      userInfo.email = editForm.email
      // 更新 store
      userStore.updateUserInfo({ nickname: editForm.nickname, email: editForm.email })
      editDialogVisible.value = false
    } catch (e: any) {
      ElMessage.error(e.message || '更新失败')
    } finally {
      editLoading.value = false
    }
  })
}

// 提交修改密码
async function submitPassword() {
  if (!pwdFormRef.value) return
  await pwdFormRef.value.validate(async (valid) => {
    if (!valid) return
    pwdLoading.value = true
    try {
      await changePassword({
        old_password: pwdForm.old_password,
        new_password: pwdForm.new_password
      })
      
      ElMessage.success('密码修改成功，请重新登录')
      // 清空表单
      pwdForm.old_password = ''
      pwdForm.new_password = ''
      pwdForm.confirm_password = ''
      // 退出登录
      setTimeout(() => {
        userStore.logout()
      }, 1500)
    } catch (e: any) {
      ElMessage.error(e.message || '密码修改失败')
    } finally {
      pwdLoading.value = false
    }
  })
}

onMounted(() => {
  loadUserInfo()
  loadUserRoles()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.profile-info {
  padding: 10px 0;
}

.info-item {
  display: flex;
  padding: 12px 0;
  border-bottom: 1px solid #ebeef5;
}

.info-item:last-child {
  border-bottom: none;
}

.info-item label {
  width: 100px;
  color: #606266;
  font-weight: 500;
}

.info-item span {
  flex: 1;
  color: #303133;
}
</style>