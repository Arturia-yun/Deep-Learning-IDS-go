<template>
  <div class="dashboard-container">
    <!-- Header -->
    <header class="glass-panel header">
      <div class="logo-area">
        <h1>概览 (Dashboard)</h1>
      </div>
      <div class="status-area">
        <span class="status-dot" :class="{ 'animate-pulse': systemStatus === 'running', 'status-offline': systemStatus !== 'running' }"></span>
        <span class="status-text">{{ systemStatus === 'running' ? '系统运行中' : 'IDS引擎未启动' }}</span>
        <span class="uptime" v-if="systemStatus === 'running'">运行时长: {{ uptime }}</span>
      </div>
    </header>

    <!-- Top Cards -->
    <div class="metrics-row">
      <!-- Card 1: Coral Glow -->
      <div class="glass-panel metric-card bg-coral">
        <h3>实时流量 Monitor</h3>
        <div class="metric-value" style="font-size: 24px;">
           <div>↓ {{ trafficIn }} <small>Mbps</small></div>
           <div>↑ {{ trafficOut }} <small>Mbps</small></div>
        </div>
      </div>
      <!-- Card 2: Sandy Brown -->
      <!-- Card 2: Sandy Brown (Redesigned) -->
      <div class="glass-panel metric-card bg-sandy custom-flex-card">
        <!-- Left: Metric -->
        <div class="card-left">
            <h3>活跃连接</h3>
            <div class="metric-value-wrapper">
                <span class="metric-number">{{ activeFlows }}</span>
                <span v-if="flowDiffVisible" 
                      class="diff-float" 
                      :class="flowDiff > 0 ? 'text-success' : 'text-danger'">
                      {{ flowDiff > 0 ? '▲' : '▼' }} {{ Math.abs(flowDiff) }}
                </span>
            </div>
            <div class="metric-sub">实时会话监控</div>
        </div>
        
        <!-- Right: Modern Flow List -->
        <div class="card-right flow-list-modern">
            <div v-for="(flow, index) in flowList" :key="index" class="flow-row-modern">
                <div class="flow-info">
                    <span class="tag-proto" :class="flow.protocol.toLowerCase()">{{ flow.protocol }}</span>
                    <span class="txt-port">{{ flow.dst_port }}</span>
                </div>
                <span class="txt-time">{{ flow.duration }}</span>
            </div>
            <div v-if="flowList.length === 0" class="empty-state">Waiting...</div>
        </div>
      </div>
      <!-- Card 3: Jasmine (Threats) -->
      <div class="glass-panel metric-card bg-jasmine threat-card-flex">
        <!-- Left: Metric -->
        <div class="card-left">
            <h3>威胁检测</h3>
             <div class="metric-value-wrapper">
                <span class="metric-number text-dark">{{ totalThreats }}</span>
            </div>
            <div class="metric-sub text-dark">{{ threatTimeLabel[threatTimeRange] }} 总计</div>
        </div>
        
        <!-- Right: Sparkline Chart -->
        <div class="card-right threat-chart-container">
            <div class="threat-toggles">
                <span 
                    v-for="range in ['Day', 'Week', 'Month']" 
                    :key="range"
                    class="toggle-pill"
                    :class="{ active: threatTimeRange === range }"
                    @click="setThreatRange(range)"
                >{{ range === 'Day' ? '日' : (range === 'Week' ? '周' : '月') }}</span>
            </div>
            <div ref="threatSparkRef" class="threat-spark-chart"></div>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="charts-row">
      <!-- Main Trend Chart -->
      <div class="glass-panel main-chart-section">
        <div class="section-header">
          <h2>网络吞吐量与威胁活动</h2>
        </div>
        <div ref="mainChartRef" class="main-chart"></div>
      </div>

      <!-- Type Distribution Chart (New) -->
      <div class="glass-panel type-chart-section">
        <div class="section-header">
            <h2>威胁类型分布</h2>
            <el-radio-group v-model="chartType" size="small" @change="renderTypeChart">
                <el-radio-button label="pie">饼图</el-radio-button>
                <el-radio-button label="bar">柱状</el-radio-button>
            </el-radio-group>
        </div>
        <div ref="typeChartRef" class="type-chart"></div>
      </div>
    </div>

    <!-- Logs Section -->
    <div class="glass-panel logs-section">
      <div class="section-header">
        <h2>实时安全日志</h2>
        <div class="header-actions">
             <el-tag type="info" effect="dark" round>实时推送中</el-tag>
        </div>
      </div>
      <el-table :data="alerts" style="width: 100%" height="300" :row-class-name="tableRowClassName">
        <el-table-column prop="timestamp" label="时间" width="180">
            <template #default="scope">
                {{ formatTime(scope.row.timestamp) }}
            </template>
        </el-table-column>
        <el-table-column prop="source_ip" label="源 IP" width="150" />
        <el-table-column prop="dest_ip" label="目的 IP" width="150" />
        <el-table-column prop="type" label="攻击类型">
             <template #default="scope">
                <el-tag :type="getSeverityType(scope.row.type)" effect="dark">{{ scope.row.type }}</el-tag>
             </template>
        </el-table-column>
        <el-table-column prop="confidence" label="置信度">
            <template #default="scope">
                <el-progress 
                    :percentage="Math.floor(scope.row.confidence * 100)" 
                    :status="scope.row.confidence > 0.8 ? 'exception' : 'warning'"
                    :show-text="true"
                    :stroke-width="15" 
                />
            </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, reactive } from 'vue'
