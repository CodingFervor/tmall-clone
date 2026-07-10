<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getPresales, payPresaleDeposit } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

onMounted(async () => {
  try { deals.value = await getPresales() } catch (e) { showToast('加载失败') } finally { loading.value = false }
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })

function remainMs(d) {
  return Math.max(0, new Date(d.deposit_end).getTime() - now.value)
}
function fmtRemain(ms) {
  const h = Math.floor(ms / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  return `${h}时${m}分`
}
// Presale timeline helpers. The deposit phase opens 48h before deposit_end (matches the seed window).
function depositStart(d) { return new Date(d.deposit_end).getTime() - 48 * 3600000 }
function balanceEnd(d) { return new Date(d.balance_start).getTime() + 48 * 3600000 }
// Current stage: deposit -> gap -> balance -> done.
function stage(d) {
  const t = now.value
  const de = new Date(d.deposit_end).getTime()
  const bs = new Date(d.balance_start).getTime()
  const be = balanceEnd(d)
  if (d.status === 'completed' || t >= be) return 'done'
  if (t < de) return 'deposit'
  if (t < bs) return 'gap'
  return 'balance'
}
// Dual-phase countdown text.
function countdownText(d) {
  const s = stage(d)
  const t = now.value
  if (s === 'deposit') return `定金还剩 ${fmtRemain(new Date(d.deposit_end).getTime() - t)}`
  if (s === 'gap') return `尾款开始 ${fmtRemain(new Date(d.balance_start).getTime() - t)}`
  if (s === 'balance') return `尾款截止 ${fmtRemain(balanceEnd(d) - t)}`
  return '已结束'
}
// Deposit doubling bonus is only available within the first 24h of the deposit phase.
function depositDoubleActive(d) {
  const t = now.value
  return t >= depositStart(d) && t < depositStart(d) + 24 * 3600000 && stage(d) === 'deposit'
}
function savePct(d) {
  if (!d.original_price || d.original_price <= d.final_price) return 0
  return Math.round((1 - d.final_price / d.original_price) * 100)
}
async function payDeposit(d) {
  try {
    await payPresaleDeposit(d.id)
    d.sold++
    showSuccessToast('定金支付成功')
  } catch (e) {
    showToast(e.response?.data?.error || '支付失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="ps-page">
    <van-nav-bar title="预售专区" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无预售活动" />
    <div v-else class="ps-list">
      <div class="banner">🎁 定金预售 · 提前锁定好价</div>
      <div v-for="d in deals" :key="d.id" class="ps-card">
        <van-image width="100" height="100" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="pc-info">
          <div class="pc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="pc-price-row">
            <span class="pc-final">¥{{ fmt(d.final_price) }}</span>
            <span class="pc-origin">¥{{ fmt(d.original_price) }}</span>
            <van-tag type="danger" round v-if="savePct(d) > 0">省{{ savePct(d) }}%</van-tag>
          </div>
          <div class="pc-deposit">定金 ¥{{ fmt(d.deposit) }} · 尾款 ¥{{ fmt(d.balance) }}</div>
          <div class="pc-stages">
            <span class="pc-stage" :class="{ active: stage(d) === 'deposit', done: ['gap','balance','done'].includes(stage(d)) }">定金期</span>
            <span class="pc-stage-sep">→</span>
            <span class="pc-stage" :class="{ active: stage(d) === 'gap', done: ['balance','done'].includes(stage(d)) }">间隔期</span>
            <span class="pc-stage-sep">→</span>
            <span class="pc-stage" :class="{ active: stage(d) === 'balance', done: stage(d) === 'done' }">尾款期</span>
            <span class="pc-stage-sep">→</span>
            <span class="pc-stage" :class="{ active: stage(d) === 'done' }">已完成</span>
          </div>
          <div class="pc-bottom">
            <span class="pc-countdown">⏰ {{ countdownText(d) }}</span>
            <van-tag type="warning" round class="pc-double" v-if="depositDoubleActive(d)">定金翻倍</van-tag>
            <van-button size="small" type="danger" round @click="payDeposit(d)" :disabled="stage(d) !== 'deposit'">付定金</van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ps-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.banner { background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.ps-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.pc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.pc-name { font-size: 14px; line-height: 20px; flex: 1; }
.pc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.pc-final { color: #ff0036; font-size: 20px; font-weight: bold; }
.pc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
.pc-deposit { font-size: 12px; color: #666; margin-bottom: 6px; }
/* Stage indicator: 定金期 → 间隔期 → 尾款期 → 已完成, current stage highlighted. */
.pc-stages { display: flex; align-items: center; flex-wrap: wrap; gap: 3px; margin-bottom: 8px; }
.pc-stage { font-size: 11px; padding: 2px 8px; border-radius: 10px; background: #f0f0f0; color: #999; border: 1px solid transparent; }
.pc-stage.done { background: #fff0f3; color: #ff0036; }
.pc-stage.active { background: #ff0036; color: #fff; border-color: #ff0036; box-shadow: 0 0 0 2px rgba(255,0,54,0.15); font-weight: bold; }
.pc-stage-sep { font-size: 11px; color: #ccc; }
.pc-bottom { display: flex; justify-content: space-between; align-items: center; gap: 6px; margin-top: auto; }
.pc-countdown { color: #ff0036; font-size: 12px; font-weight: bold; flex: 1; }
.pc-double { flex-shrink: 0; }
</style>
