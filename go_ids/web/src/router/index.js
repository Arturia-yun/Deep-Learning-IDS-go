import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ThreatsView from '../views/ThreatsView.vue'
import NetworkView from '../views/NetworkView.vue'
import SettingsView from '../views/SettingsView.vue'
import EngineView from '../views/EngineView.vue'
import AssetView from '../views/AssetView.vue'

const routes = [
    { path: '/', name: 'Dashboard', component: DashboardView },
    { path: '/threats', name: 'Threats', component: ThreatsView },
    { path: '/asset', name: 'Asset', component: AssetView },
    { path: '/engine', name: 'Engine', component: EngineView },
    { path: '/settings', name: 'Settings', component: SettingsView },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

export default router
