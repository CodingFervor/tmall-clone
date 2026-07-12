<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getCheckInStatus, getOrders, listFavorites, getMyCoupons } from '../api'

const router = useRouter()
const user = ref(null)
const cartCount = ref(0)
const loggedIn = ref(false)
const growth = ref(0) // 成长值 (derived from accumulated points)

// ---- 深色模式 (dark mode) ----
// Persisted in localStorage under 'tm_dark_mode'. The class is applied to the
// root <html> element so dark overrides cascade across the whole app.
const DARK_KEY = 'tm_dark_mode'
const darkMode = ref(false)
function applyDarkClass(on) {
  const root = document.documentElement
  if (on) root.classList.add('dark-mode')
  else root.classList.remove('dark-mode')
}
function toggleDark() {
  darkMode.value = !darkMode.value
  localStorage.setItem(DARK_KEY, String(darkMode.value))
  applyDarkClass(darkMode.value)
  showToast(darkMode.value ? '已切换深色模式' : '已切换浅色模式')
}
// Initialize from storage on mount (so the toggle UI reflects saved state).
darkMode.value = localStorage.getItem(DARK_KEY) === 'true'
applyDarkClass(darkMode.value)

// ---- 个人速览 (quick stats) ----
// 订单数 / 收藏数 / 优惠券数 / 积分数 — fetched from existing APIs.
const stats = ref({ orders: 0, favorites: 0, coupons: 0, points: 0 })
// 连续签到天数 (current check-in streak), used by the carbon-footprint card.
const streak = ref(0)
// Cached order list (kept for the spending-overview computation).
const allOrders = ref([])
async function loadStats() {
  if (!loggedIn.value) { stats.value = { orders: 0, favorites: 0, coupons: 0, points: 0 }; allOrders.value = []; return }
  try {
    const [orders, favs, coupons] = await Promise.all([
      getOrders().catch(() => []),
      listFavorites().catch(() => []),
      getMyCoupons().catch(() => []),
    ])
    allOrders.value = orders || []
    stats.value = {
      orders: allOrders.value.length,
      favorites: (favs || []).length,
      coupons: (coupons || []).length,
      points: growth.value,
    }
  } catch (_) { /* stats optional */ }
}

// ---- 消费概览 (spending stats) ----
// Derived from the cached order list. Total spent = sum of paid+ order totals
// (exclude cancelled/pending-unpaid). Average = total / paid order count. The
// 3-bar monthly chart compares the last three calendar months' spend.
const SPENDING_STATUSES = ['paid', 'shipped', 'completed']
function monthKey(d) {
  return d.getFullYear() * 100 + (d.getMonth() + 1) // YYYYMM
}
function monthLabel(mk) {
  const m = mk % 100
  return m + '月'
}
const spendingStats = computed(() => {
  const paid = allOrders.value.filter((o) => SPENDING_STATUSES.includes(o.status))
  const total = paid.reduce((s, o) => s + (Number(o.total) || 0), 0)
  const count = paid.length
  const avg = count ? total / count : 0
  // Bucket spend by YYYYMM for the last 3 months (including the current month).
  const now = new Date()
  const buckets = {}
  const keys = []
  for (let i = 2; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const k = monthKey(d)
    buckets[k] = 0
    keys.push(k)
  }
  for (const o of paid) {
    if (!o.created_at) continue
    const d = new Date(o.created_at)
    if (isNaN(d.getTime())) continue
    const k = monthKey(d)
    if (k in buckets) buckets[k] += Number(o.total) || 0
  }
  const maxMonth = Math.max(1, ...keys.map((k) => buckets[k]))
  const bars = keys.map((k) => ({
    label: monthLabel(k),
    value: buckets[k],
    height: Math.round((buckets[k] / maxMonth) * 100),
    current: k === monthKey(now),
  }))
  return { total, count, avg, bars }
})