import * as echarts from 'echarts'
import { getHistory, getStatus } from '../api'

// State
const alerts = ref([])
const totalThreats = ref(0)
const trafficIn = ref(0)
const trafficOut = ref(0)
const activeFlows = ref(0)
const flowList = ref([])
const uptime = ref('00:00:00')
const systemStatus = ref('offline') 
const chartType = ref('pie')
const typeStats = reactive({})

// Charts Refs
const mainChartRef = ref(null)
const typeChartRef = ref(null)
let mainChart = null
let typeChart = null

const threatTimeRange = ref('Day')
const threatTimeLabel = { 'Day': '过去 24 小时', 'Week': '过去 7 天', 'Month': '过去 30 天' }
const threatSparkRef = ref(null)
let threatSparkChart = null

const trafficInData = []
const trafficOutData = []
const threatData = []

// Diff Animation State
const flowDiff = ref(0)
const flowDiffVisible = ref(false)
let diffTimeout = null

const triggerFlowDiff = (diff) => {
    flowDiff.value = diff
    flowDiffVisible.value = false
    
    // Reset animation
    setTimeout(() => {
        flowDiffVisible.value = true
        // Auto hide after animation (2.5s)
        if (diffTimeout) clearTimeout(diffTimeout)
        diffTimeout = setTimeout(() => {
            flowDiffVisible.value = false
        }, 2500)
    }, 10)
}

// Formatting
const formatTime = (isoString) => {
    return new Date(isoString).toLocaleString('zh-CN', { hour12: false })
}

const getSeverityType = (type) => {
    if (['DDoS', 'Malware'].includes(type)) return 'danger'
    if (['PortScan'].includes(type)) return 'warning'
    return 'info'
}

const tableRowClassName = ({ row }) => {
  if (row.confidence > 0.9) return 'danger-row'
  return ''
}

