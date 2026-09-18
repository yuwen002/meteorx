<template>
  <el-dialog v-model="visible" title="标签管理" width="600px">
    <div class="tag-manager">
      <div class="tag-create">
        <el-input
          v-model="newTagName"
          placeholder="输入标签名称"
          style="width: 200px"
          @keyup.enter="handleCreateTag"
        />
        <el-color-picker v-model="newTagColor" />
        <el-button type="primary" @click="handleCreateTag">创建标签</el-button>
      </div>

      <el-divider />

      <div class="tag-list">
        <el-tag
          v-for="tag in tags"
          :key="tag.id"
          :style="{
            '--el-tag-bg-color': tag.color,
            '--el-tag-border-color': tag.color,
            '--el-tag-text-color': getContrastColor(tag.color)
          }"
          class="tag-item"
          closable
          @close="handleDeleteTag(tag.id)"
        >
          {{ tag.name }}
        </el-tag>
        <el-empty v-if="tags.length === 0" description="暂无标签" />
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listTags, createTag as apiCreateTag, deleteTag as apiDeleteTag, type Tag } from '@/api/modules/wiki'

const visible = ref(false)
const tags = ref<Tag[]>([])
const newTagName = ref('')
const newTagColor = ref('#409EFF')

watch(visible, async (val) => {
  if (val) {
    await loadTags()
  }
})

async function loadTags() {
  try {
    tags.value = await listTags()
  } catch (error) {
    ElMessage.error('加载标签失败')
  }
}

async function handleCreateTag() {
  if (!newTagName.value.trim()) {
    ElMessage.warning('请输入标签名称')
    return
  }

  try {
    await apiCreateTag({
      name: newTagName.value,
      color: newTagColor.value
    })
    ElMessage.success('创建成功')
    newTagName.value = ''
    await loadTags()
  } catch (error) {
    ElMessage.error('创建失败')
  }
}

function getContrastColor(hex: string) {
  const c = hex.replace('#', '')
  const r = parseInt(c.substring(0, 2), 16)
  const g = parseInt(c.substring(2, 4), 16)
  const b = parseInt(c.substring(4, 6), 16)
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  return luminance > 0.5 ? '#333333' : '#ffffff'
}

function handleDeleteTag(id: string) {
  ElMessageBox.confirm('确定要删除此标签吗？', '提示', {
    type: 'warning'
  })
    .then(async () => {
      await apiDeleteTag(id)
      ElMessage.success('删除成功')
      await loadTags()
    })
    .catch(() => {})
}

defineExpose({
  open() {
    visible.value = true
  }
})
</script>

<style scoped>
.tag-manager {
  padding: 8px 0;
}

.tag-create {
  display: flex;
  gap: 12px;
  align-items: center;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 100px;
}

.tag-item {
  margin: 0;
}
</style>