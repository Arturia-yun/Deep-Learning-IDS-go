<template>
  <div class="view-container glass-panel">
    <h1>系统设置 (Settings)</h1>
    <p>此处将管理白名单、自动封禁和通知配置。</p>
    <el-form label-position="top">
        <el-form-item label="自动封禁">
            <el-switch active-text="开启" inactive-text="关闭" />
        </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.view-container {
    padding: 40px;
    height: 100%;
}
</style>
