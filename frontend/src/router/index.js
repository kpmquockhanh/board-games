import { createRouter, createWebHistory } from 'vue-router'
import HubView from '../views/HubView.vue'
import HotpotView from '../views/HotpotView.vue'
import EkView from '../views/EkView.vue'

const routes = [
  { path: '/', name: 'hub', component: HubView },
  { path: '/hotpot', name: 'hotpot', component: HotpotView },
  { path: '/exploding-kitchen', name: 'exploding-kitchen', component: EkView },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
