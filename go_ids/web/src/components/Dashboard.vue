<template>
  <div class="dashboard-container">
    <!-- Header -->
    <header class="glass-panel header">
      <div class="logo-area">
        <el-icon :size="24" color="#fff"><Monitor /></el-icon>
        <h1>Go-IDS 监控系统</h1>
      </div>
      <div class="status-area">
        <span class="status-dot animate-pulse"></span>
        <span class="status-text">系统运行中</span>
        <span class="uptime">运行时长: {{ uptime }}</span>
      </div>
    </header>

    <!-- Top Cards -->
    <div class="metrics-row">
      <div class="glass-panel metric-card">
        <h3>实时流量</h3>
        <div class="metric-value">{{ currentTraffic }} <small>Mbps</small></div>
        <div class="chart-sparkline" ref="sparklineRef"></div>
      </div>
      <div class="glass-panel metric-card">
        <h3>活跃连接</h3>
        <div class="metric-value">{{ activeFlows }}</div>
        <div class="metric-sub">当前活跃流数量</div>
      </div>
      <div class="glass-panel metric-card danger-card">
        <h3>威胁检测</h3>
        <div class="metric-value text-danger">{{ totalThreats }}</div>
        <div class="metric-sub text-danger">过去 24 小时</div>
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
import { Monitor } from '@element-plus/icons-vue'
import { getHistory } from '../api'

// State
const alerts = ref([])
const totalThreats = ref(0)
const activeFlows = ref(840) // Mock base
const currentTraffic = ref(12.5) // Mock base
const uptime = ref('00:00:00')
const chartType = ref('pie')
const typeStats = reactive({})

// Charts Refs
const mainChartRef = ref(null)
const typeChartRef = ref(null)
let mainChart = null
let typeChart = null

const trafficData = []
const threatData = []

// Formatting
const formatTime = (isoString) => {
    return new Date(isoString).toLocaleTimeString()
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
    mainChart = echarts.init(mainChartRef.value)
    
    // Init Mock Trend Data
    const now = new Date()
    for (let i = 0; i < 60; i++) {
        trafficData.push({
            name: new Date(now - (60 - i) * 1000).toString(),
            value: [new Date(now - (60 - i) * 1000), Math.random() * 20 + 10]
        })
        threatData.push({
             name: new Date(now - (60 - i) * 1000).toString(),
             value: [new Date(now - (60 - i) * 1000), 0]
        })
    }

    const trendOptions = {
        grid: { top: 30, right: 20, bottom: 20, left: 50, containLabel: true },
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'time', splitLine: { show: false }, axisLabel: { color: '#aaa' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }, axisLabel: { color: '#aaa' } },
        series: [
            {
                name: '流量 (Mbps)',
                type: 'line',
                smooth: true,
                showSymbol: false,
                data: trafficData,
                itemStyle: { color: '#0a84ff' },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(10, 132, 255, 0.5)' },
                        { offset: 1, color: 'rgba(10, 132, 255, 0)' }
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
    typeChart = echarts.init(typeChartRef.value)
    renderTypeChart()
    
    window.addEventListener('resize', () => {
        mainChart.resize()
        typeChart.resize()
    })
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
                    itemStyle: { color: '#0a84ff', borderRadius: [5, 5, 0, 0] }
                }
            ]
        }
    }
    
    typeChart.setOption(option, true) // true = merge not allowed (reset)
}


const updateMainChart = () => {
    const now = new Date()
    // Mock Traffic
    const newTraffic = Math.max(0, currentTraffic.value + (Math.random() - 0.5) * 5)
    currentTraffic.value = parseFloat(newTraffic.toFixed(1))
    
    trafficData.shift()
    trafficData.push({ name: now.toString(), value: [now, newTraffic] })
    
    threatData.shift()
    threatData.push({ name: now.toString(), value: [now, 0] }) 
    
    mainChart.setOption({
        series: [{ data: trafficData }, { data: threatData }]
    })
}


// SSE & Logic
onMounted(async () => {
    // 1. Initial Load
    try {
        const res = await getHistory()
        alerts.value = res.data
        totalThreats.value = res.data.length
        
        // Init Stats
        res.data.forEach(a => {
            typeStats[a.type] = (typeStats[a.type] || 0) + 1
        })
    } catch (e) {
        console.error("Failed to load history", e)
    }

    // 2. Charts
    initCharts()
    renderTypeChart()
    setInterval(updateMainChart, 1000)

    // 3. SSE
    const evtSource = new EventSource('http://localhost:8080/api/events')
    
    evtSource.addEventListener('alert', (e) => {
        const newAlert = JSON.parse(e.data)
        alerts.value.unshift(newAlert)
        if (alerts.value.length > 50) alerts.value.pop()
        
        totalThreats.value++
        
        // Update Stats
        typeStats[newAlert.type] = (typeStats[newAlert.type] || 0) + 1
        renderTypeChart()
        
        // Spike in chart
        const lastIdx = threatData.length - 1
        threatData[lastIdx].value[1] = 50 
    })
    
    // Uptime Timer
    const startTime = Date.now()
    setInterval(() => {
        const diff = Math.floor((Date.now() - startTime) / 1000)
        const h = Math.floor(diff / 3600).toString().padStart(2, '0')
        const m = Math.floor((diff % 3600) / 60).toString().padStart(2, '0')
        const s = (diff % 60).toString().padStart(2, '0')
        uptime.value = `${h}:${m}:${s}`
    }, 1000)
})

onUnmounted(() => {
   if (mainChart) mainChart.dispose()
   if (typeChart) typeChart.dispose()
})

</script>

<style scoped>
.dashboard-container {
    padding: 20px;
    max-width: 1600px;
    margin: 0 auto;
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

.logo-area {
    display: flex;
    align-items: center;
    gap: 15px;
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

/* Metrics */
.metrics-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 20px;
}
.metric-card {
    padding: 24px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    height: 140px;
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
    background: rgba(255, 69, 58, 0.15) !important;
}
</style>
