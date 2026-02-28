<template>
  <div class="engine-container">
    <header class="glass-panel header">
      <div class="logo-area">
        <h1>AI 引擎调优 (Engine Tuning)</h1>
      </div>
    </header>

    <div class="content-grid">
      <!-- Left Column: Engine Status -->
      <div class="left-col">
        <div class="glass-panel engine-status-card">
          <div class="section-header">
            <h3>核心挂载状态</h3>
            <div class="status-badge" :class="engineStatus ? 'active' : 'offline'">
              <span class="dot"></span>
              {{ engineStatus ? 'ONNX Runtime Running' : 'Engine Offline' }}
            </div>
          </div>
          
          <div class="info-list">
            <div class="info-item">
              <span class="label">当前模型</span>
              <span class="value txt-code">{{ config.model_path || '加载中...' }}</span>
            </div>
            <div class="info-item">
              <span class="label">特征缩放器</span>
              <span class="value txt-code">{{ config.scaler_path || '加载中...' }}</span>
            </div>
            <div class="info-item">
              <span class="label">框架类型</span>
              <span class="value">Deep Learning (Neural Network)</span>
            </div>
            <div class="info-item">
              <span class="label">硬件加速</span>
              <span class="value text-success">CPU/GPU Auto Detection</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right Column: Tuning Controls -->
      <div class="right-col">
        <div class="glass-panel tuning-card">
          <div class="section-header">
            <h3>置信度拦截阈值 (Confidence Threshold)</h3>
          </div>
          
          <div class="slider-container">
            <div class="slider-header">
              <span class="desc-text">低于该概率的预测仅记录审计日志，不触发封禁响应。</span>
              <span class="current-value highlight">{{ displayThreshold }}%</span>
            </div>
            
            <input 
              type="range" 
              min="50" 
              max="99" 
              v-model="editThreshold" 
              class="range range-primary custom-range" 
              step="1" 
              @change="applyConfig"
            />
            
            <div class="slider-marks">
              <span>50% (激进, 易误报)</span>
              <span>75%</span>
              <span>99% (保守, 仅处理确凿攻击)</span>
            </div>
          </div>

          <div class="action-bar">
             <button class="btn btn-primary glass-btn" @click="applyConfig" :disabled="saving">
                 <el-icon v-if="saving" class="is-loading"><Loading /></el-icon>
                 {{ saving ? '应用中...' : '提交新配置' }}
             </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'

const engineStatus = ref(false)
const config = ref({})
const editThreshold = ref(80) // default 80% (0.8)
const saving = ref(false)

const displayThreshold = computed(() => {
    return editThreshold.value
})

const fetchStatus = async () => {
    try {
        const res = await axios.get('http://localhost:8080/api/engine/status')
        config.value = res.data
        editThreshold.value = Math.round(res.data.current_threshold * 100)
        engineStatus.value = true
    } catch (e) {
        console.error("Failed to fetch engine status", e)
        engineStatus.value = false
    }
}

const applyConfig = async () => {
    saving.value = true
    try {
        const payload = {
            threshold: editThreshold.value / 100.0
        }
        await axios.post('http://localhost:8080/api/engine/config', payload)
        
        ElMessage({
            message: '引擎阈值更新并持久化成功！已实时生效。',
            type: 'success',
            duration: 3000
        })
        
        // Refresh
        await fetchStatus()
    } catch (e) {
         ElMessage({
            message: '更新失败: ' + (e.response?.data?.error || e.message),
            type: 'error'
        })
    } finally {
        saving.value = false
    }
}

onMounted(() => {
    fetchStatus()
})
</script>

<style scoped>
.engine-container {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 20px;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 30px;
}

.logo-area h1 { margin: 0; font-size: 20px; font-weight: 500; letter-spacing: 0.5px; }

.content-grid {
    display: flex;
    gap: 20px;
    /* responsive */
    flex-wrap: wrap; 
}

.left-col {
    flex: 1;
    min-width: 300px;
}

.right-col {
    flex: 1.5;
    min-width: 400px;
}

.engine-status-card, .tuning-card {
    padding: 30px;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 25px;
    border-bottom: 1px solid rgba(0,0,0,0.05);
    padding-bottom: 15px;
}

.section-header h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
    color: #2c3e50;
}

/* Status Badge */
.status-badge {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    padding: 4px 12px;
    border-radius: 20px;
    background: rgba(0,0,0,0.05);
    color: #666;
    font-weight: 500;
}

.status-badge.active {
    background: rgba(46, 204, 113, 0.1);
    color: #27ae60;
}

.dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ccc;
}
.active .dot { background: #2ecc71; box-shadow: 0 0 8px #2ecc71; }

/* Info List */
.info-list {
    display: flex;
    flex-direction: column;
    gap: 15px;
}

.info-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 15px;
    background: rgba(255,255,255,0.4);
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.5);
}

.info-item .label {
    font-size: 14px;
    color: #7f8c8d;
}

.info-item .value {
    font-size: 14px;
    font-weight: 500;
    color: #34495e;
}

.txt-code {
    font-family: 'Consolas', monospace;
    font-size: 13px !important;
    background: rgba(0,0,0,0.03);
    padding: 2px 6px;
    border-radius: 4px;
}

.text-success { color: #27ae60 !important; }

/* Tuning Card */
.slider-container {
    margin-top: 10px;
    background: rgba(255,255,255,0.3);
    padding: 25px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.6);
}

.slider-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-bottom: 20px;
}

.desc-text {
    font-size: 13px;
    color: #7f8c8d;
    max-width: 70%;
    line-height: 1.5;
}

.current-value.highlight {
    font-size: 32px;
    font-weight: 700;
    color: #e67e22; /* Warning orange color */
    text-shadow: 0 2px 4px rgba(230, 126, 34, 0.2);
}

.custom-range {
    --range-shdw: #e67e22;
}
.custom-range::-webkit-slider-thumb {
     background-color: #e67e22;
}

.slider-marks {
    display: flex;
    justify-content: space-between;
    margin-top: 12px;
    font-size: 12px;
    color: #bdc3c7;
}

.action-bar {
    margin-top: auto;
    display: flex;
    justify-content: flex-end;
    padding-top: 30px;
}

.glass-btn {
    background: linear-gradient(135deg, #e67e22, #d35400);
    border: none;
    color: white;
    padding: 0 30px;
    height: 44px;
    border-radius: 22px;
    font-weight: 600;
    box-shadow: 0 4px 15px rgba(230, 126, 34, 0.3);
    transition: all 0.3s;
    display: flex;
    align-items: center;
    gap: 8px;
}

.glass-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(230, 126, 34, 0.4);
}
.glass-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
    transform: none;
}
</style>
