<template>
  <div class="share-page">
    <div v-if="loading" class="center-box">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else-if="fatal" class="center-box">
      <el-result icon="warning" :title="fatal.title" :sub-title="fatal.message">
        <template #extra>
          <el-button type="primary" @click="goHome">{{ t('wiki.shareView.backHome') }}</el-button>
        </template>
      </el-result>
    </div>

    <template v-else-if="doc">
      <header class="share-header">
        <div class="share-header__main">
          <div class="share-badge">
            <el-icon><Share /></el-icon>
            {{ t('wiki.shareView.badge') }}
          </div>
          <h1>{{ doc.title }}</h1>
          <div class="share-meta">
            <span v-if="doc.view_count">{{ t('wiki.shareView.viewCount', doc.view_count) }}</span>
            <span v-if="doc.updated_at">{{ t('wiki.shareView.lastUpdated', formatTime(doc.updated_at)) }}</span>
            <span v-if="doc.allow_download" class="tip">{{ t('wiki.shareView.allowDownloadTip') }}</span>
          </div>
        </div>
        <el-button v-if="doc.allow_download" type="primary" plain @click="downloadMarkdown">
          <el-icon style="margin-right: 4px"><Download /></el-icon>
          {{ t('wiki.shareView.downloadMarkdown') }}
        </el-button>
      </header>

      <!-- 正文已经后端转义+清洗，前端再经 DOMPurify 消毒，v-html 可信 -->
      <!-- eslint-disable-next-line vue/no-v-html -->
      <main v-if="doc.content_html" class="markdown-body" v-html="sanitizeHtml(doc.content_html)"></main>
      <pre v-else class="plain-content">{{ doc.content }}</pre>

      <footer class="share-footer">{{ t('wiki.shareView.footer') }}</footer>
    </template>

    <!-- 密码访问弹窗 -->
    <el-dialog
      v-model="passwordDialog"
      :title="t('wiki.shareView.pwdTitle')"
      width="420px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <p class="dialog-tip">{{ t('wiki.shareView.pwdTip') }}</p>
      <el-input
        v-model="password"
        type="password"
        show-password
        :placeholder="t('wiki.shareView.pwdPlaceholder')"
        @keyup.enter="submitPassword"
      />
      <div v-if="passwordError" class="password-error">{{ passwordError }}</div>
      <template #footer>
        <el-button @click="cancelPassword">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="checking" @click="submitPassword">{{ t('wiki.shareView.pwdConfirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { sanitizeHtml } from '@/utils/sanitize'
import { useI18n } from '@/composables/useI18n'
import { Download, Share } from '@element-plus/icons-vue'

interface SharedDocumentView {
  document_id: string
  node_id: string
  title: string
  content: string
  content_html: string
  format: string
  last_edited_by?: string
  updated_at?: string
  allow_download: boolean
  view_count: number
  max_views?: number
  expire_at?: string
}

// 免登录接口可能返回 { code, message, data } 包裹或直接返回文档对象
interface ShareApiResponse {
  code?: number
  message?: string
  data?: SharedDocumentView
  document_id?: string
}

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const token = route.params.token as string

const doc = ref<SharedDocumentView | null>(null)
const loading = ref(true)
const checking = ref(false)
const fatal = ref<{ title: string; message: string } | null>(null)

const passwordDialog = ref(false)
const password = ref('')
const passwordError = ref('')

// The password-protected share endpoint signals with Chinese messages,
// so match by code points to keep this source ASCII
const PWD_REQUIRED_RE = /\u9700\u8981\u5bc6\u7801/
const PWD_INVALID_RE = /\u5bc6\u7801\u9519\u8bef/

onMounted(() => {
  void fetchDoc()
})

// 免登录读取分享内容：不走 request.ts 拦截器（避免把 4xx 误判为登录失效）
async function fetchDoc(pwd?: string): Promise<void> {
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/wiki/share/${encodeURIComponent(token)}`, {
      params: pwd ? { password: pwd } : {},
      timeout: 15000
    })
    const body = res.data as ShareApiResponse
    if (body && typeof body === 'object' && body.code !== undefined && body.code !== 0 && body.code !== 200) {
      throw new Error(body.message || t('wiki.shareView.loadFailed'))
    }
    const data = (body?.data ?? body) as SharedDocumentView
    if (data && typeof data === 'object' && data.document_id) {
      doc.value = data
      fatal.value = null
      passwordDialog.value = false
      return
    }
    throw new Error(t('wiki.shareView.badFormat'))
  } catch (err) {
    const e = err as {
      response?: { status?: number; statusText?: string; data?: { message?: string } }
      message?: string
    }
    const status = e?.response?.status
    const message = e?.response?.data?.message || e?.response?.statusText || e?.message || t('wiki.shareView.loadFailed')
    if (status === 403 && (PWD_REQUIRED_RE.test(message) || PWD_INVALID_RE.test(message))) {
      // Password-protected share: stay on the password dialog (view count is only raised after a valid password)
      passwordDialog.value = true
      if (pwd) {
        passwordError.value = PWD_REQUIRED_RE.test(message) ? '' : t('wiki.shareView.pwdError')
      } else {
        passwordError.value = ''
      }
      fatal.value = null
      doc.value = null
      return
    }
    fatal.value = {
      title: status === 404 ? t('wiki.shareView.notFoundTitle') : t('wiki.shareView.errorTitle'),
      message
    }
    doc.value = null
  } finally {
    loading.value = false
  }
}

async function submitPassword() {
  if (!password.value.trim()) {
    passwordError.value = t('wiki.shareView.pwdRequired')
    return
  }
  checking.value = true
  passwordError.value = ''
  try {
    await fetchDoc(password.value)
    password.value = ''
  } catch {
    passwordError.value = t('wiki.shareView.pwdError')
  } finally {
    checking.value = false
  }
}

function cancelPassword() {
  passwordDialog.value = false
  password.value = ''
  passwordError.value = ''
  if (!doc.value && !fatal.value) {
    fatal.value = { title: t('wiki.shareView.noPwdTitle'), message: t('wiki.shareView.noPwdMessage') }
  }
}

function downloadMarkdown() {
  if (!doc.value) return
  const blob = new Blob([doc.value.content || ''], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${doc.value.title || 'share-document'}.md`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function formatTime(time: string) {
  const date = new Date(time)
  if (Number.isNaN(date.getTime())) return time
  return date.toLocaleString(locale.value, { hour12: false })
}

function goHome() {
  router.push('/')
}
</script>

<style scoped>
.share-page {
  min-height: 100vh;
  background: #f5f6f8;
  padding: 32px 16px 0;
}

.center-box {
  max-width: 860px;
  margin: 0 auto;
  background: #fff;
  border-radius: 8px;
  padding: 32px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.share-header {
  max-width: 860px;
  margin: 0 auto 20px;
  background: #fff;
  border-radius: 8px;
  padding: 24px 32px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.share-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #2563eb;
  background: #eff6ff;
  border-radius: 999px;
  padding: 2px 10px;
  margin-bottom: 10px;
}

.share-header h1 {
  margin: 0 0 8px;
  font-size: 24px;
  line-height: 1.4;
  color: #111827;
}

.share-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  color: #9ca3af;
  font-size: 12px;
}

.share-meta .tip {
  color: #059669;
}

.markdown-body {
  max-width: 860px;
  margin: 0 auto;
  background: #fff;
  border-radius: 8px;
  padding: 32px 40px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  line-height: 1.7;
  font-size: 15px;
  color: #1f2937;
  overflow-wrap: break-word;
}

/* v-html 渲染的内容需要深穿透设置排版 */
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  margin: 1.2em 0 0.6em;
  font-weight: 600;
  line-height: 1.35;
  color: #111827;
}
.markdown-body :deep(h1) {
  font-size: 1.6em;
  border-bottom: 1px solid #e5e7eb;
  padding-bottom: 0.3em;
}
.markdown-body :deep(h2) {
  font-size: 1.3em;
  border-bottom: 1px solid #f3f4f6;
  padding-bottom: 0.3em;
}
.markdown-body :deep(h3) {
  font-size: 1.1em;
}
.markdown-body :deep(p) {
  margin: 0.6em 0;
}
.markdown-body :deep(a) {
  color: #2563eb;
}
.markdown-body :deep(code) {
  background: #f3f4f6;
  border-radius: 4px;
  padding: 0.15em 0.4em;
  font-size: 0.9em;
  color: #dc2626;
}
.markdown-body :deep(pre) {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 14px 16px;
  overflow-x: auto;
}
.markdown-body :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}
.markdown-body :deep(blockquote) {
  margin: 0.8em 0;
  padding: 0.4em 1em;
  border-left: 4px solid #dbeafe;
  color: #4b5563;
  background: #f8fafc;
}
.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: 6px;
}
.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.8em 0;
}
.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid #e5e7eb;
  padding: 6px 12px;
  text-align: left;
}
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.6em;
  margin: 0.5em 0;
}
.markdown-body :deep(hr) {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 1.4em 0;
}

.plain-content {
  max-width: 860px;
  margin: 0 auto;
  background: #fff;
  border-radius: 8px;
  padding: 24px 32px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
}

.share-footer {
  max-width: 860px;
  margin: 16px auto 32px;
  text-align: center;
  color: #c3c8d0;
  font-size: 12px;
}

.dialog-tip {
  margin: 0 0 12px;
  color: #6b7280;
  font-size: 13px;
}

.password-error {
  margin-top: 10px;
  color: #ef4444;
  font-size: 13px;
}
</style>
