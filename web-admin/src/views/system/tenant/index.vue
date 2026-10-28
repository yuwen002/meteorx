<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.name"
          :placeholder="t('tenantPage.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="query.status" :placeholder="t('tenantPage.filterStatus')" clearable style="width: 120px">
          <el-option :label="t('common.status.enabled')" :value="1" />
          <el-option :label="t('common.status.disabled')" :value="0" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('tenantPage.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="openRecycleBin">
          <el-icon><Delete /></el-icon>{{ t('tenantPage.recycleBin') }}
        </el-button>
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>{{ t('tenantPage.createTenant') }}
        </el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-button type="danger" @click="handleBatchDelete">{{ t('tenantPage.batchDelete') }}</el-button>
        <el-button type="warning" @click="handleBatchDisable">{{ t('tenantPage.batchPause') }}</el-button>
        <el-button type="success" @click="handleBatchEnable">{{ t('tenantPage.batchEnable') }}</el-button>
        <span style="margin-left: 8px; color: #666">{{ t('tenantPage.selectedCount', selectedIds.length) }}</span>
      </div>

      <!-- 列表 -->
      <el-table
        v-loading="loading"
        :data="list"
        border
        stripe
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" :label="t('tenantPage.colTenantId')" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 6px;">
              <span style="font-family: monospace; color: #6b7280; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ row.id }}</span>
              <el-button
                link
                type="primary"
                size="small"
                :icon="CopyDocument"
                :title="t('tenantPage.copy')"
                @click="copyId(row.id)"
              />
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="name" :label="t('tenantPage.colName')" min-width="150" />
        <el-table-column prop="domain" :label="t('tenantPage.colDomain')" min-width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.domain }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="contact_email" :label="t('tenantPage.colContactEmail')" min-width="180" />
        <el-table-column prop="contact_phone" :label="t('tenantPage.colContactPhone')" min-width="140">
          <template #default="{ row }">
            <span v-if="row.contact_phone">{{ row.contact_phone }}</span>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="region" :label="t('tenantPage.colRegion')" width="100" />
        <el-table-column :label="t('tenantPage.colUserCount')" width="90">
          <template #default="{ row }">
            <span>{{ row.user_count ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colStatus')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'warning'">
              {{ row.status === 1 ? t('common.status.enabled') : t('tenantPage.paused') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colPlan')" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.plan_name" :type="row.plan_expired ? 'danger' : 'primary'" effect="plain" size="small">
              {{ row.plan_name }}<template v-if="row.plan_expired">{{ t('tenantPage.expiredTag') }}</template>
            </el-tag>
            <el-tag v-else type="info" effect="plain" size="small">{{ t('tenantPage.notActivated') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('tenantPage.colCreatedAt')" width="180" />
        <el-table-column :label="t('tenantPage.colAction')" width="400" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">{{ t('common.edit') }}</el-button>
            <el-button link type="info" @click="openViewUsers(row)">{{ t('tenantPage.viewUsers') }}</el-button>
            <el-button link type="warning" @click="openPlanDialog(row)">{{ t('tenantPage.planBtn') }}</el-button>
            <el-button
              link
              :type="row.status === 1 ? 'warning' : 'success'"
              @click="handleToggleStatus(row)"
            >
              {{ row.status === 1 ? t('tenantPage.pause') : t('tenantPage.resume') }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
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
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 查看用户弹窗 -->
    <el-dialog
      v-model="usersDialogVisible"
      :title="t('tenantPage.tenantUsersTitle', currentTenant?.name || '')"
      width="1000px"
      :close-on-click-modal="false"
    >
      <!-- 搜索栏 -->
      <div class="search-bar" style="margin-bottom: 16px;">
        <el-input
          v-model="userSearch.keyword"
          :placeholder="t('tenantPage.userSearchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadTenantUsers"
        />
        <el-button type="primary" @click="loadTenantUsers">
          <el-icon><Search /></el-icon>{{ t('tenantPage.query') }}
        </el-button>
        <el-button @click="resetUserSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleUserRecycleBin">
          <el-icon><Delete /></el-icon>{{ isUserRecycleBin ? t('tenantPage.backToList') : t('tenantPage.recycleBin') }}
        </el-button>
        <el-button v-if="!isUserRecycleBin" type="success" @click="openCreateUserDialog">
          <el-icon><Plus /></el-icon>{{ t('tenantPage.createUser') }}
        </el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="userSelectedIds.length > 0 && !isUserRecycleBin" class="batch-bar" style="margin-bottom: 16px;">
        <el-button type="danger" size="small" @click="handleBatchDeleteUsers">{{ t('tenantPage.batchDelete') }}</el-button>
        <el-button type="warning" size="small" @click="handleBatchDisableUsers">{{ t('tenantPage.batchDisable') }}</el-button>
        <el-button type="success" size="small" @click="handleBatchEnableUsers">{{ t('tenantPage.batchEnable') }}</el-button>
        <span style="margin-left: 8px; color: #666">{{ t('tenantPage.selectedCount', userSelectedIds.length) }}</span>
      </div>

      <!-- 用户列表 -->
      <el-table
        v-loading="userLoading"
        :data="userList"
        border
        stripe
        @selection-change="handleUserSelectionChange"
      >
        <el-table-column v-if="!isUserRecycleBin" type="selection" width="55" />
        <el-table-column prop="username" :label="t('auth.username')" min-width="120" />
        <el-table-column prop="nickname" :label="t('register.nickname')" min-width="120" />
        <el-table-column prop="email" :label="t('common.email')" min-width="180" />
        <el-table-column :label="t('tenantPage.colRole')" min-width="150">
          <template #default="{ row }">
            <el-tag
              v-for="role in row.role_list"
              :key="role.id"
              size="small"
              type="primary"
              style="margin-right: 4px; margin-bottom: 4px;"
            >
              {{ role.name }}
            </el-tag>
            <span v-if="!row.role_list || row.role_list.length === 0" style="color: #909399; font-size: 12px;">{{ t('tenantPage.noRole') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('tenantPage.colStatus')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? t('common.status.enabled') : t('common.status.disabled') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('tenantPage.colCreatedAt')" width="160" />
        <el-table-column v-if="isUserRecycleBin" prop="deleted_at" :label="t('tenantPage.colDeletedAt')" width="160" />
        <el-table-column :label="t('tenantPage.colAction')" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isUserRecycleBin">
              <el-button link type="success" @click="handleRestoreUser(row)">{{ t('tenantPage.restore') }}</el-button>
              <el-button link type="danger" @click="handlePermanentDeleteUser(row)">{{ t('tenantPage.permanentDelete') }}</el-button>
            </template>
            <template v-else>
              <el-button link type="primary" @click="openEditUserDialog(row)">{{ t('common.edit') }}</el-button>
              <el-button link type="warning" @click="openResetPasswordDialog(row)">{{ t('auth.resetPassword') }}</el-button>
              <el-button link type="info" @click="openViewUserRoles(row)">{{ t('tenantPage.viewRoles') }}</el-button>
              <el-button link type="primary" @click="openLoginLogs(row)">{{ t('tenantPage.loginLogs') }}</el-button>
              <el-button
                link
                :type="row.status === 1 ? 'warning' : 'success'"
                @click="handleToggleUserStatus(row)"
              >
                {{ row.status === 1 ? t('common.status.disabled') : t('common.status.enabled') }}
              </el-button>
              <el-button link type="danger" @click="handleDeleteUser(row)">{{ t('common.delete') }}</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination" style="margin-top: 16px;">
        <el-pagination
          v-if="!isUserRecycleBin"
          v-model:current-page="userPage"
          v-model:page-size="userPageSize"
          :total="userTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadTenantUsers"
          @current-change="loadTenantUsers"
        />
        <div v-else style="text-align: center; color: #909399; font-size: 14px;">
          {{ t('tenantPage.deletedRecordsCount', userTotal) }}
        </div>
      </div>
    </el-dialog>

    <!-- 新增/编辑用户弹窗 -->
    <!-- 重置密码弹窗 -->
    <el-dialog
      v-model="resetPwdDialogVisible"
      :title="t('auth.resetPassword')"
      width="400px"
      :close-on-click-modal="false"
      @close="closeResetPwdDialog"
    >
      <p style="margin-bottom: 16px;">{{ t('tenantPage.resetPwdTip', resetPwdUser?.username || '') }}</p>
      <el-form
        ref="resetPwdFormRef"
        :model="resetPwdForm"
        :rules="resetPwdRules"
        label-width="100px"
      >
        <el-form-item :label="t('auth.newPassword')" prop="new_password">
          <el-input
            v-model="resetPwdForm.new_password"
            type="password"
            :placeholder="t('tenantPage.newPwdPlaceholder')"
            show-password
          />
        </el-form-item>
        <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
          <el-input
            v-model="resetPwdForm.confirm_password"
            type="password"
            :placeholder="t('tenantPage.confirmPwdPlaceholder')"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeResetPwdDialog">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="resetPwdSaving" @click="submitResetPassword">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 登录日志弹窗 -->
    <el-dialog
      v-model="loginLogsDialogVisible"
      :title="t('tenantPage.loginLogsTitle', loginLogsUser?.username || '')"
      width="900px"
      :close-on-click-modal="false"
    >
      <el-table v-loading="loginLogsLoading" :data="loginLogsList" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (loginLogsPage - 1) * loginLogsPageSize + i + 1" />
        <el-table-column prop="action" :label="t('tenantPage.colAction')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.action === 'login' ? 'success' : 'info'">
              {{ row.action === 'login' ? t('tenantPage.actionLogin') : t('tenantPage.actionLogout') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result" :label="t('tenantPage.colResult')" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
              {{ row.result === 'success' ? t('tenantPage.resultSuccess') : t('tenantPage.resultFail') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" :label="t('tenantPage.colIp')" width="140" />
        <el-table-column prop="user_agent" :label="t('tenantPage.colUserAgent')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="created_at" :label="t('tenantPage.colTime')" width="180" />
      </el-table>
      <div class="pagination" style="margin-top: 16px; display: flex; justify-content: flex-end;">
        <el-pagination
          v-model:current-page="loginLogsPage"
          v-model:page-size="loginLogsPageSize"
          :page-sizes="[10, 20, 50]"
          :total="loginLogsTotal"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLoginLogs"
          @current-change="loadLoginLogs"
        />
      </div>
    </el-dialog>

    <!-- 查看用户角色弹窗 -->
    <el-dialog
      v-model="userRolesDialogVisible"
      :title="t('tenantPage.userRolesTitle', currentUser?.username || '')"
      width="500px"
      :close-on-click-modal="false"
    >
      <div v-loading="userRolesLoading">
        <div v-if="currentUserRoles.length > 0">
          <el-table :data="currentUserRoles" border stripe size="small">
            <el-table-column prop="name" :label="t('tenantPage.colRoleName')" min-width="120" />
            <el-table-column prop="code" :label="t('tenantPage.colRoleCode')" min-width="120" />
            <el-table-column :label="t('tenantPage.colAction')" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" size="small" @click="handleRemoveUserRole(row)">{{ t('tenantPage.remove') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 16px; text-align: center;">
            <el-button type="danger" @click="handleRemoveAllUserRoles">{{ t('tenantPage.removeAllRoles') }}</el-button>
          </div>
        </div>
        <el-empty v-else :description="t('tenantPage.noRolesForUser')" />
      </div>
    </el-dialog>

    <el-dialog
      v-model="userDialogVisible"
      :title="userDialogMode === 'create' ? t('tenantPage.createUser') : t('tenantPage.editUser')"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="userFormRef" :model="userForm" :rules="userRules" label-width="80px">
        <el-form-item :label="t('auth.username')" prop="username">
          <el-input v-model="userForm.username" :placeholder="t('tenantPage.usernamePlaceholder')" :disabled="userDialogMode === 'edit'" />
        </el-form-item>
        <el-form-item v-if="userDialogMode === 'create'" :label="t('auth.password')" prop="password">
          <el-input v-model="userForm.password" type="password" show-password :placeholder="t('tenantPage.passwordPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('register.nickname')" prop="nickname">
          <el-input v-model="userForm.nickname" :placeholder="t('tenantPage.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('common.email')" prop="email">
          <el-input v-model="userForm.email" :placeholder="t('tenantPage.emailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.colDepartment')" prop="department">
          <el-input v-model="userForm.department" :placeholder="t('tenantPage.departmentPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.colPosition')" prop="position">
          <el-input v-model="userForm.position" :placeholder="t('tenantPage.positionPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item v-if="userDialogMode === 'edit'" :label="t('tenantPage.colRemark')" prop="remark">
          <el-input v-model="userForm.remark" type="textarea" :rows="2" :placeholder="t('tenantPage.remarkPlaceholder')" maxlength="255" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.colRole')" prop="role_ids">
          <el-select
            v-model="userForm.role_ids"
            multiple
            :placeholder="t('tenantPage.selectRolePlaceholder')"
            style="width: 100%"
            :loading="roleLoading"
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="userSaving" @click="submitUserForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? t('tenantPage.createTenant') : t('tenantPage.editTenant')"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('tenantPage.fieldName')" prop="name">
          <el-input v-model="form.name" :placeholder="t('tenantPage.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldDomain')" prop="domain">
          <el-input v-model="form.domain" :placeholder="t('tenantPage.domainPlaceholder')" :disabled="dialogMode === 'edit'">
            <template #append>.meteorx.com</template>
          </el-input>
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldContactEmail')" prop="contact_email">
          <el-input v-model="form.contact_email" :placeholder="t('tenantPage.contactEmailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldContactPhone')" prop="contact_phone">
          <el-input v-model="form.contact_phone" :placeholder="t('tenantPage.contactPhonePlaceholder')" maxlength="20" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldIndustry')" prop="industry">
          <el-input v-model="form.industry" :placeholder="t('tenantPage.industryPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldRegion')" prop="region">
          <el-input v-model="form.region" :placeholder="t('tenantPage.regionPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('tenantPage.fieldDescription')" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" :placeholder="t('tenantPage.descriptionPlaceholder')" />
        </el-form-item>
        <el-form-item label="Logo" prop="logo">
          <el-input v-model="form.logo" :placeholder="t('tenantPage.logoPlaceholder')" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" :label="t('tenantPage.fieldStatus')" prop="status">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">{{ t('common.status.enabled') }}</el-radio>
            <el-radio :value="0">{{ t('common.status.disabled') }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 初始管理员信息（仅创建时显示） -->
        <template v-if="dialogMode === 'create'">
          <el-divider>{{ t('tenantPage.adminSection') }}</el-divider>
          <el-form-item :label="t('auth.username')" prop="admin_user.username">
            <el-input v-model="form.admin_user.username" :placeholder="t('tenantPage.adminUsernamePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('auth.password')" prop="admin_user.password">
            <el-input v-model="form.admin_user.password" type="password" show-password :placeholder="t('tenantPage.passwordPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('register.nickname')" prop="admin_user.nickname">
            <el-input v-model="form.admin_user.nickname" :placeholder="t('tenantPage.adminNicknamePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('common.email')" prop="admin_user.email">
            <el-input v-model="form.admin_user.email" :placeholder="t('tenantPage.adminEmailPlaceholder')" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 套餐管理弹窗 -->
    <el-dialog
      v-model="planDialogVisible"
      :title="t('tenantPage.planConfigTitle', currentPlanTenant?.name || '')"
      width="560px"
      :close-on-click-modal="false"
    >
      <template v-if="currentPlan">
        <div class="plan-info">
          <div class="plan-info-item">
            <span class="label">{{ t('tenantPage.planCurrent') }}</span>
            <el-tag type="primary" effect="plain">{{ currentPlan.plan_name }}</el-tag>
            <el-tag v-if="currentPlan.status === 2" type="danger" effect="plain" style="margin-left: 6px">{{ t('tenantPage.expired') }}</el-tag>
          </div>
          <div class="plan-info-item">
            <span class="label">{{ t('tenantPage.planUserUsage') }}</span>
            <el-progress :percentage="planUsagePercent" :status="planUsagePercent >= 100 ? 'exception' : undefined" style="width: 300px" />
            <span class="usage-text">{{ currentPlan.current_users }} / {{ currentPlan.user_limit === -1 ? t('tenantPage.planUnlimited') : currentPlan.user_limit }}</span>
          </div>
          <div class="plan-info-item">
            <span class="label">{{ t('tenantPage.planStorageUsage') }}</span>
            <el-progress :percentage="planStoragePercent" :status="planStoragePercent >= 100 ? 'exception' : undefined" style="width: 300px" />
            <span class="usage-text">{{ formatSize(currentPlan.current_storage) }} / {{ currentPlan.storage_limit === -1 ? t('tenantPage.planUnlimited') : formatSize(currentPlan.storage_limit) }}</span>
          </div>
          <div class="plan-info-item">
            <span class="label">{{ t('tenantPage.planWikiSpace') }}</span>
            <el-progress :percentage="planWikiPercent" :status="planWikiPercent >= 100 ? 'exception' : undefined" style="width: 300px" />
            <span class="usage-text">{{ currentPlan.current_wiki_spaces }} / {{ currentPlan.wiki_space_limit === -1 ? t('tenantPage.planUnlimited') : currentPlan.wiki_space_limit }}</span>
          </div>
          <div class="plan-info-item">
            <span class="label">{{ t('tenantPage.planStartedAt') }}</span>
            <span>{{ currentPlan.started_at || '-' }}</span>
          </div>
          <div v-if="currentPlan.expires_at" class="plan-info-item">
            <span class="label">{{ t('tenantPage.planExpiresAt') }}</span>
            <span>{{ currentPlan.expires_at }}{{ t('tenantPage.planRemainDays', currentPlan.effective_days ?? 0) }}</span>
          </div>
          <div v-else class="plan-info-item">
            <span class="label">{{ t('tenantPage.planExpiresAt') }}</span>
            <span>{{ t('tenantPage.planForever') }}</span>
          </div>
        </div>
        <el-divider />
      </template>
      <el-alert v-else :title="t('tenantPage.planNone')" type="info" show-icon :closable="false" style="margin-bottom: 16px" />
      <template v-if="planList.length > 0">
        <div class="plan-info-item">
          <span class="label">{{ t('tenantPage.planSelect') }}</span>
          <el-select v-model="assignForm.plan_id" :placeholder="t('tenantPage.planSelectPlaceholder')" style="width: 300px" :loading="planLoading">
            <el-option v-for="p in planList" :key="p.id" :label="planOptionLabel(p)" :value="p.id" />
          </el-select>
        </div>
        <div class="plan-info-item">
          <span class="label">{{ t('tenantPage.planExpiresAt') }}</span>
          <el-date-picker
            v-model="assignForm.expires_at"
            type="datetime"
            :placeholder="t('tenantPage.planExpiryPlaceholder')"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 300px"
          />
        </div>
      </template>
      <el-alert v-else :title="t('tenantPage.planNoAvailable')" type="warning" show-icon :closable="false" />
      <template #footer>
        <el-button @click="planDialogVisible = false">{{ t('tenantPage.close') }}</el-button>
        <el-button type="primary" :loading="planSaving" :disabled="!assignForm.plan_id || planList.length === 0" @click="submitPlanAssign">
          {{ t('tenantPage.planSave') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus, Delete, CopyDocument } from '@element-plus/icons-vue'
import {
  getTenantList,
  getTenantDetail,
  createTenant,
  updateTenant,
  updateTenantStatus,
  deleteTenant,
  batchUpdateTenantStatus,
  batchDeleteTenants,
  type TenantItem,
  type CreateTenantParams,
  type UpdateTenantParams
} from '@/api/modules/tenant'
import {
  getTenantUsers,
  getAllDeletedTenantUsers,
  createTenantUser,
  updateTenantUser,
  deleteTenantUser,
  updateTenantUserStatus,
  batchDeleteTenantUsers,
  batchUpdateTenantUserStatus,
  restoreTenantUser,
  permanentDeleteTenantUser,
  resetTenantUserPassword,
  type UserItem,
  type UserListParams,
  type UserCreateParams,
  type UserUpdateParams
} from '@/api/modules/user'
import { getAuditLogList, type AuditLogItem } from '@/api/modules/audit'
import { getRolesForSelect, getUserRoles, removeUserRole, removeAllUserRoles, type RoleItem } from '@/api/modules/role'
import { getPlanSelect, getTenantPlan, assignTenantPlan, type PlanItem, type CurrentPlan } from '@/api/modules/plan'
import { useTableList } from '@/composables/useTableList'
import { toPageResult, type PageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const router = useRouter()
const route = useRoute()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<TenantItem, { name: string; status?: number }>({
  fetchList: async (params) => {
    const req: { page: number; page_size: number; name?: string; status?: number } = { page: params.page, page_size: params.page_size }
    if (params.name) req.name = params.name
    if (params.status !== undefined && params.status !== null) req.status = params.status
    const res = await getTenantList(req)
    return toPageResult(res)
  },
  initialQuery: { name: '', status: undefined },
  immediate: false
})
const loadList = reload
const selectedIds = ref<string[]>([])

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)

// 查看用户相关
const usersDialogVisible = ref(false)
const currentTenant = ref<TenantItem | null>(null)
const userList = ref<UserItem[]>([])
const userPage = ref(1)
const userPageSize = ref(10)
const userTotal = ref(0)
const userLoading = ref(false)
const userSearch = reactive({ keyword: '' })
const userSelectedIds = ref<string[]>([])
const isUserRecycleBin = ref(false)

// 用户新增/编辑相关
const userDialogVisible = ref(false)
const userDialogMode = ref<'create' | 'edit'>('create')
const userFormRef = ref<FormInstance>()
const userSaving = ref(false)
const editingUserId = ref<string | null>(null)
const userForm = reactive<UserCreateParams & UserUpdateParams & { role_ids: string[] }>({
  username: '',
  password: '',
  nickname: '',
  email: '',
  department: '',
  position: '',
  remark: '',
  role_ids: []
})
const userRules = computed<FormRules>(() => ({
  username: [{ required: true, message: t('tenantPage.usernamePlaceholder'), trigger: 'blur' }],
  password: [
    { required: true, message: t('tenantPage.passwordPlaceholder'), trigger: 'blur' },
    { min: 6, max: 32, message: t('tenantPage.pwdLength'), trigger: 'blur' }
  ],
  nickname: [{ required: true, message: t('tenantPage.nicknamePlaceholder'), trigger: 'blur' }],
  role_ids: [{ required: true, message: t('tenantPage.selectRolePlaceholder'), trigger: 'change', type: 'array' }]
}))

// 角色列表
const roleList = ref<RoleItem[]>([])
const roleLoading = ref(false)

// 查看用户角色相关
const userRolesDialogVisible = ref(false)
const currentUser = ref<UserItem | null>(null)
const currentUserRoles = ref<RoleItem[]>([])
const userRolesLoading = ref(false)

// 重置密码相关
const resetPwdDialogVisible = ref(false)
const resetPwdFormRef = ref<FormInstance>()
const resetPwdSaving = ref(false)
const resetPwdUser = ref<UserItem | null>(null)
const resetPwdForm = reactive({
  new_password: '',
  confirm_password: ''
})
const resetPwdRules = computed<FormRules>(() => ({
  new_password: [
    { required: true, message: t('tenantPage.newPwdPlaceholder'), trigger: 'blur' },
    { min: 6, max: 32, message: t('tenantPage.pwdLength'), trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: t('tenantPage.pwdConfirmRequired'), trigger: 'blur' },
    { min: 6, max: 32, message: t('tenantPage.pwdLength'), trigger: 'blur' }
  ]
}))

// 登录日志相关
const loginLogsDialogVisible = ref(false)
const loginLogsLoading = ref(false)
const loginLogsUser = ref<UserItem | null>(null)
const loginLogsList = ref<AuditLogItem[]>([])
const loginLogsPage = ref(1)
const loginLogsPageSize = ref(10)
const loginLogsTotal = ref(0)

// 套餐管理相关
const planDialogVisible = ref(false)
const planLoading = ref(false)
const planSaving = ref(false)
const currentPlanTenant = ref<TenantItem | null>(null)
const currentPlan = ref<CurrentPlan | null>(null)
const planList = ref<PlanItem[]>([])
const assignForm = reactive<{ plan_id: string; expires_at: string }>({
  plan_id: '',
  expires_at: ''
})
const planUsagePercent = computed(() => {
  if (!currentPlan.value || currentPlan.value.user_limit === -1 || currentPlan.value.user_limit <= 0) return 0
  return Math.min(100, Math.round((currentPlan.value.current_users / currentPlan.value.user_limit) * 100))
})
const planStoragePercent = computed(() => {
  if (!currentPlan.value || currentPlan.value.storage_limit === -1 || currentPlan.value.storage_limit <= 0) return 0
  return Math.min(100, Math.round((currentPlan.value.current_storage / currentPlan.value.storage_limit) * 100))
})
const planWikiPercent = computed(() => {
  if (!currentPlan.value || currentPlan.value.wiki_space_limit === -1 || currentPlan.value.wiki_space_limit <= 0) return 0
  return Math.min(100, Math.round((currentPlan.value.current_wiki_spaces / currentPlan.value.wiki_space_limit) * 100))
})

// formatSize 将字节数格式化为可读大小
function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let val = bytes
  let i = 0
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return `${val.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

// planOptionLabel 拼接套餐下拉选项文本
function planOptionLabel(p: PlanItem): string {
  const limit = p.user_limit === -1 ? t('tenantPage.planUnlimited') : `${p.user_limit} ${t('tenantPage.planPeopleUnit')}`
  return t('tenantPage.planOptionLabel', p.name, limit)
}

// 打开套餐配置弹窗
async function openPlanDialog(row: TenantItem) {
  currentPlanTenant.value = row
  currentPlan.value = null
  assignForm.plan_id = ''
  assignForm.expires_at = ''
  planDialogVisible.value = true
  planLoading.value = true
  try {
    const [plans, plan] = await Promise.all([getPlanSelect(), getTenantPlan(row.id)])
    planList.value = plans || []
    if (plan) {
      currentPlan.value = plan
      assignForm.plan_id = plan.plan_id
      if (plan.expires_at) assignForm.expires_at = plan.expires_at
    }
  } catch (e) {
    planList.value = planList.value || []
  } finally {
    planLoading.value = false
  }
}

/**
 * 从订阅临期站内信（/system/tenant?tenant_id=xxx）定位该租户的套餐配置。
 * 订阅挂在租户下而不是套餐定义下：套餐管理页只有套餐本身，看不到「这个租户何时到期」。
 * 不依赖这一行在当前页：筛选、分页与列表刷新都可能把它挡在外面，所以按 ID 取详情再开弹窗。
 */
async function openPlanById(id: string) {
  try {
    const tenant = await getTenantDetail(id)
    if (tenant) await openPlanDialog(tenant)
  } catch {
    // 失败提示由拦截器统一弹出（租户可能已注销）
  }
}

// 已在本页时再点另一条到期消息只有 query 变化（不会重新挂载），因此 watch 而非 onMounted 读一次
watch(
  () => route.query.tenant_id,
  id => {
    if (typeof id === 'string' && id) openPlanById(id)
  },
  { immediate: true }
)

// 提交套餐分配
async function submitPlanAssign() {
  if (!currentPlanTenant.value?.id || !assignForm.plan_id) return
  planSaving.value = true
  try {
    await assignTenantPlan(currentPlanTenant.value.id, {
      plan_id: assignForm.plan_id,
      expires_at: assignForm.expires_at || undefined
    })
    ElMessage.success(t('tenantPage.planSaveSuccess'))
    planDialogVisible.value = false
    loadList()
  } catch (e) {
    // 错误已在拦截器处理
  } finally {
    planSaving.value = false
  }
}

const form = reactive<CreateTenantParams & UpdateTenantParams>({
  name: '',
  domain: '',
  description: '',
  contact_email: '',
  contact_phone: '',
  industry: '',
  region: '',
  logo: '',
  status: 1,
  admin_user: {
    username: '',
    password: '',
    nickname: '',
    email: ''
  }
})

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('tenantPage.namePlaceholder'), trigger: 'blur' }],
  domain: [{ required: true, message: t('tenantPage.domainPlaceholder'), trigger: 'blur' }],
  'admin_user.username': [{ required: true, message: t('tenantPage.adminUsernamePlaceholder'), trigger: 'blur' }],
  'admin_user.password': [
    { required: true, message: t('tenantPage.adminPasswordPlaceholder'), trigger: 'blur' },
    { min: 6, max: 32, message: t('tenantPage.pwdLength'), trigger: 'blur' }
  ],
  'admin_user.nickname': [{ required: true, message: t('tenantPage.adminNicknamePlaceholder'), trigger: 'blur' }]
}))

function resetSearch() {
  reset()
}

function handleSelectionChange(selection: TenantItem[]) {
  selectedIds.value = selection.map((row) => row.id).filter(Boolean) as string[]
}

async function copyId(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('tenantPage.copied'))
  } catch {
    ElMessage.error(t('tenantPage.copyFailed'))
  }
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.domain = ''
  form.description = ''
  form.contact_email = ''
  form.contact_phone = ''
  form.industry = ''
  form.region = ''
  form.logo = ''
  form.status = 1
  form.admin_user = { username: '', password: '', nickname: '', email: '' }
  dialogVisible.value = true
}

function openEditDialog(row: TenantItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.domain = row.domain
  form.description = row.description || ''
  form.contact_email = row.contact_email || ''
  form.contact_phone = row.contact_phone || ''
  form.industry = row.industry || ''
  form.region = row.region || ''
  form.logo = row.logo || ''
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (dialogMode.value === 'create') {
        await createTenant(form as CreateTenantParams)
        ElMessage.success(t('tenantPage.createSuccess'))
      } else if (editingId.value) {
        await updateTenant(editingId.value, form as UpdateTenantParams)
        ElMessage.success(t('tenantPage.updateSuccess'))
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

async function handleToggleStatus(row: TenantItem) {
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? t('tenantPage.resume') : t('tenantPage.pause')
  try {
    if (newStatus === 0) {
      await ElMessageBox.confirm(
        t('tenantPage.pauseConfirm', row.name),
        t('tenantPage.pauseTitle'),
        { type: 'warning', confirmButtonText: t('tenantPage.confirmPause'), confirmButtonClass: 'el-button--warning' }
      )
    } else {
      await ElMessageBox.confirm(t('tenantPage.resumeConfirm', row.name), t('common.tip'), {
        type: 'info'
      })
    }
    await updateTenantStatus(row.id, newStatus)
    ElMessage.success(t('tenantPage.actionSuccess', actionText))
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleDelete(row: TenantItem) {
  try {
    await ElMessageBox.confirm(t('tenantPage.deleteConfirm', row.name), t('common.tip'), { type: 'warning' })
    await deleteTenant(row.id)
    ElMessage.success(t('tenantPage.deleteSuccess'))
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('tenantPage.batchDeleteConfirm', selectedIds.value.length), t('common.tip'), {
      type: 'warning'
    })
    await batchDeleteTenants({ ids: selectedIds.value })
    ElMessage.success(t('tenantPage.batchDeleteSuccess'))
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchEnable() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('tenantPage.batchEnableConfirm', selectedIds.value.length), t('common.tip'), {
      type: 'warning'
    })
    await batchUpdateTenantStatus({ ids: selectedIds.value, status: 1 })
    ElMessage.success(t('tenantPage.batchEnableSuccess'))
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchDisable() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(
      t('tenantPage.batchPauseConfirm', selectedIds.value.length),
      t('tenantPage.batchPauseTitle'),
      { type: 'warning', confirmButtonText: t('tenantPage.confirmPause'), confirmButtonClass: 'el-button--warning' }
    )
    await batchUpdateTenantStatus({ ids: selectedIds.value, status: 0 })
    ElMessage.success(t('tenantPage.batchPauseSuccess'))
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

function openRecycleBin() {
  router.push('/system/tenant/recycle')
}

// 查看租户用户
function openViewUsers(row: TenantItem) {
  currentTenant.value = row
  usersDialogVisible.value = true
  userPage.value = 1
  userSearch.keyword = ''
  isUserRecycleBin.value = false
  loadTenantUsers()
}

function toggleUserRecycleBin() {
  isUserRecycleBin.value = !isUserRecycleBin.value
  userPage.value = 1
  loadTenantUsers()
}

async function loadTenantUsers() {
  if (!currentTenant.value) return
  userLoading.value = true
  try {
    const params: UserListParams = { page: userPage.value, page_size: userPageSize.value }
    if (userSearch.keyword) params.keyword = userSearch.keyword
    let res: PageResult<UserItem> & { data?: UserItem[]; pagination?: { total?: number } }
    if (isUserRecycleBin.value) {
      // 回收站模式：获取所有已删除的用户，然后过滤当前租户的
      res = await getAllDeletedTenantUsers(params)
      // 过滤出当前租户的用户
      const allDeletedUsers = res.data || []
      const filteredUsers = allDeletedUsers.filter((u: UserItem) => u.tenant_id === currentTenant.value?.id)
      userList.value = filteredUsers
      userTotal.value = filteredUsers.length
    } else {
      res = await getTenantUsers(currentTenant.value.id, params)
      userList.value = res.data || []
      userTotal.value = res.pagination?.total || 0
    }
  } catch (e) {
    userList.value = []
    userTotal.value = 0
  } finally {
    userLoading.value = false
  }
}

function resetUserSearch() {
  userSearch.keyword = ''
  userPage.value = 1
  loadTenantUsers()
}

function handleUserSelectionChange(selection: UserItem[]) {
  userSelectedIds.value = selection.map((row) => row.id).filter(Boolean) as string[]
}

// 加载角色列表（只加载租户级角色）
async function loadRoleList() {
  if (!currentTenant.value) return
  roleLoading.value = true
  try {
    // 使用 scope=tenant 获取租户级角色（普通用户和租户管理员角色）
    const data = await getRolesForSelect('tenant')
    roleList.value = data || []
  } catch (e) {
    roleList.value = []
  } finally {
    roleLoading.value = false
  }
}

// 打开新增用户弹窗
function openCreateUserDialog() {
  userDialogMode.value = 'create'
  editingUserId.value = null
  userForm.username = ''
  userForm.password = ''
  userForm.nickname = ''
  userForm.email = ''
  userForm.department = ''
  userForm.position = ''
  userForm.remark = ''
  userForm.role_ids = []
  loadRoleList()
  userDialogVisible.value = true
}

// 打开编辑用户弹窗
function openEditUserDialog(row: UserItem) {
  userDialogMode.value = 'edit'
  editingUserId.value = row.id
  userForm.username = row.username
  userForm.nickname = row.nickname
  userForm.email = row.email || ''
  userForm.department = row.department || ''
  userForm.position = row.position || ''
  userForm.remark = row.remark || ''
  userForm.role_ids = row.role_ids || []
  loadRoleList()
  userDialogVisible.value = true
}

// 提交用户表单
async function submitUserForm() {
  if (!userFormRef.value || !currentTenant.value) return
  const tenantId = currentTenant.value.id
  await userFormRef.value.validate(async (valid) => {
    if (!valid) return
    userSaving.value = true
    try {
      if (userDialogMode.value === 'create') {
        await createTenantUser(tenantId, {
          username: userForm.username,
          password: userForm.password,
          nickname: userForm.nickname,
          email: userForm.email,
          department: userForm.department,
          position: userForm.position,
          role_ids: userForm.role_ids
        } as UserCreateParams)
        ElMessage.success(t('tenantPage.createSuccess'))
      } else if (editingUserId.value) {
        await updateTenantUser(tenantId, editingUserId.value, {
          nickname: userForm.nickname,
          email: userForm.email,
          department: userForm.department,
          position: userForm.position,
          remark: userForm.remark,
          role_ids: userForm.role_ids
        } as UserUpdateParams)
        ElMessage.success(t('tenantPage.updateSuccess'))
      }
      userDialogVisible.value = false
      loadTenantUsers()
    } finally {
      userSaving.value = false
    }
  })
}

// 切换用户状态
async function handleToggleUserStatus(row: UserItem) {
  if (!currentTenant.value) return
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? t('common.status.enabled') : t('common.status.disabled')
  try {
    await ElMessageBox.confirm(t('tenantPage.toggleUserConfirm', actionText, row.username), t('common.tip'), {
      type: 'warning'
    })
    await updateTenantUserStatus(currentTenant.value.id, row.id, newStatus)
    ElMessage.success(t('tenantPage.actionSuccess', actionText))
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 删除用户
async function handleDeleteUser(row: UserItem) {
  if (!currentTenant.value) return

  try {
    await ElMessageBox.confirm(t('tenantPage.deleteUserConfirm', row.username), t('common.tip'), { type: 'warning' })
    await deleteTenantUser(currentTenant.value.id, row.id)
    ElMessage.success(t('tenantPage.deleteSuccess'))
    loadTenantUsers()
  } catch (e) {
    // 用户取消或删除失败
  }
}

// 恢复用户
async function handleRestoreUser(row: UserItem) {
  if (!row.id || !row.tenant_id) return
  try {
    await ElMessageBox.confirm(t('tenantPage.restoreUserConfirm', row.username), t('common.tip'), { type: 'warning' })
    await restoreTenantUser(row.tenant_id, row.id)
    ElMessage.success(t('tenantPage.restoreSuccess'))
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 永久删除用户
async function handlePermanentDeleteUser(row: UserItem) {
  if (!row.id || !row.tenant_id) return
  try {
    await ElMessageBox.confirm(t('tenantPage.permDeleteUserConfirm', row.username), t('tenantPage.dangerOp'), {
      type: 'error',
      confirmButtonText: t('tenantPage.confirmPermDelete'),
      cancelButtonText: t('common.cancel')
    })
    await permanentDeleteTenantUser(row.tenant_id, row.id)
    ElMessage.success(t('tenantPage.permDeleteSuccess'))
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 查看用户角色
async function openViewUserRoles(row: UserItem) {
  currentUser.value = row
  userRolesDialogVisible.value = true
  userRolesLoading.value = true
  currentUserRoles.value = []

  try {
    if (!row.id) return
    const roles = await getUserRoles(row.id)
    currentUserRoles.value = roles || []
  } catch (e) {
    ElMessage.error(t('tenantPage.getRolesFailed'))
  } finally {
    userRolesLoading.value = false
  }
}

// 移除用户单个角色
async function handleRemoveUserRole(role: RoleItem) {
  if (!currentUser.value || !currentTenant.value) return

  try {
    await ElMessageBox.confirm(t('tenantPage.removeRoleConfirm', role.name), t('common.tip'), { type: 'warning' })
    await removeUserRole(currentUser.value.id, role.id)
    ElMessage.success(t('tenantPage.removeSuccess'))
    // 刷新角色列表
    const roles = await getUserRoles(currentUser.value.id)
    currentUserRoles.value = roles || []
    // 刷新用户列表（更新角色显示）
    loadTenantUsers()
  } catch (e) {
    // 用户取消或操作失败
  }
}

// 移除用户所有角色
async function handleRemoveAllUserRoles() {
  if (!currentUser.value || !currentTenant.value) return

  try {
    await ElMessageBox.confirm(t('tenantPage.removeAllRolesConfirm'), t('common.tip'), { type: 'warning' })
    await removeAllUserRoles(currentUser.value.id)
    ElMessage.success(t('tenantPage.removeSuccess'))
    currentUserRoles.value = []
    // 刷新用户列表（更新角色显示）
    loadTenantUsers()
  } catch (e) {
    // 用户取消或操作失败
  }
}

// 批量删除用户
async function handleBatchDeleteUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('tenantPage.batchDeleteUsersConfirm', userSelectedIds.value.length), t('common.tip'), {
      type: 'warning'
    })
    await batchDeleteTenantUsers(currentTenant.value.id, userSelectedIds.value)
    ElMessage.success(t('tenantPage.batchDeleteSuccess'))
    userSelectedIds.value = []
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 批量启用用户
async function handleBatchEnableUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('tenantPage.batchEnableUsersConfirm', userSelectedIds.value.length), t('common.tip'), {
      type: 'warning'
    })
    await batchUpdateTenantUserStatus(currentTenant.value.id, userSelectedIds.value, 1)
    ElMessage.success(t('tenantPage.batchEnableSuccess'))
    userSelectedIds.value = []
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 批量禁用用户
async function handleBatchDisableUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('tenantPage.batchDisableUsersConfirm', userSelectedIds.value.length), t('common.tip'), {
      type: 'warning'
    })
    await batchUpdateTenantUserStatus(currentTenant.value.id, userSelectedIds.value, 0)
    ElMessage.success(t('tenantPage.batchDisableSuccess'))
    userSelectedIds.value = []
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 打开重置密码对话框
function openResetPasswordDialog(row: UserItem) {
  resetPwdUser.value = row
  resetPwdForm.new_password = ''
  resetPwdForm.confirm_password = ''
  resetPwdDialogVisible.value = true
}

// 关闭重置密码对话框
function closeResetPwdDialog() {
  resetPwdDialogVisible.value = false
  resetPwdUser.value = null
  resetPwdForm.new_password = ''
  resetPwdForm.confirm_password = ''
}

// 提交重置密码
async function submitResetPassword() {
  if (!resetPwdFormRef.value) return
  await resetPwdFormRef.value.validate(async (valid) => {
    if (!valid) return
    if (!resetPwdUser.value?.id || !currentTenant.value?.id) return

    resetPwdSaving.value = true
    try {
      await resetTenantUserPassword(currentTenant.value.id, resetPwdUser.value.id, {
        new_password: resetPwdForm.new_password,
        confirm_password: resetPwdForm.confirm_password
      })
      ElMessage.success(t('tenantPage.resetPwdSuccess'))
      closeResetPwdDialog()
    } catch (e) {
      // 错误已在拦截器处理
    } finally {
      resetPwdSaving.value = false
    }
  })
}

// 打开登录日志弹窗
function openLoginLogs(row: UserItem) {
  loginLogsUser.value = row
  loginLogsPage.value = 1
  loginLogsPageSize.value = 10
  loginLogsDialogVisible.value = true
  loadLoginLogs()
}

// 加载登录日志
async function loadLoginLogs() {
  if (!loginLogsUser.value?.id) return

  loginLogsLoading.value = true
  try {
    const res = await getAuditLogList({
      page: loginLogsPage.value,
      page_size: loginLogsPageSize.value,
      user_id: loginLogsUser.value.id,
      action: 'login'  // 只查询登录相关的日志
    })
    loginLogsList.value = res.items
    loginLogsTotal.value = res.total
  } catch (e) {
    ElMessage.error(t('tenantPage.loadLogsFailed'))
  } finally {
    loginLogsLoading.value = false
  }
}

onMounted(() => {
  loadList()
})
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

.batch-bar {
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.flex-1 {
  flex: 1;
}
</style>