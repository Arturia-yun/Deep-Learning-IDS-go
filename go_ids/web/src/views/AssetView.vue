<template>
  <div class="asset-container">
    <header class="glass-panel header">
      <div class="logo-area">
        <h1>内网资产安全画像 (Asset Profile)</h1>
      </div>
      <div class="status-area">
        <span class="info-text">数据来源: 实时溯源聚合池</span>
      </div>
    </header>

    <div class="dashboard-grid">
      <!-- 发现的受攻击资产总数概览 -->
      <!-- 发现的受攻击资产总数与拓扑概览 -->
      <div class="glass-panel overview-card">
        <div class="overview-content">
           <div class="overview-header">
               <h3>内网受害资产拓扑 (Asset Topology)</h3>
               <div class="alert-badge text-danger" v-if="targetedAssetsCount > 0">
                   发现 {{ targetedAssetsCount }} 台高危设备
               </div>
           </div>
           <div ref="topoChartRef" class="topo-chart"></div>
        </div>
      </div>

      <!-- Top Receivers / 受伤最多的内网机 -->
      <div class="glass-panel list-card top-victims">
        <div class="section-header">
           <h3>受攻击频次 Top 5 榜单 (Victims)</h3>
        </div>
        <div class="list-body">
            <div v-for="(v, index) in topVictims" :key="v.ip" class="victim-row">
                <div class="v-rank" :class="'rank-' + (index + 1)">#{{ index + 1 }}</div>
                <div class="v-ip">
                    <span class="ip-addr">{{ v.ip }}</span>
                    <el-tag size="small" type="danger" effect="dark" round v-if="index === 0">高危警戒</el-tag>
                </div>
                <div class="v-count">
                    <span class="count-num">{{ v.count }}</span> 次拦截
                </div>
                <div class="v-bar-container">
                     <div class="v-bar" :style="{ width: (v.count / maxAttackCount * 100) + '%' }"></div>
                </div>
            </div>
            <div v-if="topVictims.length === 0" class="empty-state">当前网络干净，暂无受害者资产</div>
        </div>
      </div>

       <!-- Top Attackers / 恶意扫段元凶 -->
       <div class="glass-panel list-card top-attackers">
        <div class="section-header">
           <h3>恶意源 IP 活跃度 Top 5 (Attackers)</h3>
        </div>
        <div class="list-body">
            <div v-for="(v, index) in topAttackers" :key="v.ip" class="attacker-row">
                <div class="v-ip">
                    <el-icon class="threat-icon"><WarnTriangleFilled /></el-icon>
                    <span class="ip-addr">{{ v.ip }}</span>
                </div>
                <div class="v-types">
                   <!-- 聚合显示攻击手法 -->
                   <span class="type-badge" v-for="t in v.types.slice(0,2)" :key="t">{{ t }}</span>
                   <span class="type-badge more" v-if="v.types.length > 2">+{{ v.types.length - 2 }}</span>
                </div>
                <div class="v-count text-danger">
                     发起 {{ v.count }} 次
                </div>
            </div>
             <div v-if="topAttackers.length === 0" class="empty-state">无外部活跃威胁</div>
        </div>
      </div>
      
      <!-- 雷达图：攻击手法面面观 -->
      <div class="glass-panel radar-card">
         <div class="section-header">
           <h3>威胁类型维度雷达 (Threat Radar)</h3>
        </div>
        <div ref="radarChartRef" class="radar-chart"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { WarnTriangleFilled } from '@element-plus/icons-vue'

const alerts = ref([])
const topVictims = ref([])
const topAttackers = ref([])
const maxAttackCount = ref(1)

const topoChartRef = ref(null)
let topoChart = null
const radarChartRef = ref(null)
let radarChart = null

const targetedAssetsCount = computed(() => {
    const uniqueIPs = new Set(alerts.value.map(a => a.dest_ip))
    return uniqueIPs.size
})

