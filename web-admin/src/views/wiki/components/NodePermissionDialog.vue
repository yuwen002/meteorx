<template>
  <el-dialog
    :model-value="modelValue"
    :title="`节点权限 — ${title || '未命名'}`"
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
        添加授权
      </el-button>
    </div>
    <el-alert
      v-else
      type="warning"
      :closable="false"
      style="margin-bottom: 12px"
    >
      当前账号缺少用户管理（user:list）权限，无法搜索用户添加授权。
    </el-alert>
    <el-alert type="info" :closable="false" style="margin-bottom: 12px">
      节点级授权可精确控制单个文档/目录的可见与可编辑范围，优先级高于空间默认权限。
    </el-alert>

    <el-table v-loading="loading" :data="permissions" size="small">
      <el-table-column label="用户" min-width="200">
        <template #default="{ row }">
          <div>{{ displayName(row) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="权限" width="120">
        <template #default="{ row }">
          {{ permissionLabel(row.permission) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80" align="center">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">取消</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && permissions.length === 0" description="暂无单独授权" :image-size="60" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import {
  listNodePermissions,
  setNodePermission,
  removeNodePermission,
  type NodePermission
} from '@/api/modules/wiki'
import { useUserStore } from '@/stores/user'
import UserSelect from './UserSelect.vue'

const props = defineProps<{
  modelValue: boolean
  spaceId: string
  nodeId: string
  title: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; changed: [] }>()

const userStore = useUserStore()
const canListUsers = userStore.hasPermission('user:list')

const permissionOptions = [
  { value: 'view', label: '可查看' },
  { value: 'edit', label: '可编辑' },
  { value: 'delete', label: '可删除' }
]
const permissionLabel = (p: string) =>
  permissionOptions.find((o) => o.value === p)?.label || p

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
    ElMessage.success('授权成功')
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
    ElMessage.success('已取消授权')
    await load()
    emit('changed')
  } catch {
    /* 拦截器已提示 */
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
