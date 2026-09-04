<template>
  <div class="dashboard">
    <!-- ======== 平台超级管理员：运营数据看板 ======== -->
    <template v-if="isAdmin">
      <el-row :gutter="16">
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon user-icon">
                <el-icon><OfficeBuilding /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">租户总数</div>
                <div class="stat-value">{{ overview.tenant_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon role-icon">
                <el-icon><UserFilled /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">用户总数</div>
                <div class="stat-value">{{ overview.user_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon perm-icon">
                <el-icon><Goods /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">生效订阅</div>
                <div class="stat-value">{{ overview.subscription_stats?.active ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon info-icon">
                <el-icon><Document /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">操作日志</div>
                <div class="stat-value">{{ overview.audit_stats?.total ?? 0 }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第二行：今日 / 本周 / 本月新增 -->
      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><TrendCharts /></el-icon> 新增概览</span></template>
            <el-table :data="growthRows" size="small" border>
              <el-table-column prop="label" label="指标" width="140" />
              <el-table-column prop="today" label="今日" />
              <el-table-column prop="week" label="本周" />
              <el-table-column prop="month" label="本月" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Coin /></el-icon> 订阅分布</span></template>
            <div class="sub-stats">
              <div class="sub-item">
                <div class="sub-num">{{ overview.subscription_stats?.total ?? 0 }}</div>
                <div class="sub-label">订阅总数</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#10b981">{{ overview.subscription_stats?.active ?? 0 }}</div>
                <div class="sub-label">生效中</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#f59e0b">{{ overview.subscription_stats?.expired ?? 0 }}</div>
                <div class="sub-label">已到期</div>
              </div>
              <div class="sub-item">
                <div class="sub-num" style="color:#ef4444">{{ overview.subscription_stats?.cancelled ?? 0 }}</div>
                <div class="sub-label">已取消</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 第三行：审计操作分布 -->
      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><Odometer /></el-icon> 操作类型统计</span></template>
            <el-empty v-if="!actionRows.length" description="暂无数据" :image-size="60" />
            <el-table v-else :data="actionRows" size="small" border>
              <el-table-column prop="key" label="操作类型" />
              <el-table-column prop="value" label="次数" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="section-card">
            <template #header><span><el-icon><DataAnalysis /></el-icon> 模块访问统计</span></template>
            <el-empty v-if="!moduleRows.length" description="暂无数据" :image-size="60" />
            <el-table v-else :data="moduleRows" size="small" border>
              <el-table-column prop="key" label="模块" />
              <el-table-column prop="value" label="访问次数" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- ======== 普通租户管理员：工作台 ======== -->
    <template v-else>
      <el-row :gutter="16">
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon user-icon">
                <el-icon><User /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">我的租户用户</div>
                <div class="stat-value">{{ stats.users }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon role-icon">
                <el-icon><UserFilled /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">角色总数</div>
                <div class="stat-value">{{ stats.roles }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon perm-icon">
                <el-icon><Lock /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">我的权限</div>
                <div class="stat-value">{{ stats.my_permission }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon info-icon">
                <el-icon><InfoFilled /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">当前租户</div>
                <div class="stat-value">{{ userStore.userInfo?.tenant_id || '-' }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <el-card class="welcome-card" shadow="never">
      <template #header>
        <span><el-icon><TrendCharts /></el-icon>{{ isAdmin ? '平台概览' : '工作台' }}</span>
      </template>
      <div class="welcome">
        <h3>欢迎，{{ userStore.userInfo?.nickname || userStore.userInfo?.username || '用户' }}！</h3>
        <p v-if="isAdmin">这是一个基于 Vue 3 + Element Plus + Go 后端（Chi + GORM）的多租户 SaaS 平台。</p>
        <p v-else>您已登录租户管理后台，可以管理本租户下的用户和角色。</p>
        <el-divider />
        <div class="feature-list">
          <template v-if="isAdmin">
            <div>✅ 数据看板：平台运营数据总览</div>
            <div>✅ 租户管理：新增、禁用/启用、注销审批</div>
            <div>✅ 用户管理：系统管理员管理、用户统计</div>
            <div>✅ 通知公告：面向全平台与租户发布公告</div>
            <div>✅ 套餐管理：订阅与配额管理</div>
            <div>✅ 审计日志：全平台操作留痕</div>
          </template>
          <template v-else>
            <div>👥 用户管理：管理本租户下的用户</div>
            <div>🔐 角色权限：查看和分配用户角色</div>
            <div>📊 数据隔离：仅显示本租户数据</div>
            <div>🔑 安全登录：支持租户 ID + 账号密码登录</div>
          </template>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  OfficeBuilding,
  UserFilled,
  Goods,
  Document,
  TrendCharts,
  Coin,
  Odometer,
  DataAnalysis,
  User,
  Lock,
  InfoFilled
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useUserStore } from '@/stores/user'
import { getUserStats } from '@/api/modules/user'
import { getRBACStats } from '@/api/modules/role'
import { getDashboardOverview, type DashboardOverview } from '@/api/modules/dashboard'

const userStore = useUserStore()
const stats = ref({ users: 0, roles: 0, permissions: 0, my_permission: 0 })
const overview = ref<DashboardOverview>({} as DashboardOverview)

const isAdmin = computed(() => !!userStore.userInfo?.is_master)

// 新增概览行数据
const growthRows = computed(() => [
  { label: '新增租户', today: overview.value.tenant_stats?.today_new ?? 0, week: overview.value.tenant_stats?.week_new ?? 0, month: overview.value.tenant_stats?.month_new ?? 0 },
  { label: '新增用户', today: overview.value.user_stats?.today_new ?? 0, week: overview.value.user_stats?.week_new ?? 0, month: overview.value.user_stats?.month_new ?? 0 }
])

// 操作类型 / 模块统计行
const actionRows = computed(() => Object.entries(overview.value.audit_stats?.action_stats ?? {}).map(([key, value]) => ({ key, value })))
const moduleRows = computed(() => Object.entries(overview.value.audit_stats?.module_stats ?? {}).map(([key, value]) => ({ key, value })))

onMounted(async () => {
  try {
    if (isAdmin.value) {
      // 平台超级管理员：加载运营数据看板
      try {
        overview.value = await getDashboardOverview()
      } catch {
        ElMessage.warning('运营数据看板加载失败')
      }
      // 兼容旧统计接口，作为兜底
      const rbacStats = await getRBACStats()
      if (rbacStats) {
        stats.value.roles = rbacStats.role_count ?? 0
        stats.value.permissions = rbacStats.permission_count ?? 0
        stats.value.my_permission = rbacStats.my_permission ?? 0
      }
    } else {
      const userStats = await getUserStats()
      stats.value.users = userStats.user_count ?? 0
      stats.value.roles = 0
      stats.value.permissions = 0
      stats.value.my_permission = userStore.permissions.length
    }
  } catch (e) {
    ElMessage.warning('部分统计数据加载失败')
  }
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.stat-card {
  height: 120px;
}
.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 100%;
}
.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
}
.user-icon { background: #3b82f6; }
.role-icon { background: #10b981; }
.perm-icon { background: #f59e0b; }
.info-icon { background: #8b5cf6; }
.stat-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 26px;
  font-weight: 600;
  color: #111827;
}
.section-card {
  min-height: 180px;
}
.sub-stats {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.sub-item {
  flex: 1;
  text-align: center;
  padding: 12px 0;
  background: #f8fafc;
  border-radius: 8px;
}
.sub-num {
  font-size: 22px;
  font-weight: 600;
  color: #111827;
}
.sub-label {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
.welcome-card {
  margin-top: 0;
}
.welcome h3 {
  margin: 0 0 6px;
  color: #111827;
}
.welcome p {
  margin: 0;
  color: #4b5563;
  font-size: 14px;
}
.feature-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 24px;
  color: #374151;
  font-size: 14px;
}
</style>
