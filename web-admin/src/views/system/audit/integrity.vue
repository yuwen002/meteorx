<template>
  <div class="page">
    <!-- 说明与操作 -->
    <el-card shadow="never" class="intro-card">
      <div class="intro-bar">
        <div class="intro-text">
          <div class="intro-title">{{ t('integrityPage.intro.title') }}</div>
          <div class="intro-desc">
            {{ t('integrityPage.intro.desc') }}
          </div>
        </div>
        <el-button type="primary" :loading="loading" @click="runVerify">
          <el-icon><CircleCheck /></el-icon>{{ t('integrityPage.verify.start') }}
        </el-button>
        <el-button v-if="canRehash" type="warning" :loading="rehashing" @click="runRehash">
          <el-icon><Refresh /></el-icon>{{ t('integrityPage.rehash.btn') }}
        </el-button>
      </div>
    </el-card>

    <!-- 未配置提示：pepper 缺失时整卡降级，不阻断，但此处明确告知管理员功能未生效 -->
    <el-alert
      v-if="result && !result.enabled"
      type="warning"
      show-icon
      :closable="false"
      class="state-alert"
      :title="t('integrityPage.alert.disabledTitle')"
      :description="t('integrityPage.alert.disabledDesc')"
    />

    <!-- 结论横幅 -->
    <el-alert
      v-if="result && result.enabled"
      :type="verdictAlertType"
      show-icon
      :closable="false"
      class="state-alert"
      :title="verdictTitle"
      :description="result.message || verdictDesc"
    />

    <!-- 统计概览 -->
    <el-row v-if="result && result.enabled" :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon checked-icon"><el-icon><Document /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">{{ t('integrityPage.stat.checked') }}</div>
              <div class="stat-value">{{ result.checked_rows }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon legacy-icon"><el-icon><Clock /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">{{ t('integrityPage.stat.legacy') }}</div>
              <div class="stat-value">{{ result.legacy_rows }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon tampered-icon"><el-icon><CircleClose /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">{{ t('integrityPage.stat.tampered') }}</div>
              <div class="stat-value" :class="{ danger: result.tampered_rows > 0 }">{{ result.tampered_rows }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon anchor-icon"><el-icon><Connection /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">{{ t('integrityPage.stat.anchors') }}</div>
              <div class="stat-value">{{ result.anchors_checked }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 覆盖范围 -->
    <el-card v-if="result && result.enabled && (result.covered_from || result.covered_to)" shadow="never" class="cover-card">
      <el-descriptions :column="2" border>
        <el-descriptions-item :label="t('integrityPage.cover.from')">{{ result.covered_from || '—' }}</el-descriptions-item>
        <el-descriptions-item :label="t('integrityPage.cover.to')">{{ result.covered_to || '—' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 待密钥重算明细：仅在轮换过渡期（pending_rehash）展示，与篡改异常区分 -->
    <el-card
      v-if="result && result.enabled && result.verdict === 'pending_rehash'"
      shadow="never"
      class="anomaly-card"
    >
      <template #header>
        <div class="card-header">
          <span>{{ t('integrityPage.pending.title') }}</span>
          <el-tag type="warning" size="small">{{ t('integrityPage.pending.tag') }}</el-tag>
        </div>
      </template>
      <div class="anomaly-block">
        <div class="block-title">
          <el-icon><Warning /></el-icon>
          {{ t('integrityPage.pending.blockTitle', result.pending_rehash_rows) }}
        </div>
        <div v-if="result.pending_rehash_samples.length > 0" class="tag-wrap">
          <el-tag v-for="id in result.pending_rehash_samples" :key="'p' + id" type="warning" size="small" class="id-tag">
            {{ id }}
          </el-tag>
        </div>
      </div>
    </el-card>

    <!-- 异常明细 -->
    <el-card v-if="result && result.enabled && hasAnomaly" shadow="never" class="anomaly-card">
      <template #header>
        <div class="card-header">
          <span>{{ t('integrityPage.anomaly.title') }}</span>
          <el-tag type="danger" size="small">{{ t('integrityPage.anomaly.tag') }}</el-tag>
        </div>
      </template>

      <div v-if="result.anomalies.length > 0" class="anomaly-block">
        <div class="block-title">{{ t('integrityPage.anomaly.descTitle') }}</div>
        <ul class="anomaly-list">
          <li v-for="(a, i) in result.anomalies" :key="'a' + i">{{ a }}</li>
        </ul>
      </div>

      <div v-if="result.tampered_samples.length > 0" class="anomaly-block">
        <div class="block-title">{{ t('integrityPage.anomaly.samplesTitle') }}</div>
        <div class="tag-wrap">
          <el-tag v-for="id in result.tampered_samples" :key="'t' + id" type="danger" size="small" class="id-tag">
            {{ id }}
          </el-tag>
        </div>
      </div>

      <div v-if="result.anomaly_anchor_ids.length > 0" class="anomaly-block">
        <div class="block-title">{{ t('integrityPage.anomaly.anchorTitle') }}</div>
        <div class="tag-wrap">
          <el-tag v-for="id in result.anomaly_anchor_ids" :key="'n' + id" type="warning" size="small" class="id-tag">
            {{ id }}
          </el-tag>
        </div>
      </div>
    </el-card>

    <!-- 首次进入空状态 -->
    <el-empty v-if="!loading && !result" :description="t('integrityPage.empty.hint')" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  CircleCheck, CircleClose, Document, Clock, Connection, Refresh, Warning
} from '@element-plus/icons-vue'
import { verifyAuditIntegrity, rehashAuditIntegrity } from '@/api/modules/audit'
import type { IntegrityVerifyResult } from '@/api/modules/audit'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const result = ref<IntegrityVerifyResult | null>(null)
const loading = ref(false)
const rehashing = ref(false)

const hasAnomaly = computed(() => {
  const r = result.value
  if (!r) return false
  return r.tampered_rows > 0 || r.tampered_samples.length > 0 || r.anomaly_anchor_ids.length > 0 || r.anomalies.length > 0
})

// 处于密钥轮换过渡期（verdict=pending_rehash）才允许 rehash：把仅匹配旧 pepper 的历史行迁移到新密钥
const canRehash = computed(() => !!result.value?.enabled && result.value?.verdict === 'pending_rehash')

const verdictAlertType = computed<'success' | 'error' | 'warning' | 'info'>(() => {
  switch (result.value?.verdict) {
    case 'verified':
      return 'success'
    case 'tampered':
      return 'error'
    case 'pending_rehash':
      return 'warning'
    default:
      return 'info'
  }
})

const verdictTitle = computed(() => {
  switch (result.value?.verdict) {
    case 'verified':
      return t('integrityPage.verdict.verified.title')
    case 'tampered':
      return t('integrityPage.verdict.tampered.title')
    case 'pending_rehash':
      return t('integrityPage.verdict.pending.title')
    default:
      return t('integrityPage.verdict.unknown.title')
  }
})

const verdictDesc = computed(() => {
  switch (result.value?.verdict) {
    case 'verified':
      return t('integrityPage.verdict.verified.desc')
    case 'tampered':
      return t('integrityPage.verdict.tampered.desc')
    case 'pending_rehash':
      return t('integrityPage.verdict.pending.desc')
    default:
      return ''
  }
})

async function runVerify() {
  loading.value = true
  try {
    result.value = await verifyAuditIntegrity()
    if (result.value.enabled && result.value.verdict === 'tampered') {
      ElMessage.error(t('integrityPage.msg.tampered'))
    } else if (result.value.enabled && result.value.verdict === 'verified') {
      ElMessage.success(t('integrityPage.msg.verified'))
    } else if (result.value.enabled && result.value.verdict === 'pending_rehash') {
      ElMessage.warning(t('integrityPage.msg.pending'))
    }
  } catch (error) {
    console.error('failed to verify integrity', error)
  } finally {
    loading.value = false
  }
}

async function runRehash() {
  try {
    await ElMessageBox.confirm(
      t('integrityPage.rehash.confirmMsg'),
      t('integrityPage.rehash.confirmTitle'),
      { type: 'warning', confirmButtonText: t('integrityPage.rehash.confirmBtn'), cancelButtonText: t('common.cancel') }
    )
  } catch {
    return
  }
  rehashing.value = true
  try {
    const res = await rehashAuditIntegrity()
    ElMessage.success(res.message || t('integrityPage.rehash.done'))
    // 重算后立即复检，预期回到 verified
    await runVerify()
  } catch (error) {
    console.error('failed to rehash integrity', error)
  } finally {
    rehashing.value = false
  }
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.intro-card {
  margin-bottom: 16px;

  .intro-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .intro-text {
    flex: 1;
    min-width: 240px;
  }

  .intro-title {
    font-size: 16px;
    font-weight: 600;
    color: #262626;
    margin-bottom: 6px;
  }

  .intro-desc {
    font-size: 13px;
    color: #8c8c8c;
    line-height: 1.6;
  }
}

.state-alert {
  margin-bottom: 16px;
}

.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  .stat-content {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .stat-icon {
    width: 48px;
    height: 48px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;

    &.checked-icon { background: #e6f7ff; color: #1890ff; }
    &.legacy-icon { background: #f5f5f5; color: #8c8c8c; }
    &.tampered-icon { background: #fff2f0; color: #f56c6c; }
    &.anchor-icon { background: #f6ffed; color: #52c41a; }
  }

  .stat-info {
    flex: 1;
  }

  .stat-label {
    font-size: 14px;
    color: #8c8c8c;
    margin-bottom: 4px;
  }

  .stat-value {
    font-size: 24px;
    font-weight: 600;
    color: #262626;

    &.danger { color: #f56c6c; }
  }
}

.cover-card {
  margin-bottom: 16px;
}

.anomaly-card {
  margin-top: 16px;

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .anomaly-block {
    margin-bottom: 16px;

    &:last-child { margin-bottom: 0; }
  }

  .block-title {
    font-size: 13px;
    font-weight: 600;
    color: #595959;
    margin-bottom: 8px;
  }

  .anomaly-list {
    margin: 0;
    padding-left: 20px;

    li {
      font-size: 13px;
      color: #434343;
      line-height: 1.8;
    }
  }

  .tag-wrap {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .id-tag {
    font-family: 'Menlo', 'Consolas', monospace;
  }
}
</style>