// 数据预处理核心逻辑
const processData = (data) => {
    alerts.value = data
    
    // 聚合受害者 (Dest IP)
    const destMap = {}
    const srcMap = {}
    const typeCountMap = {}
    
    data.forEach(item => {
        // Victim
        if (!destMap[item.dest_ip]) destMap[item.dest_ip] = 0
        destMap[item.dest_ip]++
        
        // Attacker
        if (!srcMap[item.source_ip]) srcMap[item.source_ip] = { count: 0, types: new Set() }
        srcMap[item.source_ip].count++
        srcMap[item.source_ip].types.add(item.type)
        
        // Types
        if (!typeCountMap[item.type]) typeCountMap[item.type] = 0
        typeCountMap[item.type]++
    })
    
    // 排序并取 Top 5 Victims
    const sortedVictims = Object.keys(destMap).map(ip => ({ ip, count: destMap[ip] }))
                               .sort((a,b) => b.count - a.count)
    topVictims.value = sortedVictims.slice(0, 5)
    maxAttackCount.value = sortedVictims.length > 0 ? sortedVictims[0].count : 1
    
    // 提取全局高危节点（凡是遭受攻击的都是高危节点）
    const highRiskIPs = new Set(sortedVictims.map(v => v.ip))
    
    // 排序 Attackers
    const sortedAttackers = Object.keys(srcMap).map(ip => ({ 
        ip, 
        count: srcMap[ip].count,
        types: Array.from(srcMap[ip].types)
    })).sort((a,b) => b.count - a.count)
    topAttackers.value = sortedAttackers.slice(0, 5)
    
    // 渲染图表
    renderRadar(typeCountMap)
    
    // 基于受害者和攻击者关系提取拓扑 links (为了视图清晰，我们以 topAttacker 为核心衍生边)
    const topoLinks = []
    const topoNodes = new Map()
    
    // 恒定中心节点：防火墙/网关
    topoNodes.set('Gateway', { name: '企业网关', symbolSize: 40, category: 0 })
    
    // 将发现的所有受害 IP 接入网关，并标红
    sortedVictims.forEach(v => {
        topoNodes.set(v.ip, { name: v.ip, symbolSize: 20 + Math.min(v.count*2, 30), category: 1, value: v.count })
        topoLinks.push({ source: 'Gateway', target: v.ip })
    })

    // 画线：Attacker -> Victim
    // 只取头部攻击源防止连线爆炸
    topAttackers.value.slice(0, 3).forEach(atk => {
        topoNodes.set(atk.ip, { name: atk.ip, symbolSize: 25, category: 2 })
        // 查找此攻击者攻击了谁
        const targets = [...new Set(data.filter(d => d.source_ip === atk.ip).map(d => d.dest_ip))]
        targets.forEach(t => {
            if(topoNodes.has(t)) {
                topoLinks.push({ source: atk.ip, target: t, lineStyle: { color: '#e74c3c', width: 2, curveness: 0.2 } })
            }
        })
    })
    
    // 填充内网良性节点（模拟展示网段环境）
    if(sortedVictims.length < 8) {
         for(let i=1; i<= (8 - sortedVictims.length); i++) {
             let mockIp = `192.168.1.${100 + i}`
             if(!topoNodes.has(mockIp)) {
                 topoNodes.set(mockIp, { name: mockIp, symbolSize: 15, category: 3 })
                 topoLinks.push({ source: 'Gateway', target: mockIp })
             }
         }
    }
    
    const nodesArray = Array.from(topoNodes.values())
    renderTopo(nodesArray, topoLinks)
}

