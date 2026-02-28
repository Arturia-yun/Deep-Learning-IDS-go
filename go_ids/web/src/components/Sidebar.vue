<template>
  <div 
    class="sidebar glass-panel" 
    :class="{ 'is-collapsed': isCollapse }"
    @mouseenter="isCollapse = false"
    @mouseleave="isCollapse = true"
  >
    <div class="logo-container">
        <div class="logo-icon">
            <el-icon :size="20"><Lock /></el-icon>
        </div>
        <span class="logo-text">Go-IDS</span>
    </div>
    
    <nav class="glass-menu">
      <router-link to="/" class="menu-item" :class="{ 'is-active': activePath === '/' }">
        <el-icon><Odometer /></el-icon>
        <span>仪表盘</span>
      </router-link>
      <router-link to="/threats" class="menu-item" :class="{ 'is-active': activePath === '/threats' }">
        <el-icon><Warning /></el-icon>
        <span>威胁审计</span>
      </router-link>
      <router-link to="/asset" class="menu-item" :class="{ 'is-active': activePath === '/asset' }">
        <el-icon><House /></el-icon>
        <span>资产画像</span>
      </router-link>
      <router-link to="/engine" class="menu-item" :class="{ 'is-active': activePath === '/engine' }">
        <el-icon><Cpu /></el-icon>
        <span>AI 引擎</span>
      </router-link>
      <router-link to="/settings" class="menu-item" :class="{ 'is-active': activePath === '/settings' }">
        <el-icon><Setting /></el-icon>
        <span>系统设置</span>
      </router-link>
    </nav>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Lock, Odometer, Warning, Connection, Setting, Cpu, House } from '@element-plus/icons-vue'

const route = useRoute()
const activePath = computed(() => route.path)
const isCollapse = ref(true)
</script>

<style scoped>
.sidebar {
    height: 100vh;
    width: 240px;
    display: flex;
    flex-direction: column;
    background: var(--sidebar-bg);
    border-radius: 0;
    border-left: none;
    z-index: 100;
    
    /* Optimization for Reflow */
    position: absolute;
    top: 0;
    left: 0;
    transition: width 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
    will-change: width;
    
    overflow: hidden; 
    box-shadow: 4px 0 15px rgba(243, 137, 98, 0.2);
}

.sidebar.is-collapsed {
    width: 64px;
}

.logo-container {
    height: 60px;
    display: flex;
    align-items: center;
    padding: 0 16px; 
    gap: 20px; /* Increased to 20px to push text effectively out of 64px bounds */
    border-bottom: 1px solid rgba(255,255,255,0.05);
    min-width: 240px; /* Ensure inner content doesn't shrink */
    transition: padding 0.3s ease;
}

.logo-icon {
    width: 32px;
    height: 32px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    flex-shrink: 0;
}

.logo-text {
    font-size: 18px;
    font-weight: 600;
    letter-spacing: 0.5px;
    white-space: nowrap; 
    color: white;
}

.glass-menu {
    flex-grow: 1;
    padding-top: 20px;
    width: 240px; /* Force menu to be full width inside the overflow container */
    display: flex;
    flex-direction: column;
}

.menu-item {
    background: transparent;
    color: rgba(255, 255, 255, 0.8);
    margin: 4px 12px;
    border-radius: 8px;
    height: 50px;
    display: flex;
    align-items: center;
    white-space: nowrap; /* Critical for menu text */
    text-decoration: none;
    transition: background 0.2s, color 0.2s;
}

.menu-item:hover {
    background: rgba(255,255,255,0.05);
    color: #fff;
}

.menu-item.is-active {
    background: rgba(255, 255, 255, 0.2);
    color: #fff;
}

/* Ensure icons stay fixed */
.menu-item .el-icon {
    font-size: 18px;
    flex-shrink: 0; 
    margin-right: 15px;
    margin-left: 10px;
    transition: margin 0.3s;
}

/* Hide text in collapsed state */
.is-collapsed .menu-item span {
    opacity: 0;
    visibility: hidden;
    transition: opacity 0.1s;
}

.menu-item span {
    opacity: 1;
    transition: opacity 0.3s 0.1s; /* Delay fade in */
    white-space: nowrap;
}
</style>