// ---- 碳足迹 (carbon footprint) ----
// Green points are earned from eco-actions: the current check-in streak and
// completed orders. Each point converts into a fixed CO₂ reduction, and the
// reduction is also expressed as an equivalent number of trees planted.
const CARBON_PER_POINT_KG = 0.5   // 每"绿色积分"约减少 0.5kg 碳排放
const CO2_PER_TREE_KG = 20         // 一棵树年固碳约 20kg
const greenPoints = computed(() => (streak.value * 5) + (stats.value.orders * 3))
const carbonReducedKg = computed(() => Math.round(greenPoints.value * CARBON_PER_POINT_KG * 10) / 10)
const treesEquivalent = computed(() => Math.max(0, Math.floor(carbonReducedKg.value / CO2_PER_TREE_KG)))

// ---- 会员等级体系 (member growth levels) ----
// thresholds: cumulative growth points required to reach each level
const LEVELS = [
  { name: '白银', min: 0,     color: 'silver',   cssVar: '--lv-silver' },
  { name: '黄金', min: 1000,  color: 'gold',     cssVar: '--lv-gold' },
  { name: '铂金', min: 5000,  color: 'cyan',     cssVar: '--lv-cyan' },
  { name: '钻石', min: 20000, color: 'purple',   cssVar: '--lv-purple' },
  { name: '王者', min: 50000, color: 'gradient', cssVar: '--lv-gradient' },
]

const currentLevel = computed(() => {
  let lv = LEVELS[0]
  for (const l of LEVELS) if (growth.value >= l.min) lv = l
  return lv
})
const nextLevel = computed(() => {
  const idx = LEVELS.findIndex((l) => l.name === currentLevel.value.name)
  return idx < LEVELS.length - 1 ? LEVELS[idx + 1] : null
})
// progress (%) toward next level
const progressPct = computed(() => {
  if (!nextLevel.value) return 100 // max level
  const span = nextLevel.value.min - currentLevel.value.min
  const done = growth.value - currentLevel.value.min
  return Math.max(0, Math.min(100, Math.round((done / span) * 100)))
})
const pointsToNext = computed(() => {
  if (!nextLevel.value) return 0
  return Math.max(0, nextLevel.value.min - growth.value)
})

async function load() {
  loggedIn.value = !!localStorage.getItem('tm_token')
  if (!loggedIn.value) {
    growth.value = 0
    streak.value = 0
    return
  }
  try {
    const res = await getProfile(); user.value = res.user; cartCount.value = res.cart_count || 0
    try {
      const st = await getCheckInStatus()
      growth.value = st.total_points || 0
      streak.value = (st.last && st.last.streak) || 0
    } catch (_) { streak.value = 0 /* points optional */ }
    await loadStats()
  } catch (e) { loggedIn.value = false }
}
onMounted(load); onActivated(load)
function logout() { localStorage.removeItem('tm_token'); localStorage.removeItem('tm_user'); loggedIn.value = false; user.value = null; growth.value = 0; streak.value = 0; allOrders.value = []; stats.value = { orders: 0, favorites: 0, coupons: 0, points: 0 }; showToast('已退出登录') }

// ---- 消息红点 (notification dots) ----
// Each tracked cell records its last-visit timestamp in localStorage. A red dot
// shows when more than an hour has elapsed since the last visit. Visiting the
// page clears the dot by updating the timestamp to now.
const NOTIF_KEY = 'tm_notif_visits'
const NOTIF_INTERVAL = 60 * 60 * 1000 // 1 hour
// Trigger a re-render of dots when visits change so they clear immediately.
const notifTick = ref(0)

function notifVisits() {
  try { return JSON.parse(localStorage.getItem(NOTIF_KEY) || '{}') } catch (_) { return {} }
}
function setNotifVisits(v) {
  try { localStorage.setItem(NOTIF_KEY, JSON.stringify(v)) } catch (_) {}
}
// true when the cell should show a red dot (last visit older than 1h or never).
function hasNotif(key) {
  void notifTick.value // depend on tick for reactivity
  const visits = notifVisits()
  const last = visits[key]
  if (!last) return true // never visited
  return (Date.now() - last) > NOTIF_INTERVAL
}
// Navigate to a route and record the visit so the dot clears.
function goWithVisit(route, key) {
  const visits = notifVisits()
  visits[key] = Date.now()
  setNotifVisits(visits)
  notifTick.value++
  router.push(route)
}
function fmt(n) { return Number(n || 0).toFixed(2) }