// Chart Logic
const initCharts = () => {
    // 1. Trend Chart
    if (!mainChartRef.value) return
    mainChart = echarts.init(mainChartRef.value)
    
    // Init Mock Trend Data (Zero fill)
    const now = new Date()
    for (let i = 0; i < 60; i++) {
        const t = new Date(now - (60 - i) * 1000)
        trafficInData.push({ name: t.toString(), value: [t, 0] })
        trafficOutData.push({ name: t.toString(), value: [t, 0] })
        threatData.push({ name: t.toString(), value: [t, 0] })
    }

    const trendOptions = {
        grid: { top: 30, right: 20, bottom: 20, left: 50, containLabel: true },
        tooltip: { trigger: 'axis' },
        legend: { data: ['下载流量', '上传流量', '威胁评分'], icon: 'circle', top: 0, right: 20, textStyle: { color: '#666' } },
        xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#666' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'rgba(0,0,0,0.05)' } }, axisLabel: { color: '#666' } },
        series: [
            {
                name: '下载流量', // Download (Green)
                type: 'line',
                smooth: true,
                showSymbol: false,
                data: trafficInData,
                itemStyle: { color: '#2ecc71' }, 
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(46, 204, 113, 0.4)' },
                        { offset: 1, color: 'rgba(46, 204, 113, 0)' }
                    ])
                }
            },
            {
                name: '上传流量', // Upload (Blue)
                type: 'line',
                smooth: true,
                showSymbol: false,
                data: trafficOutData,
                itemStyle: { color: '#3498db' },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(52, 152, 219, 0.4)' },
                        { offset: 1, color: 'rgba(52, 152, 219, 0)' }
                    ])
                }
            },
            {
                name: '威胁评分',
                type: 'bar',
                data: threatData,
                itemStyle: { color: '#ff453a' },
                barWidth: 5
            }
        ]
    }
    mainChart.setOption(trendOptions)
    
    // 2. Type Distribution Chart
    if (!typeChartRef.value) return
    typeChart = echarts.init(typeChartRef.value)
    renderTypeChart()
    
    // window resize handled by ResizeObserver in onMounted
}

// 3. Threat Spark Chart Logic
const renderThreatSpark = async () => {
    if (!threatSparkRef.value) return
    if (!threatSparkChart) threatSparkChart = echarts.init(threatSparkRef.value)
    
    const range = threatTimeRange.value
    let data = []
    let axis = []
    
    try {
        // Fetch Real Data using axios (assumed available as per getStatus)
        // If exact helper isn't available, using basic fetch or relative path
        const response = await fetch(`http://localhost:8080/api/stats/threats?range=${range}`)
        const points = await response.json()
        
        if (points && points.length > 0) {
           axis = points.map(p => p.label)
           data = points.map(p => p.count)
           // 正确计算指定时域内的发生总数，而不是取被分页限制的表格长度
           totalThreats.value = data.reduce((a, b) => a + b, 0)
        } else {
             // Fallback minimal data if empty
            data = [0,0,0,0,0,0]
            axis = ['','','','','','']
            totalThreats.value = 0
        }
    } catch (e) {
        console.error("Failed to fetch threat stats", e)
        // Keep mock or empty on error
        data = [0,0,0,0,0]
        axis = ['','','','','']
    }

    const option = {
        grid: { top: 5, right: 5, bottom: 5, left: 5 },
        tooltip: { 
            trigger: 'axis',
            confine: true, // 防止 tooltip 被外部容器剪切或覆盖
            backgroundColor: 'rgba(255, 255, 255, 0.85)',
            borderColor: 'rgba(211, 84, 0, 0.4)',
            borderWidth: 1,
            padding: [8, 12],
            textStyle: {
                color: '#2c3e50',
                fontSize: 12,
                fontWeight: 500
            },
            extraCssText: 'box-shadow: 0 4px 12px rgba(211, 84, 0, 0.15); border-radius: 8px; backdrop-filter: blur(8px);',
            formatter: function (params) {
                const data = params[0];
                return `<div style="display:flex; flex-direction:column; gap:4px;">
                            <span style="font-size:10px; color:#7f8c8d;">时间: ${data.name}</span>
                            <div style="display:flex; align-items:center; gap:6px;">
                                <span style="display:inline-block; width:8px; height:8px; border-radius:50%; background-color:#d35400;"></span>
                                <span style="font-weight:700; color:#d35400; font-size:14px;">${data.value} <span style="font-size:10px; font-weight:normal; color:#95a5a6;">次拦截</span></span>
                            </div>
                        </div>`;
            }
        },
        xAxis: { type: 'category', data: axis, show: false },
        yAxis: { type: 'value', show: false },
        series: [{
            type: 'line',
            data: data,
            smooth: true,
            showSymbol: false,
            lineStyle: { width: 2, color: '#d35400' },
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(211, 84, 0, 0.4)' },
                    { offset: 1, color: 'rgba(211, 84, 0, 0)' }
                ])
            }
        }]
    }
    threatSparkChart.setOption(option)
}

