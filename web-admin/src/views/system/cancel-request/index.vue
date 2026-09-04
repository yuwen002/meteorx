<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          placeholder="搜索租户名称/ID"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 130px">
          <el-option label="待审批" :value="1" />
          <el-option label="已通过" :value="2" />
          <el-option label="已驳回" :value="3" />
          <el-option label="已完成" :value="4" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
      </div>

      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="tenant_name" label="租户名称" min-width="140" />
        <el-table-column prop="tenant_id" label="租户ID" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span style="font-family: monospace; color: #6b7280;">{{ row.tenant_id }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="注销原因" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="getStatusType(row.status)">{{ row.status_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="effective_at" label="计划执行时间" width="160">
          <template #default="{ row }">{{ row.effective_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="applied_at" label="申请时间" width="160" />
        <el-table-column prop="review_remark" label="审批备注" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.review_remark || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 1">
              <el-button link type="success" @click="openApprove(row)">通过</el-button>
              <el-button link type="danger" @click="openReject(row)">驳回</el-button>
            </template>
            <template v-else>
              <el-button link type="info" disabled>已处理</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 通过注销申请弹窗 -->
    <el-dialog v-model="approveVisible" title="通过注销申请" width="520px" :close-on-click-modal="false">
      <el-alert
        title="通过后，租户将在指定时间后自动注销（删除租户及数据、取消订阅）。"
        type="warning"
        show-icon
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-form ref="approveFormRef" :model="approveForm" :rules="approveRules" label-width="90px">
        <el-form-item label="生效时间" prop="effective_days">
          <el-radio-group v-model="approveForm.effective_days">
            <el-radio-button :value="0">立即生效</el-radio-button>
            <el-radio-button :value="3">3天后</el-radio-button>
            <el-radio-button :value="7">7天后</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="审批备注" prop="review_remark">
          <el-input
            v-model="approveForm.review_remark"
            type="textarea"
            :rows="3"
            placeholder="请输入审批备注（可选）"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="approveVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitApprove">确定通过</el-button>
      </template>
    </el-dialog>

    <!-- 驳回注销申请弹窗 -->
    <el-dialog v-model="rejectVisible" title="驳回注销申请" width="520px" :close-on-click-modal="false">
      <el-form ref="rejectFormRef" :model="rejectForm" :rules="rejectRules" label-width="90px">
        <el-form-item label="驳回原因" prop="review_remark">
          <el-input
            v-model="rejectForm.review_remark"
            type="textarea"
            :rows="4"
            placeholder="请输入驳回原因（必填）"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectVisible = false">取消</el-button>
        <el-button type="danger" :loading="submitting" @click="submitReject">确定驳回</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  getCancelRequestList,
  approveCancelRequest,
  rejectCancelRequest,
  type CancelRequestItem
} from '@/api/modules/tenant'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<CancelRequestItem, { keyword: string; status?: number }>({
  fetchList: async (params) =>
    toPageResult(
      await getCancelRequestList({
        page: params.page,
        page_size: params.page_size,
        keyword: params.keyword || undefined,
        status: params.status
      })
    ),
  initialQuery: { keyword: '', status: undefined }
})
const loadList = reload

const approveVisible = ref(false)
const rejectVisible = ref(false)
const submitting = ref(false)
const approveFormRef = ref<FormInstance>()
const rejectFormRef = ref<FormInstance>()
const currentRow = ref<CancelRequestItem | null>(null)

const approveForm = reactive({ effective_days: 7, review_remark: '' })
const rejectForm = reactive({ review_remark: '' })

const approveRules: FormRules = {
  effective_days: [{ required: true, message: '请选择生效时间', trigger: 'change' }]
}
const rejectRules: FormRules = {
  review_remark: [{ required: true, message: '请输入驳回原因', trigger: 'blur' }]
}

function resetSearch() {
  reset()
}

function openApprove(row: CancelRequestItem) {
  currentRow.value = row
  approveForm.effective_days = 7
  approveForm.review_remark = ''
  approveVisible.value = true
}

function openReject(row: CancelRequestItem) {
  currentRow.value = row
  rejectForm.review_remark = ''
  rejectVisible.value = true
}

async function submitApprove() {
  if (!approveFormRef.value || !currentRow.value) return
  try {
    await approveFormRef.value.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    await approveCancelRequest(currentRow.value.id, {
      effective_days: approveForm.effective_days,
      review_remark: approveForm.review_remark || undefined
    })
    ElMessage.success('已通过注销申请')
    approveVisible.value = false
    loadList()
  } catch (e) {
    // 错误已在拦截器处理
  } finally {
    submitting.value = false
  }
}

async function submitReject() {
  if (!rejectFormRef.value || !currentRow.value) return
  try {
    await rejectFormRef.value.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    await rejectCancelRequest(currentRow.value.id, {
      review_remark: rejectForm.review_remark
    })
    ElMessage.success('已驳回注销申请')
    rejectVisible.value = false
    loadList()
  } catch (e) {
    // 错误已在拦截器处理
  } finally {
    submitting.value = false
  }
}

function getStatusType(status: number): string {
  const map: Record<number, string> = { 1: 'warning', 2: 'success', 3: 'danger', 4: 'info' }
  return map[status] || 'info'
}
</script>

<style scoped lang="scss">
.page {
  padding: 20px;
}
.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}
.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
