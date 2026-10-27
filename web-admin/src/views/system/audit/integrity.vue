<template>
  <div class="page">
    <!-- 说明与操作 -->
    <el-card shadow="never" class="intro-card">
      <div class="intro-bar">
        <div class="intro-text">
          <div class="intro-title">审计日志防篡改完整性校验</div>
          <div class="intro-desc">
            对每行审计日志重算 HMAC 行摘要，并沿锚点链逐段核对窗口折叠摘要，检出内容篡改、删行、插行、重序与断链。
            校验为只读操作，不修改任何数据。
          </div>
        </div>
        <el-button type="primary" :loading="loading" @click="runVerify">
          <el-icon><CircleCheck /></el-icon>开始校验
        </el-button>
        <el-button v-if="canRehash" type="warning" :loading="rehashing" @click="runRehash">
          <el-icon><Refresh /></el-icon>密钥重算（rehash）
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
      title="完整性校验未启用"
      description="服务端未配置完整性密钥（audit.integrity.secret_key / METEORX_AUDIT_INTEGRITY_SECRET_KEY）。历史与新写入的审计日志均不含行摘要，无法进行防篡改校验。"
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
              <div class="stat-label">重算行数</div>
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
              <div class="stat-label">旧数据（无摘要）</div>
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
              <div class="stat-label">篡改行数</div>
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
              <div class="stat-label">校验锚点</div>
              <div class="stat-value">{{ result.anchors_checked }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 覆盖范围 -->
    <el-card v-if="result && result.enabled && (result.covered_from || result.covered_to)" shadow="never" class="cover-card">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="锚点链覆盖起始">{{ result.covered_from || '—' }}</el-descriptions-item>
        <el-descriptions-item label="锚点链覆盖结束">{{ result.covered_to || '—' }}</el-descriptions-item>
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
          <span>待密钥重算</span>
          <el-tag type="warning" size="small">可执行 rehash</el-tag>
        </div>
      </template>
      <div class="anomaly-block">
        <div class="block-title">
          <el-icon><Warning /></el-icon>
          仅匹配旧 pepper、待重算到新密钥的历史行：{{ result.pending_rehash_rows }}
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
          <span>异常明细</span>
          <el-tag type="danger" size="small">需排查</el-tag>
        </div>
      </template>

      <div v-if="result.anomalies.length > 0" class="anomaly-block">
        <div class="block-title">异常描述</div>
        <ul class="anomaly-list">
          <li v-for="(a, i) in result.anomalies" :key="'a' + i">{{ a }}</li>
        </ul>
      </div>

      <div v-if="result.tampered_samples.length > 0" class="anomaly-block">
        <div class="block-title">篡改行样本 ID（最多若干条）</div>
        <div class="tag-wrap">
          <el-tag v-for="id in result.tampered_samples" :key="'t' + id" type="danger" size="small" class="id-tag">
            {{ id }}
          </el-tag>
        </div>
      </div>

      <div v-if="result.anomaly_anchor_ids.length > 0" class="anomaly-block">
        <div class="block-title">异常锚点 ID</div>
        <div class="tag-wrap">
          <el-tag v-for="id in result.anomaly_anchor_ids" :key="'n' + id" type="warning" size="small" class="id-tag">
            {{ id }}
          </el-tag>
        </div>
      </div>
    </el-card>

    <!-- 首次进入空状态 -->
    <el-empty v-if="!loading && !result" description='点击"开始校验"检查审计日志是否被篡改' />
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
      return '校验通过：未发现篡改'
    case 'tampered':
      return '警告：检测到数据被篡改'
    case 'pending_rehash':
      return '提示：检测到待密钥重算数据'
    default:
      return '未知校验结果'
  }
})

const verdictDesc = computed(() => {
  switch (result.value?.verdict) {
    case 'verified':
      return '所有参与校验的审计日志行摘要与锚点链均自洽。'
    case 'tampered':
      return '存在行摘要重算不符或锚点链断裂，请结合下方异常明细排查。'
    case 'pending_rehash':
      return '存在仅在旧 pepper 下可验证的历史行（密钥轮换所致，非篡改），可执行密钥重算迁移到新密钥并重建锚点链。'
    default:
      return ''
  }
})

async function runVerify() {
  loading.value = true
  try {
    result.value = await verifyAuditIntegrity()
    if (result.value.enabled && result.value.verdict === 'tampered') {
      ElMessage.error('检测到审计日志被篡改，请排查异常明细')
    } else if (result.value.enabled && result.value.verdict === 'verified') {
      ElMessage.success('完整性校验通过')
    } else if (result.value.enabled && result.value.verdict === 'pending_rehash') {
      ElMessage.warning('检测到待密钥重算的历史行，可执行密钥重算')
    }
  } catch (error) {
    console.error('完整性校验失败', error)
  } finally {
    loading.value = false
  }
}

async function runRehash() {
  try {
    await ElMessageBox.confirm(
      '将把仅匹配旧 pepper 的历史行重算到新 pepper，并清空重建整条锚点链。此操作会修改审计日志的行摘要列，请确认已完成密钥轮换配置。',
      '确认执行密钥重算（rehash）？',
      { type: 'warning', confirmButtonText: '执行', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  rehashing.value = true
  try {
    const res = await rehashAuditIntegrity()
    ElMessage.success(res.message || '密钥重算完成')
    // 重算后立即复检，预期回到 verified
    await runVerify()
  } catch (error) {
    console.error('密钥重算失败', error)
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