const setThreatRange = (range) => {
    threatTimeRange.value = range
    renderThreatSpark()
}

const renderTypeChart = () => {
    if (!typeChart) return
    
    // Convert stats to Array
    const data = Object.keys(typeStats).map(k => ({ value: typeStats[k], name: k }))
    if (data.length === 0) {
        // Mock data for display if empty
        data.push({ value: 0, name: '暂无数据' })
    }

    let option = {}
    
    if (chartType.value === 'pie') {
        option = {
            tooltip: { trigger: 'item' },
            legend: { top: '5%', left: 'center', textStyle: { color: '#ccc' } },
            series: [
                {
                    name: '攻击类型',
                    type: 'pie',
                    radius: ['40%', '70%'],
                    itemStyle: {
                        borderRadius: 10,
                        borderColor: '#000',
                        borderWidth: 2
                    },
                    label: { show: false },
                    data: data
                }
            ]
        }
    } else {
        option = {
            tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
            grid: { top: 30, right: 30, bottom: 20, left: 30, containLabel: true },
            xAxis: { 
                type: 'category', 
                data: data.map(d => d.name),
                axisLabel: { color: '#aaa' }
            },
            yAxis: { 
                type: 'value',
                axisLabel: { color: '#aaa' },
                splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
            },
            series: [
                {
                    data: data.map(d => d.value),
                    type: 'bar',
                    itemStyle: { color: '#f38962', borderRadius: [5, 5, 0, 0] }
                }
            ]
        }
    }
    
    typeChart.setOption(option, true) // true = merge not allowed (reset)
}


let currentThreatCount = 0

const updateMainChart = async () => {
    if (!mainChart) return
    const now = new Date()
    
    let inMbps = 0
    let outMbps = 0

    // Fetch Real Traffic Logic
    if (systemStatus.value === 'running') {
        try {
            const res = await getStatus()
            systemStatus.value = 'running'
            
            // Backend returns traffic_in and traffic_out
            if (res.data.traffic_in !== undefined) {
                inMbps = parseFloat(res.data.traffic_in.toFixed(2))
                outMbps = parseFloat(res.data.traffic_out.toFixed(2))
            }
            // Active Flows
            if (res.data.active_flows !== undefined) {
                const newVal = res.data.active_flows
                const diff = newVal - activeFlows.value
                
                if (diff !== 0) {
                    triggerFlowDiff(diff)
                }
                activeFlows.value = newVal
            }
            // Flow List
            if (res.data.flow_list) {
                flowList.value = res.data.flow_list
            }
        } catch (e) {
             console.warn("Traffic Sync Failed, maybe offline")
        }
    }
    
    // Update State
    trafficIn.value = inMbps
    trafficOut.value = outMbps
    
    // Shift & Push
    trafficInData.shift()
    trafficInData.push({ name: now.toString(), value: [now, inMbps] })
    
    trafficOutData.shift()
    trafficOutData.push({ name: now.toString(), value: [now, outMbps] })
    
    // 提取本秒累积到的威胁数量，然后清零计数器，形成脉冲波柱状图
    const threatsThisSec = currentThreatCount
    currentThreatCount = 0
    
    threatData.shift()
    threatData.push({ name: now.toString(), value: [now, threatsThisSec] }) 
    
    mainChart.setOption({
        series: [
            { data: trafficInData }, 
            { data: trafficOutData }, 
            { data: threatData }
        ]
    })
}


