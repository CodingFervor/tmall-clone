<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getGroupBuys, joinGroupBuy } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

onMounted(async () => {
  try { deals.value = await getGroupBuys() } catch (e) { showToast('加载失败') } finally { loading.value = false }
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
  return Math.min(100, Math.round((d.joined / d.required) * 100))
}
// ---- 拼团圆环倒计时 (group-buy ring countdown) ----
// An SVG circle whose stroke depletes (stroke-dashoffset grows) as the deal's
// end_time approaches. The total duration spans created_at → end_time so the
// arc represents elapsed time. Center text shows the remaining HH:MM. The ring
// color steps green (>50%) → orange (<50%) → red (<25%) → gray (expired).
const RING_R = 26          // radius of the countdown ring (viewBox 64x64)
const RING_C = 2 * Math.PI * RING_R // circumference for stroke-dashoffset math
// Full duration of the deal (created_at → end_time) in ms.
function ringTotalMs(d) {
  const start = new Date(d.created_at).getTime()
  const end = new Date(d.end_time).getTime()
  if (isNaN(start) || isNaN(end) || end <= start) return 1
  return end - start
}
// Remaining fraction 0..1 of the deal window (1 = just started, 0 = ended).
function ringFraction(d) {
  const remain = remainMs(d)
  const total = ringTotalMs(d)
  if (total <= 0) return 0
  return Math.max(0, Math.min(1, remain / total))
}
// stroke-dashoffset: 0 = full ring drawn, C = fully depleted (empty).
function ringOffset(d) {
  return RING_C * (1 - ringFraction(d))
}
// Center label: remaining HH:MM (hours may exceed 99 for long windows).
function ringLabel(d) {
  const ms = remainMs(d)
  if (ms <= 0) return '结束'
  const totalMin = Math.floor(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}
// Ring color by remaining-time fraction: green>50% / orange<50% / red<25% / gray expired.
function ringColor(d) {
  if (d.status === 'expired' || remainMs(d) <= 0) return '#bbb'
  const f = ringFraction(d)
  if (f > 0.5) return '#07c160' // green
  if (f > 0.25) return '#ff9800' // orange
  return '#ff0036' // red
}
function ringExpired(d) {
  return d.status === 'expired' || remainMs(d) <= 0
}
// Dicebear avatars: deterministic per deal so re-renders stay stable, 2-4 members, first is the leader.
function memberAvatars(d) {
  const count = Math.min(4, Math.max(2, d.joined || 2))
  const seedBase = Number(d.id) || 1
  const avatars = []
  for (let i = 0; i < count; i++) {
    const seed = `gb${seedBase}-${i}`
    avatars.push(`https://api.dicebear.com/7.x/adventurer/svg?seed=${encodeURIComponent(seed)}`)
  }
  return avatars
}
async function join(d) {
  try {
    const res = await joinGroupBuy(d.id)
    d.joined = res.data.joined
    d.status = res.data.status
    showSuccessToast(res.message)
  } catch (e) {
    showToast(e.response?.data?.error || '参团失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="gb-page">
    <van-nav-bar title="超值拼团" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无拼团活动" />
    <div v-else class="gb-list">
      <div class="banner">👥 超值拼团 · 人多更便宜</div>
      <div v-for="d in deals" :key="d.id" class="gb-card">
        <van-image width="100" height="100" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="gc-info">
          <div class="gc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="gc-price-row">
            <span class="gc-group">¥{{ fmt(d.group_price) }}</span>
            <span class="gc-origin">¥{{ fmt(d.original_price) }}</span>
          </div>
          <div class="gc-members">
            <div class="avatar-stack">
              <img
                v-for="(avatar, i) in memberAvatars(d)"
                :key="i"
                :src="avatar"
                class="member-avatar"
                :class="{ leader: i === 0 }"
                :style="{ left: (i * 16) + 'px', zIndex: memberAvatars(d).length - i }"
                alt="拼团成员"
              />
            </div>
            <span class="members-text">{{ d.joined || memberAvatars(d).length }}人已参团</span>
          </div>
          <div class="gc-progress">
            <div class="gp-bar"><div class="gp-fill" :style="{ width: progress(d) + '%' }"></div></div>
            <span class="gp-text">{{ d.joined }}/{{ d.required }}人</span>
          </div>
          <div class="gc-bottom">
            <!-- 拼团圆环倒计时 (ring countdown): depleting arc + HH:MM center -->
            <div class="gc-ring" :class="{ expired: ringExpired(d) }">
              <svg width="64" height="64" viewBox="0 0 64 64">
                <circle cx="32" cy="32" :r="RING_R" fill="none" stroke="#f0f0f0" stroke-width="5" />
                <circle
                  cx="32" cy="32" :r="RING_R" fill="none"
                  :stroke="ringColor(d)" stroke-width="5" stroke-linecap="round"
                  :stroke-dasharray="RING_C"
                  :stroke-dashoffset="ringOffset(d)"
                  :transform="'rotate(-90 32 32)'"
                  class="ring-progress"
                />
              </svg>
              <span class="ring-label" :style="{ color: ringColor(d) }">{{ ringLabel(d) }}</span>
            </div>
            <van-button size="small" type="danger" round :disabled="d.status !== 'active'" @click="join(d)">
              {{ d.status === 'success' ? '已成团' : '去拼团' }}
            </van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gb-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.banner { background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.gb-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.gc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.gc-name { font-size: 14px; line-height: 20px; flex: 1; }
.gc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.gc-group { color: #ff0036; font-size: 20px; font-weight: bold; }
.gc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
/* Member avatar stack: overlapping circular avatars with a gold border on the leader. */
.gc-members { display: flex; align-items: center; margin-bottom: 6px; }
.avatar-stack { position: relative; height: 30px; width: calc(30px + 16px * 3); }
.member-avatar { position: absolute; top: 0; width: 30px; height: 30px; border-radius: 50%; border: 2px solid #fff; background: #eee; object-fit: cover; box-shadow: 0 1px 3px rgba(0,0,0,0.15); }
.member-avatar.leader { border-color: #ffc107; box-shadow: 0 0 0 1px #ffc107, 0 1px 3px rgba(0,0,0,0.15); }
.members-text { margin-left: 12px; font-size: 12px; color: #666; }
.gc-progress { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.gp-bar { flex: 1; height: 12px; background: #ffe0e0; border-radius: 6px; overflow: hidden; }
.gp-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #ff0036); transition: width 0.3s; }
.gp-text { font-size: 11px; color: #ff0036; white-space: nowrap; }
.gc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
/* 拼团圆环倒计时 (group-buy ring countdown) */
.gc-ring { position: relative; width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; }
.gc-ring .ring-progress { transition: stroke-dashoffset 1s linear, stroke 0.4s; }
.gc-ring .ring-label { position: absolute; font-size: 13px; font-weight: bold; font-variant-numeric: tabular-nums; letter-spacing: -0.5px; }
.gc-ring.expired { opacity: 0.7; }
</style>
