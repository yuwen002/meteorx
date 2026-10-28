<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('wiki.perm.title', title || t('wiki.perm.untitled'))"
    width="600px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="load"
  >
    <div v-if="canListUsers" class="perm-toolbar">
      <UserSelect ref="userSelectRef" v-model="newUserId" class="user-picker" />
      <el-select v-model="newPermission" class="perm-picker">
        <el-option
          v-for="opt in permissionOptions"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>
      <el-button type="primary" :disabled="!newUserId" :loading="submitting" @click="addPermission">
        {{ t('wiki.perm.add') }}
      </el-button>
    </div>
    <el-alert
      v-else
      type="warning"
      :closable="false"
      style="margin-bottom: 12px"
    >
      {{ t('wiki.perm.noUserPerm') }}
    </el-alert>
    <el-alert type="info" :closable="false" style="margin-bottom: 12px">
      {{ t('wiki.perm.hint') }}
    </el-alert>

    <el-table v-loading="loading" :data="permissions" size="small">
      <el-table-column :label="t('wiki.perm.colUser')" min-width="200">
        <template #default="{ row }">
          <div>{{ displayName(row) }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="t('wiki.perm.colPerm')" width="120">
        <template #default="{ row }">
          {{ permissionLabel(row.permission) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="80" align="center">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">{{ t('wiki.perm.revoke') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && permissions.length === 0" :description="t('wiki.perm.empty')" :image-size="60" />
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">{{ t('common.close') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import {
  listNodePermissions,
  setNodePermission,
  removeNodePermission,
  type NodePermission
} from '@/api/modules/wiki'
import { useUserStore } from '@/stores/user'
import UserSelect from './UserSelect.vue'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
  spaceId: string
  nodeId: string
  title: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; changed: [] }>()

const userStore = useUserStore()
const canListUsers = userStore.hasPermission('user:list')

const permissionOptions = computed(() => [
  { value: 'view', label: t('wiki.perm.opt.view') },
  { value: 'edit', label: t('wiki.perm.opt.edit') },
  { value: 'delete', label: t('wiki.perm.opt.delete') }
])
const permissionLabel = (p: string) =>
  permissionOptions.value.find((o) => o.value === p)?.label || p

const permissions = ref<NodePermission[]>([])
const loading = ref(false)
const submitting = ref(false)
const newUserId = ref('')
const newPermission = ref('view')

function displayName(row: NodePermission) {
  return (row.user_name || row.user_id) as string
}

async function load() {
  if (!props.spaceId || !props.nodeId) return
  loading.value = true
  try {
    permissions.value = (await listNodePermissions(props.spaceId, props.nodeId)) ?? []
  } catch {
    permissions.value = []
  } finally {
    loading.value = false
  }
}

async function addPermission() {
  if (!newUserId.value) return
  submitting.value = true
  try {
    await setNodePermission(props.spaceId, props.nodeId, {
      user_id: newUserId.value,
      permission: newPermission.value
    })
    ElMessage.success(t('wiki.perm.grantSuccess'))
    newUserId.value = ''
    newPermission.value = 'view'
    await load()
    emit('changed')
  } finally {
    submitting.value = false
  }
}

async function remove(row: NodePermission) {
  try {
    await removeNodePermission(
      props.spaceId,
      props.nodeId,
      row.user_id,
      row.permission
    )
    ElMessage.success(t('wiki.perm.revokeSuccess'))
    await load()
    emit('changed')
  } catch {
    /* interceptor already reported the error */
  }
}
</script>

<style scoped>
.perm-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.user-picker {
  flex: 1;
}
.perm-picker {
  width: 120px;
}
</style>