// SSE & Logic
onMounted(async () => {
    let startTimestamp = Date.now()

    // 1. Initial Load & Sync Time
    try {
        const [histRes, statusRes] = await Promise.all([
            getHistory(),
            getStatus()
        ])
        
        // History
        alerts.value = histRes.data
        histRes.data.forEach(a => {
            typeStats[a.type] = (typeStats[a.type] || 0) + 1
        })
        
        // Status & Start Time
        if (statusRes.data.start_time) {
            startTimestamp = new Date(statusRes.data.start_time).getTime()
        }
        
        // API is reachable, set status to running immediately
        systemStatus.value = 'running'

    } catch (e) {
        console.error("Failed to load initial data", e)
    }

    // 2. Charts
    initCharts()
    renderTypeChart()
    renderThreatSpark()
    setInterval(updateMainChart, 1000)

    // 3. SSE
    const evtSource = new EventSource('http://localhost:8080/api/events')
    
    // Status Logic: Event-Driven (Zero Polling)
    evtSource.onopen = () => {
        systemStatus.value = 'running'
    }
    
    evtSource.onerror = () => {
        systemStatus.value = 'offline'
    }
    
    evtSource.addEventListener('alert', (e) => {
        const newAlert = JSON.parse(e.data)
        alerts.value.unshift(newAlert)
        if (alerts.value.length > 50) alerts.value.pop()
        
        totalThreats.value++
        
        // Update Stats
        typeStats[newAlert.type] = (typeStats[newAlert.type] || 0) + 1
        renderTypeChart()
        
        // 累加本秒度周期内的威胁触发次数，交给下一秒定时的 updateMainChart 归档显示
        currentThreatCount++
    })
    
    // Uptime Timer (Synced with Backend)
    setInterval(() => {
        const diff = Math.floor((Date.now() - startTimestamp) / 1000)
        const h = Math.floor(diff / 3600).toString().padStart(2, '0')
        const m = Math.floor((diff % 3600) / 60).toString().padStart(2, '0')
        const s = (diff % 60).toString().padStart(2, '0')
        uptime.value = `${h}:${m}:${s}`
    }, 1000)

    // Auto Resize Charts using ResizeObserver
    const resizeObserver = new ResizeObserver(() => {
        mainChart?.resize()
        typeChart?.resize()
        threatSparkChart?.resize()
    })
    if (mainChartRef.value) resizeObserver.observe(mainChartRef.value.parentElement)
    if (typeChartRef.value) resizeObserver.observe(typeChartRef.value.parentElement)
    if (threatSparkRef.value) resizeObserver.observe(threatSparkRef.value.parentElement)
})

onUnmounted(() => {
   if (mainChart) mainChart.dispose()
   if (typeChart) typeChart.dispose()
   if (threatSparkChart) threatSparkChart.dispose()
})

</script>

<style scoped>
.dashboard-container {
    width: 100%;
    /* Removed padding to let RouterView handle layout if needed, 
       but for now keeping internal padding */
    display: flex;
    flex-direction: column;
    gap: 20px;
}

/* Modern Side-by-Side Card Layout */
/* Modern Side-by-Side Card Layout */
.custom-flex-card {
    display: flex !important;
    flex-direction: row !important;
    align-items: center;
    justify-content: space-between;
    padding: 24px !important; /* Match other cards (24px) */
    gap: 15px;
    height: 160px; /* Consistently increased */
}

.card-left {
    flex: 0 0 45%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    height: 100%; /* Fill vertical space */
}

.metric-value-wrapper {
    position: relative;
    display: flex;
    align-items: baseline;
    margin: 2px 0; /* Tighter vertical spacing */
}

/* Right Side List - Better Glassmorphism */
.card-right.flow-list-modern {
    flex: 1;
    height: 100%; /* Fill available height */
    background: rgba(255, 255, 255, 0.1); /* More transparent */
    border-radius: 8px;
    padding: 6px;
    overflow-y: auto;
    backdrop-filter: blur(8px); /* Stronger blur */
    box-shadow: inset 0 1px 1px rgba(255,255,255,0.1);
    border: 1px solid rgba(255,255,255,0.15);
}

.flow-row-modern {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 8px;
    margin-bottom: 4px;
    background: rgba(255,255,255,0.15); /* Subtle item bg */
    border-radius: 4px;
    border-bottom: none;
    transition: all 0.2s;
    color: rgba(0,0,0,0.7);
}
.flow-row-modern:hover {
    background: rgba(255,255,255,0.4);
    transform: translateX(2px);
}

