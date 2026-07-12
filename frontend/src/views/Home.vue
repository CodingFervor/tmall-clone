<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProducts, getBrands, getCategories, getProduct } from '../api'

const router = useRouter()
const products = ref([])
const brands = ref([])
const categories = ref([])
const loading = ref(false)
// Tracks whether real content has loaded so it can fade in over the skeleton.
const contentReady = ref(false)
// Fixed skeleton placeholders rendered while loading.
const skeletonProducts = [0, 1, 2]
const skeletonSeckill = [0, 1, 2, 3]
// Ripple feedback for category icons: holds the key being pressed.
const rippleCat = ref(null)

// ---- 限时抢购倒计时 (flash sale countdown to next top-of-hour) ----
const countdown = ref('00:00:00')
const flashing = ref(false) // "正在抢购中!" brief state at 0
let timer = null
let flashTimer = null

function tick() {
  const now = new Date()
  const next = new Date(now)
  next.setMinutes(0, 0, 0)
  next.setHours(now.getHours() + 1) // next top-of-hour
  let ms = next.getTime() - now.getTime()
  if (ms <= 0) {
    // hit the hour: show flashing state, then reset
    if (!flashing.value) {
      flashing.value = true
      countdown.value = '正在抢购中!'
      clearTimeout(flashTimer)
      flashTimer = setTimeout(() => { flashing.value = false }, 1500)
    }
    return
  }
  const h = Math.floor(ms / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  const s = Math.floor((ms % 60000) / 1000)
  countdown.value = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

onMounted(async () => {
  loading.value = true
  try {
    const [pl, brs, cats] = await Promise.all([getProducts({ page: 1, page_size: 20 }), getBrands(), getCategories()])
    products.value = pl.data
    brands.value = (brs || []).slice(0, 8)
    categories.value = (cats || []).slice(0, 10)
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
    // Trigger fade-in of real content on the next frame, after the skeleton is removed.
    requestAnimationFrame(() => { requestAnimationFrame(() => { contentReady.value = true }) })
  }
  tick()
  timer = setInterval(tick, 1000)
  // Surface the price-track floating ball (比价悬浮球) after content loads.
  loadPriceTrack()
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (flashTimer) clearTimeout(flashTimer)
  if (longPressTimer) clearTimeout(longPressTimer)
})
function fmt(n) { return Number(n).toFixed(2) }

// New product (新品限时标签): products created within the last 7 days.
function isNew(p) {
  if (!p || !p.created_at) return false
  const created = new Date(p.created_at)
  if (isNaN(created.getTime())) return false
  return (Date.now() - created.getTime()) <= 7 * 24 * 3600 * 1000
}

// ---- 价格趋势标签 (price trend tags) ----
// 降价: original price more than 10% above the current price.
// 秒杀/正品: derived from the product's flag fields (is_seckill / is_genuine).
function isPriceDrop(p) {
  return !!(p && p.original_price && p.original_price > p.price * 1.1)
}
function isSeckill(p) { return !!(p && p.is_seckill) }
function isGenuine(p) { return !!(p && p.is_genuine) }
// ---- 限量抢购闪烁 (limited stock flash) ----
// Products with fewer than 10 units left surface a red, pulsing "⚡仅剩N件" badge
// to create urgency. Stock of 0 (or missing) is treated as "not low stock" so we
// don't badge out-of-stock items here (they're better handled elsewhere).
function lowStock(p) {
  const s = Number(p && p.stock)
  if (!s || s <= 0) return null
  if (s < 10) return s
  return null
}

// ---- 分类图标美化 (category icon gradients + ripple) ----
// Each category icon gets a soft circular gradient backdrop. A pool of
// festive red/orange/purple tones matching the Tmall brand is cycled.
const catGradients = [
  'linear-gradient(135deg, #ff0036, #ff5a5f)',
  'linear-gradient(135deg, #ff7a18, #ffb347)',
  'linear-gradient(135deg, #a06bff, #c89bff)',
  'linear-gradient(135deg, #ff4d8d, #ff9a9e)',
  'linear-gradient(135deg, #ff9800, #ffc46b)',
]
function catStyle(i) { return { background: catGradients[i % catGradients.length] } }
function onCatTouchStart(c) { rippleCat.value = (c && c.id != null) ? c.id : 'all' }
function onCatTouchEnd() { rippleCat.value = null }

// ---- 热门标签 (hot tags discovery feed) ----
// Clickable chips that route to /search?q=<tag>. Five rotating gradient
// colors cycle across the chips for a colorful, festive look.
const hotTags = ['新品上市', '限时特惠', '品质好物', '夏日必备', '居家优选', '数码达人', '美妆护肤', '食品生鲜']
const tagColors = [
  'linear-gradient(135deg, #ff0036, #ff5a5f)', // red
  'linear-gradient(135deg, #ff7a18, #ffb347)', // orange
  'linear-gradient(135deg, #5b8def, #7ec8ff)', // blue
  'linear-gradient(135deg, #00c9a7, #4cd9b5)', // green
  'linear-gradient(135deg, #a06bff, #c89bff)', // purple
]
function tagStyle(i) {
  return { background: tagColors[i % tagColors.length] }
}
function searchTag(tag) {
  router.push({ name: 'search', query: { q: tag } })
}

// ---- 首页跑马灯 (home promotional marquee) ----
// A continuously scrolling promotional bar rendered below the flash banner.
// The track is duplicated so the CSS translateX animation loops seamlessly.
// Tapping a message shows a toast (tappable toast).
const marqueeMessages = ['🎉满199减15', '📦全国包邮', '⚡限时秒杀', '🎁新人礼包', '💎88VIP折扣']
const marqueeTrack = computed(() => [...marqueeMessages, ...marqueeMessages])
function tapMarquee(msg) {
  showToast(msg.replace(/[🎉📦⚡🎁💎]/g, '').trim())
}

// ---- 比价悬浮球 (price track floating ball) ----
// Shows the last viewed product (saved from ProductDetail) as a 48px floating
// ball. We re-fetch the live price and compare it to the stored price to render
// an ↑/↓ trend indicator. Tap routes to the product detail; a long-press
// (touch hold / mouse hold for 500ms) dismisses it for this session.
const PRICE_TRACK_KEY = 'tm_price_track'
const PRICE_TRACK_DISMISS_KEY = 'tm_price_track_dismissed'
const priceTrack = ref(null)        // { id, name, image, price(stored), ts }
const livePrice = ref(null)         // current price re-fetched from the API
const ballHidden = ref(true)        // whether to show the ball
let longPressTimer = null

// Trend: 1 = price went up, -1 = went down, 0 = unchanged.
const priceTrend = computed(() => {
  if (priceTrack.value == null || livePrice.value == null) return 0
  const diff = Number(livePrice.value) - Number(priceTrack.value.price)
  if (diff > 0) return 1
  if (diff < 0) return -1
  return 0
})

async function loadPriceTrack() {
  // Respect a per-session dismissal.
  if (localStorage.getItem(PRICE_TRACK_DISMISS_KEY) === '1') { ballHidden.value = true; return }
  let raw = null
  try { raw = localStorage.getItem(PRICE_TRACK_KEY) } catch (_) { return }
  if (!raw) { ballHidden.value = true; return }
  let parsed = null
  try { parsed = JSON.parse(raw) } catch (_) { return }
  if (!parsed || !parsed.id) { ballHidden.value = true; return }
  priceTrack.value = parsed
  ballHidden.value = false
  // Fetch the live price for the trend indicator.
  try {
    const res = await getProduct(parsed.id)
    if (res && res.data) {
      livePrice.value = Number(res.data.price)
      // Keep the stored price in sync so the trend reflects the latest snapshot.
      try {
        localStorage.setItem(PRICE_TRACK_KEY, JSON.stringify({
          id: parsed.id, name: parsed.name, image: parsed.image,
          price: Number(res.data.price), ts: Date.now(),
        }))
      } catch (_) { /* ignore */ }
    }
  } catch (_) {
    // Network failure: fall back to the stored price (no trend shown).
    livePrice.value = Number(parsed.price)
  }
}
function displayPrice() {
  return livePrice.value != null ? livePrice.value : (priceTrack.value ? priceTrack.value.price : 0)
}
function tapBall() {
  if (priceTrack.value) router.push('/product/' + priceTrack.value.id)
}
function startLongPress(e) {
  if (longPressTimer) clearTimeout(longPressTimer)
  longPressTimer = setTimeout(() => {
    longPressTimer = null
    dismissBall()
    // Prevent the pending click that follows a touch-hold from navigating.
    if (e && e.preventDefault) e.preventDefault()
  }, 500)
}
function cancelLongPress() {
  if (longPressTimer) { clearTimeout(longPressTimer); longPressTimer = null }
}
function dismissBall() {
  ballHidden.value = true
  try { localStorage.setItem(PRICE_TRACK_DISMISS_KEY, '1') } catch (_) {}
  showToast('已移除比价悬浮球')
}

// ---- 天气小卡片 (home weather widget) ----
// Deterministic weather derived from today's date hash so the card is stable
// for the whole day (no flicker, no API call). The hash picks one of three
// conditions (晴/多云/雨) and maps to a temperature band + an emoji. Each
// condition carries a festive shopping tip encouraging users to browse.
const WEATHER_STATES = [
  { icon: '☀️', name: '晴', tempBase: 26, tip: '阳光明媚，适合购物' },
  { icon: '☁️', name: '多云', tempBase: 21, tip: '舒适宜人，适合购物' },
  { icon: '🌧️', name: '小雨', tempBase: 17, tip: '雨天宅家，正好购物' },
]
const SHOPPING_TIPS = ['好物限时抢', '正品低价', '新人有礼', '满减专场', '品质优选']
function dateHash() {
  // Stable per-day hash from YYYY-MM-DD.
  const d = new Date()
  const key = `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
  let h = 0
  for (let i = 0; i < key.length; i++) {
    h = (h * 31 + key.charCodeAt(i)) >>> 0
  }
  return h
}
const weather = computed(() => {
  const h = dateHash()
  const state = WEATHER_STATES[h % WEATHER_STATES.length]
  // Vary the temperature ±4°C deterministically from the same hash.
  const variance = (h % 9) - 4
  const temp = state.tempBase + variance
  const tip = SHOPPING_TIPS[(h >> 3) % SHOPPING_TIPS.length]
  return { ...state, temp, tip }
})
</script>

<template>
  <div class="home">
    <van-sticky>
      <div class="topbar">
        <span class="logo">天猫</span>
        <van-search class="search" placeholder="理想生活上天猫" shape="round" readonly @click="router.push('/search')" />
      </div>
    </van-sticky>

    <div class="banner">
      <van-image width="100%" height="160" fit="cover" src="https://img.alicdn.com/imgextra/s1180x270/tmall-banner.jpg" />
    </div>

    <!-- 天气小卡片 (home weather widget): deterministic weather + temperature + shopping tip -->
    <div class="weather-card">
      <div class="w-left">
        <span class="w-icon">{{ weather.icon }}</span>
        <div class="w-meta">
          <div class="w-temp"><span class="w-deg">{{ weather.temp }}</span><span class="w-unit">°C</span></div>
          <div class="w-name">{{ weather.name }}</div>
        </div>
      </div>
      <div class="w-tip">
        <span class="w-tip-badge">🛍️ {{ weather.tip }}</span>
        <span class="w-tip-sub">{{ weather.name }}天 · {{ weather.tip }}</span>
      </div>
    </div>

    <div class="cat-grid">
      <div
        v-for="(c, i) in categories"
        :key="c.id"
        class="cat-item"
        :class="{ rippling: rippleCat === c.id }"
        @click="router.push({ name: 'category', query: { id: c.id } })"
        @touchstart.passive="onCatTouchStart(c)"
        @touchend.passive="onCatTouchEnd"
        @mousedown="onCatTouchStart(c)"
        @mouseup="onCatTouchEnd"
        @mouseleave="onCatTouchEnd"
      >
        <div class="cat-icon" :style="catStyle(i)">{{ c.icon }}</div>
        <div class="cat-name">{{ c.name }}</div>
      </div>
      <div
        class="cat-item"
        :class="{ rippling: rippleCat === 'all' }"
        @click="router.push({ name: 'category' })"
        @touchstart.passive="onCatTouchStart({ id: 'all' })"
        @touchend.passive="onCatTouchEnd"
        @mousedown="onCatTouchStart({ id: 'all' })"
        @mouseup="onCatTouchEnd"
        @mouseleave="onCatTouchEnd"
      >
        <div class="cat-icon cat-all">»</div>
        <div class="cat-name">全部</div>
      </div>
    </div>

    <!-- 限时抢购倒计时横幅 (flash sale countdown banner) -->
    <div class="flash-banner" :class="{ flashing: flashing }" @click="router.push('/seckill')">
      <span class="fb-text">⚡ 限时秒杀 距开抢还有</span>
      <span class="fb-countdown" :class="{ flashing: flashing }">{{ countdown }}</span>
    </div>

    <!-- 首页跑马灯 (home promotional marquee): infinite CSS translateX scroll -->
    <div class="marquee">
      <span class="mq-icon">📣</span>
      <div class="mq-viewport">
        <div class="mq-track">
          <span
            v-for="(m, i) in marqueeTrack"
            :key="i"
            class="mq-item"
            @click="tapMarquee(m)"
          >{{ m }}</span>
        </div>
      </div>
    </div>

    <!-- 限时秒杀入口 -->
    <div v-if="loading" class="seckill-entry skeleton-seckill">
      <div v-for="i in skeletonSeckill" :key="'sk' + i" class="sk-sk-item">
        <div class="sk-circle shimmer"></div>
        <div class="sk-line sk-line-s shimmer"></div>
      </div>
    </div>
    <div v-else class="seckill-entry" :class="{ 'fade-in': contentReady }" @click="router.push('/seckill')">
      <span class="se-icon">⚡</span>
      <span class="se-title">限时秒杀</span>
      <span class="se-sub">5折抢购 · 先到先得</span>
      <span class="se-go">去看看 ›</span>
    </div>

    <!-- 品牌馆入口 -->
    <div class="section">
      <div class="section-head"><span class="tm-red">品牌旗舰</span><span class="more" @click="router.push('/brands')">全部 ›</span></div>
      <div class="brand-scroll">
        <div v-for="b in brands" :key="b.id" class="brand-card" @click="router.push('/brand/' + b.id)">
          <div class="brand-logo">{{ b.logo }}</div>
          <div class="brand-name van-ellipsis">{{ b.name.replace('官方旗舰店', '') }}</div>
          <div class="brand-fans">{{ (b.followers / 10000).toFixed(0) }}万粉丝</div>
        </div>
      </div>
    </div>

    <!-- 热门标签 (hot tags discovery feed) -->
    <div class="tag-section">
      <div class="tag-head">
        <span class="tag-title">🔥 热门标签</span>
        <span class="tag-sub">发现好物 一键直达</span>
      </div>
      <div class="tag-chips">
        <span
          v-for="(t, i) in hotTags"
          :key="t"
          class="tag-chip"
          :style="tagStyle(i)"
          @click="searchTag(t)"
        >{{ t }}</span>
      </div>
    </div>

    <div class="section">
      <div class="section-head"><span>为你推荐</span></div>
      <!-- Skeleton product cards while loading (骨架屏) -->
      <div v-if="loading" class="product-grid">
        <div v-for="i in skeletonProducts" :key="'sp' + i" class="product-card sk-product-card">
          <div class="sk-img shimmer"></div>
          <div class="sk-line sk-line-l shimmer"></div>
          <div class="sk-line sk-line-m shimmer"></div>
          <div class="sk-bottom">
            <div class="sk-line sk-price shimmer"></div>
            <div class="sk-line sk-sales shimmer"></div>
          </div>
        </div>
      </div>
      <!-- Real content fades in once loaded -->
      <div v-else class="product-grid" :class="{ 'fade-in': contentReady }">
        <div v-for="p in products" :key="p.id" class="product-card" @click="router.push('/product/' + p.id)">
          <div class="new-badge" v-if="isNew(p)">NEW</div>
          <!-- 限量抢购闪烁 (limited stock flash): red pulsing "⚡仅剩N件" badge -->
          <div v-if="lowStock(p) !== null" class="low-stock-badge">⚡仅剩{{ lowStock(p) }}件</div>
          <van-image width="100%" height="170" :src="p.image" fit="cover" radius="6" />
          <div class="p-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="p-shop van-ellipsis">{{ p.shop }}</div>
          <div class="p-bottom">
            <span class="price">¥{{ fmt(p.price) }}</span>
            <span class="p-tags-inline">
              <span v-if="isPriceDrop(p)" class="trend-tag trend-down">📉降价</span>
              <span v-if="isSeckill(p)" class="trend-tag trend-seckill">⚡秒杀</span>
              <span v-if="isGenuine(p)" class="trend-tag trend-genuine">✓正品</span>
            </span>
          </div>
          <div class="p-sales-row"><span class="sales">{{ p.sales }}人付款</span></div>
        </div>
      </div>
    </div>

    <!-- 比价悬浮球 (price track floating ball) -->
    <div
      v-if="!ballHidden && priceTrack"
      class="price-ball"
      @click="tapBall"
      @touchstart.passive="startLongPress"
      @touchend="cancelLongPress"
      @touchmove="cancelLongPress"
      @touchcancel="cancelLongPress"
      @mousedown="startLongPress"
      @mouseup="cancelLongPress"
      @mouseleave="cancelLongPress"
    >
      <van-image class="pb-thumb" fit="cover" radius="6" :src="priceTrack.image" />
      <span class="pb-price">¥{{ fmt(displayPrice()) }}</span>
      <span v-if="priceTrend > 0" class="pb-trend pb-up">↑</span>
      <span v-else-if="priceTrend < 0" class="pb-trend pb-down">↓</span>
    </div>
  </div>
</template>

<style scoped>
.topbar { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: #fff; }
.logo { color: #ff0036; font-weight: bold; font-size: 18px; }
.search { flex: 1; padding: 0; }
.banner { margin: 8px; border-radius: 8px; overflow: hidden; }
/* 天气小卡片 (home weather widget) */
.weather-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0 8px 8px;
  padding: 12px 14px;
  border-radius: 10px;
  background: linear-gradient(135deg, #e3f2fd 0%, #fff8e1 60%, #ffeef0 100%);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}
.w-left { display: flex; align-items: center; gap: 10px; }
.w-icon { font-size: 34px; line-height: 1; filter: drop-shadow(0 2px 3px rgba(0, 0, 0, 0.12)); }
.w-meta { display: flex; flex-direction: column; }
.w-temp { display: flex; align-items: baseline; gap: 1px; line-height: 1; }
.w-deg { font-size: 24px; font-weight: bold; color: #ff0036; font-variant-numeric: tabular-nums; }
.w-unit { font-size: 12px; color: #ff0036; font-weight: bold; }
.w-name { font-size: 12px; color: #666; margin-top: 3px; }
.w-tip { display: flex; flex-direction: column; align-items: flex-end; gap: 4px; }
.w-tip-badge {
  font-size: 13px;
  font-weight: bold;
  color: #fff;
  background: linear-gradient(135deg, #ff0036, #ff5577);
  padding: 5px 12px;
  border-radius: 999px;
  box-shadow: 0 2px 6px rgba(255, 0, 54, 0.3);
  white-space: nowrap;
}
.w-tip-sub { font-size: 11px; color: #999; }
.flash-banner { display: flex; align-items: center; justify-content: center; gap: 10px; margin: 0 0 8px; padding: 14px 16px; background: linear-gradient(90deg, #ff0036, #ff2f5a); color: #fff; font-size: 16px; font-weight: bold; cursor: pointer; transition: background 0.3s; }
.flash-banner.flashing { background: linear-gradient(90deg, #ffb300, #ff0036); }
.fb-text { letter-spacing: 0.5px; }
.fb-countdown { font-family: 'Courier New', Consolas, monospace; font-size: 20px; letter-spacing: 1px; font-variant-numeric: tabular-nums; padding: 2px 10px; border-radius: 6px; background: rgba(0, 0, 0, 0.22); }
.fb-countdown.flashing { font-family: inherit; font-size: 16px; background: rgba(255, 255, 255, 0.25); animation: blink 0.5s ease-in-out infinite alternate; }
@keyframes blink { from { opacity: 1; } to { opacity: 0.6; } }
/* 首页跑马灯 (home promotional marquee) */
.marquee { display: flex; align-items: center; gap: 6px; margin: 0 8px 8px; padding: 8px 12px; background: linear-gradient(90deg, #fff5e6, #fff0f3); border-radius: 8px; overflow: hidden; }
.mq-icon { font-size: 14px; flex-shrink: 0; }
.mq-viewport { flex: 1; overflow: hidden; }
.mq-track { display: inline-flex; align-items: center; gap: 28px; white-space: nowrap; will-change: transform; animation: mq-scroll 16s linear infinite; }
.mq-track:hover { animation-play-state: paused; }
.mq-item { font-size: 13px; color: #ff0036; font-weight: bold; cursor: pointer; user-select: none; }
.mq-item:active { opacity: 0.6; }
@keyframes mq-scroll { from { transform: translateX(0); } to { transform: translateX(-50%); } }
.seckill-entry { display: flex; align-items: center; gap: 8px; margin: 8px; background: linear-gradient(90deg, #ff0036, #ff5577); border-radius: 8px; padding: 12px 16px; color: #fff; cursor: pointer; }
.se-icon { font-size: 20px; }
.se-title { font-size: 16px; font-weight: bold; }
.se-sub { font-size: 12px; opacity: 0.9; flex: 1; }
.se-go { font-size: 13px; }
.cat-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 14px 0; padding: 16px 8px; background: #fff; margin: 0 8px 8px; border-radius: 8px; }
.cat-item { text-align: center; cursor: pointer; user-select: none; -webkit-tap-highlight-color: transparent; }
.cat-item .cat-icon {
  font-size: 24px;
  width: 48px; height: 48px;
  margin: 0 auto;
  display: flex; align-items: center; justify-content: center;
  color: #fff;
  border-radius: 50%;
  box-shadow: 0 3px 8px rgba(255, 0, 54, 0.18);
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}
.cat-item .cat-all { background: linear-gradient(135deg, #888, #bbb); font-weight: bold; font-size: 22px; letter-spacing: -2px; }
.cat-item.rippling .cat-icon { transform: scale(0.88); box-shadow: 0 1px 3px rgba(255, 0, 54, 0.25); }
.cat-item.rippling::after {
  content: '';
  position: absolute;
}
.cat-name { font-size: 12px; color: #666; margin-top: 6px; }
.section { background: #fff; margin: 0 8px 8px; border-radius: 8px; padding: 12px; }
.section-head { display: flex; justify-content: space-between; align-items: center; font-size: 16px; font-weight: bold; margin-bottom: 10px; }
.more { font-size: 12px; color: #999; font-weight: normal; }
.brand-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; }
.brand-card { flex-shrink: 0; width: 76px; text-align: center; }
.brand-logo { width: 56px; height: 56px; line-height: 56px; margin: 0 auto; background: #fff5f6; color: #ff0036; border: 1px solid #ffd6dd; border-radius: 50%; font-weight: bold; font-size: 14px; }
.brand-name { font-size: 12px; margin-top: 6px; }
.brand-fans { font-size: 10px; color: #999; }
.product-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.product-card { background: #fafafa; border-radius: 8px; overflow: hidden; padding-bottom: 6px; position: relative; }
.new-badge { position: absolute; top: 6px; left: 6px; z-index: 5; background: #ff0036; color: #fff; font-size: 11px; font-weight: bold; line-height: 1; padding: 3px 7px; border-radius: 999px; box-shadow: 0 1px 4px rgba(255, 0, 54, 0.45); letter-spacing: 0.5px; }
/* 限量抢购闪烁 (limited stock flash): red pulsing badge */
.low-stock-badge { position: absolute; top: 6px; right: 6px; z-index: 5; background: #ff0036; color: #fff; font-size: 11px; font-weight: bold; line-height: 1; padding: 4px 8px; border-radius: 999px; box-shadow: 0 1px 6px rgba(255, 0, 54, 0.55); animation: ls-flash 0.9s ease-in-out infinite; }
@keyframes ls-flash { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.45; transform: scale(0.92); } }
.p-tag { padding: 2px 6px; }
.p-name { font-size: 13px; line-height: 18px; padding: 0 6px; height: 36px; }
.p-shop { font-size: 11px; color: #999; padding: 0 6px; }
.p-bottom { display: flex; align-items: center; justify-content: flex-start; gap: 6px; padding: 2px 6px; }
.p-bottom .price { font-size: 16px; }
.p-tags-inline { display: inline-flex; align-items: center; gap: 3px; flex-wrap: wrap; }
.p-sales-row { padding: 0 6px 2px; }
.sales { font-size: 11px; color: #999; }
/* 价格趋势标签 (price trend tags) */
.trend-tag { display: inline-flex; align-items: center; font-size: 10px; line-height: 1; font-weight: bold; padding: 3px 5px; border-radius: 3px; white-space: nowrap; }
.trend-down { color: #07c160; background: #e6f9ee; border: 1px solid #b6e8c9; }
.trend-seckill { color: #ff0036; background: #fff0f3; border: 1px solid #ffc6cf; }
.trend-genuine { color: #1677ff; background: #eef5ff; border: 1px solid #bcd6ff; }
.loading { text-align: center; padding: 20px; }

/* ---- 骨架屏 + 渐入动画 (skeleton shimmer + fade-in) ---- */
.shimmer {
  background: linear-gradient(90deg, #ececec 25%, #f5f5f5 37%, #ececec 63%);
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}
@keyframes shimmer {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}
/* Skeleton product card */
.sk-product-card { padding: 0 0 8px; }
.sk-img { width: 100%; height: 170px; border-radius: 6px 6px 0 0; }
.sk-line { height: 12px; border-radius: 4px; margin: 8px 6px; }
.sk-line-l { height: 16px; }
.sk-line-m { width: 60%; }
.sk-bottom { display: flex; align-items: center; justify-content: space-between; padding: 0 6px; margin-top: 4px; }
.sk-price { width: 40%; height: 18px; }
.sk-sales { width: 30%; height: 12px; }
/* Skeleton seckill row */
.skeleton-seckill { display: flex; gap: 10px; justify-content: space-between; padding: 14px 16px; background: #f2f2f2 !important; cursor: default; }
.sk-sk-item { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.sk-circle { width: 48px; height: 48px; border-radius: 50%; }
.skeleton-seckill .sk-line-s { width: 80%; margin: 0; height: 10px; }
/* Fade-in real content */
.fade-in { animation: fadeInUp 0.45s ease both; }
@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 热门标签 (hot tags discovery feed) */
.tag-section {
  margin: 0 8px 8px;
  border-radius: 8px;
  padding: 14px 12px 16px;
  background: linear-gradient(135deg, #fff0f3 0%, #fff6e6 50%, #f0f6ff 100%);
}
.tag-head { display: flex; align-items: baseline; gap: 10px; margin-bottom: 12px; }
.tag-title { font-size: 16px; font-weight: bold; color: #ff0036; }
.tag-sub { font-size: 11px; color: #999; }
.tag-chips { display: flex; flex-wrap: wrap; gap: 10px; }
.tag-chip {
  color: #fff;
  font-size: 13px;
  font-weight: bold;
  padding: 7px 16px;
  border-radius: 999px;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  user-select: none;
}
.tag-chip:active { transform: scale(0.94); box-shadow: 0 1px 3px rgba(0, 0, 0, 0.18); }

/* 比价悬浮球 (price track floating ball) */
.price-ball {
  position: fixed;
  right: 16px;
  bottom: 80px;
  z-index: 50;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.22);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  user-select: none;
  -webkit-tap-highlight-color: transparent;
}
.pb-thumb { width: 100%; height: 100%; }
.pb-price {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  background: rgba(255, 0, 54, 0.92);
  color: #fff;
  font-size: 9px;
  font-weight: bold;
  line-height: 14px;
  text-align: center;
  padding: 1px 0;
}
.pb-trend {
  position: absolute;
  top: -2px;
  right: -2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  color: #fff;
  font-size: 12px;
  font-weight: bold;
  line-height: 18px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border: 1.5px solid #fff;
}
.pb-up { background: #ff0036; }
.pb-down { background: #07c160; }
</style>
