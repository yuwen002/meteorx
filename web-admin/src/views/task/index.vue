<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.pending }}</div>
          <div class="stat-label">{{ t('taskPage.stat.pending') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.in_progress }}</div>
          <div class="stat-label">{{ t('taskPage.stat.in_progress') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.completed }}</div>
          <div class="stat-label">{{ t('taskPage.stat.completed') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card stat-card--clickable" @click="filterOverdue">
          <div class="stat-num danger">{{ stats.overdue }}</div>
          <div class="stat-label">{{ t('taskPage.stat.overdue') }}<span class="drill-hint">{{ t('taskPage.stat.drillHint') }}</span></div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ (stats.completion_rate * 100).toFixed(1) }}%</div>
          <div class="stat-label">{{ t('taskPage.stat.completionRate') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ formatDuration(stats.avg_handle_seconds) }}</div>
          <div class="stat-label">{{ t('taskPage.stat.avgHandle') }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('taskPage.title') }}</span>
          <div>
            <el-radio-group v-model="viewMode" size="small" style="margin-right: 12px;" @change="handleViewModeChange">
              <el-radio-button label="list">{{ t('taskPage.view.list') }}</el-radio-button>
              <el-radio-button label="board">{{ t('taskPage.view.board') }}</el-radio-button>
            </el-radio-group>
            <el-tooltip :disabled="aiEnabled" :content="t('taskPage.aiDisabledTip')" placement="bottom">
              <span>
                <el-button :disabled="!aiEnabled" @click="openSummary">
                  <el-icon><MagicStick /></el-icon>{{ t('taskPage.aiSummary') }}
                </el-button>
              </span>
            </el-tooltip>
            <el-button @click="openTrash">
              <el-icon><Files /></el-icon>{{ t('taskPage.trash') }}
            </el-button>
            <el-button type="primary" @click="openDialog()">
              <el-icon><Plus /></el-icon>{{ t('taskPage.create') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-if="viewMode === 'board'" v-model="groupBy" :placeholder="t('taskPage.board.groupBy.label')" style="width: 130px; margin-right: 12px;" @change="handleGroupByChange">
          <el-option :label="t('taskPage.board.groupBy.status')" value="status" />
          <el-option :label="t('taskPage.board.groupBy.priority')" value="priority" />
          <el-option :label="t('taskPage.board.groupBy.assignee')" value="assignee" />
          <el-option :label="t('taskPage.board.groupBy.tag')" value="tag" />
        </el-select>
        <el-select v-model="visibilityFilter" :placeholder="t('taskPage.filter.scopePlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.scope.all')" value="" />
          <el-option :label="t('taskPage.scope.personal')" value="personal" />
          <el-option :label="t('taskPage.scope.tenant')" value="tenant" />
        </el-select>
        <!-- 看板视图下状态列由列本身表达，隐藏状态筛选避免“筛某状态→其余列全空”的歧义 -->
        <el-select v-if="viewMode === 'list'" v-model="statusFilter" :placeholder="t('taskPage.filter.statusPlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.stat.pending')" value="pending" />
          <el-option :label="t('taskPage.stat.in_progress')" value="in_progress" />
          <el-option :label="t('taskPage.stat.completed')" value="completed" />
          <el-option :label="t('taskPage.stat.overdue')" value="overdue" />
        </el-select>
        <el-select v-model="priorityFilter" :placeholder="t('taskPage.filter.priorityPlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.priority.low')" value="low" />
          <el-option :label="t('taskPage.priority.normal')" value="normal" />
          <el-option :label="t('taskPage.priority.high')" value="high" />
          <el-option :label="t('taskPage.priority.urgent')" value="urgent" />
        </el-select>
        <el-input v-model="tagFilter" :placeholder="t('taskPage.filter.tagPlaceholder')" clearable style="width: 150px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-date-picker
          v-model="dueRange"
          type="daterange"
          :range-separator="t('taskPage.filter.rangeSeparator')"
          :start-placeholder="t('taskPage.filter.dueStart')"
          :end-placeholder="t('taskPage.filter.dueEnd')"
          value-format="YYYY-MM-DD"
          clearable
          style="width: 240px; margin-right: 12px;"
          @change="loadList"
        />
        <el-input v-model="keyword" :placeholder="t('taskPage.filter.keywordPlaceholder')" clearable style="width: 200px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList">{{ t('common.query') }}</el-button>
      </div>

      <div v-if="selected.length" class="batch-bar">
        <span class="batch-tip">{{ t('taskPage.batch.selected', selected.length) }}</span>
        <el-dropdown trigger="click" class="batch-dropdown" @command="handleBatchStatus">
          <el-button plain size="small">{{ t('taskPage.batch.changeStatus') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="pending">{{ t('taskPage.stat.pending') }}</el-dropdown-item>
              <el-dropdown-item command="in_progress">{{ t('taskPage.stat.in_progress') }}</el-dropdown-item>
              <el-dropdown-item command="completed">{{ t('taskPage.stat.completed') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" @click="openBatchAssign">{{ t('taskPage.batch.assign') }}</el-button>
        <el-button type="success" size="small" @click="handleBatchComplete">{{ t('taskPage.batch.complete') }}</el-button>
        <el-button type="danger" size="small" @click="handleBatchDelete">{{ t('taskPage.batch.delete') }}</el-button>
      </div>

      <!-- 看板视图：按 groupBy 分列；仅“按状态”分组时卡片拖拽换列即改状态（复用 /tasks/batch/status） -->
      <div v-if="viewMode === 'board'" v-loading="boardLoading" class="board">
        <div
          v-for="col in boardColumns"
          :key="col.key"
          class="board-col"
          :class="{ 'board-col--dragover': dragEnabled && dragOverKey === col.key }"
          @dragover.prevent="dragEnabled && (dragOverKey = col.key)"
          @dragleave="dragOverKey = ''"
          @drop.prevent="handleDrop(col.key)"
        >
          <div class="board-col__header">
            <span v-if="col.dotClass" class="board-col__dot" :class="col.dotClass" />
            <span class="board-col__title">{{ col.title }}</span>
            <span class="board-col__count">{{ col.tasks.length }}</span>
          </div>
          <div
            v-for="task in col.tasks"
            :key="task.id"
            class="board-card"
            :class="{ 'board-card--dragging': dragTaskId === task.id }"
            :draggable="dragEnabled && !boardSubmitting"
            @dragstart="handleDragStart(task)"
            @dragend="handleDragEnd"
          >
            <div class="board-card__title">{{ task.title }}</div>
            <div class="board-card__meta">
              <el-tag v-if="blockedCount(task) > 0" size="small" type="danger" effect="plain">{{ t('taskPage.dep.blockedBadge', blockedCount(task)) }}</el-tag>
              <el-tag size="small" :type="priorityTagType(task.priority)" effect="plain">{{ priorityLabel(task.priority) }}</el-tag>
              <el-tag v-if="task.recurrence" size="small" type="success" effect="plain">{{ recurrenceLabel(task.recurrence) }}</el-tag>
              <span v-if="task.due_date" class="board-card__due" :class="{ 'board-card__due--overdue': isOverdue(task) }">{{ task.due_date }}</span>
            </div>
          </div>
          <div v-if="!col.tasks.length" class="board-col__empty">{{ t('taskPage.board.emptyCol') }}</div>
        </div>
        <div v-if="boardTruncatedTip" class="board-truncated">{{ boardTruncatedTip }}</div>
      </div>

      <el-table v-if="viewMode === 'list'" v-loading="loading" :data="list" stripe row-key="id" max-height="calc(100vh - 300px)" style="width: 100%;" @selection-change="handleSelectionChange" @expand-change="handleExpandChange">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="subtask-panel">
              <div v-if="childrenLoading[row.id]" class="subtask-loading">{{ t('taskPage.subtask.loading') }}</div>
              <template v-else>
                <div v-for="child in childrenMap[row.id] || []" :key="child.id" class="subtask-row">
                  <el-tag size="small" type="info" effect="plain">{{ t('taskPage.subtask.tag') }}</el-tag>
                  <span class="subtask-title">{{ child.title }}</span>
                  <el-tag :type="statusTagType(child.status)" size="small">{{ statusLabel(child.status) }}</el-tag>
                  <el-tag :type="priorityTagType(child.priority)" size="small" effect="plain">{{ priorityLabel(child.priority) }}</el-tag>
                  <span class="subtask-assignee">{{ child.assignee_name || '-' }}</span>
                  <span class="subtask-due">{{ child.due_date || '-' }}</span>
                  <el-button type="primary" link size="small" @click="openDialog(child)">{{ t('taskPage.action.edit') }}</el-button>
                  <el-button type="danger" link size="small" @click="handleDelete(child)">{{ t('taskPage.action.delete') }}</el-button>
                </div>
                <div v-if="!(childrenMap[row.id] || []).length" class="subtask-empty">{{ t('taskPage.subtask.empty') }}</div>
              </template>
            </div>
          </template>
        </el-table-column>
        <el-table-column type="selection" width="45" />
        <el-table-column prop="title" :label="t('taskPage.col.title')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag v-if="row.recurrence" size="small" type="success" effect="plain" style="margin-right: 6px;">{{ recurrenceLabel(row.recurrence) }}</el-tag>
            <el-tooltip v-if="blockedCount(row) > 0" :content="blockerTooltip(row)" placement="top">
              <el-tag size="small" type="danger" effect="plain" class="dep-badge">{{ t('taskPage.dep.blockedBadge', blockedCount(row)) }}</el-tag>
            </el-tooltip>
            <span>{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.scope')" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.visibility === 'tenant' ? 'warning' : 'info'">
              {{ row.visibility === 'tenant' ? t('taskPage.scope.tagTenant') : t('taskPage.scope.tagPersonal') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.status')" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.priority')" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="priorityTagType(row.priority)" size="small" effect="plain">{{ priorityLabel(row.priority) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.tags')" min-width="140">
          <template #default="{ row }">
            <el-tag v-for="tag in row.tags" :key="tag" size="small" type="info" style="margin-right: 4px;">{{ tag }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.assignee')" min-width="120">
          <template #default="{ row }">
            <span>{{ row.assignee_name || '-' }}</span>
            <el-tag v-if="extraAssigneeCount(row) > 0" size="small" type="info" class="assignee-more">+{{ extraAssigneeCount(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.dueDate')" min-width="150">
          <template #default="{ row }">{{ row.due_date || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('taskPage.col.createdAt')" min-width="150" />
        <el-table-column :label="t('taskPage.col.action')" width="220" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'completed'" type="success" link size="small" @click="handleComplete(row)">{{ t('taskPage.action.complete') }}</el-button>
            <el-button v-else type="warning" link size="small" @click="handleReopen(row)">{{ t('taskPage.action.reopen') }}</el-button>
            <el-button type="primary" link size="small" @click="openDialog(row)">{{ t('taskPage.action.edit') }}</el-button>
            <el-button link size="small" @click="openCreateSubtask(row)">{{ t('taskPage.subtask.add') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('taskPage.action.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="viewMode === 'list'" class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingId ? t('taskPage.dialog.editTitle') : t('taskPage.dialog.createTitle')" width="520px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item v-if="!editingId && aiEnabled" :label="t('taskPage.form.aiParseLabel')">
          <div class="ai-parse">
            <el-input v-model="nlText" type="textarea" :rows="2" maxlength="500" :placeholder="t('taskPage.form.aiParsePlaceholder')" />
            <el-button class="ai-parse__btn" type="primary" plain :loading="aiParsing" @click="handleAIParse">
              <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiParseBtn') }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.titleLabel')" prop="title">
          <el-input v-model="form.title" :placeholder="t('taskPage.form.titlePlaceholder')" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('taskPage.form.descLabel')">
          <el-input v-model="form.description" type="textarea" :rows="3" :placeholder="t('taskPage.form.descPlaceholder')" maxlength="2000" show-word-limit />
          <el-button v-if="aiEnabled" class="ai-inline-btn" size="small" plain :loading="aiExpanding" @click="handleAIExpand">
            <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiExpandBtn') }}
          </el-button>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.visibilityLabel')" prop="visibility">
          <el-radio-group v-model="form.visibility">
            <el-radio-button label="personal">{{ t('taskPage.scope.personal') }}</el-radio-button>
            <el-radio-button label="tenant">{{ t('taskPage.scope.tenant') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.parentLabel')">
          <el-select
            v-model="form.parent_id"
            filterable
            remote
            reserve-keyword
            clearable
            :remote-method="searchParents"
            :loading="parentLoading"
            :placeholder="t('taskPage.form.parentPlaceholder')"
            style="width: 100%;"
          >
            <el-option v-for="opt in parentOptions" :key="opt.id" :label="opt.title" :value="opt.id" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.form.parentTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.dep.label')">
          <el-select
            v-model="form.blocked_by"
            multiple
            filterable
            remote
            reserve-keyword
            collapse-tags
            collapse-tags-tooltip
            clearable
            :remote-method="searchBlockers"
            :loading="depLoading"
            :placeholder="t('taskPage.dep.placeholder')"
            style="width: 100%;"
          >
            <el-option v-for="opt in depOptions" :key="opt.id" :label="opt.title" :value="opt.id" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.dep.tip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.priorityLabel')">
          <div class="ai-field-row">
            <el-select v-model="form.priority" style="flex: 1;">
              <el-option :label="t('taskPage.priority.low')" value="low" />
              <el-option :label="t('taskPage.priority.normal')" value="normal" />
              <el-option :label="t('taskPage.priority.high')" value="high" />
              <el-option :label="t('taskPage.priority.urgent')" value="urgent" />
            </el-select>
            <el-button v-if="aiEnabled" size="small" plain :loading="aiSuggesting" @click="handleAISuggest">
              <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiSuggestBtn') }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item v-if="form.visibility === 'tenant'" :label="t('taskPage.form.assigneeLabel')">
          <el-select
            v-model="form.assignees"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            :loading="assignableLoading"
            :placeholder="t('taskPage.form.assigneesPlaceholder')"
            style="width: 100%;"
          >
            <el-option v-for="m in assignableMembers" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.form.assigneesTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.dueLabel')">
          <el-date-picker
            v-model="form.due_date"
            type="datetime"
            :placeholder="t('taskPage.form.duePlaceholder')"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%;"
          />
        </el-form-item>
        <el-form-item :label="t('taskPage.form.remindLabel')">
          <el-input v-model="form.remind_before" :placeholder="t('taskPage.form.remindPlaceholder')" clearable />
          <div class="form-tip">{{ t('taskPage.form.remindTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.recurrenceLabel')">
          <el-select v-model="form.recurrence" :placeholder="t('taskPage.form.recurrencePlaceholder')" style="width: 100%;">
            <el-option :label="t('taskPage.recurrence.none')" value="" />
            <el-option :label="t('taskPage.recurrence.daily')" value="daily" />
            <el-option :label="t('taskPage.recurrence.weekly')" value="weekly" />
            <el-option :label="t('taskPage.recurrence.monthly')" value="monthly" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.form.recurrenceTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.tagsLabel')">
          <el-select v-model="form.tags" multiple filterable allow-create default-first-option :placeholder="t('taskPage.form.tagsPlaceholder')" style="width: 100%;">
            <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitLoading" @click="submitForm">{{ editingId ? t('taskPage.form.submitSave') : t('taskPage.form.submitCreate') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="assignVisible" :title="t('taskPage.assign.title')" width="440px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item :label="t('taskPage.assign.idLabel')">
          <el-select
            v-model="assignUserIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            :loading="assignableLoading"
            :placeholder="t('taskPage.assignees.placeholder')"
            style="width: 100%;"
          >
            <el-option v-for="m in assignableMembers" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.assignees.batchTip') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="assignLoading" @click="submitBatchAssign">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="trashVisible" :title="t('taskPage.trash.title')" size="600px">
      <el-table v-loading="trashLoading" :data="trashList" stripe style="width: 100%;">
        <el-table-column prop="title" :label="t('taskPage.trash.colTitle')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('taskPage.trash.colDeletedAt')" min-width="150">
          <template #default="{ row }">{{ row.deleted_at || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('taskPage.trash.colAction')" width="150" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleRestore(row)">{{ t('taskPage.trash.restore') }}</el-button>
            <el-button type="danger" link size="small" @click="handlePermanentDelete(row)">{{ t('taskPage.trash.permanentDelete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="trashPage"
          v-model:page-size="trashPageSize"
          :total="trashTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTrash"
          @current-change="loadTrash"
        />
      </div>
    </el-drawer>

    <el-dialog v-model="summaryVisible" :title="t('taskPage.summary.title')" width="560px">
      <div class="summary-toolbar">
        <el-radio-group v-model="summaryScope" size="small" @change="loadSummary">
          <el-radio-button label="today">{{ t('taskPage.summary.scope.today') }}</el-radio-button>
          <el-radio-button label="week">{{ t('taskPage.summary.scope.week') }}</el-radio-button>
          <el-radio-button label="overdue">{{ t('taskPage.summary.scope.overdue') }}</el-radio-button>
          <el-radio-button label="all">{{ t('taskPage.summary.scope.all') }}</el-radio-button>
        </el-radio-group>
        <span v-if="summary" class="summary-tip">{{ t('taskPage.summary.totalCount', summary.total) }}</span>
      </div>
      <div v-loading="summaryLoading" class="summary-body">
        <template v-if="summary">
          <p class="summary-text">{{ summary.text }}</p>
          <div v-if="summary.highlights?.length" class="summary-block">
            <div class="summary-block__title">{{ t('taskPage.summary.highlights') }}</div>
            <ul><li v-for="(h, i) in summary.highlights" :key="'h'+i">{{ h }}</li></ul>
          </div>
          <div v-if="summary.suggested_focus?.length" class="summary-block">
            <div class="summary-block__title">{{ t('taskPage.summary.suggestedFocus') }}</div>
            <ul><li v-for="(f, i) in summary.suggested_focus" :key="'f'+i">{{ f }}</li></ul>
          </div>
        </template>
        <el-empty v-else-if="!summaryLoading" :description="t('taskPage.summary.empty')" />
      </div>
      <template #footer>
        <el-button @click="summaryVisible = false">{{ t('taskPage.summary.close') }}</el-button>
        <el-button type="primary" :loading="summaryLoading" @click="loadSummary">{{ t('taskPage.summary.regenerate') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus, Files, MagicStick, ArrowDown } from '@element-plus/icons-vue'
import {
  getTaskList,
  getTaskBoard,
  getTask,
  createTask,
  updateTask,
  completeTask,
  reopenTask,
  deleteTask,
  getTaskStats,
  getTaskTrash,
  restoreTask,
  permanentDeleteTask,
  batchCompleteTasks,
  batchDeleteTasks,
  batchUpdateTaskStatus,
  batchAssignTasks,
  getAssignableMembers,
  type TaskItem,
  type TaskStats,
  type TaskStatus,
  type TaskPriority,
  type TaskVisibility,
  type TaskRecurrence,
  type AssignableMember,
} from '@/api/modules/task'
import {
  getAICapabilities,
  parseTaskByNL,
  suggestTaskFields,
  expandTaskDescription,
  summarizeTasks,
  type TaskSummary,
  type TaskSummaryScope,
} from '@/api/modules/task-ai'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const route = useRoute()

const loading = ref(false)
const list = ref<TaskItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref<'' | TaskStatus | 'overdue'>('')
const priorityFilter = ref<'' | TaskPriority>('')
const visibilityFilter = ref<'' | TaskVisibility>('')
const tagFilter = ref('')
const dueRange = ref<[string, string] | null>(null)

// 子任务懒加载：childrenMap 缓存父 ID → 直接子任务，childrenLoading 标记展开中。
// 仅单层，故展开面板只列子任务本身，不再对子行提供展开。
const childrenMap = reactive<Record<string, TaskItem[]>>({})
const childrenLoading = reactive<Record<string, boolean>>({})

const stats = reactive<TaskStats>({ pending: 0, in_progress: 0, completed: 0, overdue: 0, total: 0, completion_rate: 0, avg_handle_seconds: 0 })

// 批量选择
const selected = ref<TaskItem[]>([])
function handleSelectionChange(rows: TaskItem[]) {
  selected.value = rows
}

// handleExpandChange 行首展开时按需懒加载该根任务的直接子任务（已缓存则不重复拉）。
// el-table 的 expand-change 在展开/收起均触发，expanded 为当前展开行数组（或布尔），仅展开时取数。
async function handleExpandChange(row: TaskItem, expanded: TaskItem[] | boolean) {
  const isExpanded = Array.isArray(expanded)
    ? expanded.some((r) => r.id === row.id)
    : !!expanded
  if (!isExpanded || childrenMap[row.id]) return
  childrenLoading[row.id] = true
  try {
    const raw = await getTaskList({ parent_id: row.id, page: 1, page_size: 100 })
    childrenMap[row.id] = toPageResult<TaskItem>(raw).list
  } catch {
    childrenMap[row.id] = []
  } finally {
    childrenLoading[row.id] = false
  }
}

// 回收站
const trashVisible = ref(false)
const trashLoading = ref(false)
const trashList = ref<TaskItem[]>([])
const trashPage = ref(1)
const trashPageSize = ref(10)
const trashTotal = ref(0)

function statusLabel(s: string) {
  return ({ pending: t('taskPage.stat.pending'), in_progress: t('taskPage.stat.in_progress'), completed: t('taskPage.stat.completed') } as Record<string, string>)[s] || s
}
function statusTagType(s: string) {
  return ({ pending: 'info', in_progress: 'warning', completed: 'success' } as Record<string, string>)[s] || 'info'
}
function priorityLabel(p: string) {
  return ({ low: t('taskPage.priority.low'), normal: t('taskPage.priority.normal'), high: t('taskPage.priority.high'), urgent: t('taskPage.priority.urgent') } as Record<string, string>)[p] || p
}
function priorityTagType(p: string) {
  return ({ low: 'info', normal: '', high: 'warning', urgent: 'danger' } as Record<string, string>)[p] || ''
}
function recurrenceLabel(r: string) {
  return ({ daily: t('taskPage.recurrence.daily'), weekly: t('taskPage.recurrence.weekly'), monthly: t('taskPage.recurrence.monthly') } as Record<string, string>)[r] || r
}
// formatDuration 将秒数格式化为“X天/X小时/X分钟”可读时长；0 或无效返回 “-”。
function formatDuration(seconds: number) {
  if (!seconds || seconds <= 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return t('taskPage.duration.days', d, h)
  if (h > 0) return t('taskPage.duration.hours', h, m)
  if (m > 0) return t('taskPage.duration.minutes', m)
  return t('taskPage.duration.seconds', Math.floor(seconds))
}

async function loadStats() {
  try {
    const data = await getTaskStats(visibilityFilter.value ? { visibility: visibilityFilter.value } : undefined)
    Object.assign(stats, data)
  } catch (e) {
    console.error(t('taskPage.loadStatsFailed'), e)
  }
}

async function loadList() {
  loading.value = true
  // 列表默认只展示根任务，子任务通过行首展开箭头懒加载，避免父子混杂
  Object.keys(childrenMap).forEach((k) => delete childrenMap[k])
  try {
    const raw = await getTaskList({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
      status: statusFilter.value || undefined,
      priority: priorityFilter.value || undefined,
      visibility: visibilityFilter.value || undefined,
      tag: tagFilter.value || undefined,
      due_from: dueRange.value?.[0] || undefined,
      due_to: dueRange.value?.[1] || undefined,
      root_only: true,
    })
    const result = toPageResult<TaskItem>(raw)
    list.value = result.list
    total.value = result.total
    loadStats()
  } catch (e) {
    ElMessage.error(t('taskPage.loadListFailed'))
  } finally {
    loading.value = false
  }
}

// ===== 看板视图（三列 = 状态，拖拽换列即改状态） =====
const VIEW_MODE_KEY = 'meteorx.task.viewMode'
function initialViewMode(): 'list' | 'board' {
  try {
    return localStorage.getItem(VIEW_MODE_KEY) === 'board' ? 'board' : 'list'
  } catch {
    return 'list'
  }
}
const viewMode = ref<'list' | 'board'>(initialViewMode())
// 看板分组维度：status（默认，按状态分列且支持拖拽换列）/priority/assignee/tag，持久化到 localStorage。
const GROUP_BY_KEY = 'meteorx.task.groupBy'
type BoardGroupBy = 'status' | 'priority' | 'assignee' | 'tag'
function initialGroupBy(): BoardGroupBy {
  try {
    const v = localStorage.getItem(GROUP_BY_KEY)
    if (v === 'priority' || v === 'assignee' || v === 'tag') return v
  } catch {
    /* 隐私模式读取失败回落默认分组 */
  }
  return 'status'
}
const groupBy = ref<BoardGroupBy>(initialGroupBy())
// 仅按状态分组时“列=状态”语义成立才允许拖拽换列；其余维度禁用拖拽避免歧义。
const dragEnabled = computed(() => groupBy.value === 'status')
const boardLoading = ref(false)
const boardTasks = ref<TaskItem[]>([])
const boardTotal = ref(0)
// 看板走独立 /tasks/board 端点一次拉足当前筛选范围内的任务（后端上限 BoardMax=500）；
// truncated 为真表示匹配总数超过上限，仅展示了其中一部分。
const boardTruncated = ref(false)

// 拖拽不依赖 dataTransfer 读数据（jsdom/浏览器差异），仅用组件内状态追踪拖起项与悬停列。
const dragTaskId = ref('')
const dragOverKey = ref('')
const boardSubmitting = ref(false)

async function loadBoard() {
  boardLoading.value = true
  try {
    const res = await getTaskBoard({
      keyword: keyword.value || undefined,
      priority: priorityFilter.value || undefined,
      visibility: visibilityFilter.value || undefined,
      tag: tagFilter.value || undefined,
      due_from: dueRange.value?.[0] || undefined,
      due_to: dueRange.value?.[1] || undefined,
      root_only: true,
      // 看板以状态分列，不传 status（尤其规避 overdue 伪状态过滤掉列）；默认仅含根任务保持分列清爽
    })
    boardTasks.value = res.items || []
    boardTotal.value = res.total || 0
    boardTruncated.value = !!res.truncated
    loadStats()
  } catch {
    ElMessage.error(t('taskPage.loadListFailed'))
  } finally {
    boardLoading.value = false
  }
}

interface BoardColumn {
  key: string
  title: string
  /** 仅状态分组时提供色点样式，其余维度为空不渲染色点 */
  dotClass?: string
  tasks: TaskItem[]
}

// boardColumns 按当前分组维度将看板任务分桶：
// status → 固定三列；priority → urgent/high/normal/low 固定四列；
// assignee → 按主负责人动态列；tag → 按标签动态列（多标签出现在每列）+ 未分类列。
const boardColumns = computed<BoardColumn[]>(() => {
  switch (groupBy.value) {
    case 'priority': {
      const order: TaskPriority[] = ['urgent', 'high', 'normal', 'low']
      return order.map((p) => ({
        key: p,
        title: priorityLabel(p),
        tasks: boardTasks.value.filter((t) => t.priority === p),
      }))
    }
    case 'assignee': {
      const map = new Map<string, BoardColumn>()
      for (const task of boardTasks.value) {
        const key = task.assignee_id || '__unassigned__'
        if (!map.has(key)) {
          map.set(key, {
            key,
            title: task.assignee_id ? task.assignee_name || task.assignee_id : t('taskPage.board.unassigned'),
            tasks: [],
          })
        }
        map.get(key)!.tasks.push(task)
      }
      return [...map.values()].sort((a, b) => b.tasks.length - a.tasks.length)
    }
    case 'tag': {
      const map = new Map<string, TaskItem[]>()
      const untagged: TaskItem[] = []
      for (const task of boardTasks.value) {
        if (!task.tags || !task.tags.length) {
          untagged.push(task)
          continue
        }
        for (const tag of task.tags) {
          if (!map.has(tag)) map.set(tag, [])
          map.get(tag)!.push(task)
        }
      }
      const cols: BoardColumn[] = [...map.entries()]
        .sort((a, b) => a[0].localeCompare(b[0]))
        .map(([tag, tasks]) => ({ key: tag, title: tag, tasks }))
      cols.push({ key: '__untagged__', title: t('taskPage.board.untagged'), tasks: untagged })
      return cols
    }
    default: {
      const buckets: Record<TaskStatus, TaskItem[]> = { pending: [], in_progress: [], completed: [] }
      for (const task of boardTasks.value) {
        if (buckets[task.status]) buckets[task.status].push(task)
      }
      return (['pending', 'in_progress', 'completed'] as TaskStatus[]).map((s) => ({
        key: s,
        title: statusLabel(s),
        dotClass: `board-col__dot--${s}`,
        tasks: buckets[s],
      }))
    }
  }
})

const boardTruncatedTip = computed(() => {
  // 仅当后端标记截断（匹配总数超过看板上限 BoardMax）时提示，不再因分页丢失而误报。
  if (boardTruncated.value) {
    return t('taskPage.board.truncated', boardTasks.value.length, boardTotal.value)
  }
  return ''
})

// isOverdue 卡片逾期判定：有截止日、未完成且已过当前时刻。
function isOverdue(task: TaskItem) {
  if (!task.due_date || task.status === 'completed') return false
  const d = new Date(task.due_date.replace(' ', 'T'))
  return !Number.isNaN(d.getTime()) && d.getTime() < Date.now()
}

function handleViewModeChange(mode: 'list' | 'board') {
  try {
    localStorage.setItem(VIEW_MODE_KEY, mode)
  } catch {
    /* 隐私模式下写入失败仅影响下次默认视图，忽略 */
  }
  // 切换视图清空列表选择态，避免残留的批量条在看板下造成误操作
  selected.value = []
  if (mode === 'board') loadBoard()
  else loadList()
}

// refreshCurrent 供跨视图共享操作（如新建）刷新当前可见数据。
function refreshCurrent() {
  return viewMode.value === 'board' ? loadBoard() : loadList()
}

// handleGroupByChange 切换看板分组维度：仅持久化偏好，分桶由 boardColumns 响应式重算，无需重拉数据。
function handleGroupByChange(val: BoardGroupBy) {
  try {
    localStorage.setItem(GROUP_BY_KEY, val)
  } catch {
    /* 隐私模式写入失败仅影响下次默认分组，忽略 */
  }
  // 离开状态分组时清空残留拖拽态，避免高亮/拖起项悬空
  dragTaskId.value = ''
  dragOverKey.value = ''
}

function handleDragStart(task: TaskItem) {
  if (!dragEnabled.value || boardSubmitting.value) return
  dragTaskId.value = task.id
}

function handleDragEnd() {
  dragTaskId.value = ''
  dragOverKey.value = ''
}

// handleDrop 仅在按状态分组时生效：卡片放入不同状态列复用批量改状态接口（单条）改状态；
// 无论成败都重拉看板，避免乐观移动与后端不一致（失败提示由拦截器统一弹出）。
async function handleDrop(colKey: string) {
  dragOverKey.value = ''
  if (!dragEnabled.value) return
  const id = dragTaskId.value
  dragTaskId.value = ''
  if (!id || boardSubmitting.value) return
  const status = colKey as TaskStatus
  const task = boardTasks.value.find((x) => x.id === id)
  if (!task || task.status === status) return
  boardSubmitting.value = true
  try {
    await batchUpdateTaskStatus({ ids: [id], status })
    ElMessage.success(t('taskPage.board.moved', statusLabel(status)))
  } catch {
    /* 错提由拦截器弹出 */
  } finally {
    await loadBoard()
    boardSubmitting.value = false
  }
}

const dialogVisible = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const editingId = ref('')
const form = reactive({
  title: '',
  description: '',
  visibility: 'personal' as TaskVisibility,
  priority: 'normal' as TaskPriority,
  assignees: [] as string[],
  parent_id: '',
  blocked_by: [] as string[],
  due_date: '',
  remind_before: '',
  recurrence: '' as '' | TaskRecurrence,
  tags: [] as string[],
})
const rules = computed<FormRules>(() => ({
  title: [{ required: true, message: t('taskPage.form.titleRequired'), trigger: 'blur' }],
}))

// 可指派成员：加载当前租户存活成员供负责人多选（仅需登录，不依赖 user:list 权限）。
const assignableMembers = ref<AssignableMember[]>([])
const assignableLoading = ref(false)
async function loadAssignableMembers() {
  assignableLoading.value = true
  try {
    assignableMembers.value = (await getAssignableMembers()) || []
  } catch {
    assignableMembers.value = []
  } finally {
    assignableLoading.value = false
  }
}

// extraAssigneeCount 返回除主负责人外的额外负责人数（列表“负责人”列以 +N 折叠展示）。
function extraAssigneeCount(row: TaskItem): number {
  const n = row.assignees?.length ?? 0
  return n > 1 ? n - 1 : 0
}

// 阻塞关系徽标：未完成前置（status !== completed）才计入“被 N 阻塞”。
function incompleteBlockers(row: TaskItem) {
  return (row.blocked_by || []).filter((b) => b.status !== 'completed')
}
function blockedCount(row: TaskItem): number {
  return incompleteBlockers(row).length
}
function blockerTooltip(row: TaskItem): string {
  return t('taskPage.dep.blockedBy', incompleteBlockers(row).map((b) => b.title).join('、'))
}

// 父任务选择器：远程搜索本人可见的根任务（排除自身），仅用于建/改时挂父。
const parentOptions = ref<TaskItem[]>([])
const parentLoading = ref(false)
async function searchParents(query: string) {
  parentLoading.value = true
  try {
    const raw = await getTaskList({ keyword: query || undefined, root_only: true, page: 1, page_size: 20 })
    parentOptions.value = toPageResult<TaskItem>(raw).list.filter((tk) => tk.id !== editingId.value)
  } catch {
    parentOptions.value = []
  } finally {
    parentLoading.value = false
  }
}

// 前置任务选择器：远程搜索本人可见的根任务（排除自身）供建/改时选前置。
// depOptions 仅存 id/title（TaskDependencyRef 形状），已选项回显与搜索结果共用此列表。
const depOptions = ref<{ id: string; title: string }[]>([])
const depLoading = ref(false)
async function searchBlockers(query: string) {
  depLoading.value = true
  try {
    const raw = await getTaskList({ keyword: query || undefined, root_only: true, page: 1, page_size: 20 })
    const results = toPageResult<TaskItem>(raw).list
      .filter((tk) => tk.id !== editingId.value)
      .map((tk) => ({ id: tk.id, title: tk.title }))
    // 保留已选中但不在本次结果中的前置，避免其标签因远程刷新而丢标题
    const keep = depOptions.value.filter((o) => form.blocked_by.includes(o.id) && !results.some((r) => r.id === o.id))
    depOptions.value = [...keep, ...results]
  } catch {
    /* 保留已有选项 */
  } finally {
    depLoading.value = false
  }
}

function openDialog(row?: TaskItem) {
  editingId.value = row?.id || ''
  if (row) {
    form.title = row.title
    form.description = row.description
    form.visibility = row.visibility
    form.priority = row.priority
    // 负责人多选：优先取集合，无集合时回落主负责人（自我负责也展示为一个可取消的选项）
    form.assignees = row.assignees?.length ? [...row.assignees] : (row.assignee_id ? [row.assignee_id] : [])
    form.parent_id = row.parent_id || ''
    // 前置依赖回显：取 blocked_by 的 id 集合，并以其标题初始化下拉选项（无需再拉取）
    form.blocked_by = (row.blocked_by || []).map((b) => b.id)
    depOptions.value = (row.blocked_by || []).map((b) => ({ id: b.id, title: b.title }))
    form.due_date = row.due_date || ''
    form.remind_before = row.remind_before || ''
    form.recurrence = (row.recurrence as '' | TaskRecurrence) || ''
    form.tags = [...(row.tags || [])]
    // 已挂父任务时回填一个下拉选项使其可见（标题异步拉取，失败则先以 ID 占位）
    parentOptions.value = row.parent_id ? [{ ...row, id: row.parent_id, title: row.parent_id } as TaskItem] : []
    if (row.parent_id) {
      getTask(row.parent_id)
        .then((p) => { parentOptions.value = [{ ...p }] })
        .catch(() => { /* 保留 ID 占位 */ })
    }
  } else {
    form.title = ''
    form.description = ''
    form.visibility = 'personal'
    form.priority = 'normal'
    form.assignees = []
    form.parent_id = ''
    form.blocked_by = []
    form.due_date = ''
    form.remind_before = ''
    form.recurrence = ''
    form.tags = []
    parentOptions.value = []
    depOptions.value = []
  }
  dialogVisible.value = true
  // 首次打开时懒加载可指派成员（已加载则不重复）
  if (!assignableMembers.value.length) loadAssignableMembers()
}

// openCreateSubtask 从某行直接新建其子任务：预填 parent_id（父任务即该行）并继承其可见范围。
function openCreateSubtask(row: TaskItem) {
  openDialog()
  form.parent_id = row.id
  form.visibility = row.visibility
  parentOptions.value = [{ ...row }]
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      const payload = {
        title: form.title,
        description: form.description,
        visibility: form.visibility,
        priority: form.priority,
        assignees: form.visibility === 'tenant' ? form.assignees : undefined,
        due_date: form.due_date || undefined,
        remind_before: form.remind_before || undefined,
        recurrence: form.recurrence,
        tags: form.tags,
        parent_id: form.parent_id,
        blocked_by: form.blocked_by,
      }
      if (editingId.value) {
        await updateTask(editingId.value, payload)
        ElMessage.success(t('taskPage.updated'))
      } else {
        await createTask(payload)
        ElMessage.success(t('taskPage.created'))
      }
      dialogVisible.value = false
      refreshCurrent()
    } catch (e) {
      ElMessage.error((e as Error).message || t('taskPage.opFailed'))
    } finally {
      submitLoading.value = false
    }
  })
}

async function handleComplete(row: TaskItem) {
  try {
    await completeTask(row.id)
    ElMessage.success(t('taskPage.completed'))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.opFailed'))
  }
}

async function handleReopen(row: TaskItem) {
  try {
    await reopenTask(row.id, 'pending')
    ElMessage.success(t('taskPage.reopened'))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.opFailed'))
  }
}

async function handleDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(t('taskPage.deleteConfirm', row.title), t('taskPage.deleteTitle'), { type: 'warning' })
    await deleteTask(row.id)
    ElMessage.success(t('taskPage.deleted'))
    loadList()
  } catch { /* cancelled */ }
}

async function handleBatchComplete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    const res = await batchCompleteTasks(ids)
    ElMessage.success(t('taskPage.batchCompleted', res.affected))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchCompleteFailed'))
  }
}

// 批量改状态/批量指派：复用后端已有的 /tasks/batch/status 与 /tasks/batch/assign，
// 两者均逐条做归属校验，只返回实际生效数，失败项不阻断整体。
const assignVisible = ref(false)
const assignLoading = ref(false)
const assignUserIds = ref<string[]>([])

async function handleBatchStatus(status: string) {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    const res = await batchUpdateTaskStatus({ ids, status: status as TaskStatus })
    ElMessage.success(t('taskPage.batchStatusChanged', statusLabel(status), res.affected))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchStatusFailed'))
  }
}

function openBatchAssign() {
  assignUserIds.value = []
  assignVisible.value = true
  if (!assignableMembers.value.length) loadAssignableMembers()
}

async function submitBatchAssign() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  const assignees = assignUserIds.value.filter((v) => v.trim() !== '')
  if (!assignees.length) {
    ElMessage.warning(t('taskPage.assigneeIdRequired'))
    return
  }
  assignLoading.value = true
  try {
    const res = await batchAssignTasks({ ids, assignees })
    ElMessage.success(t('taskPage.batchAssigned', res.affected))
    assignVisible.value = false
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchAssignFailed'))
  } finally {
    assignLoading.value = false
  }
}

async function handleBatchDelete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    await ElMessageBox.confirm(t('taskPage.batchDeleteConfirm', ids.length), t('taskPage.batchDeleteTitle'), { type: 'warning' })
    const res = await batchDeleteTasks(ids)
    ElMessage.success(t('taskPage.batchDeleted', res.affected))
    loadList()
  } catch { /* cancelled */ }
}

// filterOverdue 点击"已逾期"统计卡：将状态筛选切为逾期伪状态并重查（再次点击取消）。
function filterOverdue() {
  // 逾期是列表专属的伪状态筛选，看板下点卡片先切回列表视图再下钻。
  if (viewMode.value !== 'list') {
    viewMode.value = 'list'
    try {
      localStorage.setItem(VIEW_MODE_KEY, 'list')
    } catch {
      /* ignore */
    }
  }
  statusFilter.value = statusFilter.value === 'overdue' ? '' : 'overdue'
  page.value = 1
  loadList()
}

function openTrash() {
  trashVisible.value = true
  trashPage.value = 1
  loadTrash()
}

async function loadTrash() {
  trashLoading.value = true
  try {
    const raw = await getTaskTrash({ page: trashPage.value, page_size: trashPageSize.value })
    const result = toPageResult<TaskItem>(raw)
    trashList.value = result.list
    trashTotal.value = result.total
  } catch {
    ElMessage.error(t('taskPage.loadTrashFailed'))
  } finally {
    trashLoading.value = false
  }
}

async function handleRestore(row: TaskItem) {
  try {
    await restoreTask(row.id)
    ElMessage.success(t('taskPage.restored'))
    loadTrash()
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.restoreFailed'))
  }
}

async function handlePermanentDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(t('taskPage.permanentDeleteConfirm', row.title), t('taskPage.permanentDeleteTitle'), { type: 'warning' })
    await permanentDeleteTask(row.id)
    ElMessage.success(t('taskPage.permanentDeleted'))
    loadTrash()
  } catch { /* cancelled */ }
}

// ===== AI 辅助（可选叠加层，未启用时相关入口隐藏/禁用） =====
const aiEnabled = ref(false)
const aiModel = ref('')
const nlText = ref('')
const aiParsing = ref(false)
const aiSuggesting = ref(false)
const aiExpanding = ref(false)

const summaryVisible = ref(false)
const summaryLoading = ref(false)
const summaryScope = ref<TaskSummaryScope>('all')
const summary = ref<TaskSummary | null>(null)

// loadAICapabilities 探测后端 AI 可用性；失败静默降级为不可用（不阻断任务页）。
async function loadAICapabilities() {
  try {
    const cap = await getAICapabilities()
    aiEnabled.value = cap.enabled
    aiModel.value = cap.model
  } catch {
    aiEnabled.value = false
  }
}

// handleAIParse 自然语言解析并预填新建表单（仅新建态使用）。
async function handleAIParse() {
  const text = nlText.value.trim()
  if (!text) {
    ElMessage.warning(t('taskPage.aiDescRequired'))
    return
  }
  aiParsing.value = true
  try {
    const draft = await parseTaskByNL(text)
    if (draft.title) form.title = draft.title
    if (draft.priority) form.priority = draft.priority
    if (draft.due_date) form.due_date = draft.due_date
    if (draft.recurrence) form.recurrence = draft.recurrence
    if (draft.tags?.length) form.tags = [...draft.tags]
    ElMessage.success(t('taskPage.aiPrefilled'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiParseFailed'))
  } finally {
    aiParsing.value = false
  }
}

// handleAISuggest 依据当前标题/描述给出优先级与标签建议。
async function handleAISuggest() {
  if (!form.title.trim()) {
    ElMessage.warning(t('taskPage.aiTitleRequired'))
    return
  }
  aiSuggesting.value = true
  try {
    const s = await suggestTaskFields(form.title, form.description)
    if (s.priority) form.priority = s.priority
    if (s.tags?.length) form.tags = Array.from(new Set([...form.tags, ...s.tags]))
    if (s.reason) ElMessage.info(t('taskPage.aiSuggestion', s.reason))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiSuggestFailed'))
  } finally {
    aiSuggesting.value = false
  }
}

// handleAIExpand 扩写描述并追加拆解步骤。
async function handleAIExpand() {
  if (!form.title.trim()) {
    ElMessage.warning(t('taskPage.aiTitleRequired'))
    return
  }
  aiExpanding.value = true
  try {
    const e2 = await expandTaskDescription(form.title, form.description)
    let text = e2.description || ''
    if (e2.steps?.length) text += (text ? '\n\n' : '') + e2.steps.map((s, i) => `${i + 1}. ${s}`).join('\n')
    if (text) form.description = text
    else ElMessage.info(t('taskPage.aiNoContent'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiExpandFailed'))
  } finally {
    aiExpanding.value = false
  }
}

// openSummary 打开摘要对话框并按当前范围生成。
function openSummary() {
  summaryVisible.value = true
  loadSummary()
}

async function loadSummary() {
  summaryLoading.value = true
  try {
    summary.value = await summarizeTasks(summaryScope.value)
  } catch (e) {
    summary.value = null
    ElMessage.error((e as Error).message || t('taskPage.summaryFailed'))
  } finally {
    summaryLoading.value = false
  }
}

onMounted(() => {
  refreshCurrent()
  loadAICapabilities()
})

/**
 * 从收件箱的任务提醒（link 带 task_id）定位任务。
 * 不依赖当前列表里恰好有它：筛选条件、分页与列表刷新都可能把它挡在外面。
 */
async function openTaskById(id: string) {
  try {
    openDialog(await getTask(id))
  } catch {
    // 失败提示由拦截器统一弹出（任务可能已删除，或已不属于当前用户）
  }
}

// 已在任务页时再点另一条提醒只会改 query（不重新 mount），所以用 watch
watch(
  () => route.query.task_id,
  id => {
    if (typeof id === 'string' && id) openTaskById(id)
  },
  { immediate: true }
)
</script>

<style scoped>
.stat-row {
  margin-bottom: 16px;
}
.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 2px;
}
.stat-card {
  text-align: center;
}
.stat-card--clickable {
  cursor: pointer;
  transition: box-shadow 0.2s;
}
.stat-card--clickable:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.12);
}
.drill-hint {
  margin-left: 4px;
  font-size: 11px;
  color: #c0c4cc;
}
.stat-num {
  font-size: 28px;
  font-weight: 600;
  color: #3b82f6;
}
.stat-num.danger {
  color: #ef4444;
}
.stat-label {
  margin-top: 4px;
  color: #6b7280;
  font-size: 13px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.batch-bar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.batch-tip {
  margin-right: 12px;
  color: #6b7280;
  font-size: 13px;
}
.batch-dropdown {
  margin-right: 12px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.ai-parse {
  width: 100%;
}
.ai-parse__btn {
  margin-top: 8px;
}
.ai-inline-btn {
  margin-top: 6px;
}
.ai-field-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.summary-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.summary-tip {
  font-size: 12px;
  color: #909399;
}
.summary-body {
  min-height: 120px;
}
.summary-text {
  white-space: pre-wrap;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  margin: 0 0 12px;
}
.summary-block__title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 4px;
  color: #606266;
}
.summary-block ul {
  margin: 0 0 12px;
  padding-left: 20px;
  color: #606266;
  line-height: 1.6;
}
/* ===== 子任务展开面板 ===== */
.subtask-panel {
  padding: 8px 16px 8px 48px;
  background: var(--el-fill-color-lighter);
}
.subtask-loading,
.subtask-empty {
  font-size: 12px;
  color: #909399;
  line-height: 1.8;
}
.subtask-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.subtask-row:last-child {
  border-bottom: none;
}
.subtask-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}
.subtask-assignee,
.subtask-due {
  font-size: 12px;
  color: #6b7280;
}
/* 列表“负责人”列多负责人折叠标记 +N，与主负责人名同一行展示 */
.assignee-more {
  margin-left: 4px;
}
/* 列表“标题”列阻塞徽标，与标题文本同行前置展示 */
.dep-badge {
  margin-right: 6px;
}
/* ===== 看板视图 ===== */
.board {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  overflow-x: auto;
  padding-bottom: 8px;
}
.board-col {
  flex: 1 0 260px;
  min-width: 260px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 8px;
  transition: background 0.2s, box-shadow 0.2s;
}
.board-col--dragover {
  background: var(--el-color-primary-light-9);
  box-shadow: inset 0 0 0 2px var(--el-color-primary-light-5);
}
.board-col__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px 10px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.board-col__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #909399;
}
.board-col__dot--pending {
  background: #909399;
}
.board-col__dot--in_progress {
  background: #e6a23c;
}
.board-col__dot--completed {
  background: #67c23a;
}
.board-col__title {
  flex: 1;
}
.board-col__count {
  font-size: 12px;
  color: #909399;
  background: var(--el-bg-color-overlay);
  border-radius: 10px;
  padding: 0 8px;
  line-height: 18px;
}
.board-card {
  background: var(--el-bg-color-overlay);
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  cursor: grab;
}
.board-card:active {
  cursor: grabbing;
}
.board-card--dragging {
  opacity: 0.5;
}
.board-card__title {
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
  word-break: break-word;
}
.board-card__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.board-card__due {
  font-size: 12px;
  color: #909399;
}
.board-card__due--overdue {
  color: #ef4444;
}
.board-col__empty {
  padding: 16px;
  text-align: center;
  color: #c0c4cc;
  font-size: 13px;
}
.board-truncated {
  width: 100%;
  padding: 8px;
  text-align: center;
  color: #e6a23c;
  font-size: 12px;
}
</style>