.flow-info {
    display: flex;
    align-items: center;
    gap: 6px;
}

.tag-proto {
    font-size: 9px;
    padding: 1px 4px;
    border-radius: 3px;
    font-weight: 700;
    text-transform: uppercase;
    box-shadow: 0 1px 2px rgba(0,0,0,0.1);
}
.tag-proto.tcp { background: rgba(52, 152, 219, 0.9); color: white; }
.tag-proto.udp { background: rgba(230, 126, 34, 0.9); color: white; }

.txt-port {
    font-family: 'Consolas', monospace;
    font-weight: 600;
    font-size: 11px;
    color: #2c3e50;
    opacity: 0.9;
}

.txt-time {
    font-size: 10px;
    color: rgba(0,0,0,0.5);
    font-weight: 500;
}

.empty-state {
    text-align: center;
    font-size: 10px;
    color: rgba(0,0,0,0.4);
    margin-top: 30px;
}

/* Custom Scrollbar */
.flow-list-modern::-webkit-scrollbar { width: 3px; }
.flow-list-modern::-webkit-scrollbar-track { background: transparent; }
.flow-list-modern::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.3); border-radius: 3px; }
.flow-list-modern::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.5); }

/* --- RESTORED GLOBAL STYLES --- */
.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 30px;
}

.logo-area h1 { margin: 0; font-size: 20px; font-weight: 500; letter-spacing: 0.5px; }

.status-area {
    display: flex;
    align-items: center;
    gap: 15px;
    font-size: 14px;
    color: var(--text-secondary);
}
.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--success-color);
    box-shadow: 0 0 10px var(--success-color);
}
.animate-pulse { animation: pulse 2s infinite; }
@keyframes pulse {
    0% { opacity: 1; box-shadow: 0 0 0 0 rgba(48, 209, 88, 0.7); }
    70% { opacity: 0.7; box-shadow: 0 0 0 6px rgba(48, 209, 88, 0); }
    100% { opacity: 1; box-shadow: 0 0 0 0 rgba(48, 209, 88, 0); }
}
.status-offline {
    background-color: var(--text-secondary);
    box-shadow: none;
}

/* Metrics */
.metrics-row {
    display: grid;
    /* Card 1 (Traffic) narrow, Card 2 (Flows) medium, Card 3 (Threats) wide */
    grid-template-columns: 0.8fr 1.1fr 1.3fr;
    gap: 20px;
}
.metric-card {
    padding: 24px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    height: 160px; /* Increased from 140px for charts */
}
.metric-card h3 {
    margin: 0;
    font-size: 14px;
    color: var(--text-secondary);
    font-weight: normal;
}
.metric-value {
    font-size: 36px;
    font-weight: 600;
}
.metric-number {
    font-size: 36px; /* Match standard .metric-value */
    font-weight: 600;
    line-height: 1;
}
.metric-value small { font-size: 16px; font-weight: normal; color: var(--text-secondary); }
.text-danger { color: var(--danger-color); }
.metric-sub { font-size: 13px; color: var(--text-secondary); margin-top: auto; }

/* Charts Row - Split Layout */
.charts-row {
    display: flex;
    gap: 20px;
    height: 400px;
}

.main-chart-section {
    padding: 24px;
    flex: 2; /* 66% width */
    display: flex;
    flex-direction: column;
}
.main-chart {
    width: 100%;
    flex-grow: 1;
}

.type-chart-section {
    padding: 24px;
    flex: 1; /* 33% width */
    display: flex;
    flex-direction: column;
}
.type-chart {
    width: 100%;
    flex-grow: 1;
}


/* Threat Card Styles */
.threat-card-flex {
    display: flex !important;
    flex-direction: row !important;
    align-items: center;
    justify-content: space-between; /* Metric Left, Chart Right */
    padding: 24px !important;
    gap: 15px;
    height: 160px;
    /* Removed box-sizing: border-box to match other cards */
}

