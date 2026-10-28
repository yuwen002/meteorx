<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span class="title">{{ t('tenantSettingsPage.title') }}</span>
              <el-tag v-if="loading" type="info" size="small">{{ t('common.loading') }}</el-tag>
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
              <el-tab-pane :label="t('tenantSettingsPage.tab.basic')" name="basic">
                <el-form-item :label="t('tenantSettingsPage.form.tenantName')" prop="welcome_text">
                  <el-input v-model="form.welcome_text" :placeholder="t('tenantSettingsPage.form.welcomePlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.description')" prop="description">
                  <el-input
                    v-model="form.description"
                    type="textarea"
                    :rows="3"
                    :placeholder="t('tenantSettingsPage.form.descriptionPlaceholder')"
                  />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.contactName')" prop="contact_name">
                  <el-input v-model="form.contact_name" :placeholder="t('tenantSettingsPage.form.contactNamePlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.contactEmail')" prop="contact_email">
                  <el-input v-model="form.contact_email" :placeholder="t('tenantSettingsPage.form.contactEmailPlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.contactPhone')" prop="contact_phone">
                  <el-input v-model="form.contact_phone" :placeholder="t('tenantSettingsPage.form.contactPhonePlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.address')" prop="address">
                  <el-input v-model="form.address" :placeholder="t('tenantSettingsPage.form.addressPlaceholder')" />
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane :label="t('tenantSettingsPage.tab.branding')" name="branding">
                <el-form-item label="Logo" prop="logo">
                  <el-input v-model="form.logo" placeholder="Logo URL" />
                  <div v-if="form.logo" class="preview">
                    <img :src="form.logo" alt="Logo" class="preview-logo" />
                  </div>
                </el-form-item>
                <el-form-item label="Favicon" prop="favicon">
                  <el-input v-model="form.favicon" placeholder="Favicon URL" />
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.primaryColor')" prop="primary_color">
                  <div class="color-picker-wrapper">
                    <el-color-picker v-model="form.primary_color" />
                    <el-input
                      v-model="form.primary_color"
                      placeholder="#667eea"
                      style="width: 120px; margin-left: 12px;"
                    />
                  </div>
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.theme')" prop="theme">
                  <el-select v-model="form.theme" :placeholder="t('tenantSettingsPage.form.themePlaceholder')" style="width: 200px;">
                    <el-option :label="t('tenantSettingsPage.theme.light')" value="light" />
                    <el-option :label="t('tenantSettingsPage.theme.dark')" value="dark" />
                    <el-option :label="t('tenantSettingsPage.theme.auto')" value="auto" />
                  </el-select>
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane :label="t('tenantSettingsPage.tab.localization')" name="localization">
                <el-form-item :label="t('tenantSettingsPage.form.language')" prop="language">
                  <el-select v-model="form.language" :placeholder="t('tenantSettingsPage.form.languagePlaceholder')" style="width: 200px;">
                    <el-option :label="t('tenantSettingsPage.lang.zh')" value="zh-CN" />
                    <el-option :label="t('tenantSettingsPage.lang.en')" value="en-US" />
                    <el-option :label="t('tenantSettingsPage.lang.ja')" value="ja-JP" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('tenantSettingsPage.form.timezone')" prop="timezone">
                  <el-select v-model="form.timezone" :placeholder="t('tenantSettingsPage.form.timezonePlaceholder')" style="width: 200px;">
                    <el-option :label="t('tenantSettingsPage.timezone.shanghai')" value="Asia/Shanghai" />
                    <el-option :label="t('tenantSettingsPage.timezone.london')" value="Europe/London" />
                    <el-option :label="t('tenantSettingsPage.timezone.newyork')" value="America/New_York" />
                    <el-option :label="t('tenantSettingsPage.timezone.losangeles')" value="America/Los_Angeles" />
                    <el-option :label="t('tenantSettingsPage.timezone.tokyo')" value="Asia/Tokyo" />
                  </el-select>
                </el-form-item>
              </el-tab-pane>

              <el-tab-pane :label="t('tenantSettingsPage.tab.extra')" name="extra">
                <el-form-item :label="t('tenantSettingsPage.form.extra')" prop="extra">
                  <el-input
                    v-model="form.extra"
                    type="textarea"
                    :rows="8"
                    placeholder='{"key": "value"}'
                  />
                  <div class="extra-tip">{{ t('tenantSettingsPage.extraTip') }}</div>
                </el-form-item>
              </el-tab-pane>
            </el-tabs>

            <div class="form-actions">
              <el-button type="primary" :loading="saving" @click="handleSave">
                {{ t('tenantSettingsPage.action.save') }}
              </el-button>
              <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
            </div>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never">
          <template #header>
            <span class="title">{{ t('tenantSettingsPage.tab.preview') }}</span>
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
              <span v-else class="preview-header-title">{{ form.welcome_text || t('tenantSettingsPage.preview.defaultHeader') }}</span>
            </div>
            <div class="preview-body">
              <div class="preview-welcome">
                <h3>{{ form.welcome_text || t('tenantSettingsPage.preview.defaultWelcome') }}</h3>
                <p>{{ form.description || t('tenantSettingsPage.preview.defaultDesc') }}</p>
              </div>
              <div class="preview-info">
                <div v-if="form.contact_name" class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.contact') }}</span>
                  <span class="info-value">{{ form.contact_name }}</span>
                </div>
                <div v-if="form.contact_email" class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.email') }}</span>
                  <span class="info-value">{{ form.contact_email }}</span>
                </div>
                <div v-if="form.contact_phone" class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.phone') }}</span>
                  <span class="info-value">{{ form.contact_phone }}</span>
                </div>
                <div v-if="form.address" class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.address') }}</span>
                  <span class="info-value">{{ form.address }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.language') }}</span>
                  <span class="info-value">{{ getLanguageLabel(form.language) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('tenantSettingsPage.preview.timezone') }}</span>
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
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus/es/components/message/index'
import { getTenantSettings, updateTenantSettings } from '@/api/modules/tenant'
import type { TenantSettings, UpdateTenantSettingsReq } from '@/api/modules/tenant'
import { useBrand } from '@/composables/useBrand'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const formRef = ref<FormInstance>()
const loading = ref(false)
const saving = ref(false)
const activeTab = ref('basic')

const form = reactive<TenantSettings>({
  id: '',
  tenant_id: '',
  tenant_name: '',
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

const formRules = computed<FormRules>(() => ({
  contact_email: [
    { type: 'email', message: t('tenantSettingsPage.rules.emailInvalid'), trigger: 'blur' }
  ],
  extra: [
    {
      validator: (_rule: unknown, value: string, callback: (error?: string | Error) => void) => {
        if (value && value.trim()) {
          try {
            JSON.parse(value)
          } catch {
            callback(new Error(t('tenantSettingsPage.rules.jsonInvalid')))
            return
          }
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}))

onMounted(() => {
  loadSettings()
})

async function loadSettings() {
  loading.value = true
  try {
    const res = await getTenantSettings()
    Object.assign(form, res)
  } catch (error) {
    console.error('failed to load tenant settings', error)
    ElMessage.warning(t('tenantSettingsPage.msg.loadFailed'))
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
      // 保存成功后即时将品牌配置应用到全局 UI（多租户白标）
      useBrand().applyServerBrand(res)
      ElMessage.success(t('tenantSettingsPage.msg.saveSuccess'))
    } catch (error) {
      ElMessage.error((error as { response?: { data?: { message?: string } } })?.response?.data?.message || t('tenantSettingsPage.msg.saveFailed'))
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
    'zh-CN': t('tenantSettingsPage.lang.zh'),
    'en-US': t('tenantSettingsPage.lang.en'),
    'ja-JP': t('tenantSettingsPage.lang.ja')
  }
  return map[lang] || lang
}

function getTimezoneLabel(tz: string): string {
  const map: Record<string, string> = {
    'Asia/Shanghai': t('tenantSettingsPage.timezone.shanghai'),
    'Europe/London': t('tenantSettingsPage.timezone.london'),
    'America/New_York': t('tenantSettingsPage.timezone.newyork'),
    'America/Los_Angeles': t('tenantSettingsPage.timezone.losangeles'),
    'Asia/Tokyo': t('tenantSettingsPage.timezone.tokyo')
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