<script setup>
import { ref, onMounted, onActivated, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCoupons, claimCoupon, getMyCoupons } from '../api'
const router = useRouter()
const coupons = ref([])
const myCoupons = ref([])
const loading = ref(true)
const loggedIn = ref(false)
async function load() {
  loggedIn.value = !!localStorage.getItem('tm_token')
  loading.value = true
  try {
    coupons.value = await getCoupons()
    coupons.value.forEach((c) => { c._endTs = parseEnd(c.end_date) })
    if (loggedIn.value) myCoupons.value = await getMyCoupons()
    myCoupons.value.forEach((mc) => { mc._endTs = parseEnd(mc.coupon && mc.coupon.end_date) })
  } catch (e) { showToast('加载失败') } finally { loading.value = false }
}
onMounted(load); onActivated(load)
async function claim(c) {
  if (!loggedIn.value) { router.push('/login'); return }
  try { await claimCoupon(c.id); c.is_claimed = true; myCoupons.value = await getMyCoupons(); showSuccessToast('领取成功') }
  catch (e) { showToast(e.response?.data?.error || '领取失败') }
}
function couponValue(c) { return c.coupon_type === 'discount' ? (c.value * 10).toFixed(1) + '折' : '¥' + c.value }
async function shareCoupon(c) {
  const link = window.location.origin + '/#/coupons?claim=' + c.id
  try { await navigator.clipboard.writeText(link); showSuccessToast('分享链接已复制') }
  catch (e) {
    const ta = document.createElement('textarea'); ta.value = link; document.body.appendChild(ta); ta.select()
    try { document.execCommand('copy'); showSuccessToast('分享链接已复制') } catch { showToast('复制失败') } finally { document.body.removeChild(ta) }
  }
}
// Coupon countdown (优惠券倒计时): a live ticking `now` ref drives per-coupon
// remaining-time computation against each coupon's end_date. Thresholds:
//   <24h → red blinking "即将过期 HH:MM:SS"
//   <3d  → orange "X天后过期"
//   else → gray remaining days.
// Expired coupons show a gray "已过期".
const now = ref(Date.now())
let timer = null
function startTimer() {
  stopTimer()
  timer = setInterval(() => { now.value = Date.now() }, 1000)
}
function stopTimer() { if (timer) { clearInterval(timer); timer = null } }
onMounted(startTimer)
onUnmounted(stopTimer)
// Parse a coupon's end_date into a timestamp. Accepts "YYYY-MM-DD" (treated as
// end of that day) and full ISO strings. Returns NaN when unparseable.
function parseEnd(endDate) {
  if (!endDate) return NaN
  const s = String(endDate).trim()
  if (/^\d{4}-\d{1,2}-\d{1,2}$/.test(s)) {
    const d = new Date(s + 'T23:59:59')
    return d.getTime()
  }
  return new Date(s).getTime()
}
function countdown(c) {
  const end = c._endTs
  if (!end || isNaN(end)) return null
  const ms = end - now.value
  if (ms <= 0) return { level: 'expired', text: '已过期' }
  const totalSec = Math.floor(ms / 1000)
  const days = Math.floor(totalSec / 86400)
  if (ms < 24 * 60 * 60 * 1000) {
    const h = String(Math.floor(totalSec / 3600)).padStart(2, '0')
    const m = String(Math.floor((totalSec % 3600) / 60)).padStart(2, '0')
    const s = String(totalSec % 60).padStart(2, '0')
    return { level: 'urgent', text: `即将过期 ${h}:${m}:${s}` }
  }
  if (days < 3) return { level: 'soon', text: `${days + 1}天后过期` }
  return { level: 'normal', text: `剩余${days + 1}天` }
}
</script>
<template>
  <div class="coupons-page">
    <van-nav-bar title="领券中心" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else>
      <div class="section-head">可领取的优惠券</div>
      <div v-for="c in coupons" :key="c.id" class="coupon-card" :class="{ claimed: c.is_claimed }">
        <div class="cc-left"><div class="cc-value">{{ couponValue(c) }}</div><div class="cc-threshold" v-if="c.threshold > 0">满{{ c.threshold }}元可用</div><div class="cc-threshold" v-else>无门槛</div></div>
        <div class="cc-right"><div class="cc-title">{{ c.title }}</div><div class="cc-date">{{ c.end_date }} 到期</div><div v-if="countdown(c)" class="cc-countdown" :class="['cd-' + countdown(c).level, { blink: countdown(c).level === 'urgent' }]">{{ countdown(c).text }}</div><div class="cc-actions"><van-button v-if="!c.is_claimed" size="small" type="danger" round @click="claim(c)">立即领取</van-button><van-tag v-else type="success">已领取</van-tag><van-button size="small" plain round @click="shareCoupon(c)">分享</van-button></div></div>
      </div>
      <template v-if="loggedIn && myCoupons.length">
        <div class="section-head" style="margin-top: 16px">我的优惠券</div>
        <div v-for="mc in myCoupons" :key="mc.id" class="coupon-card" :class="{ used: mc.is_used }">
          <div class="cc-left"><div class="cc-value">{{ mc.coupon ? couponValue(mc.coupon) : '' }}</div><div class="cc-threshold" v-if="mc.coupon && mc.coupon.threshold > 0">满{{ mc.coupon.threshold }}元可用</div></div>
          <div class="cc-right"><div class="cc-title">{{ mc.coupon ? mc.coupon.title : '' }}</div><div class="cc-date">{{ mc.coupon ? mc.coupon.end_date : '' }} 到期</div><div v-if="!mc.is_used && countdown(mc)" class="cc-countdown" :class="['cd-' + countdown(mc).level, { blink: countdown(mc).level === 'urgent' }]">{{ countdown(mc).text }}</div><van-tag :type="mc.is_used ? 'default' : 'success'">{{ mc.is_used ? '已使用' : '未使用' }}</van-tag></div>
        </div>
      </template>
    </div>
  </div>
</template>
<style scoped>
.coupons-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.section-head { padding: 12px 16px; font-size: 14px; color: #666; font-weight: bold; }
.coupon-card { display: flex; margin: 8px; border-radius: 8px; overflow: hidden; background: #fff; }
.coupon-card.claimed, .coupon-card.used { opacity: 0.5; }
.cc-left { background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; padding: 16px 12px; text-align: center; width: 100px; display: flex; flex-direction: column; justify-content: center; }
.cc-value { font-size: 22px; font-weight: bold; }
.cc-threshold { font-size: 11px; opacity: 0.9; margin-top: 4px; }
.cc-right { flex: 1; padding: 12px 16px; display: flex; flex-direction: column; justify-content: center; gap: 4px; }
.cc-title { font-size: 14px; font-weight: bold; }
.cc-date { font-size: 12px; color: #999; }
.cc-actions { display: flex; align-items: center; gap: 8px; }
/* Coupon countdown (优惠券倒计时) */
.cc-countdown { font-size: 11px; font-weight: bold; padding: 2px 8px; border-radius: 4px; display: inline-block; width: fit-content; margin-top: 2px; }
.cc-countdown.cd-urgent { color: #ff0036; background: #fff0f2; border: 1px solid #ffd6df; }
.cc-countdown.cd-soon { color: #ff9800; background: #fff7e6; border: 1px solid #ffe0b3; }
.cc-countdown.cd-normal { color: #999; background: #f7f7f7; border: 1px solid #eee; }
.cc-countdown.cd-expired { color: #999; background: #f7f7f7; border: 1px solid #eee; }
.cc-countdown.blink { animation: cd-blink 1s ease-in-out infinite; }
@keyframes cd-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
</style>
