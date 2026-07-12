<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog } from 'vant'
import { getPointShop, redeemPoints, getRedemptions } from '../api'

const router = useRouter()
const products = ref([])
const points = ref(0)
const loading = ref(true)
const redemptions = ref([])

async function load() {
  loading.value = true
  try {
    const res = await getPointShop()
    products.value = res.data || []
    points.value = res.points || 0
    redemptions.value = await getRedemptions()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- 兑换老虎机动画 (slot-machine spin before reveal) ----
// On redeem we show a 3-column slot machine that spins, then stops column by
// column left→right, before revealing the result. The columns cycle emoji
// symbols during the spin (a fast CSS blur), then each locks onto the gift 🎁.
const SLOT_SYMBOLS = ['🎁', '⭐', '💎', '🎉', '🍀']
// Per-column spin state: the landed symbol index (null while still spinning).
const slotCols = ref([null, null, null])
const slotReels = ref([[], [], []])   // rendered symbols per column
const slotOffsets = ref([0, 0, 0])    // translateY offset (in symbol heights)
const showSlot = ref(false)           // controls the slot popup
const slotSpinning = ref(false)
const slotResult = ref(null)          // the product being redeemed, for the reveal
const slotError = ref('')             // non-empty when the redeem failed
let spinTimers = []

// Build a repeating strip that ends on the target symbol.
function buildReel(targetIdx, length) {
  const reel = []
  for (let i = 0; i < length; i++) {
    reel.push((Math.floor(Math.random() * SLOT_SYMBOLS.length) + i) % SLOT_SYMBOLS.length)
  }
  reel[length - 1] = targetIdx // force the final cell to the target
  return reel
}
function stopColumn(c) {
  const reel = slotReels.value[c]
  slotOffsets.value[c] = reel.length - 1
  slotCols.value[c] = reel[reel.length - 1]
  if (slotCols.value.every((v) => v !== null)) {
    slotSpinning.value = false
    revealSlot()
  }
}
// Start the spin with a target of all-🎁 (index 0) for a celebratory reveal.
function startSlotSpin() {
  slotSpinning.value = true
  slotResult.value = null
  slotError.value = ''
  const target = 0 // 🎁
  for (let c = 0; c < 3; c++) {
    slotCols.value[c] = null
    const len = 20 + c * 4 // outer columns spin a bit longer
    slotReels.value[c] = buildReel(target, len)
    slotOffsets.value[c] = 0
  }
  for (let c = 0; c < 3; c++) {
    spinTimers.push(setTimeout(() => stopColumn(c), 700 + c * 600))
  }
}
function closeSlot() {
  spinTimers.forEach((t) => clearTimeout(t))
  spinTimers = []
  slotSpinning.value = false
  showSlot.value = false
}
async function revealSlot() {
  // After the reels settle, surface the outcome (success or error) and reload.
  if (slotError.value) {
    showToast(slotError.value)
  } else {
    showSuccessToast('兑换成功')
  }
  await load()
  // Keep the popup open a beat so the user sees the result, then auto-close.
  setTimeout(() => { showSlot.value = false }, 1200)
}

async function redeem(p) {
  try {
    await showConfirmDialog({ title: '确认兑换', message: `兑换「${p.name}」将扣除 ${p.points} 积分？` })
    // Open the slot machine and kick off the spin immediately; the API call runs
    // in parallel so the spin and the network request resolve together.
    slotResult.value = p
    slotError.value = ''
    showSlot.value = true
    startSlotSpin()
    try {
      const res = await redeemPoints(p.id)
      points.value = res.points
      // The spin will resolve on its own; nothing to do here on success.
    } catch (e) {
      // Record the error so revealSlot() surfaces it once the reels stop.
      slotError.value = (e && e.response && e.response.data && e.response.data.error) || '兑换失败'
      // Cancel the remaining stop timers so we don't show a false success.
      spinTimers.forEach((t) => clearTimeout(t))
      spinTimers = []
      // Force the reels to settle immediately on the error.
      for (let c = 0; c < 3; c++) {
        if (slotCols.value[c] === null) {
          const reel = slotReels.value[c]
          if (reel && reel.length) {
            slotOffsets.value[c] = reel.length - 1
            slotCols.value[c] = reel[reel.length - 1]
          } else {
            slotCols.value[c] = 0
          }
        }
      }
      slotSpinning.value = false
      revealSlot()
    }
  } catch (e) {
    if (e && e.message !== 'cancel' && e !== 'cancel') {
      showToast(e.response?.data?.error || '兑换失败')
    }
  }
}
</script>

<template>
  <div class="shop-page">
    <van-nav-bar title="积分商城" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="points-banner">
      <div class="pb-label">我的可用积分</div>
      <div class="pb-value">{{ points }}</div>
      <van-button size="mini" plain round color="#fff" @click="router.push('/checkin')">去签到赚积分</van-button>
    </div>

    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else>
      <div class="section-title">积分好礼</div>
      <div class="product-grid">
        <div v-for="p in products" :key="p.id" class="product-card">
          <van-image width="100%" height="100" radius="8" :src="p.image" fit="cover" />
          <div class="pc-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="pc-points">{{ p.points }} 积分</div>
          <div class="pc-stock">剩余 {{ p.stock }} 件</div>
          <van-button size="small" type="danger" round block :disabled="points < p.points" @click="redeem(p)">
            {{ points < p.points ? '积分不足' : '立即兑换' }}
          </van-button>
        </div>
      </div>

      <div v-if="redemptions.length" class="section-title">兑换记录</div>
      <div v-if="redemptions.length" class="record-list">
        <div v-for="r in redemptions" :key="r.id" class="record-item">
          <div class="ri-info">
            <div class="ri-name">{{ r.product_name }}</div>
            <div class="ri-time">{{ r.created_at?.slice(0, 16).replace('T', ' ') }}</div>
          </div>
          <div class="ri-cost">-{{ r.points_cost }} 积分</div>
        </div>
      </div>
    </template>

    <!-- 兑换老虎机动画 (slot-machine spin before reveal) -->
    <van-popup v-model:show="showSlot" round closeable close-icon-position="top-right" :style="{ width: '86%' }" :close-on-click-overlay="false">
      <div class="slot-popup">
        <div class="slot-hero">
          <div class="slot-title">🎁 积分兑换</div>
          <div class="slot-sub">{{ slotSpinning ? '正在兑换...' : (slotError ? '兑换失败' : '兑换成功') }}</div>
        </div>
        <!-- 3-column slot machine -->
        <div class="slot-machine">
          <div v-for="(reel, c) in slotReels" :key="c" class="slot-col">
            <div
              class="slot-strip"
              :class="{ 'slot-spinning': slotSpinning && slotCols[c] === null, 'slot-stopped': slotCols[c] !== null }"
              :style="{ transform: `translateY(-${slotOffsets[c] * (100 / (reel.length || 1))}%)` }"
            >
              <div v-for="(sym, r) in reel" :key="r" class="slot-symbol">{{ SLOT_SYMBOLS[sym] }}</div>
            </div>
          </div>
        </div>
        <div v-if="slotResult && !slotSpinning" class="slot-reveal" :class="{ 'reveal-error': slotError }">
          <template v-if="!slotError">🎉 恭喜兑换「{{ slotResult.name }}」</template>
          <template v-else>😅 {{ slotError }}</template>
        </div>
        <div v-if="!slotSpinning" class="slot-actions">
          <van-button block round type="danger" @click="closeSlot">完成</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.shop-page { min-height: 100vh; padding-bottom: 20px; }
.loading { text-align: center; padding: 80px; }
.points-banner { background: linear-gradient(135deg, #ff0036, #ff5577); padding: 24px 20px; text-align: center; color: #fff; }
.pb-label { font-size: 13px; opacity: 0.9; }
.pb-value { font-size: 36px; font-weight: bold; margin: 6px 0 10px; }
.section-title { font-size: 15px; font-weight: bold; padding: 16px 16px 8px; }
.product-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; padding: 0 12px; }
.product-card { background: #fff; border-radius: 10px; padding: 10px; }
.pc-name { font-size: 13px; line-height: 18px; height: 36px; margin-top: 8px; }
.pc-points { color: #ff0036; font-size: 16px; font-weight: bold; margin-top: 4px; }
.pc-stock { color: #999; font-size: 11px; margin: 4px 0 8px; }
.record-list { margin: 0 12px; background: #fff; border-radius: 10px; padding: 4px 12px; }
.record-item { display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #f5f5f5; }
.record-item:last-child { border-bottom: none; }
.ri-name { font-size: 14px; }
.ri-time { font-size: 11px; color: #999; margin-top: 2px; }
.ri-cost { color: #ff0036; font-size: 14px; font-weight: bold; }
/* 兑换老虎机动画 (slot-machine spin before reveal) */
.slot-popup { padding: 0 0 18px; overflow: hidden; }
.slot-hero { background: linear-gradient(135deg, #ff0036 0%, #ff5577 50%, #ffb347 100%); color: #fff; padding: 22px 18px 16px; text-align: center; }
.slot-title { font-size: 20px; font-weight: bold; letter-spacing: 0.5px; }
.slot-sub { font-size: 13px; opacity: 0.95; margin-top: 6px; }
.slot-machine { display: flex; gap: 8px; justify-content: center; margin: 20px 16px; background: #222; border-radius: 12px; padding: 10px; box-shadow: inset 0 2px 8px rgba(0,0,0,0.5); }
.slot-col { width: 64px; height: 64px; overflow: hidden; border-radius: 8px; background: #fff; position: relative; }
.slot-symbol { height: 64px; line-height: 64px; text-align: center; font-size: 38px; }
.slot-strip { will-change: transform; }
.slot-strip.slot-spinning { animation: slot-spin 0.12s steps(1) infinite; }
.slot-strip.slot-stopped { transition: transform 0.5s cubic-bezier(0.15, 0.85, 0.25, 1); }
@keyframes slot-spin { 0% { transform: translateY(0); } 100% { transform: translateY(-64px); } }
.slot-reveal { text-align: center; font-size: 16px; font-weight: bold; color: #ff0036; padding: 4px 0 12px; animation: slot-pop 0.4s ease; }
.slot-reveal.reveal-error { color: #999; }
@keyframes slot-pop { 0% { transform: scale(0.6); opacity: 0; } 60% { transform: scale(1.15); } 100% { transform: scale(1); opacity: 1; } }
.slot-actions { padding: 0 18px; }
</style>
