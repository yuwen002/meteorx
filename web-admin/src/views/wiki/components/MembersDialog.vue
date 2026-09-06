<template>
  <el-dialog
    :model-value="modelValue"
    title="成员管理"
    width="640px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="load"
  >
    <div v-if="canManage && canListUsers" class="member-toolbar">
      <span class="tip">添加协作者：</span>
      <UserSelect ref="userSelectRef" v-model="newUserId" class="user-picker" />
      <el-select v-model="newRole" class="role-picker">
        <el-option
          v-for="opt in roleOptions"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>
      <el-button type="primary" :disabled="!newUserId" :loading="submitting" @click="addUser">
        添加成员
      </el-button>
    </div>
    <el-alert
      v-if="!canListUsers"
      type="warning"
      :closable="false"
      style="margin-bottom: 12px"
    >
      当前账号缺少用户管理（user:list）权限，无法搜索用户添加新成员。
    </el-alert>

    <el-table v-loading="loading" :data="members" size="small">
      <el-table-column label="用户" min-width="200">
        <template #default="{ row }">
          <div>{{ displayName(row) }}</div>
          <div class="sub">{{ row.user_email || row.user_id }}</div>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="150">
        <template #default="{ row }">
          <el-select
            v-if="canManage"
            :model-value="row.role"
            :disabled="isSelfOwner(row)"
            size="small"
            @update:model-value="(v: string) => changeRole(row, v)"
          >
            <el-option
              v-for="opt in roleOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
          <span v-else>{{ roleLabel(row.role) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="90" align="center">
        <template #default="{ row }">
          <el-tooltip
            v-if="canManage"
            :disabled="!isSelfOwner(row)"
            content="所有者不能移除自己，请先提升其他人为所有者"
          >
            <span>
              <el-button
                link
                type="danger"
                :disabled="isSelfOwner(row)"
                @click="removeUser(row)"
              >
                移除
              </el-button>
            </span>
          </el-tooltip>
          <span v-else>—</span>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && members.length === 0" description="暂无成员" :image-size="60" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import {
  listMembers,
  addMember,
  removeMember,
  type WikiSpaceMember
} from '@/api/modules/wiki'
import { useUserStore } from '@/stores/user'
import UserSelect from './UserSelect.vue'

const props = defineProps<{
  modelValue: boolean
  spaceId: string
  myUserId: string
  myRole: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; changed: [] }>()

const userStore = useUserStore()
const canListUsers = userStore.hasPermission('user:list')
const canManage = computed(() => props.myRole === 'owner' || props.myRole === 'admin')

const roleOptions = [
  { value: 'owner', label: '所有者' },
  { value: 'admin', label: '管理员' },
  { value: 'editor', label: '编辑者' },
  { value: 'viewer', label: '访客（只读）' }
]
const roleLabel = (r: string) => roleOptions.find((o) => o.value === r)?.label || r

const members = ref<WikiSpaceMember[]>([])
const loading = ref(false)
const submitting = ref(false)
const newUserId = ref('')
const newRole = ref('viewer')

const displayName = (row: WikiSpaceMember) => row.user_name || row.user_email || row.user_id
const isSelfOwner = (row: WikiSpaceMember) => row.user_id === props.myUserId && row.role === 'owner'

async function load() {
  if (!props.spaceId) return
  loading.value = true
  try {
    members.value = (await listMembers(props.spaceId)) ?? []
  } catch {
    members.value = []
  } finally {
    loading.value = false
  }
}

async function addUser() {
  if (!newUserId.value) return
  submitting.value = true
  try {
    await addMember(props.spaceId, { user_id: newUserId.value, role: newRole.value as never })
    ElMessage.success('已添加成员')
    newUserId.value = ''
    await load()
    emit('changed')
  } finally {
    submitting.value = false
  }
}

async function changeRole(row: WikiSpaceMember, role: string) {
  try {
    await addMember(props.spaceId, { user_id: row.user_id, role: role as never })
    ElMessage.success('角色已更新')
    await load()
    emit('changed')
  } catch {
    await load()
  }
}

async function removeUser(row: WikiSpaceMember) {
  await ElMessageBox.confirm(`确定将 ${displayName(row)} 移出该知识库吗？`, '提示', {
    type: 'warning'
  }).catch(() => Promise.reject(new Error('canceled')))
  try {
    await removeMember(props.spaceId, row.user_id)
    ElMessage.success('已移除')
    await load()
    emit('changed')
  } catch (e) {
    if ((e as Error)?.message === 'canceled') return
  }
}
</script>

<style scoped>
.member-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.tip {
  color: #6b7280;
  font-size: 13px;
  white-space: nowrap;
}
.user-picker {
  flex: 1;
}
.role-picker {
  width: 120px;
}
.sub {
  color: #9ca3af;
  font-size: 12px;
}
</style>
