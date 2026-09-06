<template>
  <el-select
    :model-value="modelValue"
    :disabled="disabled"
    filterable
    remote
    clearable
    reserve-keyword
    :remote-method="searchUsers"
    :loading="loading"
    placeholder="输入用户名搜索"
    style="width: 100%"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  >
    <el-option
      v-for="u in options"
      :key="u.id"
      :label="u.nickname || u.username"
      :value="u.id"
    >
      <span>{{ u.nickname || u.username }}</span>
      <span v-if="u.email" class="sub">{{ u.email }}</span>
    </el-option>
    <template #empty>
      <span style="color: #9ca3af">无匹配用户</span>
    </template>
  </el-select>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getUserList, type UserItem } from '@/api/modules/user'
import { toPageResult } from '@/types/pagination'

defineProps<{ modelValue: string; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const options = ref<UserItem[]>([])
const loading = ref(false)
const searched = ref(false)

async function searchUsers(keyword: string) {
  loading.value = true
  try {
    const res = toPageResult<UserItem>(await getUserList({ page: 1, page_size: 20, keyword: keyword.trim() }))
    options.value = res?.list ?? []
    searched.value = true
  } catch {
    options.value = []
  } finally {
    loading.value = false
  }
}

// 首次聚焦/打开时拉取候选（无关键字场景）
async function ensureLoaded() {
  if (!searched.value) await searchUsers('')
}
defineExpose({ ensureLoaded })
</script>

<style scoped>
.sub {
  float: right;
  color: #9ca3af;
  font-size: 12px;
}
</style>