/* Specific Override: Shrink left side for Threat Card to give Chart more space */
.threat-card-flex .card-left {
    flex: 0 0 25% !important; /* Down from 45% */
    min-width: 100px;
}

.threat-chart-container {
    flex: 1;
    height: 100%;
    position: relative;
    background: rgba(255, 255, 255, 0.4);
    border-radius: 8px;
    overflow: hidden; /* For chart */
    backdrop-filter: blur(4px);
    box-shadow: inset 0 2px 4px rgba(0,0,0,0.03);
    border: 1px solid rgba(255,255,255,0.3);
    display: flex;
    flex-direction: column;
}

.threat-toggles {
    position: absolute;
    top: 5px;
    right: 5px;
    z-index: 10;
    display: flex;
    gap: 4px;
    background: rgba(255,255,255,0.5);
    padding: 2px;
    border-radius: 12px;
}

.toggle-pill {
    font-size: 9px;
    padding: 2px 8px;
    border-radius: 8px;
    cursor: pointer;
    color: #7f8c8d;
    transition: all 0.2s;
    font-weight: 500;
}
.toggle-pill:hover {
    color: #2c3e50;
    background: rgba(255,255,255,0.8);
}
.toggle-pill.active {
    background: white;
    color: #d35400;
    font-weight: 700;
    box-shadow: 0 1px 2px rgba(0,0,0,0.1);
}

.threat-spark-chart {
    flex: 1;
    width: 100%;
    margin-top: 5px; /* Space for toggles */
}

/* Logs */
.logs-section {
    padding: 24px;
    flex-grow: 1;
}
.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
}
.section-header h2 { margin: 0; font-size: 18px; font-weight: 600; }

/* Table Overrides for Glass */
:deep(.el-table), :deep(.el-table__expanded-cell) {
    background-color: transparent !important;
}
:deep(.el-table tr), :deep(.el-table th.el-table__cell) {
    background-color: transparent !important;
    color: var(--text-primary);
}
:deep(.el-table--enable-row-hover .el-table__body tr:hover>td.el-table__cell) {
    background-color: rgba(255, 255, 255, 0.1) !important;
}
:deep(.el-table td.el-table__cell), :deep(.el-table th.el-table__cell.is-leaf) {
    border-bottom: 1px solid rgba(255,255,255,0.05) !important;
}
:deep(.danger-row) {
    background: rgba(255, 69, 58, 0.1) !important;
}

/* NexaVerse Card Colors */
.bg-coral { background: var(--coral-glow); color: white; }
.bg-sandy { background: var(--sandy-brown); color: #2c3e50; }
.bg-jasmine { background: var(--jasmine); color: #2c3e50; }

.bg-coral h3, .bg-coral .metric-value small, .bg-coral .metric-sub { color: rgba(255,255,255,0.9); }
.bg-sandy h3, .bg-sandy .metric-value small, .bg-sandy .metric-sub { color: rgba(0,0,0,0.6); }
.bg-jasmine h3, .bg-jasmine .metric-value small, .bg-jasmine .metric-sub { color: rgba(0,0,0,0.6); }

.text-dark { color: #2c3e50 !important; }

/* Remove old Glass Table overrides that forced transparency */
:deep(.el-table) { background-color: transparent; }
:deep(.el-table tr), :deep(.el-table th.el-table__cell) { background-color: transparent; color: var(--text-primary); }
:deep(.el-table td.el-table__cell) { border-bottom: 1px solid rgba(0,0,0,0.05); }


/* Floating Diff Animation */
.diff-float {
    display: inline-block;
    margin-left: 8px; 
    font-size: 20px; 
    font-weight: bold;
    animation: floatUp 2.5s ease-out forwards; 
    pointer-events: none;
    vertical-align: top;
}
.text-success { color: #2ecc71; }
.text-danger { color: #e74c3c; }

@keyframes floatUp {
    0% { opacity: 1; transform: translateY(0); }
    20% { opacity: 1; transform: translateY(-5px); } 
    100% { opacity: 0; transform: translateY(-30px); }
}</style>