const renderTopo = (nodes, links) => {
    if (!topoChartRef.value) return
    if (!topoChart) topoChart = echarts.init(topoChartRef.value)
    
    const option = {
        tooltip: { formatter: '{b}' },
        legend: {
            data: ['核心网关', '发现受害机 (高危)', '外部攻击源', '普通终端'],
            bottom: 0, textStyle: { color: '#7f8c8d', fontSize: 10 }
        },
        color: ['#3498db', '#e74c3c', '#8e44ad', '#2ecc71'],
        series: [{
            type: 'graph',
            layout: 'force',
            force: { 
                repulsion: [100, 300], // 动态斥力
                edgeLength: [50, 80],
                gravity: 0.1,
                friction: 0.1 // 降低摩擦系数，让拖拽更平滑，松手后有回弹感
            },
            roam: true, // 允许缩放和平移
            draggable: true, // 允许拖拽节点
            edgeSymbol: ['none', 'arrow'], // 连线显示箭头
            edgeSymbolSize: [4, 8],
            label: { show: true, position: 'right', fontSize: 10, color: '#34495e' },
            data: nodes.map(n => ({
                name: n.name,
                symbolSize: n.symbolSize,
                category: n.category,
                // 高亮选中效果
                itemStyle: n.category === 1 ? { shadowBlur: 10, shadowColor: '#e74c3c' } : null
            })),
            links: links,
            lineStyle: { color: 'rgba(0,0,0,0.15)', curveness: 0.1, opacity: 0.8 },
            categories: [
                { name: '核心网关' }, { name: '发现受害机 (高危)' },
                { name: '外部攻击源' }, { name: '普通终端' }
            ]
        }]
    }
    topoChart.setOption(option)
}

const renderRadar = (typeMap) => {
    if (!radarChartRef.value) return
    if (!radarChart) {
        radarChart = echarts.init(radarChartRef.value)
    }
    
    const types = Object.keys(typeMap)
    let indicators = []
    let values = []
    let tMax = 1
    
    if (types.length === 0) {
        // Mock fallback if empty
        indicators = [
            { name: 'DDoS', max: 100 }, { name: 'PortScan', max: 100 }, 
            { name: 'Malware', max: 100 }, { name: 'Injection', max: 100 },
            { name: 'BruteForce', max: 100 }
        ]
        values = [0, 0, 0, 0, 0]
    } else {
        // 动态计算 max
        tMax = Math.max(...Object.values(typeMap)) * 1.2
        if (tMax < 5) tMax = 5
        
        indicators = types.map(t => ({ name: t, max: tMax }))
        values = types.map(t => typeMap[t])
    }

    const option = {
        tooltip: { trigger: 'item' },
        radar: {
            indicator: indicators,
            shape: 'circle',
            splitNumber: 4,
            axisName: { color: '#34495e', fontSize: 11, fontWeight: 500 },
            splitLine: { lineStyle: { color: ['rgba(230, 126, 34, 0.1)', 'rgba(230, 126, 34, 0.2)', 'rgba(230, 126, 34, 0.4)', 'rgba(230, 126, 34, 0.6)'] } },
            splitArea: { show: false },
            axisLine: { lineStyle: { color: 'rgba(230, 126, 34, 0.4)' } }
        },
        series: [{
            name: '威胁维度落点',
            type: 'radar',
            data: [{
                value: values,
                name: '发现的攻击手段',
                symbol: 'circle',
                symbolSize: 6,
                itemStyle: { color: '#e67e22' },
                areaStyle: {
                    color: new echarts.graphic.RadialGradient(0.5, 0.5, 1, [
                         { color: 'rgba(230, 126, 34, 0.1)', offset: 0 },
                         { color: 'rgba(230, 126, 34, 0.5)', offset: 1 }
                    ])
                },
                lineStyle: { width: 2 }
            }]
        }]
    }
    
    radarChart.setOption(option)
}

const fetchData = async () => {
    try {
        // 我们利用历史报警端点拉取近 500 条数据进行资产分析
        const res = await axios.get('http://localhost:8080/api/alerts?limit=500')
        processData(res.data)
    } catch(e) {
        console.error("Asset profiling error", e)
    }
}

onMounted(() => {
    fetchData()
    // 监听实时报警流进行重新计算
    const evtSource = new EventSource('http://localhost:8080/api/events')
    evtSource.addEventListener('alert', () => {
        // 为了简单，接到警报后节流重新请求一次全量数据
        fetchData()
    })
    
    const resizeObserver = new ResizeObserver(() => {
        if(radarChart) radarChart.resize()
        if(topoChart) topoChart.resize()
    })
    if (radarChartRef.value) resizeObserver.observe(radarChartRef.value.parentElement)
    if (topoChartRef.value) resizeObserver.observe(topoChartRef.value.parentElement)
})