// ---- 生日卡片 (birthday card) ----
// Reads the user's birthday from localStorage 'tm_birthday' (format MM-DD,
// MM/DD, or YYYY-MM-DD — we only use the month & day). When it matches today,
// a festive "🎂生日快乐" card is shown. `setBirthday` is a demo helper for users
// who haven't set one yet so they can see the effect.
const BIRTHDAY_KEY = 'tm_birthday'
const isBirthday = computed(() => {
  const raw = localStorage.getItem(BIRTHDAY_KEY)
  if (!raw) return false
  // Accept MM-DD, MM/DD, and YYYY-MM-DD; take the trailing MM-DD.
  const parts = raw.trim().split(/[-/]/)
  const md = parts.length >= 2 ? parts.slice(-2) : parts
  const m = Number(md[0])
  const d = Number(md[1])
  if (!m || !d) return false
  const now = new Date()
  return now.getMonth() + 1 === m && now.getDate() === d
})
function setBirthday() {
  const raw = window.prompt('请输入你的生日（格式：MM-DD，例如 06-15）')
  if (!raw || !raw.trim()) return
  const val = raw.trim()
  // Basic validation: a MM-DD-ish or YYYY-MM-DD string.
  const parts = val.split(/[-/]/)
  const md = parts.length >= 2 ? parts.slice(-2) : parts
  const m = Number(md[0])
  const d = Number(md[1])
  if (!m || !d || m < 1 || m > 12 || d < 1 || d > 31) { showToast('格式不正确，请使用 MM-DD'); return }
  localStorage.setItem(BIRTHDAY_KEY, val)
  // Force reactivity: computed reads localStorage at evaluation; bump by toggling
  // growth (a cheap re-render trigger) is overkill, so we re-read via a tick by
  // nudging notifTick which the template also depends on.
  notifTick.value++
  showToast(isBirthday.value ? '🎂 生日快乐！' : '已保存，生日当天将收到祝福')
}
</script>

