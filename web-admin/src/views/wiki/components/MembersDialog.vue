<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('wiki.member.title')"
    width="640px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="load"
  >
    <div v-if="canManage && canListUsers" class="member-toolbar">
      <span class="tip">{{ t('wiki.member.addLabel') }}</span>
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
        {{ t('wiki.member.add') }}
      </el-button>
    </div>
    <el-alert
      v-if="!canListUsers"
      type="warning"
      :closable="false"
      style="margin-bottom: 12px"
    >
      {{ t('wiki.member.noUserPerm') }}
    </el-alert>

    <el-table v-loading="loading" :data="members" size="small">
      <el-table-column :label="t('wiki.member.colUser')" min-width="200">
        <template #default="{ row }">
          <div>{{ displayName(row) }}</div>
          <div class="sub">{{ row.user_email || row.user_id }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="t('wiki.member.colRole')" width="150">
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
      <el-table-column :label="t('common.actions')" width="90" align="center">
        <template #default="{ row }">
          <el-tooltip
            v-if="canManage"
            :disabled="!isSelfOwner(row)"
            :content="t('wiki.member.ownerTip')"
          >
            <span>
              <el-button
                link
                type="danger"
                :disabled="isSelfOwner(row)"
                @click="removeUser(row)"
              >
                {{ t('wiki.member.remove') }}
              </el-button>
            </span>
          </el-tooltip>
          <span v-else>—</span>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && members.length === 0" :description="t('wiki.member.empty')" :image-size="60" />
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">{{ t('common.close') }}</el-button>
    </template>
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
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

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

const roleOptions = computed(() => [
  { value: 'owner', label: t('wiki.member.role.owner') },
  { value: 'admin', label: t('wiki.member.role.admin') },
  { value: 'editor', label: t('wiki.member.role.editor') },
  { value: 'viewer', label: t('wiki.member.role.viewer') }
])
const roleLabel = (r: string) => roleOptions.value.find((o) => o.value === r)?.label || r

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
    ElMessage.success(t('wiki.member.addSuccess'))
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
    ElMessage.success(t('wiki.member.roleUpdated'))
    await load()
    emit('changed')
  } catch {
    await load()
  }
}

async function removeUser(row: WikiSpaceMember) {
  await ElMessageBox.confirm(t('wiki.member.removeConfirm', displayName(row)), t('common.tip'), {
    type: 'warning'
  }).catch(() => Promise.reject(new Error('canceled')))
  try {
    await removeMember(props.spaceId, row.user_id)
    ElMessage.success(t('wiki.member.removeSuccess'))
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