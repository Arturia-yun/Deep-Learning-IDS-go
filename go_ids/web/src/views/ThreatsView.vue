<template>
  <div class="threats-container h-full flex flex-col gap-6 p-6">
    <!-- Header Section -->
    <div class="glass-panel p-6 flex justify-between items-center rounded-2xl">
      <div>
        <h1 class="text-3xl font-extrabold text-[#2c3e50] tracking-tight">威胁审计</h1>
        <p class="text-sm text-[#7f8c8d] mt-1">
          实时监控与历史攻击溯源，点击列表查看链路报文(Payload)。
        </p>
      </div>
      <div>
        <button class="btn btn-outline border-[var(--coral-glow)] text-[var(--coral-glow)] hover:bg-[var(--coral-glow)] hover:border-[var(--coral-glow)] hover:text-white" @click="fetchAlerts">
          <el-icon class="mr-2"><Refresh /></el-icon> 刷新数据
        </button>
      </div>
    </div>

    <!-- Main Content: Table -->
    <div class="glass-panel rounded-2xl flex-grow flex flex-col overflow-hidden relative">
      <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/50 backdrop-blur-sm">
         <span class="loading loading-spinner text-[var(--coral-glow)] loading-lg"></span>
      </div>

      <!-- Use container queries or static transform to prevent table from reflowing content width during transition -->
      <div class="absolute inset-0 overflow-x-auto overflow-y-auto px-1">
        <!-- We use transform scale and fixed width to avoid flex squeeze -->
        <table class="table table-zebra table-pin-rows static-fast-table">
          <thead>
            <tr class="bg-base-200 text-[#2c3e50]">
              <th>ID</th>
              <th>时间</th>
              <th>源 IP</th>
              <th>目的 IP</th>
              <th>攻击类别</th>
              <th>置信度</th>
              <th class="text-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="alert in alerts" :key="alert.id" class="hover cursor-pointer" @click="viewDetails(alert)">
              <td class="font-mono text-xs text-gray-500">#{{ alert.id }}</td>
              <td>{{ formatTime(alert.timestamp) }}</td>
              <td>
                <div class="badge badge-error gap-1 bg-red-100 text-red-700 border-red-200">
                  {{ alert.source_ip }}
                </div>
              </td>
              <td>
                <div class="badge badge-info gap-1 bg-blue-100 text-blue-700 border-blue-200">
                  {{ alert.dest_ip }}
                </div>
              </td>
              <td class="font-semibold text-[var(--danger-color)]">{{ alert.type }}</td>
              <td>
                <div class="radial-progress text-[var(--danger-color)] text-xs" 
                     :style="`--value:${Math.round(alert.confidence*100)}; --size:2.5rem; --thickness: 4px;`">
                  {{ Math.round(alert.confidence * 100) }}%
                </div>
              </td>
              <td class="text-center">
                <button class="btn btn-sm btn-ghost text-[var(--coral-glow)]" @click.stop="viewDetails(alert)">
                  判研分析
                </button>
              </td>
            </tr>
            <tr v-if="alerts.length === 0 && !loading">
              <td colspan="7" class="text-center py-12 text-gray-400">
                暂无捕获到的威胁记录。
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Details Modal / Drawer -->
    <dialog id="payload_modal" class="modal modal-bottom sm:modal-middle">
      <div class="modal-box w-11/12 max-w-5xl h-[80vh] flex flex-col p-0 overflow-hidden bg-white shadow-2xl rounded-2xl">
        <div class="bg-[var(--coral-glow)] text-white p-6 flex justify-between items-center">
          <div>
            <h3 class="font-bold text-2xl flex items-center gap-2">
              <el-icon><WarningFilled /></el-icon> 攻击判研分析
            </h3>
            <p class="text-white/80 text-sm mt-1">Alert ID: #{{ selectedAlert?.id }} | 发生于: {{ formatTime(selectedAlert?.timestamp) }}</p>
          </div>
          <form method="dialog">
            <button class="btn btn-sm btn-circle btn-ghost text-white hover:bg-white/20">✕</button>
          </form>
        </div>
        
        <div class="p-6 overflow-y-auto flex-grow flex flex-col gap-6 bg-gray-50">
           <!-- Metadata Grid -->
           <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div class="stat bg-white rounded-xl shadow-sm border border-gray-100 p-4">
                <div class="stat-title text-xs text-gray-500">源 IP (攻击者)</div>
                <div class="stat-value text-lg text-red-500 truncate">{{ selectedAlert?.source_ip }}</div>
              </div>
              <div class="stat bg-white rounded-xl shadow-sm border border-gray-100 p-4">
                <div class="stat-title text-xs text-gray-500">目的 IP (受害者)</div>
                <div class="stat-value text-lg text-blue-500 truncate">{{ selectedAlert?.dest_ip }}</div>
              </div>
              <div class="stat bg-white rounded-xl shadow-sm border border-gray-100 p-4">
                <div class="stat-title text-xs text-gray-500">攻击模式判定</div>
                <div class="stat-value text-lg font-bold text-gray-800">{{ selectedAlert?.type }}</div>
              </div>
              <div class="stat bg-white rounded-xl shadow-sm border border-gray-100 p-4">
                <div class="stat-title text-xs text-gray-500">模型置信率</div>
                <div class="stat-value text-lg text-green-600">{{ Math.round((selectedAlert?.confidence || 0) * 100) }}%</div>
              </div>
           </div>

           <!-- Payload Inspector -->
           <div class="flex-grow flex flex-col bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
             <div class="bg-gray-100 border-b border-gray-200 px-4 py-2 flex justify-between items-center">
                <span class="font-semibold text-gray-700 text-sm flex items-center gap-2">
                  <el-icon><Document /></el-icon> 原始负载截获 (Payload)
                </span>
                <div class="join">
                   <button class="btn btn-xs join-item" :class="fmtMode==='text'?'btn-active':''" @click="fmtMode='text'">Text</button>
                   <button class="btn btn-xs join-item" :class="fmtMode==='hex'?'btn-active':''" @click="fmtMode='hex'">Hex Dump</button>
                </div>
             </div>
             <div class="p-4 bg-[#1e1e1e] flex-grow overflow-y-auto font-mono text-sm">
                <!-- If no payload -->
                <div v-if="!selectedAlert?.payload" class="h-full flex flex-col items-center justify-center text-gray-500">
                   <el-icon class="text-4xl mb-2"><Discount /></el-icon>
                   <span>无应用层载荷或为空</span>
                </div>
                <!-- Payload Display -->
                <pre v-else class="text-[#d4d4d4] whitespace-pre-wrap word-break">{{ formatPayload(selectedAlert.payload) }}</pre>
             </div>
           </div>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { getHistory } from '../api'
