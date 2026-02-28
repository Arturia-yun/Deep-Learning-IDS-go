<script setup>
import Sidebar from './components/Sidebar.vue'
</script>

<template>
  <div class="app-layout">
    <Sidebar />
    <div class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>
  </div>
</template>

<style>
/* Reset body margin */
body { margin: 0; padding: 0; }
#app { width: 100vw; height: 100vh; overflow: hidden; }

.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  position: relative;
}

.main-content {
  flex-grow: 1;
  padding: 20px;
  overflow-y: auto;
  position: relative;
  /* Add hardware acceleration and static margin to isolate reflow */
  margin-left: 64px; /* default collapsed size */
  transition: margin-left 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  will-change: margin-left;
  width: calc(100vw - 64px);
}

.app-layout:has(.sidebar:not(.is-collapsed)) .main-content {
  margin-left: 240px;
  width: calc(100vw - 240px);
}

/* Page Transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