<template>
  <div class="mine-page">
    <!-- 生日卡片 (birthday card): shown when tm_birthday matches today -->
    <div v-if="isBirthday" class="birthday-card">
      <div class="bday-content">
        <span class="bday-cake">🎂</span>
        <div class="bday-text">
          <div class="bday-title">生日快乐！</div>
          <div class="bday-sub">天猫祝你生日快乐，愿你拥有美好的一天 🎁</div>
        </div>
        <span class="bday-confetti">🎉</span>
      </div>
      <div class="bday-gift">专属生日礼包已到账，立即去积分商城看看 ›</div>
    </div>
    <div class="mine-header">
      <div class="header-row">
        <div v-if="loggedIn && user" class="user-info">
          <van-image round width="60" height="60" :src="user.avatar || 'https://via.placeholder.com/60'" />
          <div class="u-text"><div class="u-name">{{ user.nickname || user.username }}</div><div class="u-id">用户名: {{ user.username }}</div></div>
        </div>
        <div v-else class="user-info" @click="router.push('/login')">
          <van-image round width="60" height="60" src="https://via.placeholder.com/60" />
          <div class="u-text"><div class="u-name">登录/注册</div><div class="u-id">理想生活上天猫</div></div>
        </div>
        <!-- 深色模式开关 (dark mode toggle) -->
        <div class="dark-toggle" @click="toggleDark">
          <span class="dt-icon">{{ darkMode ? '🌙' : '☀️' }}</span>
          <span class="dt-label">{{ darkMode ? '深色' : '浅色' }}</span>
        </div>
      </div>
    </div>
    <!-- 个人速览 (quick stats) — 订单数/收藏数/优惠券数/积分数 -->
    <div class="stats-row">
      <div class="stat-item" @click="router.push('/orders')">
        <div class="stat-num">{{ stats.orders }}</div>
        <div class="stat-label">订单数</div>
      </div>
      <div class="stat-item" @click="router.push('/favorites')">
        <div class="stat-num">{{ stats.favorites }}</div>
        <div class="stat-label">收藏数</div>
      </div>
      <div class="stat-item" @click="router.push('/coupons')">
        <div class="stat-num">{{ stats.coupons }}</div>
        <div class="stat-label">优惠券数</div>
      </div>
      <div class="stat-item" @click="router.push('/points-shop')">
        <div class="stat-num">{{ stats.points }}</div>
        <div class="stat-label">积分数</div>
      </div>
    </div>
    <div class="growth-section">
      <div class="growth-head">
        <span class="growth-title">成长值</span>
        <span class="growth-value">{{ growth }}</span>
      </div>

      <!-- current level badge (prominent) -->
      <div class="badge-row">
        <div v-if="loggedIn" class="lv-badge" :class="'lv-' + currentLevel.color">
          <span class="lv-star">★</span>
          <span class="lv-name">{{ currentLevel.name }}会员</span>
        </div>
        <div v-else class="lv-badge lv-silver">
          <span class="lv-star">★</span>
          <span class="lv-name">登录查看等级</span>
        </div>
      </div>

      <!-- progress bar to next level -->
      <div class="bar-wrap">
        <div class="bar-track">
          <div class="bar-fill" :style="{ width: progressPct + '%' }"></div>
        </div>
        <div class="bar-labels">
          <span>{{ currentLevel.name }}</span>
          <span v-if="nextLevel">{{ nextLevel.name }}</span>
          <span v-else>已达最高等级</span>
        </div>
      </div>

      <div class="growth-foot" v-if="loggedIn">
        距下一等级还需 <b>{{ pointsToNext }}</b> 积分
        <span v-if="!nextLevel" class="max-tip">（已是最高等级）</span>
      </div>
      <div class="growth-foot" v-else>
        <span class="login-tip" @click="router.push('/login')">登录后查看成长进度 ›</span>
      </div>

      <!-- all level badges overview -->
      <div class="badges-overview">
        <div
          v-for="l in LEVELS" :key="l.name"
          class="ov-badge"
          :class="['lv-' + l.color, { active: l.name === currentLevel.name }]"
        >
          <div class="ov-icon">★</div>
          <div class="ov-name">{{ l.name }}</div>
          <div class="ov-min">{{ l.min }}</div>
        </div>
      </div>
    </div>

    <!-- 碳足迹 (carbon footprint) — green-themed card driven by check-in streak + orders -->
    <div class="carbon-card" :class="{ 'cc-locked': !loggedIn }">
      <div class="cc-head">
        <span class="cc-icon">🌱</span>
        <span class="cc-title">碳足迹</span>
        <span class="cc-points">绿色积分 <b>{{ greenPoints }}</b></span>
      </div>
      <div class="cc-body">
        <div class="cc-stat">
          <div class="cc-num">减少 {{ carbonReducedKg }} <small>kg碳排放</small></div>
          <div class="cc-sub">相当于种了 {{ treesEquivalent }} 棵树 🌳</div>
        </div>
        <div class="cc-breakdown">
          <span class="cb-item">签到 {{ streak }} 天 × 5</span>
          <span class="cb-sep">·</span>
          <span class="cb-item">订单 {{ stats.orders }} 笔 × 3</span>
        </div>
        <div v-if="!loggedIn" class="cc-login-tip" @click="router.push('/login')">登录后查看你的环保贡献 ›</div>
      </div>
    </div>

    <!-- 消费概览 (spending stats) — total spent + average + 3-bar monthly chart -->
    <div class="spend-card" :class="{ 'sp-locked': !loggedIn }">
      <div class="sp-head">
        <span class="sp-icon">💰</span>
        <span class="sp-title">消费概览</span>
        <span class="sp-count">已购 {{ spendingStats.count }} 笔</span>
      </div>
      <template v-if="loggedIn">
        <div class="sp-summary">
          <div class="sp-stat">
            <div class="sp-num">¥{{ fmt(spendingStats.total) }}</div>
            <div class="sp-label">累计消费</div>
          </div>
          <div class="sp-divider"></div>
          <div class="sp-stat">
            <div class="sp-num">¥{{ fmt(spendingStats.avg) }}</div>
            <div class="sp-label">笔均消费</div>
          </div>
        </div>
        <!-- 近三月消费柱状图 (3-bar monthly chart) -->
        <div class="sp-chart">
          <div v-for="(b, i) in spendingStats.bars" :key="i" class="sp-bar-col">
            <div class="sp-bar-track">
              <div class="sp-bar" :class="{ 'sp-bar-cur': b.current }" :style="{ height: b.height + '%' }"></div>
            </div>
            <span class="sp-bar-val">¥{{ fmt(b.value) }}</span>
            <span class="sp-bar-label">{{ b.label }}</span>
          </div>
        </div>
      </template>
      <div v-else class="sp-login-tip" @click="router.push('/login')">登录后查看消费明细 ›</div>
    </div>

    <van-cell-group inset title="我的订单">
      <div class="order-entries">
        <div class="oe-item" @click="router.push('/orders')"><van-icon name="balance-pay-o" size="28" color="#ffa300" /><span>待付款</span></div>
        <div class="oe-item" @click="router.push('/orders')"><van-icon name="logistics" size="28" color="#07c160" /><span>待发货</span></div>
        <div class="oe-item" @click="router.push('/orders')"><van-icon name="gift-o" size="28" color="#ff0036" /><span>待收货</span></div>
        <div class="oe-item" @click="router.push('/orders')"><van-icon name="comment-o" size="28" color="#1989fa" /><span>待评价</span></div>
      </div>
    </van-cell-group>
    <van-cell-group inset title="常用功能">
      <van-cell title="购物车" :value="cartCount + '件'" is-link class="notif-cell" :class="{ 'notif-on': hasNotif('cart') }" @click="goWithVisit('/cart', 'cart')" icon="cart-o" />
      <van-cell title="我的订单" is-link class="notif-cell" :class="{ 'notif-on': hasNotif('orders') }" @click="goWithVisit('/orders', 'orders')" icon="orders-o" />
      <van-cell title="我的收藏" is-link @click="router.push('/favorites')" icon="star-o" />
      <van-cell title="浏览历史" is-link @click="router.push('/history')" icon="clock-o" />
      <van-cell title="每日签到" is-link class="notif-cell" :class="{ 'notif-on': hasNotif('checkin') }" @click="goWithVisit('/checkin', 'checkin')" icon="calendar-o" />
      <van-cell title="积分商城" is-link @click="router.push('/points-shop')" icon="gold-coin-o" />
      <van-cell title="积分抽奖" is-link @click="router.push('/lottery')" icon="gem-o" />
      <van-cell title="超值拼团" is-link @click="router.push('/group-buy')" icon="friends-o" />
      <van-cell title="超值套餐" is-link @click="router.push('/bundles')" icon="gift-o" />
      <van-cell title="礼品卡" is-link @click="router.push('/gift-card')" icon="card" />
      <van-cell title="预售专区" is-link @click="router.push('/presale')" icon="underway-o" />
      <van-cell title="售后服务" is-link @click="router.push('/refunds')" icon="after-sale" />
      <van-cell title="优惠券" is-link class="notif-cell" :class="{ 'notif-on': hasNotif('coupons') }" @click="goWithVisit('/coupons', 'coupons')" icon="coupon-o" />
      <van-cell title="收货地址" is-link icon="location-o" @click="router.push('/addresses')" />
      <van-cell title="编辑资料" is-link icon="edit" @click="router.push('/profile')" />
      <van-cell title="我的生日" is-link icon="gift-o" @click="setBirthday" />
      <van-cell title="我的关注" is-link icon="like-o" @click="router.push('/brands')" />
      <van-cell title="88VIP" is-link icon="diamond-o" @click="showToast('演示功能')" />
      <van-cell title="管理后台" is-link @click="router.push('/admin')" icon="setting-o" />
    </van-cell-group>
    <div v-if="loggedIn" style="margin: 20px"><van-button block plain type="danger" @click="logout">退出登录</van-button></div>
  </div>