onUnmounted(() => {
    if (radarChart) radarChart.dispose()
    if (topoChart) topoChart.dispose()
})
</script>

<style scoped>
.asset-container {
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
.info-text { color: #7f8c8d; font-size: 13px; }

.dashboard-grid {
    display: grid;
    /* 采用 CSS grid 画布，两列布局 */
    grid-template-columns: 1fr 1fr;
    grid-template-rows: auto auto;
    gap: 20px;
}

/* Card 1: 资产总数揽件与拓扑 (占据左上一整块位置) */
.overview-card {
    padding: 20px;
    display: flex;
    flex-direction: column;
    background: linear-gradient(135deg, rgba(236, 240, 241, 0.4), rgba(255, 255, 255, 0.2));
    min-height: 300px; /* 改为 min-height，让 Grid 的默认 align-items: stretch 促使其等高 */
}
.overview-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    width: 100%;
}
.overview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid rgba(0,0,0,0.05);
    padding-bottom: 10px;
    margin-bottom: 10px;
}
.overview-header h3 { font-size: 15px; font-weight: 600; color: #2c3e50; margin: 0; }
.alert-badge {
    background: rgba(231, 76, 60, 0.1);
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
}
.topo-chart {
    flex: 1;
    width: 100%;
}

/* Radar Chart (右上角) */
.radar-card {
    padding: 20px;
    height: 320px;
    display: flex;
    flex-direction: column;
}
.radar-chart { flex: 1; width: 100%; }

/* Lists Styles */
.list-card {
    padding: 25px;
    display: flex;
    flex-direction: column;
    min-height: 300px;
}
.section-header { margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid rgba(0,0,0,0.05); }
.section-header h3 { font-size: 15px; font-weight: 600; color: #2c3e50; margin: 0; }

.list-body { flex: 1; display: flex; flex-direction: column; gap: 15px; }

/* Victim Rows */
.victim-row {
    position: relative;
    padding: 12px;
    background: rgba(255,255,255,0.5);
    border-radius: 8px;
    display: flex;
    align-items: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.02);
}
.v-rank {
    font-size: 16px; font-weight: 800; color: #ccc; width: 40px; text-align: center;
}
.rank-1 { color: #e74c3c; font-size: 20px; }
.rank-2 { color: #e67e22; font-size: 18px; }
.rank-3 { color: #f1c40f; }

.v-ip { flex: 1; display: flex; align-items: center; gap: 10px; }
.ip-addr { font-family: 'Consolas', monospace; font-weight: 600; font-size: 14px; color: #34495e; }

.v-count { font-size: 13px; color: #7f8c8d; width: 80px; text-align: right; z-index: 2; }
.count-num { font-size: 16px; font-weight: 700; color: #2c3e50; }

.v-bar-container {
    position: absolute;
    bottom: 0;
    left: 40px;
    right: 90px;
    height: 4px;
    background: transparent;
    border-radius: 2px;
    overflow: hidden;
}
.v-bar {
    height: 100%;
    background: linear-gradient(90deg, #f39c12, #e74c3c);
    border-radius: 2px;
    transition: width 0.5s cubic-bezier(0.25, 0.8, 0.25, 1);
}

/* Attacker Rows */
.attacker-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 15px;
    border-left: 3px solid #e74c3c;
    background: linear-gradient(90deg, rgba(231, 76, 60, 0.05), transparent);
    margin-bottom: 8px;
}
.threat-icon { color: #e74c3c; font-size: 16px; margin-right: 8px; margin-top: 2px;}

.v-types { display: flex; gap: 6px; }
.type-badge { font-size: 10px; background: rgba(0,0,0,0.05); color: #666; padding: 2px 6px; border-radius: 4px; border: 1px solid rgba(0,0,0,0.1); }
.type-badge.more { background: transparent; border: none; font-weight: bold; }

.empty-state { text-align: center; font-size: 13px; color: #aaa; margin-top: 40px; }
</style>
