<template>
  <div class="dashboard">
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon user-icon">
              <el-icon><User /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">用户总数</div>
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
              <div class="stat-label">权限总数</div>
              <div class="stat-value">{{ stats.permissions }}</div>
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
              <div class="stat-label">我的权限</div>
              <div class="stat-value">{{ stats.my_permission }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="welcome-card" shadow="never">
      <template #header>
        <span><el-icon><TrendCharts /></el-icon> 系统信息</span>
      </template>
      <div class="welcome">
        <h3>欢迎，{{ userStore.userInfo?.nickname || userStore.userInfo?.username || '用户' }}！</h3>
        <p>这是一个基于 Vue 3 + Element Plus + Go 后端（Chi + GORM）的多租户 RBAC 管理系统。</p>
        <el-divider />
        <div class="feature-list">
          <div>✅ 用户管理：新增、编辑、删除、启用/禁用</div>
          <div>✅ 角色管理：角色绑定权限、角色绑定用户</div>
          <div>✅ 权限管理：基于 code 的权限码，自动注册到数据库</div>
          <div>✅ 权限控制：路由级 + 按钮级双重控制</div>
          <div>✅ 多租户：租户隔离 + 系统管理员（is_master）</div>
        </div>
        <el-divider />
        <div class="next-steps">
          <div style="font-weight: 600; margin-bottom: 8px">接下来你可以：</div>
          <el-tag type="info" style="margin: 4px">在"用户管理"中新增一个用户</el-tag>
          <el-tag type="info" style="margin: 4px">在"角色管理"中创建角色并绑定权限</el-tag>
          <el-tag type="info" style="margin: 4px">在"权限管理"中查看所有系统权限</el-tag>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { getUserStats } from '@/api/modules/user'
import { getRBACStats } from '@/api/modules/role'

const userStore = useUserStore()
const stats = ref({ users: 0, roles: 0, permissions: 0, my_permission: 0 })

onMounted(async () => {
  try {
    // 获取 RBAC 统计信息
    const rbacStats = await getRBACStats()
    // 获取用户总数
    const userStats = await getUserStats()
    
    stats.value = {
      users: userStats.data.user_count ?? 0,
      roles: rbacStats.data.role_count ?? 0,
      permissions: rbacStats.data.permission_count ?? 0,
      my_permission: rbacStats.data.my_permission ?? 0
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
.next-steps {
  color: #374151;
  font-size: 14px;
}
</style>