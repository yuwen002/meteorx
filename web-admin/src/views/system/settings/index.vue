<template>
  <div class="page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
      :title="t('settingsPage.alertTitle')"
      :description="t('settingsPage.alertDesc')"
    />

    <el-card shadow="never">
      <div class="toolbar">
        <div class="flex-1">
          <span v-if="dirtyCount > 0" class="dirty-hint">{{ t('settingsPage.dirtyHint', dirtyCount) }}</span>
        </div>
        <el-button :disabled="dirtyCount === 0" @click="resetDraft">{{ t('settingsPage.reset') }}</el-button>
        <el-button type="primary" :disabled="dirtyCount === 0" :loading="saving" @click="handleSave">
          {{ t('settingsPage.save') }}
        </el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="key" :label="t('settingsPage.col.key')" min-width="180">
          <template #default="{ row }">
            <el-tag effect="plain" size="small">{{ row.key }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" :label="t('settingsPage.col.description')" min-width="240" show-overflow-tooltip />
        <el-table-column prop="value_type" :label="t('settingsPage.col.type')" width="100">
          <template #default="{ row }">{{ typeLabel(row.value_type) }}</template>
        </el-table-column>
        <el-table-column :label="t('settingsPage.col.value')" min-width="220">
          <template #default="{ row }">
            <el-switch
              v-if="row.value_type === 'bool'"
              :model-value="draft[row.key] === 'true'"
              @update:model-value="(v: boolean) => (draft[row.key] = v ? 'true' : 'false')"
            />
            <el-input-number
              v-else-if="row.value_type === 'int'"
              :model-value="Number(draft[row.key]) || 0"
              :min="0"
              controls-position="right"
              style="width: 140px"
              @update:model-value="(v?: number) => (draft[row.key] = String(v ?? 0))"
            />
            <el-input v-else v-model="draft[row.key]" :placeholder="row.default_value" />
          </template>
        </el-table-column>
        <el-table-column :label="t('settingsPage.col.source')" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.source === 'db'" type="success" size="small" effect="plain">{{ t('settingsPage.source.custom') }}</el-tag>
            <el-tag v-else type="info" size="small" effect="plain">{{ t('settingsPage.source.default') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="default_value" :label="t('settingsPage.col.defaultValue')" min-width="140" show-overflow-tooltip />
        <el-table-column prop="updated_at" :label="t('settingsPage.col.updatedAt')" width="170" />
      </el-table>

      <el-empty v-if="!loading && list.length === 0" :description="t('settingsPage.empty')" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useI18n } from '@/composables/useI18n'
import {
  listSystemSettings,
  updateSystemSettings,
  type SettingItem,
  type SettingUpdateItem
} from '@/api/modules/settings'

const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const list = ref<SettingItem[]>([])
// draft[key] 为用户编辑中的字符串值；与 list 中的 value 不同的键即为待保存项
const draft = reactive<Record<string, string>>({})

const dirtyCount = computed(
  () => list.value.filter((item) => draft[item.key] !== undefined && draft[item.key] !== item.value).length
)

function typeLabel(vt: string) {
  const key = `settingsPage.type.${vt}`
  const label = t(key)
  return label === key ? vt : label
}

// 以后端返回为准重建草稿基线（保存成功后回到干净状态）
function reseedDraft(items: SettingItem[]) {
  list.value = items
  for (const key of Object.keys(draft)) delete draft[key]
  for (const item of items) draft[item.key] = item.value
}

async function loadList() {
  loading.value = true
  try {
    const items = await listSystemSettings()
    reseedDraft(items || [])
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

function resetDraft() {
  for (const item of list.value) draft[item.key] = item.value
}

async function handleSave() {
  const changes: SettingUpdateItem[] = list.value
    .filter((item) => draft[item.key] !== item.value)
    .map((item) => ({ key: item.key, value: draft[item.key] }))
  if (changes.length === 0) return

  saving.value = true
  try {
    const items = await updateSystemSettings(changes)
    reseedDraft(items || [])
    ElMessage.success(t('settingsPage.savedSuccess'))
  } catch (e) {
    // 后端整批校验：400 时合法项也未落库，保留草稿供用户修正
    ElMessage.error((e as Error)?.message || t('settingsPage.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(loadList)
</script>

<style scoped>
.page {
  padding: 0;
}
.toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.flex-1 {
  flex: 1;
}
.dirty-hint {
  font-size: 13px;
  color: #e6a23c;
}
</style>
