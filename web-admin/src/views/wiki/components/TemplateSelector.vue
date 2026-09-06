<template>
  <el-dialog v-model="visible" title="选择文档模板" width="900px">
    <div class="template-selector">
      <div class="filter-bar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索模板..."
          style="width: 240px"
          clearable
        />
        <el-select
          v-model="selectedCategory"
          placeholder="选择分类"
          clearable
          style="width: 160px"
        >
          <el-option
            v-for="cat in categories"
            :key="cat"
            :label="cat"
            :value="cat"
          />
        </el-select>
      </div>

      <el-row :gutter="16" style="margin-top: 16px">
        <el-col v-for="template in filteredTemplates" :key="template.id" :span="8" style="margin-bottom: 16px">
          <el-card shadow="hover" class="template-card" @click="selectTemplate(template)">
            <div class="template-header">
              <el-icon :size="24"><Document /></el-icon>
              <span class="template-name">{{ template.name }}</span>
            </div>
            <div class="template-desc">{{ template.description || '暂无描述' }}</div>
            <div class="template-footer">
              <el-tag size="small">{{ template.category }}</el-tag>
              <el-tag v-if="template.is_public" size="small" type="success">公开</el-tag>
              <el-tag v-else size="small" type="info">私有</el-tag>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-empty v-if="filteredTemplates.length === 0" description="暂无模板" />
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Document } from '@element-plus/icons-vue'
import { listTemplates, type DocumentTemplate } from '@/api/modules/wiki'
import { ElMessage } from 'element-plus'

const emit = defineEmits<{
  select: [template: DocumentTemplate]
}>()

const visible = ref(false)
const templates = ref<DocumentTemplate[]>([])
const searchKeyword = ref('')
const selectedCategory = ref('')

const categories = computed(() => {
  const cats = new Set(templates.value.map((t) => t.category))
  return Array.from(cats)
})

const filteredTemplates = computed(() => {
  let result = templates.value

  if (selectedCategory.value) {
    result = result.filter((t) => t.category === selectedCategory.value)
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(
      (t) =>
        t.name.toLowerCase().includes(keyword) ||
        (t.description && t.description.toLowerCase().includes(keyword))
    )
  }

  return result
})

watch(visible, async (val) => {
  if (val) {
    await loadTemplates()
  }
})

async function loadTemplates() {
  try {
    templates.value = await listTemplates()
  } catch (error) {
    ElMessage.error('加载模板失败')
  }
}

function selectTemplate(template: DocumentTemplate) {
  emit('select', template)
  visible.value = false
}

defineExpose({
  open() {
    visible.value = true
  }
})
</script>

<style scoped>
.template-selector {
  padding: 8px 0;
}

.filter-bar {
  display: flex;
  gap: 12px;
}

.template-card {
  cursor: pointer;
  transition: transform 0.2s;
  height: 100%;
}

.template-card:hover {
  transform: translateY(-2px);
}

.template-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.template-name {
  font-size: 15px;
  font-weight: 600;
  color: #1f2937;
}

.template-desc {
  font-size: 13px;
  color: #6b7280;
  min-height: 36px;
  margin-bottom: 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.template-footer {
  display: flex;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid #f3f4f6;
}
</style>