</template>

<style scoped>
.mine-page { min-height: 100vh; padding-bottom: 20px; }
/* 生日卡片 (birthday card) */
.birthday-card { margin: 8px; border-radius: 12px; overflow: hidden; background: linear-gradient(135deg, #ff0036 0%, #ff5577 50%, #ffb347 100%); color: #fff; box-shadow: 0 4px 14px rgba(255, 0, 54, 0.3); }
.bday-content { display: flex; align-items: center; gap: 12px; padding: 16px; }
.bday-cake { font-size: 40px; line-height: 1; animation: bday-bounce 1.4s ease-in-out infinite; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2)); }
@keyframes bday-bounce { 0%, 100% { transform: translateY(0) rotate(-3deg); } 50% { transform: translateY(-6px) rotate(3deg); } }
.bday-text { flex: 1; }
.bday-title { font-size: 20px; font-weight: bold; letter-spacing: 1px; }
.bday-sub { font-size: 12px; opacity: 0.95; margin-top: 4px; line-height: 18px; }
.bday-confetti { font-size: 24px; animation: bday-spin 2s linear infinite; }
@keyframes bday-spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.bday-gift { padding: 10px 16px; background: rgba(0, 0, 0, 0.12); font-size: 12px; line-height: 18px; }
.mine-header { background: linear-gradient(135deg, #ff0036, #ff5577); padding: 30px 20px; color: #fff; }
.header-row { display: flex; align-items: center; justify-content: space-between; }
.user-info { display: flex; align-items: center; gap: 14px; }
.u-name { font-size: 18px; font-weight: bold; }
.u-id { font-size: 12px; opacity: 0.8; margin-top: 4px; }
/* 深色模式开关 (dark mode toggle) */
.dark-toggle { display: flex; align-items: center; gap: 4px; padding: 6px 12px; background: rgba(255,255,255,0.2); border-radius: 999px; cursor: pointer; font-size: 13px; }
.dt-icon { font-size: 16px; }
/* 个人速览 (quick stats) */
.stats-row { display: flex; background: #fff; margin: -12px 12px 12px; border-radius: 12px; padding: 16px 0; box-shadow: 0 2px 12px rgba(0,0,0,0.06); position: relative; z-index: 2; }
.stat-item { flex: 1; text-align: center; cursor: pointer; }
.stat-num { font-size: 20px; font-weight: bold; color: #ff0036; }
.stat-label { font-size: 12px; color: #666; margin-top: 4px; }
.order-entries { display: flex; padding: 16px 0; }
.oe-item { flex: 1; text-align: center; font-size: 12px; color: #666; }
.oe-item span { display: block; margin-top: 4px; }

/* 消息红点 (notification dots) */
/* Render an 8px red dot in the top-right corner of a cell when its
   notif-on class is present. Uses the free ::before pseudo-element
   (the bottom hairline is on ::after, so they don't clash). */
.notif-cell.notif-on :deep(::before) {
  content: '';
  position: absolute;
  top: 50%;
  right: 26px;
  transform: translateY(-50%);
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ff0036;
  box-shadow: 0 0 0 2px rgba(255, 0, 54, 0.18);
  z-index: 3;
}
/* ---- 会员成长值 (member growth chart) ---- */
.growth-section { margin: 12px; background: #fff; border-radius: 12px; padding: 16px; }
.growth-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 12px; }
.growth-title { font-size: 16px; font-weight: bold; color: #333; }
.growth-value { font-size: 22px; font-weight: bold; color: #ff0036; font-variant-numeric: tabular-nums; }

.badge-row { display: flex; justify-content: center; margin-bottom: 14px; }
.lv-badge { display: inline-flex; align-items: center; gap: 6px; padding: 8px 18px; border-radius: 20px; color: #fff; font-size: 15px; font-weight: bold; box-shadow: 0 2px 8px rgba(0,0,0,0.12); transform: scale(1.05); }
.lv-star { font-size: 16px; }

.lv-silver { background: linear-gradient(135deg, #b0b0b0, #8a8a8a); }
.lv-gold { background: linear-gradient(135deg, #ffd700, #ffb300); }
.lv-cyan { background: linear-gradient(135deg, #22d3ee, #0891b2); }
.lv-purple { background: linear-gradient(135deg, #a855f7, #7e22ce); }
.lv-gradient { background: linear-gradient(135deg, #ff0036, #a855f7, #22d3ee, #ffd700); background-size: 200% 200%; animation: shimmer 3s ease infinite; }
@keyframes shimmer { 0% { background-position: 0% 50%; } 50% { background-position: 100% 50%; } 100% { background-position: 0% 50%; } }

.bar-wrap { margin-bottom: 10px; }
.bar-track { height: 10px; background: #f0f0f0; border-radius: 5px; overflow: hidden; }
.bar-fill { height: 100%; background: linear-gradient(90deg, #ffd700, #ff0036); border-radius: 5px; transition: width 0.4s ease; }
.bar-labels { display: flex; justify-content: space-between; font-size: 11px; color: #999; margin-top: 4px; }

.growth-foot { font-size: 13px; color: #666; text-align: center; }
.growth-foot b { color: #ff0036; font-size: 15px; }
.max-tip { color: #999; font-size: 12px; }
.login-tip { color: #ff0036; cursor: pointer; }

.badges-overview { display: flex; justify-content: space-between; gap: 6px; margin-top: 14px; padding-top: 14px; border-top: 1px dashed #eee; }
.ov-badge { flex: 1; text-align: center; padding: 8px 2px; border-radius: 8px; opacity: 0.55; transition: opacity 0.2s; }
.ov-badge.active { opacity: 1; }
.ov-icon { width: 34px; height: 34px; line-height: 34px; margin: 0 auto 4px; border-radius: 50%; color: #fff; font-size: 16px; }
.ov-badge.lv-silver .ov-icon { background: linear-gradient(135deg, #b0b0b0, #8a8a8a); }
.ov-badge.lv-gold .ov-icon { background: linear-gradient(135deg, #ffd700, #ffb300); }
.ov-badge.lv-cyan .ov-icon { background: linear-gradient(135deg, #22d3ee, #0891b2); }
.ov-badge.lv-purple .ov-icon { background: linear-gradient(135deg, #a855f7, #7e22ce); }
.ov-badge.lv-gradient .ov-icon { background: linear-gradient(135deg, #ff0036, #a855f7, #22d3ee, #ffd700); }
.ov-name { font-size: 12px; color: #333; }
.ov-min { font-size: 10px; color: #999; }

/* 碳足迹 (carbon footprint) — green-themed card */
.carbon-card { margin: 12px; border-radius: 12px; padding: 16px; color: #fff; background: linear-gradient(135deg, #07c160 0%, #16a34a 60%, #15803d 100%); box-shadow: 0 4px 14px rgba(7, 193, 96, 0.3); overflow: hidden; position: relative; }
.carbon-card::after { content: '🌳'; position: absolute; right: -10px; bottom: -18px; font-size: 90px; opacity: 0.12; pointer-events: none; }
.cc-head { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.cc-icon { font-size: 20px; }
.cc-title { font-size: 16px; font-weight: bold; flex: 1; }
.cc-points { font-size: 12px; background: rgba(255,255,255,0.22); padding: 4px 10px; border-radius: 999px; }
.cc-points b { font-size: 14px; }
.cc-body { position: relative; z-index: 1; }
.cc-stat { margin-bottom: 10px; }
.cc-num { font-size: 22px; font-weight: bold; line-height: 1.3; }
.cc-num small { font-size: 13px; font-weight: normal; opacity: 0.9; }
.cc-sub { font-size: 13px; opacity: 0.95; margin-top: 4px; }
.cc-breakdown { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; font-size: 11px; opacity: 0.9; padding-top: 8px; border-top: 1px solid rgba(255,255,255,0.2); }
.cb-sep { opacity: 0.6; }
.cc-login-tip { font-size: 13px; margin-top: 8px; text-decoration: underline; cursor: pointer; }
.carbon-card.cc-locked { opacity: 0.85; }

/* 深色模式下的个人页样式 (scoped dark overrides) */
:global(html.dark-mode) .growth-section { background: #2a2a2a; }
:global(html.dark-mode) .growth-title { color: #fff; }
:global(html.dark-mode) .bar-track { background: #3a3a3a; }
:global(html.dark-mode) .bar-labels,
:global(html.dark-mode) .growth-foot { color: #aaa; }
:global(html.dark-mode) .badges-overview { border-top-color: #444; }
:global(html.dark-mode) .ov-name { color: #ddd; }
:global(html.dark-mode) .ov-min { color: #888; }
:global(html.dark-mode) .stats-row { background: #2a2a2a; }
:global(html.dark-mode) .stat-label { color: #aaa; }
</style>
