<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getSeckillDeals, grabSeckill } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

onMounted(async () => {
  try { deals.value = await getSeckillDeals() } catch (e) { showToast('加载失败') } finally { loading.value = false }
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })

function remainMs(d) {
  return Math.max(0, new Date(d.end_time).getTime() - now.value)
}
function fmtRemain(ms) {
  const h = Math.floor(ms / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  const s = Math.floor((ms % 60000) / 1000)
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
function progress(d) {
  if (d.stock <= 0) return 100
  return Math.min(100, Math.round((d.sold / d.stock) * 100))
}
// Stock below 20% remaining triggers the urgent shake effect.
function lowStock(d) {
  if (!d.stock || d.stock <= 0) return progress(d) >= 80
  return (d.stock - d.sold) / d.stock < 0.2
}
async function grab(d) {
  try {
    const res = await grabSeckill(d.id)
    d.sold = res.data.sold
    showSuccessToast('抢购成功！')
  } catch (e) {
    showToast(e.response?.data?.error || '抢购失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="seckill-page">
    <div class="flash-overlay"></div>
    <van-nav-bar title="天猫秒杀" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无秒杀活动" />
    <div v-else class="deal-list">
      <div class="banner">⚡ 限时秒杀 · 抢完即止</div>
      <div v-for="d in deals" :key="d.id" class="deal-card">
        <van-image width="110" height="110" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="dc-info">
          <div class="dc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="dc-price-row">
            <span class="dc-sk-price">¥{{ fmt(d.seckill_price) }}</span>
            <span class="dc-origin">¥{{ fmt(d.original_price) }}</span>
          </div>
          <div class="dc-progress">
            <div class="dp-bar" :class="{ shake: lowStock(d) && progress(d) < 100 }"><div class="dp-fill" :style="{ width: progress(d) + '%' }"></div></div>
            <span class="dp-text">已抢{{ progress(d) }}%</span>
          </div>
          <div class="dc-bottom">
            <span class="dc-countdown">⏰ {{ fmtRemain(remainMs(d)) }}</span>
            <van-button size="small" type="danger" round class="grab-btn" :disabled="progress(d) >= 100" @click="grab(d)">
              {{ progress(d) >= 100 ? '已抢光' : '马上抢' }}
            </van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.seckill-page { min-height: 100vh; position: relative; }
.loading { text-align: center; padding: 80px; }
/* Periodic white flash overlay: a 5s infinite loop fires the flash once per cycle (pure CSS). */
.flash-overlay { position: fixed; inset: 0; background: #fff; pointer-events: none; z-index: 9999; opacity: 0; animation: seckill-flash 5s ease-out infinite; }
@keyframes seckill-flash {
  0% { opacity: 0; }
  2% { opacity: 0.85; }
  10% { opacity: 0; }
  100% { opacity: 0; }
}
.banner { background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.deal-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.dc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.dc-name { font-size: 14px; line-height: 20px; flex: 1; }
.dc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.dc-sk-price { color: #ff0036; font-size: 20px; font-weight: bold; }
.dc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
.dc-progress { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.dp-bar { flex: 1; height: 12px; background: #ffe0e0; border-radius: 6px; overflow: hidden; }
.dp-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #ff0036); transition: width 0.3s; }
.dp-text { font-size: 11px; color: #ff0036; white-space: nowrap; }
/* Urgent shake on the progress bar when stock < 20% remaining (pure CSS). */
.dp-bar.shake { animation: seckill-shake 0.4s linear infinite; }
@keyframes seckill-shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-3px); }
  40% { transform: translateX(3px); }
  60% { transform: translateX(-2px); }
  80% { transform: translateX(2px); }
}
.dc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
.dc-countdown { color: #ff0036; font-size: 14px; font-weight: bold; font-variant-numeric: tabular-nums; }
/* Pulsing glow on the "马上抢" button (pure CSS). */
.grab-btn { animation: seckill-pulse 1.2s ease-in-out infinite; }
@keyframes seckill-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(255,0,54,0.7), 0 0 6px 1px rgba(255,0,54,0.6); transform: scale(1); }
  50% { box-shadow: 0 0 0 8px rgba(255,0,54,0), 0 0 14px 3px rgba(255,0,54,0.9); transform: scale(1.06); }
}
.grab-btn:disabled { animation: none; }
</style>