import { Refresh, WarningFilled, Document, Discount } from '@element-plus/icons-vue'

const alerts = ref([])
const loading = ref(false)
const selectedAlert = ref(null)
const fmtMode = ref('text')

const fetchAlerts = async () => {
  loading.value = true
  try {
    const res = await getHistory(100)
    if (res.data) {
      alerts.value = res.data
    }
  } catch (error) {
    console.error('Failed to fetch alerts:', error)
  } finally {
    loading.value = false
  }
}

const viewDetails = (alert) => {
  selectedAlert.value = alert
  fmtMode.value = 'text'
  document.getElementById('payload_modal').showModal()
}

const formatTime = (ts) => {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleString()
}

// Hex Dump Formatter Helper
const toHexDump = (str) => {
  if (!str) return '';
  let dump = '';
  for (let i = 0; i < str.length; i += 16) {
    let chunk = str.slice(i, i + 16);
    let hex = '';
    let ascii = '';
    
    // Convert current 16 chars to Hex and ASCII equivalent
    for (let j = 0; j < 16; j++) {
      if (j < chunk.length) {
        const code = chunk.charCodeAt(j);
        hex += code.toString(16).padStart(2, '0') + ' ';
        // Only printable ASCII
        ascii += (code >= 32 && code <= 126) ? chunk[j] : '.';
      } else {
        hex += '   ';
        ascii += ' ';
      }
      if (j === 7) hex += ' '; // middle gap
    }
    
    // Address prefix
    const addr = i.toString(16).padStart(8, '0');
    dump += `${addr}  ${hex} |${ascii}|\n`;
  }
  return dump;
}

const formatPayload = (payload) => {
  if (fmtMode.value === 'hex') {
    return toHexDump(payload)
  }
  return payload
}

let evtSource = null

onMounted(() => {
  fetchAlerts()
  
  // 主动倾听系统引擎实时的安全风暴警报，并将其不刷新地插入视图前端
  evtSource = new EventSource('http://localhost:8080/api/events')
  evtSource.addEventListener('alert', (e) => {
    try {
        const newAlert = JSON.parse(e.data)
        alerts.value.unshift(newAlert)
        
        // 如果列表过长，控制内存
        if (alerts.value.length > 500) {
            alerts.value.pop()
        }
    } catch(err) {
        console.error("Parse SSE alert error", err)
    }
  })
})

onUnmounted(() => {
    if (evtSource) {
        evtSource.close()
    }
})
</script>

<style scoped>
/* Glass Panel style overrides */
.glass-panel {
  background: var(--bg-card);
  border: none;
  box-shadow: var(--card-shadow);
  border-radius: var(--card-radius);
  /* Make internal elements smooth */
  transition: all 0.3s ease;
}

/* Base customizer for table to avoid extreme zebra contrasting in light theme */
.table-zebra tbody tr:nth-child(even) {
  background-color: #fafbfc;
}
.table-zebra tbody tr:hover {
  background-color: #f1f3f5;
}

/* Smooth modal interactions */
.modal-box {
  animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
@keyframes slideIn {
  0% { transform: translateY(20px) scale(0.98); opacity: 0; }
  100% { transform: translateY(0) scale(1); opacity: 1; }
}

/* Custom scrollbar for pre block */
pre::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
pre::-webkit-scrollbar-track {
  background: rgba(255,255,255,0.05);
}
pre::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.2);
  border-radius: 4px;
}

/* Optimization for zero-reflow animation:
   Use a fixed wide width and animate with transform scaleX to avoid layout reflow triggers.
   100vw - 64px(sidebar collapsed) - 40px(padding)
*/
.static-fast-table {
  /* State 1: Collapsed Sidebar (Default) -> the "widest" natural state */
  width: calc(100vw - 64px - 40px);
  transform-origin: left top;
  transition: transform 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  will-change: transform;
  table-layout: fixed; /* prevent row text from resizing */
}

/* We listen to the .app-layout parent dynamically if menu expands */
:root:has(.sidebar:not(.is-collapsed)) .static-fast-table,
.app-layout:has(.sidebar:not(.is-collapsed)) .static-fast-table {
  /* State 2: Expanded Sidebar -> Instead of changing width, we scale it down 
     Scale Factor = WidthWhenExpanded / WidthWhenCollapsed
     WidthWhenExpanded = 100vw - 240px - 40px
     WidthWhenCollapsed = 100vw - 64px - 40px
  */
  transform: scaleX(calc((100vw - 280px) / (100vw - 104px)));
}
</style>
