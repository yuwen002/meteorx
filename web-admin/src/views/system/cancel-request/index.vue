<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          :placeholder="t('cancelPage.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="query.status" :placeholder="t('cancelPage.statusFilter')" clearable style="width: 130px">
          <el-option :label="t('cancelPage.status.pending')" :value="1" />
          <el-option :label="t('cancelPage.status.approved')" :value="2" />
          <el-option :label="t('cancelPage.status.rejected')" :value="3" />
          <el-option :label="t('cancelPage.status.completed')" :value="4" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('common.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
      </div>

      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="tenant_name" :label="t('cancelPage.col.tenantName')" min-width="140" />
        <el-table-column prop="tenant_id" :label="t('cancelPage.col.tenantId')" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span style="font-family: monospace; color: #6b7280;">{{ row.tenant_id }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="reason" :label="t('cancelPage.col.reason')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" :label="t('cancelPage.col.status')" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="getStatusType(row.status)">{{ row.status_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="effective_at" :label="t('cancelPage.col.effectiveAt')" width="160">
          <template #default="{ row }">{{ row.effective_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="applied_at" :label="t('cancelPage.col.appliedAt')" width="160" />
        <el-table-column prop="review_remark" :label="t('cancelPage.col.reviewRemark')" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.review_remark || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('cancelPage.col.operation')" width="180" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 1">
              <el-button link type="success" @click="openApprove(row)">{{ t('cancelPage.action.approve') }}</el-button>
              <el-button link type="danger" @click="openReject(row)">{{ t('cancelPage.action.reject') }}</el-button>
            </template>
            <template v-else>
              <el-button link type="info" disabled>{{ t('cancelPage.action.processed') }}</el-button>
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
    <el-dialog v-model="approveVisible" :title="t('cancelPage.approveTitle')" width="520px" :close-on-click-modal="false">
      <el-alert
        :title="t('cancelPage.approveAlert')"
        type="warning"
        show-icon
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-form ref="approveFormRef" :model="approveForm" :rules="approveRules" label-width="90px">
        <el-form-item :label="t('cancelPage.form.effectiveLabel')" prop="effective_days">
          <el-radio-group v-model="approveForm.effective_days">
            <el-radio-button :value="0">{{ t('cancelPage.form.effectiveNow') }}</el-radio-button>
            <el-radio-button :value="3">{{ t('cancelPage.form.effective3') }}</el-radio-button>
            <el-radio-button :value="7">{{ t('cancelPage.form.effective7') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('cancelPage.form.remarkLabel')" prop="review_remark">
          <el-input
            v-model="approveForm.review_remark"
            type="textarea"
            :rows="3"
            :placeholder="t('cancelPage.form.remarkPlaceholder')"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="approveVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitApprove">{{ t('cancelPage.submitApprove') }}</el-button>
      </template>
    </el-dialog>

    <!-- 驳回注销申请弹窗 -->
    <el-dialog v-model="rejectVisible" :title="t('cancelPage.rejectTitle')" width="520px" :close-on-click-modal="false">
      <el-form ref="rejectFormRef" :model="rejectForm" :rules="rejectRules" label-width="90px">
        <el-form-item :label="t('cancelPage.form.rejectReasonLabel')" prop="review_remark">
          <el-input
            v-model="rejectForm.review_remark"
            type="textarea"
            :rows="4"
            :placeholder="t('cancelPage.form.rejectReasonPlaceholder')"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="danger" :loading="submitting" @click="submitReject">{{ t('cancelPage.submitReject') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { useI18n } from '@/composables/useI18n'
import {
  getCancelRequestList,
  approveCancelRequest,
  rejectCancelRequest,
  type CancelRequestItem
} from '@/api/modules/tenant'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const { t } = useI18n()

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

const approveRules = computed<FormRules>(() => ({
  effective_days: [{ required: true, message: t('cancelPage.form.effectiveRequired'), trigger: 'change' }]
}))
const rejectRules = computed<FormRules>(() => ({
  review_remark: [{ required: true, message: t('cancelPage.form.rejectReasonRequired'), trigger: 'blur' }]
}))

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
    ElMessage.success(t('cancelPage.approved'))
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
    ElMessage.success(t('cancelPage.rejected'))
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
