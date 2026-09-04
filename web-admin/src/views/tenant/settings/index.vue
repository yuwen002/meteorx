<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span class="title">租户设置</span>
              <el-tag v-if="loading" type="info" size="small">加载中...</el-tag>
            </div>
          </template>

          <el-form
            ref="formRef"
            :model="form"
            :rules="formRules"
            label-width="120px"
            class="settings-form"
          >
            <el-tabs v-model="activeTab">
              <el-tab-pane label="基本信息" name="basic">
                <el-form-item label="租户名称" prop="welcome_text">
                  <el-input v-model="form.welcome_text" placeholder="设置租户欢迎语" />
                </el-form-item>
                <el-form-item label="租户描述" prop="description">
                  <el-input
                    v-model="form.description"
                    type="textarea"
                    :rows="3"
                    placeholder="设置租户描述信息"
                  />
                </el-form-item>
                <el-form-item label="联系姓名" prop="contact_name">
                  <el-input v-model="form.contact_name" placeholder="联系人姓名" />
                </el-form-item>
                <el-form-item label="联系邮箱" prop="contact_email">
                  <el-input v-model="form.contact_email" placeholder="联系邮箱" />
                </el-form-item>
                <el-form-item label="联系电话" prop="contact_phone">
                  <el-input v-model="form.contact_phone" placeholder="联系电话" />
                </el-form-item>
                <el-form-item label="地址" prop="address">
                  <el-input v-model="form.address" placeholder="地址" />
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane label="品牌定制" name="branding">
                <el-form-item label="Logo" prop="logo">
                  <el-input v-model="form.logo" placeholder="Logo URL" />
                  <div v-if="form.logo" class="preview">
                    <img :src="form.logo" alt="Logo" class="preview-logo" />
                  </div>
                </el-form-item>
                <el-form-item label="Favicon" prop="favicon">
                  <el-input v-model="form.favicon" placeholder="Favicon URL" />
                </el-form-item>
                <el-form-item label="主题色" prop="primary_color">
                  <div class="color-picker-wrapper">
                    <el-color-picker v-model="form.primary_color" />
                    <el-input
                      v-model="form.primary_color"
                      placeholder="#667eea"
                      style="width: 120px; margin-left: 12px;"
                    />
                  </div>
                </el-form-item>
                <el-form-item label="主题风格" prop="theme">
                  <el-select v-model="form.theme" placeholder="选择主题" style="width: 200px;">
                    <el-option label="明亮主题" value="light" />
                    <el-option label="暗色主题" value="dark" />
                    <el-option label="跟随系统" value="auto" />
                  </el-select>
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane label="本地化" name="localization">
                <el-form-item label="默认语言" prop="language">
                  <el-select v-model="form.language" placeholder="选择语言" style="width: 200px;">
                    <el-option label="简体中文" value="zh-CN" />
                    <el-option label="English" value="en-US" />
                    <el-option label="日本語" value="ja-JP" />
                  </el-select>
                </el-form-item>
                <el-form-item label="时区" prop="timezone">
                  <el-select v-model="form.timezone" placeholder="选择时区" style="width: 200px;">
                    <el-option label="(GMT+08:00) 北京" value="Asia/Shanghai" />
                    <el-option label="(GMT+00:00) 伦敦" value="Europe/London" />
                    <el-option label="(GMT-05:00) 纽约" value="America/New_York" />
                    <el-option label="(GMT-08:00) 洛杉矶" value="America/Los_Angeles" />
                    <el-option label="(GMT+09:00) 东京" value="Asia/Tokyo" />
                  </el-select>
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane label="扩展配置" name="extra">
                <el-form-item label="扩展配置" prop="extra">
                  <el-input
                    v-model="form.extra"
                    type="textarea"
                    :rows="8"
                    placeholder='{"key": "value"}'
                  />
                  <div class="extra-tip">支持 JSON 格式的扩展配置</div>
                </el-form-item>
              </el-tab-pane>
            </el-tabs>

            <div class="form-actions">
              <el-button type="primary" :loading="saving" @click="handleSave">
                保存设置
              </el-button>
              <el-button @click="handleReset">重置</el-button>
            </div>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never">
          <template #header>
            <span class="title">预览</span>
          </template>
          <div class="preview-panel">
            <div
              class="preview-header"
              :style="{ background: form.primary_color || '#667eea' }"
            >
              <img
                v-if="form.logo"
                :src="form.logo"
                alt="Logo"
                class="preview-header-logo"
              />
              <span v-else class="preview-header-title">{{ form.welcome_text || 'MeteorX 平台' }}</span>
            </div>
            <div class="preview-body">
              <div class="preview-welcome">
                <h3>{{ form.welcome_text || '欢迎使用' }}</h3>
                <p>{{ form.description || '租户描述将显示在此处' }}</p>
              </div>
              <div class="preview-info">
                <div v-if="form.contact_name" class="info-item">
                  <span class="info-label">联系人:</span>
                  <span class="info-value">{{ form.contact_name }}</span>
                </div>
                <div v-if="form.contact_email" class="info-item">
                  <span class="info-label">邮箱:</span>
                  <span class="info-value">{{ form.contact_email }}</span>
                </div>
                <div v-if="form.contact_phone" class="info-item">
                  <span class="info-label">电话:</span>
                  <span class="info-value">{{ form.contact_phone }}</span>
                </div>
                <div v-if="form.address" class="info-item">
                  <span class="info-label">地址:</span>
                  <span class="info-value">{{ form.address }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">语言:</span>
                  <span class="info-value">{{ getLanguageLabel(form.language) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">时区:</span>
                  <span class="info-value">{{ getTimezoneLabel(form.timezone) }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { getTenantSettings, updateTenantSettings } from '@/api/modules/tenant'
import type { TenantSettings, UpdateTenantSettingsReq } from '@/api/modules/tenant'

const formRef = ref<FormInstance>()
const loading = ref(false)
const saving = ref(false)
const activeTab = ref('basic')

const form = reactive<TenantSettings>({
  id: '',
  tenant_id: '',
  logo: '',
  favicon: '',
  primary_color: '#667eea',
  theme: 'light',
  language: 'zh-CN',
  timezone: 'Asia/Shanghai',
  description: '',
  welcome_text: '',
  contact_name: '',
  contact_email: '',
  contact_phone: '',
  address: '',
  extra: '',
  created_at: '',
  updated_at: ''
})

const formRules: FormRules = {
  contact_email: [
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  extra: [
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (value && value.trim()) {
          try {
            JSON.parse(value)
          } catch {
            callback(new Error('请输入有效的 JSON 格式'))
            return
          }
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

onMounted(() => {
  loadSettings()
})

async function loadSettings() {
  loading.value = true
  try {
    const res = await getTenantSettings()
    Object.assign(form, res)
  } catch (error) {
    console.error('加载租户设置失败', error)
    ElMessage.warning('加载设置失败，使用默认配置')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      const data: UpdateTenantSettingsReq = {
        logo: form.logo,
        favicon: form.favicon,
        primary_color: form.primary_color,
        theme: form.theme,
        language: form.language,
        timezone: form.timezone,
        description: form.description,
        welcome_text: form.welcome_text,
        contact_name: form.contact_name,
        contact_email: form.contact_email,
        contact_phone: form.contact_phone,
        address: form.address,
        extra: form.extra
      }
      const res = await updateTenantSettings(data)
      Object.assign(form, res)
      ElMessage.success('保存成功')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.message || '保存失败')
    } finally {
      saving.value = false
    }
  })
}

function handleReset() {
  if (formRef.value) {
    formRef.value.resetFields()
  }
  Object.assign(form, {
    logo: '',
    favicon: '',
    primary_color: '#667eea',
    theme: 'light',
    language: 'zh-CN',
    timezone: 'Asia/Shanghai',
    description: '',
    welcome_text: '',
    contact_name: '',
    contact_email: '',
    contact_phone: '',
    address: '',
    extra: ''
  })
}

function getLanguageLabel(lang: string): string {
  const map: Record<string, string> = {
    'zh-CN': '简体中文',
    'en-US': 'English',
    'ja-JP': '日本語'
  }
  return map[lang] || lang
}

function getTimezoneLabel(tz: string): string {
  const map: Record<string, string> = {
    'Asia/Shanghai': '(GMT+08:00) 北京',
    'Europe/London': '(GMT+00:00) 伦敦',
    'America/New_York': '(GMT-05:00) 纽约',
    'America/Los_Angeles': '(GMT-08:00) 洛杉矶',
    'Asia/Tokyo': '(GMT+09:00) 东京'
  }
  return map[tz] || tz
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .title {
    font-size: 16px;
    font-weight: 500;
  }
}

.settings-form {
  padding: 0 16px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.color-picker-wrapper {
  display: flex;
  align-items: center;
}

.extra-tip {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 4px;
}

.preview {
  margin-top: 8px;
}

.preview-logo {
  max-height: 60px;
  max-width: 200px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  padding: 4px;
}

.preview-panel {
  .preview-header {
    height: 80px;
    border-radius: 8px 8px 0 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    overflow: hidden;

    .preview-header-logo {
      max-height: 60px;
      max-width: 180px;
    }

    .preview-header-title {
      font-size: 18px;
      font-weight: 500;
    }
  }

  .preview-body {
    padding: 16px;
    background: #fafafa;
    border-radius: 0 0 8px 8px;
  }

  .preview-welcome {
    margin-bottom: 16px;

    h3 {
      margin: 0 0 8px;
      font-size: 18px;
      color: #262626;
    }

    p {
      margin: 0;
      font-size: 14px;
      color: #595959;
    }
  }

  .preview-info {
    .info-item {
      display: flex;
      font-size: 13px;
      margin-bottom: 8px;

      .info-label {
        color: #8c8c8c;
        min-width: 60px;
      }

      .info-value {
        color: #262626;
      }
    }
  }
}
</style>