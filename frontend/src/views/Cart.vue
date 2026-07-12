<script setup>
import { ref, computed, onMounted, onActivated, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog, showDialog } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getMyCoupons, getAddresses, requestInvoice, getTieredDiscounts, getProducts, addToCart, toggleFavorite, getCheckInStatus } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
const coupons = ref([])
const selectedCouponId = ref(null)
const showCouponPicker = ref(false)
const remark = ref('')
// ---- 感谢快递员小费 (checkout courier tip) ----
// Three quick-tip buttons (¥1/¥3/¥5) append a "💰感谢快递员 ¥N" line to the order
// remark so it travels with the order. Tapping the active tip removes it.
const TIP_AMOUNTS = [1, 3, 5]
const selectedTip = ref(0)
function applyTip(amount) {
  // Strip any previously-appended tip line so we don't stack duplicates.
  const base = remark.value.replace(/\s*💰感谢快递员\s*¥\d+\s*/g, '').trim()
  if (selectedTip.value === amount) {
    // Tapping the active tip toggles it off.
    selectedTip.value = 0
    remark.value = base
    showToast('已取消打赏')
    return
  }
  selectedTip.value = amount
  remark.value = (base ? base + ' ' : '') + `💰感谢快递员 ¥${amount}`
  showToast(`已添加打赏 ¥${amount}`)
}
// Recommended products (猜你喜欢).
const recommendations = ref([])
// Address picker (收货地址).
const addresses = ref([])
const selectedAddressId = ref(null)
const showAddressPicker = ref(false)
// Invoice form (发票).
const invoiceEnabled = ref(false)
const showInvoiceForm = ref(false)
const invoiceType = ref('personal')
const invoiceTitle = ref('')
const invoiceTaxNo = ref('')
const invoiceEmail = ref('')
// Tiered discounts (阶梯满减) — store-wide promotion tiers.
const tiers = ref([])
// Group buy invite (好友拼单邀请) — social shopping invite popup.
const showGroupBuy = ref(false)
// Demo counter of how many friends have been "invited" (persisted in
// localStorage across sessions).
const GROUP_KEY = 'tm_groupbuy_invited'
const invitedCount = ref(Number(localStorage.getItem(GROUP_KEY) || 0))
// Generated invite text: item count + total + a fake join link.
const inviteText = computed(() =>
  `🎉 我正在天猫发起拼单，${selectedCount.value}件好物合计¥${fmt(selectedTotal.value)}，快来一起拼单享优惠！\n👉 点击加入拼单：https://tmall.example.com/group?c=${selectedCount.value}&t=${Date.now()}`
)
function openGroupBuy() {
  if (!items.value.length) { showToast('购物车是空的'); return }
  showGroupBuy.value = true
}
async function copyInvite() {
  try {
    await navigator.clipboard.writeText(inviteText.value)
    showSuccessToast('邀请已复制')
  } catch (e) {
    // Clipboard API may be unavailable (HTTP, older browsers): fall back to a
    // transient selection-based copy via a hidden textarea.
    const ta = document.createElement('textarea')
    ta.value = inviteText.value
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try { document.execCommand('copy'); showSuccessToast('邀请已复制') } catch (_) { showToast('复制失败，请手动复制') }
    document.body.removeChild(ta)
  }
}
async function shareInvite() {
  if (navigator.share) {
    try { await navigator.share({ title: '好友拼单邀请', text: inviteText.value }) } catch (e) {}
  } else {
    await copyInvite()
    showToast('当前环境不支持系统分享，已为你复制邀请')
  }
}
function addInvitee() {
  invitedCount.value += 1
  localStorage.setItem(GROUP_KEY, String(invitedCount.value))
  showToast(`已邀请${invitedCount.value}人`)
}

const usableCoupons = computed(() =>
  (coupons.value || []).filter((c) => c.is_used === 0 && (!c.coupon || selectedTotal.value >= c.coupon.threshold))
)
const discount = computed(() => {
  const uc = usableCoupons.value.find((c) => c.id === selectedCouponId.value)
  if (!uc || !uc.coupon) return 0
  if (uc.coupon.coupon_type === 'discount') return Math.round(selectedTotal.value * (1 - uc.coupon.value) * 100) / 100
  return uc.coupon.value
})
// ---- 最佳优惠券自动选用 (best coupon auto-apply) ----
// Compute the discount a usable coupon yields against the current subtotal so we
// can rank them. Discount-type coupons take a percentage off; fixed coupons
// subtract their face value. Returns 0 when the coupon can't be evaluated.
function couponDiscount(uc) {
  if (!uc || !uc.coupon) return 0
  if (uc.coupon.coupon_type === 'discount') return Math.round(selectedTotal.value * (1 - uc.coupon.value) * 100) / 100
  return uc.coupon.value
}
// The usable coupon with the largest discount (ties broken by id for stability).
const bestCoupon = computed(() => {
  const list = usableCoupons.value
  if (!list.length) return null
  return list.reduce((best, uc) => (couponDiscount(uc) > couponDiscount(best) ? uc : best), list[0])
})
// Auto-select the best coupon whenever the usable set changes (e.g. after the
// cart loads or the subtotal crosses a threshold). Only fires a toast the first
// time per cart load to avoid spamming on every selection toggle; we track the
// last subtotal+coupon signature we announced.
let lastAutoApplySig = ''
watch(bestCoupon, (best) => {
  if (!best) return
  // Signature guards against re-announcing for the same effective state.
  const sig = selectedTotal.value + ':' + best.id
  // Only auto-apply (and announce) when the user hasn't manually picked one
  // OR their pick is no longer the best value.
  if (selectedCouponId.value === best.id) { lastAutoApplySig = sig; return }
  selectedCouponId.value = best.id
  if (lastAutoApplySig !== sig) {
    lastAutoApplySig = sig
    showSuccessToast('已自动选用最佳优惠')
  }
})
const finalTotal = computed(() => Math.max(0, selectedTotal.value - discount.value))
const selectedCount = computed(() => items.value.filter((i) => i.selected === 1).length)
// Top-up hints: claimed coupons just below the threshold (凑单提示).
const topupHints = computed(() => {
  const hints = []
  for (const c of (coupons.value || [])) {
    if (c.is_used === 0 && c.coupon && selectedTotal.value < c.coupon.threshold) {
      const diff = Math.ceil(c.coupon.threshold - selectedTotal.value)
      if (diff <= 50) hints.push({ id: c.id, diff, label: c.coupon.coupon_type === 'discount' ? `${(c.coupon.value * 10).toFixed(1)}折券` : `满${c.coupon.threshold}减${c.coupon.value}` })
    }
  }
  return hints.sort((a, b) => a.diff - b.diff)
})
// Tiered discount summary (阶梯满减) — active tiers joined with " | ".
const tierSummary = computed(() =>
  (tiers.value || []).map((t) => `满${Math.round(t.threshold)}减${Math.round(t.discount)}`).join(' | ')
)
// The next unreached tier above the current selected subtotal (再买¥X减¥Y).
const nextTier = computed(() => {
  for (const t of tiers.value || []) {
    if (selectedTotal.value < t.threshold) {
      return { diff: Math.ceil(t.threshold - selectedTotal.value), discount: t.discount, threshold: t.threshold }
    }
  }
  return null
})
// Whether the selected subtotal has reached at least one tier (已达满减).
// Returns the highest reached tier (with the largest threshold) or null.
const reachedTier = computed(() => {
  let best = null
  for (const t of tiers.value || []) {
    if (selectedTotal.value >= t.threshold && (!best || t.threshold > best.threshold)) best = t
  }
  return best
})
// Smart top-up picks (智能凑单): when there's a next tier to chase, surface
// recommended products whose price roughly covers the remaining gap. Products
// are ranked by closeness to the gap; small additions (≤ gap*2) that finish the
// tier are promoted first. Empty when no tier to chase or nothing matches.
const topupPicks = computed(() => {
  if (!nextTier.value) return []
  const gap = nextTier.value.diff
  return (recommendations.value || [])
    .filter((p) => p.price > 0)
    .map((p) => ({ ...p, _delta: p.price - gap }))
    .filter((p) => p._delta <= gap) // don't suggest items wildly over the gap
    .sort((a, b) => Math.abs(a._delta) - Math.abs(b._delta))
    .slice(0, 4)
})
async function quickTopup(p) {
  try { await addToCart(p.id, 1); await load(); showSuccessToast('已加入购物车 凑单成功') }
  catch (e) { showToast(e.response?.data?.error || '加入失败') }
}
// ---- 搭配购买 (cart companion suggestions) ----
// "买了A的人还买了": for each cart item we surface one related product from the
// recommendation pool. The pairing is deterministic (a stable hash of the item's
// product_id picks an index into the pool) so a given item always maps to the
// same companion, and we skip the item itself. We show at most one companion per
// item and dedupe companions already in the cart.
function hashPick(seed) {
  let h = (seed >>> 0) || 1
  h = (h * 2654435761) >>> 0
  return h
}
const companionSuggestions = computed(() => {
  const pool = (recommendations.value || []).filter((p) => p.id)
  if (!pool.length) return []
  const inCartIds = new Set(items.value.map((i) => i.product_id))
  const used = new Set()
  const out = []
  for (const it of items.value) {
    // Deterministically pick a starting index for this item.
    const base = hashPick(Number(it.product_id) || 0) % pool.length
    let pick = null
    for (let step = 0; step < pool.length; step++) {
      const cand = pool[(base + step) % pool.length]
      if (cand.id === it.product_id) continue
      if (inCartIds.has(cand.id)) continue
      if (used.has(cand.id)) continue
      pick = cand
      break
    }
    if (pick) {
      used.add(pick.id)
      out.push({ cartItem: it, product: pick })
    }
  }
  return out
})
async function quickAddCompanion(p) {
  try { await addToCart(p.id, 1); await load(); showSuccessToast('已加入购物车') }
  catch (e) { showToast(e.response?.data?.error || '加入失败') }
}
// ---- 下单抽奖 (order lucky draw) ----
// After a successful checkout a "🎁幸运抽奖" popup appears. It's a 3-column slot
// machine: the columns scroll, then stop one-by-one left → right. The three
// landed symbols decide the prize:
//   - all 3 match → 88VIP折扣券
//   - exactly 2 match → 10元券
//   - otherwise → 谢谢参与
// Winners trigger a confetti burst. The first draw is free; "再来一次" costs 50
// 积分. Points are seeded from the check-in status (total_points) and tracked
// locally for the session so replays can spend them.
const SLOT_SYMBOLS = ['🍒', '🍋', '🔔', '⭐', '💎']
const REPLAY_COST = 50
const LUCKY_POINTS_KEY = 'tm_lucky_points'
const showLuckyDraw = ref(false)
const drawSpinning = ref(false)
const drawResult = ref(null)         // { prize: '88VIP折扣券'|'10元券'|'谢谢参与', win: bool }
const drawPoints = ref(0)
// Per-column state: the landed symbol index (null while still spinning).
const slotCols = ref([null, null, null])
// Vertical offset (in symbol-height units) used to drive the scrolling animation.
// Each column animates by translating its strip upward; the final offset places
// the target symbol centered.
const slotOffsets = ref([0, 0, 0])
const slotReels = ref([[], [], []])  // each column's rendered symbol list
let spinTimers = []

function loadDrawPoints() {
  // Prefer the server's total_points; fall back to a locally persisted value.
  if (localStorage.getItem('tm_token')) {
    getCheckInStatus().then((res) => {
      drawPoints.value = Number(res.total_points) || 0
      localStorage.setItem(LUCKY_POINTS_KEY, String(drawPoints.value))
    }).catch(() => {
      drawPoints.value = Number(localStorage.getItem(LUCKY_POINTS_KEY) || 0)
    })
  } else {
    drawPoints.value = Number(localStorage.getItem(LUCKY_POINTS_KEY) || 0)
  }
}
// Build a long repeating strip ending on the target symbol so the CSS
// transition can scroll from top to that symbol.
function buildReel(targetIdx, length) {
  const reel = []
  for (let i = 0; i < length; i++) {
    reel.push((Math.floor(Math.random() * SLOT_SYMBOLS.length) + i) % SLOT_SYMBOLS.length)
  }
  // Force the final cell to the target so the landing is deterministic.
  reel[length - 1] = targetIdx
  return reel
}
// Pre-pick the three landed symbols with a light weighting so a 3-match is rare
// but possible. Returns the indices of the three symbols.
function pickLanding() {
  const r = Math.random()
  if (r < 0.12) {
    // ~12%: all three match.
    const s = Math.floor(Math.random() * SLOT_SYMBOLS.length)
    return [s, s, s]
  }
  if (r < 0.45) {
    // ~33%: exactly two match (cols 0&1, 0&2, or 1&2).
    const s = Math.floor(Math.random() * SLOT_SYMBOLS.length)
    const other = (s + 1 + Math.floor(Math.random() * (SLOT_SYMBOLS.length - 1))) % SLOT_SYMBOLS.length
    const pairSlot = Math.floor(Math.random() * 3)
    const res = [other, other, other]
    res[pairSlot] = s
    res[(pairSlot + 1) % 3] = s
    return res
  }
  // otherwise: no matches (all distinct).
  const shuffled = [...SLOT_SYMBOLS.keys()]
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  return shuffled.slice(0, 3)
}
function startDraw(isReplay) {
  if (drawSpinning.value) return
  if (isReplay) {
    if (drawPoints.value < REPLAY_COST) { showToast('积分不足，需要' + REPLAY_COST + '积分'); return }
    drawPoints.value -= REPLAY_COST
    localStorage.setItem(LUCKY_POINTS_KEY, String(drawPoints.value))
  }
  drawSpinning.value = true
  drawResult.value = null
  const landing = pickLanding()
  // Reset columns to spinning state.
  for (let c = 0; c < 3; c++) {
    slotCols.value[c] = null
    const len = 20 + c * 4 // longer reels for outer columns -> they spin longer
    slotReels.value[c] = buildReel(landing[c], len)
    slotOffsets.value[c] = 0
  }
  // Stop the columns left → right with a stagger.
  for (let c = 0; c < 3; c++) {
    const stopDelay = 700 + c * 600
    spinTimers.push(setTimeout(() => stopColumn(c), stopDelay))
  }
}
function stopColumn(c) {
  // Set the offset so the final symbol sits centered, then lock the column.
  const reel = slotReels.value[c]
  // The final symbol is the last cell; translate up by (len - 1) symbol heights.
  slotOffsets.value[c] = reel.length - 1
  slotCols.value[c] = reel[reel.length - 1]
  // If this is the last column, resolve the result.
  if (slotCols.value.every((v) => v !== null)) {
    drawSpinning.value = false
    resolveDraw()
  }
}
function resolveDraw() {
  const [a, b, c] = slotCols.value
  let win = false
  let prize = '谢谢参与'
  if (a === b && b === c) { prize = '88VIP折扣券'; win = true }
  else if (a === b || b === c || a === c) { prize = '10元券'; win = true }
  drawResult.value = { prize, win }
  if (win) {
    showSuccessToast('恭喜中奖：' + prize)
    burstConfetti()
  } else {
    showToast('谢谢参与')
  }
}
function closeLuckyDraw() {
  // Cancel any pending spin timers if the user closes mid-spin.
  spinTimers.forEach((t) => clearTimeout(t))
  spinTimers = []
  drawSpinning.value = false
  showLuckyDraw.value = false
}
// ---- Confetti (彩纸) for winners ----
// A lightweight DOM-based confetti: spawn N colored chips that fall and fade,
// then remove themselves. No external dependency.
function burstConfetti() {
  const host = document.getElementById('tm-confetti')
  if (!host) return
  const colors = ['#ff0036', '#ff8a00', '#ffc107', '#07c160', '#1989fa', '#7232dd']
  for (let i = 0; i < 60; i++) {
    const chip = document.createElement('div')
    chip.className = 'tm-confetti-chip'
    chip.style.left = Math.random() * 100 + '%'
    chip.style.background = colors[i % colors.length]
    chip.style.animationDelay = (Math.random() * 0.5) + 's'
    chip.style.animationDuration = (1.6 + Math.random() * 1.4) + 's'
    chip.style.transform = `rotate(${Math.random() * 360}deg)`
    host.appendChild(chip)
    setTimeout(() => chip.remove(), 3200)
  }
}
// Whether a cart item has dropped in price (降价 tag) — original_price > price * 1.1.
function isPriceDrop(item) {
  return item.original_price && item.original_price > item.price * 1.1
}

async function load() {
  loading.value = true
  try {
    const [cartRes, myCoupons, addrList, tierList, recs] = await Promise.all([
      getCart(),
      getMyCoupons().catch(() => []),
      getAddresses().catch(() => []),
      getTieredDiscounts().catch(() => []),
      getProducts({ page: 1, page_size: 4 }).then((r) => r.data).catch(() => []),
    ])
    items.value = cartRes.data || []
    selectedTotal.value = cartRes.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
    coupons.value = myCoupons || []
    addresses.value = addrList || []
    tiers.value = tierList || []
    recommendations.value = recs || []
    // Default to the marked default address (or the first).
    if (selectedAddressId.value === null) {
      const def = addresses.value.find((a) => a.is_default === 1) || addresses.value[0]
      selectedAddressId.value = def ? def.id : null
    }
  } catch (e) { if (e.response?.status === 401) router.replace('/login') } finally { loading.value = false }
}
onMounted(() => { load(); loadDrawPoints() }); onActivated(load)

async function toggleSelect(item) {
  const ns = item.selected === 1 ? 0 : 1
  try { await updateCart(item.id, item.quantity, ns); item.selected = ns; await load() } catch (e) {}
}
async function changeQty(item, qty) {
  if (qty < 1) return
  try { await updateCart(item.id, qty, item.selected); item.quantity = qty; await load() } catch (e) {}
}
async function removeItem(item) {
  try {
    await deleteCart(item.id)
    // 摇一摇撤销 (shake to undo): remember the just-removed item so a shake
    // gesture within the undo window can restore it by re-adding to the cart.
    lastDeleted.value = { product_id: item.product_id, name: item.product_name, at: Date.now() }
    await load(); showSuccessToast('已删除')
  } catch (e) {}
}
// ---- 摇一摇撤销删除 (shake to undo delete) ----
// Detect a shake via DeviceMotionEvent (acceleration magnitude > 25). On shake
// we prompt "摇一摇撤销删除？"; confirming re-fetches the cart to restore the
// item. iOS 13+ requires requesting permission first, which we attempt on the
// first user gesture. Where DeviceMotion is unavailable, a fallback toast hints
// the user can swipe items back instead.
const lastDeleted = ref(null)
const SHAKING = { THRESHOLD: 25, WINDOW_MS: 8000 } // 8s undo window
let lastShakeAt = 0
let motionHandler = null
function handleMotion(e) {
  const a = e.accelerationIncludingGravity || e.acceleration
  if (!a) return
  const mag = Math.sqrt((a.x || 0) ** 2 + (a.y || 0) ** 2 + (a.z || 0) ** 2)
  if (mag > SHAKING.THRESHOLD) {
    // Debounce repeated triggers within the same gesture.
    const now = Date.now()
    if (now - lastShakeAt < 1200) return
    lastShakeAt = now
    onShake()
  }
}
function onShake() {
  // Only offer undo when a recent delete is still within the window.
  const del = lastDeleted.value
  if (!del || Date.now() - del.at > SHAKING.WINDOW_MS) {
    showToast('摇一摇：摇晃手机可撤销刚刚的删除')
    return
  }
  showConfirmDialog({
    title: '摇一摇撤销',
    message: `摇一摇撤销删除？\n将恢复「${del.name}」到购物车`,
    confirmButtonText: '撤销删除',
    confirmButtonColor: '#ff0036',
    cancelButtonText: '不了',
  }).then(async () => {
    try {
      await addToCart(del.product_id, 1)
      await load()
      lastDeleted.value = null
      showSuccessToast('已恢复')
    } catch (e) {
      showToast(e.response?.data?.error || '恢复失败')
    }
  }).catch(() => {
    // User declined — clear the undo record so a later shake falls back to toast.
    lastDeleted.value = null
  })
}
async function setupShake() {
  // iOS 13+ requires an explicit permission request triggered by a user gesture.
  const DME = window.DeviceMotionEvent
  if (!DME) {
    showToast('当前设备不支持摇一摇')
    return
  }
  if (typeof DME.requestPermission === 'function') {
    try {
      const state = await DME.requestPermission()
      if (state !== 'granted') { showToast('未授权运动传感器，摇一摇不可用'); return }
    } catch (_) { showToast('运动传感器授权失败'); return }
  }
  if (motionHandler) return // already listening
  motionHandler = handleMotion
  window.addEventListener('devicemotion', motionHandler)
  showToast('摇一摇已开启，可撤销删除')
}
onUnmounted(() => {
  if (motionHandler) { window.removeEventListener('devicemotion', motionHandler); motionHandler = null }
})
// 收藏 (favorite) — called from the swipe action of a cart item.
async function favoriteItem(item) {
  try {
    await toggleFavorite(item.product_id)
    showSuccessToast('已收藏')
  } catch (e) { showToast(e.response?.data?.error || '操作失败') }
}
async function quickAdd(p) { try { await addToCart(p.id, 1); await load(); showSuccessToast('已加入购物车') } catch (e) { showToast(e.response?.data?.error || '加入失败') } }
async function toggleAll() {
  const t = allSelected.value ? 0 : 1
  for (const it of items.value) { if (it.selected !== t) await updateCart(it.id, it.quantity, t) }
  await load()
}
// Invert selection (反选) — flip every item's selected state.
async function invertSelection() {
  for (const it of items.value) { await updateCart(it.id, it.quantity, it.selected === 1 ? 0 : 1) }
  await load()
}
const selectedAddress = computed(() => addresses.value.find((a) => a.id === selectedAddressId.value) || null)
function addressLabel() {
  const a = selectedAddress.value
  if (!a) return '请选择收货地址'
  return `${a.name} ${a.phone} ${a.detail}`
}
function pickAddress(a) {
  selectedAddressId.value = a.id
  showAddressPicker.value = false
}
function invoiceLabel() {
  if (!invoiceEnabled.value) return '不开发票'
  if (invoiceType.value === 'company') {
    if (invoiceTitle.value) return `单位发票 · ${invoiceTitle.value}`
    return '单位发票'
  }
  return invoiceTitle.value ? `个人发票 · ${invoiceTitle.value}` : '个人发票'
}
function openInvoiceForm() {
  showInvoiceForm.value = true
}
function saveInvoice() {
  if (invoiceType.value === 'company' && !invoiceTitle.value) { showToast('请填写单位名称'); return }
  if (invoiceEmail.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(invoiceEmail.value)) { showToast('邮箱格式不正确'); return }
  invoiceEnabled.value = true
  showInvoiceForm.value = false
}
function closeInvoiceForm() {
  showInvoiceForm.value = false
}
// ---- 结算错误恢复 (checkout error recovery) ----
// When checkout fails we distinguish two failure modes:
//  - product info changed (e.g. price/stock/selection changed server-side):
//    show "商品信息已更新" with a Retry button.
//  - network timeout: show "3秒后自动重试..." with a live countdown that
//    auto-retries when it reaches 0.
// We cap attempts at 3 (MAX_CHECKOUT_RETRIES); after that the dialog shows
// "请联系客服". The selected-item snapshot is rebuilt from fresh cart data on
// each retry so stale product info never gets re-submitted.
const MAX_CHECKOUT_RETRIES = 3
const checkoutRetries = ref(0)
const showCheckoutError = ref(false)
const checkoutErrorMsg = ref('')
const checkoutIsTimeout = ref(false)
const checkoutCountdown = ref(0)       // seconds remaining for auto-retry
const checkoutMaxedOut = ref(false)    // all retries exhausted
let countdownTimer = null

// A network/timeout failure: axios sets err.code === 'ECONNABORTED' and there is
// no response (the request never reached the server). Treat 5xx as transient
// network/server errors too so the countdown auto-retry kicks in.
function isTimeoutErr(e) {
  if (e && (e.code === 'ECONNABORTED' || e.code === 'ETIMEDOUT')) return true
  if (e && !e.response) return true // request never reached the server
  const s = e && e.response && e.response.status
  return s === 502 || s === 503 || s === 504 || s === 408
}

function clearCountdown() {
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
}

// Build the order payload fresh from current cart state on each attempt so we
// always submit up-to-date product info (prices/stock/selection).
function buildCheckoutPayload() {
  const sel = items.value
    .filter((i) => i.selected === 1)
    .map((i) => ({ product_id: i.product_id, quantity: i.quantity }))
  const a = selectedAddress.value
  const addressText = `${a.name} ${a.phone} ${a.detail}`
  return {
    items: sel,
    address: addressText,
    user_coupon_id: selectedCouponId.value || undefined,
    remark: remark.value,
  }
}

async function performCheckout() {
  const order = await createOrder(buildCheckoutPayload())
  // Optionally request an invoice for the new order.
  if (invoiceEnabled.value) {
    await requestInvoice(order.id, {
      invoice_type: invoiceType.value,
      title: invoiceTitle.value,
      tax_no: invoiceTaxNo.value,
      email: invoiceEmail.value,
    }).catch(() => {})
  }
  return order
}

// Core retry driver. Recurses via the auto/manual retry entry points so that
// the retry counter, countdown and exhausted state all stay consistent.
async function checkout() {
  if (selectedTotal.value <= 0) { showToast('请选择商品'); return }
  if (!selectedAddress.value) { showToast('请选择收货地址'); return }
  // Reset retry bookkeeping for a fresh checkout attempt.
  checkoutRetries.value = 0
  checkoutMaxedOut.value = false
  await runCheckoutAttempt()
}

async function runCheckoutAttempt() {
  try {
    showCheckoutError.value = false
    clearCountdown()
    const order = await performCheckout()
    clearCountdown()
    // Reload the cart (order consumed the selected items) and offer the lucky
    // draw before navigating away.
    await load()
    showSuccessToast('下单成功')
    openLuckyDraw()
  } catch (e) {
    handleCheckoutError(e)
  }
}
// Offer the post-order lucky draw. The first spin is free; we refresh points
// first so the "再来一次" cost reflects the latest balance.
function openLuckyDraw() {
  loadDrawPoints()
  drawResult.value = null
  slotCols.value = [null, null, null]
  slotOffsets.value = [0, 0, 0]
  slotReels.value = [[], [], []]
  showLuckyDraw.value = true
}

function handleCheckoutError(e) {
  // Reload the cart first so the dialog/retry works against fresh data.
  load().catch(() => {})

  const timeout = isTimeoutErr(e)
  checkoutIsTimeout.value = timeout

  if (checkoutRetries.value >= MAX_CHECKOUT_RETRIES) {
    // Exhausted all retries -> contact customer service.
    checkoutMaxedOut.value = true
    checkoutErrorMsg.value = '多次重试失败，请联系客服'
    clearCountdown()
    showCheckoutError.value = true
    return
  }
  checkoutRetries.value += 1

  if (timeout) {
    // Network timeout: auto-retry after a 3s countdown.
    checkoutErrorMsg.value = '网络连接超时'
    showCheckoutError.value = true
    checkoutCountdown.value = 3
    clearCountdown()
    countdownTimer = setInterval(() => {
      checkoutCountdown.value -= 1
      if (checkoutCountdown.value <= 0) {
        clearCountdown()
        // Auto-retry without user interaction.
        runCheckoutAttempt()
      }
    }, 1000)
  } else {
    // Product info changed (price/stock/selection) or other server error:
    // surface "商品信息已更新" with a manual Retry button.
    const msg = e && e.response && e.response.data && e.response.data.error
    checkoutErrorMsg.value = msg ? `商品信息已更新：${msg}` : '商品信息已更新，请重试'
    showCheckoutError.value = true
  }
}

// Manual retry (Retry button in the dialog). Cancels any pending auto-countdown.
function retryCheckout() {
  clearCountdown()
  showCheckoutError.value = false
  runCheckoutAttempt()
}

function closeCheckoutError() {
  clearCountdown()
  showCheckoutError.value = false
}

onUnmounted(() => { clearCountdown() })
function couponLabel(uc) {
  if (!uc.coupon) return ''
  const c = uc.coupon
  if (c.coupon_type === 'discount') return `${(c.value * 10).toFixed(1)}折券 · 满${c.threshold}可用`
  return `满${c.threshold}减${c.value}`
}
function fmt(n) { return Number(n).toFixed(2) }
// ---- 预估重量 (cart weight estimator) ----
// Each product gets a deterministic weight in 0.1–5.0kg derived from a hash of
// its product_id (so the same product always reports the same weight). The
// selected-items total is the sum of weight × quantity; >10kg surfaces a
// "大件商品" warning to suggest consolidated / heavy-goods shipping.
function productWeight(productId) {
  let h = (Number(productId) || 0) >>> 0
  h = (h * 2654435761) >>> 0
  h = (h ^ (h >>> 13)) >>> 0
  // Map into [0.1, 5.0] kg (one decimal of precision, deterministic).
  const w = 0.1 + ((h % 490) / 100) // 0.1 + [0.00..4.89] -> 0.10..4.99
  return Math.round(w * 10) / 10
}
const estimatedWeight = computed(() => {
  let total = 0
  for (const it of items.value) {
    if (it.selected !== 1) continue
    total += productWeight(it.product_id) * (it.quantity || 1)
  }
  return Math.round(total * 10) / 10
})
const isHeavyOrder = computed(() => estimatedWeight.value > 10)
</script>

<template>
  <div class="cart-page">
    <van-nav-bar title="购物车" fixed placeholder />
    <!-- Recommendations (猜你喜欢) — horizontal scroll above the cart items. -->
    <div v-if="!loading && recommendations.length" class="rec-section">
      <div class="rec-head">猜你喜欢</div>
      <div class="rec-scroll">
        <div v-for="p in recommendations" :key="p.id" class="rec-card" @click="router.push('/product/' + p.id)">
          <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
          <div class="rec-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="rec-bottom">
            <span class="price">¥{{ fmt(p.price) }}</span>
            <van-button size="mini" type="danger" round @click.stop="quickAdd(p)">加入购物车</van-button>
          </div>
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading /></div>
    <!-- 购物车空状态 (cart empty state): animated emoji bag + 去逛逛 + recommendations -->
    <div v-else-if="!items.length" class="cart-empty">
      <div class="ce-bag-wrap">
        <div class="ce-bag">🛍️</div>
        <div class="ce-bag-shadow"></div>
      </div>
      <div class="ce-title">购物车空空如也</div>
      <div class="ce-sub">还没有添加任何商品，快去挑选心仪好物吧</div>
      <van-button class="ce-go-btn" type="danger" round icon="shop-o" @click="router.push('/home')">去逛逛</van-button>
      <!-- 为你推荐 (recommendations when empty) -->
      <div v-if="recommendations.length" class="ce-recommend">
        <div class="ce-rec-head"><span class="ce-rec-line"></span><span class="ce-rec-title">为你推荐</span><span class="ce-rec-line"></span></div>
        <div class="ce-rec-grid">
          <div v-for="p in recommendations" :key="p.id" class="ce-rec-card" @click="router.push('/product/' + p.id)">
            <van-image width="100%" height="120" radius="8" :src="p.image" fit="cover" />
            <div class="ce-rec-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="ce-rec-bottom">
              <span class="ce-rec-price">¥{{ fmt(p.price) }}</span>
              <van-button size="mini" type="danger" round @click.stop="quickAdd(p)">加入</van-button>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else>
      <van-swipe-cell v-for="it in items" :key="it.id">
        <div class="cart-item">
          <van-checkbox :model-value="it.selected === 1" @click="toggleSelect(it)" />
          <van-image width="80" height="80" radius="6" :src="it.product_image" fit="cover" @click="router.push('/product/' + it.product_id)" />
          <div class="ci-info">
            <div class="ci-name van-multi-ellipsis--l2">{{ it.product_name }}</div>
            <div class="ci-bottom"><span class="price">¥{{ fmt(it.price) }}</span><van-tag v-if="isPriceDrop(it)" type="danger" size="medium" round>降价</van-tag><van-stepper v-model="it.quantity" :min="1" :max="it.stock" @change="(v) => changeQty(it, v)" /><van-icon name="delete-o" size="20" @click="removeItem(it)" /></div>
          </div>
        </div>
        <template #right>
          <div class="swipe-actions">
            <div class="swipe-btn swipe-fav" @click="favoriteItem(it)">收藏</div>
            <div class="swipe-btn swipe-del" @click="removeItem(it)">删除</div>
          </div>
        </template>
      </van-swipe-cell>
      <!-- 搭配购买 (cart companion suggestions): "买了A的人还买了" -->
      <div v-if="companionSuggestions.length" class="companion-section">
        <div class="comp-head">🛍 搭配购买</div>
        <div v-for="(c, idx) in companionSuggestions" :key="idx" class="comp-item">
          <div class="comp-anchor">
            <van-image width="44" height="44" radius="6" :src="c.cartItem.product_image" fit="cover" />
            <span class="comp-anchor-name van-ellipsis">{{ c.cartItem.product_name }}</span>
          </div>
          <div class="comp-arrow">→</div>
          <div class="comp-pick" @click="router.push('/product/' + c.product.id)">
            <van-image width="44" height="44" radius="6" :src="c.product.image" fit="cover" />
            <div class="comp-pick-info">
              <span class="comp-pick-name van-ellipsis">买了它的人还买了</span>
              <span class="comp-pick-title van-ellipsis">{{ c.product.name }}</span>
              <span class="comp-pick-price">¥{{ fmt(c.product.price) }}</span>
            </div>
            <van-button size="mini" type="danger" round @click.stop="quickAddCompanion(c.product)">加入</van-button>
          </div>
        </div>
      </div>
      <!-- Coupon selector -->
      <van-cell
        title="优惠券"
        :value="selectedCouponId ? couponLabel(usableCoupons.find((c) => c.id === selectedCouponId)) : (usableCoupons.length ? usableCoupons.length + '张可用' : '暂无可用')"
        is-link
        @click="showCouponPicker = true"
        style="margin-top: 8px"
      />
      <van-field v-model="remark" label="订单备注" placeholder="选填，如送货时间、发票等" style="margin-top: 8px" />
      <!-- 感谢快递员小费 (checkout courier tip): quick-tip buttons → remark -->
      <div class="tip-row">
        <span class="tip-label">💰 感谢快递员</span>
        <div class="tip-btns">
          <span
            v-for="amt in TIP_AMOUNTS"
            :key="amt"
            class="tip-btn"
            :class="{ active: selectedTip === amt }"
            @click="applyTip(amt)"
          >¥{{ amt }}</span>
        </div>
      </div>
      <!-- Address picker (收货地址) -->
      <van-cell
        title="收货地址"
        :value="addressLabel()"
        is-link
        @click="showAddressPicker = true"
        style="margin-top: 8px"
      />
      <!-- 地址地图预览 (checkout map preview): a CSS-only faux map showing the
           selected delivery address with a 📍 pin overlay and a "查看大地图" toast. -->
      <div class="addr-map" @click="showToast('查看大地图')">
        <div class="am-canvas">
          <div class="am-grid"></div>
          <div class="am-road am-road-h"></div>
          <div class="am-road am-road-v"></div>
          <div class="am-river"></div>
          <div class="am-pin">📍
            <div class="am-pin-pulse"></div>
          </div>
        </div>
        <div class="am-overlay">
          <div class="am-line">
            <span class="am-name">{{ selectedAddress ? selectedAddress.name : '收货人' }}</span>
            <span class="am-phone">{{ selectedAddress ? selectedAddress.phone : '' }}</span>
          </div>
          <div class="am-addr van-ellipsis">{{ addressLabel() }}</div>
          <span class="am-bigbtn">查看大地图 ›</span>
        </div>
      </div>
      <!-- Invoice cell (发票) -->
      <van-cell
        title="发票"
        :value="invoiceLabel()"
        is-link
        @click="openInvoiceForm"
        style="margin-top: 8px"
      />
      <!-- Top-up hints (凑单提示) -->
      <div v-if="topupHints.length" class="topup-hints">
        <div v-for="h in topupHints.slice(0, 2)" :key="h.id" class="th-item">💡 再买 <b>¥{{ h.diff }}</b> 可使用 {{ h.label }}，<span class="th-go" @click="router.push('/home')">去凑单 ›</span></div>
      </div>
      <div v-if="discount > 0" class="discount-line">优惠券抵扣 -¥{{ fmt(discount) }}</div>
      <!-- Tiered discount banner (阶梯满减) -->
      <div v-if="tierSummary" class="tier-banner">
        <span class="tier-icon">🎁</span>
        <span class="tier-summary">{{ tierSummary }}</span>
        <span v-if="reachedTier" class="tier-met">✓ 已达满减</span>
        <span v-else-if="nextTier" class="tier-next">再买<b>¥{{ nextTier.diff }}</b>减¥{{ nextTier.discount }}</span>
      </div>
      <!-- Smart quantity / 凑单 quick-add (智能数量) -->
      <div v-if="reachedTier" class="smart-met">
        <span class="sm-check">✓</span>
        <span class="sm-text">已达满减 满{{ Math.round(reachedTier.threshold) }}减{{ Math.round(reachedTier.discount) }}</span>
      </div>
      <div v-else-if="nextTier && topupPicks.length" class="smart-topup">
        <div class="st-head">💡 再买 <b>¥{{ nextTier.diff }}</b> 即可凑满减</div>
        <div class="st-scroll">
          <div v-for="p in topupPicks" :key="p.id" class="st-card" @click="quickTopup(p)">
            <van-image width="70" height="70" radius="6" :src="p.image" fit="cover" />
            <div class="st-name van-ellipsis">{{ p.name }}</div>
            <div class="st-price">¥{{ fmt(p.price) }}</div>
            <div class="st-add">+ 凑单</div>
          </div>
        </div>
      </div>
      <!-- 预估重量 (cart weight estimator) for the selected items -->
      <div class="weight-bar" :class="{ heavy: isHeavyOrder }">
        <span class="wb-text">📦 预估重量: {{ estimatedWeight.toFixed(1) }}kg</span>
        <span v-if="isHeavyOrder" class="wb-warn">⚠️ 大件商品，建议联系物流或拆分下单</span>
      </div>
      <van-submit-bar :price="finalTotal * 100" :button-text="'结算 (' + selectedCount + '件)'" @submit="checkout">
        <van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox>
        <span class="invert-btn" @click="invertSelection">反选</span>
        <span class="group-btn" @click="openGroupBuy">👥 邀请拼单</span>
        <span class="shake-btn" @click="setupShake">📳 摇一摇</span>
        <span class="selected-count">已选{{ selectedCount }}件</span>
      </van-submit-bar>

      <van-popup v-model:show="showCouponPicker" position="bottom" round>
        <div class="coupon-picker">
          <div class="cp-head">选择优惠券</div>
          <div v-if="!usableCoupons.length" class="cp-empty">暂无可用优惠券</div>
          <div v-for="uc in usableCoupons" :key="uc.id" class="cp-item" :class="{ active: selectedCouponId === uc.id }" @click="selectedCouponId = (selectedCouponId === uc.id ? null : uc.id); showCouponPicker = false">
            <div class="cp-value" v-if="uc.coupon.coupon_type === 'discount'">{{ (uc.coupon.value * 10).toFixed(1) }}折</div>
            <div class="cp-value" v-else>¥{{ uc.coupon.value }}</div>
            <div class="cp-info"><div class="cp-title">{{ uc.coupon.title }}</div><div class="cp-sub">{{ couponLabel(uc) }}</div></div>
          </div>
        </div>
      </van-popup>

      <!-- Address picker popup (收货地址) -->
      <van-popup v-model:show="showAddressPicker" position="bottom" round>
        <div class="addr-picker">
          <div class="cp-head">选择收货地址</div>
          <div v-if="!addresses.length" class="cp-empty">暂无收货地址，请先在地址管理中添加</div>
          <div
            v-for="a in addresses"
            :key="a.id"
            class="addr-item"
            :class="{ active: selectedAddressId === a.id }"
            @click="pickAddress(a)"
          >
            <div class="addr-top"><span class="addr-name">{{ a.name }}</span><span class="addr-phone">{{ a.phone }}</span><van-tag v-if="a.is_default === 1" type="danger" size="mini">默认</van-tag></div>
            <div class="addr-detail">{{ a.detail }}</div>
          </div>
        </div>
      </van-popup>

      <!-- Invoice form popup (发票) -->
      <van-popup v-model:show="showInvoiceForm" position="bottom" round :style="{ maxHeight: '80%' }">
        <div class="invoice-form">
          <div class="cp-head">开具发票</div>
          <van-cell-group inset>
            <van-field label="是否开票">
              <template #input>
                <van-switch v-model="invoiceEnabled" />
              </template>
            </van-field>
            <template v-if="invoiceEnabled">
              <van-field label="发票抬头类型" is-link readonly :model-value="invoiceType === 'company' ? '单位' : '个人'" @click="invoiceType = (invoiceType === 'company' ? 'personal' : 'company')" />
              <van-field v-model="invoiceTitle" :label="invoiceType === 'company' ? '单位名称' : '抬头名称'" :placeholder="invoiceType === 'company' ? '请输入单位名称' : '选填，如个人姓名'" />
              <van-field v-if="invoiceType === 'company'" v-model="invoiceTaxNo" label="税号" placeholder="选填，单位税号" />
              <van-field v-model="invoiceEmail" label="电子邮箱" placeholder="选填，发票将发送至该邮箱" />
            </template>
          </van-cell-group>
          <div class="inv-actions">
            <van-button block @click="closeInvoiceForm">取消</van-button>
            <van-button block type="danger" @click="saveInvoice">确定</van-button>
          </div>
        </div>
      </van-popup>

      <!-- Group buy invite popup (好友拼单邀请) -->
      <van-popup v-model:show="showGroupBuy" round closeable close-icon-position="top-right" :style="{ width: '86%', maxHeight: '85%' }">
        <div class="gb-popup">
          <div class="gb-hero">
            <div class="gb-title">邀请好友一起拼单</div>
            <div class="gb-sub">🎉 拼着买 更划算</div>
          </div>
          <div class="gb-counter">已邀请 <b>{{ invitedCount }}</b> 人</div>
          <div class="gb-card">
            <div class="gb-card-line">🛒 {{ selectedCount }} 件好物</div>
            <div class="gb-card-line">💰 合计 ¥{{ fmt(selectedTotal) }}</div>
          </div>
          <div class="gb-invite-text">{{ inviteText }}</div>
          <div class="gb-actions">
            <van-button block round type="danger" @click="copyInvite">复制邀请</van-button>
            <van-button block round plain type="danger" @click="shareInvite">系统分享</van-button>
          </div>
          <div class="gb-demo" @click="addInvitee">+ 模拟一位好友加入拼单</div>
        </div>
      </van-popup>
    </div>

    <!-- 下单抽奖 (order lucky draw) popup -->
    <van-popup v-model:show="showLuckyDraw" round closeable close-icon-position="top-right" :style="{ width: '88%' }" :close-on-click-overlay="false" @click-close-icon="closeLuckyDraw">
      <div class="ld-popup">
        <div class="ld-hero">
          <div class="ld-title">🎁 幸运抽奖</div>
          <div class="ld-sub">下单成功！转一转赢好礼</div>
        </div>
        <!-- 3-column slot machine -->
        <div class="ld-machine">
          <div v-for="(reel, c) in slotReels" :key="c" class="ld-col">
            <div
              class="ld-strip"
              :class="{ 'ld-spinning': drawSpinning && slotCols[c] === null, 'ld-stopped': slotCols[c] !== null }"
              :style="{ transform: `translateY(-${slotOffsets[c] * (100 / (reel.length || 1))}%)` }"
            >
              <div v-for="(sym, r) in reel" :key="r" class="ld-symbol">{{ SLOT_SYMBOLS[sym] }}</div>
            </div>
          </div>
        </div>
        <div class="ld-points">我的积分：{{ drawPoints }}</div>
        <!-- Result banner -->
        <div v-if="drawResult" class="ld-result" :class="{ win: drawResult.win }">
          <template v-if="drawResult.win">🎉 恭喜中奖：{{ drawResult.prize }}</template>
          <template v-else>😅 {{ drawResult.prize }}</template>
        </div>
        <!-- Controls: first spin is free; replays cost 积分 -->
        <div class="ld-actions">
          <van-button v-if="!drawResult" block round type="danger" :loading="drawSpinning" @click="startDraw(false)">{{ drawSpinning ? '转动中...' : '免费抽奖' }}</van-button>
          <template v-else>
            <van-button block round type="danger" @click="startDraw(true)">再来一次 ({{ REPLAY_COST }}积分)</van-button>
            <van-button block round plain @click="closeLuckyDraw(); router.push('/orders')">查看订单</van-button>
          </template>
        </div>
        <div class="ld-rules">3个相同=88VIP折扣券 · 2个相同=10元券 · 其他=谢谢参与</div>
      </div>
    </van-popup>

    <!-- 结算错误恢复 (checkout error recovery) dialog -->
    <van-dialog
      v-model:show="showCheckoutError"
      :title="checkoutMaxedOut ? '结算失败' : (checkoutIsTimeout ? '网络超时' : '商品信息已更新')"
      :show-confirm-button="false"
      :close-on-click-overlay="false"
    >
      <div class="co-error">
        <div class="co-msg">{{ checkoutErrorMsg }}</div>
        <div v-if="checkoutIsTimeout && !checkoutMaxedOut" class="co-countdown">{{ checkoutCountdown }}秒后自动重试...</div>
        <div v-if="checkoutMaxedOut" class="co-countdown co-contact">请联系客服</div>
        <div v-if="!checkoutMaxedOut" class="co-retries">剩余重试次数 {{ MAX_CHECKOUT_RETRIES - checkoutRetries }}/{{ MAX_CHECKOUT_RETRIES }}</div>
      </div>
      <div class="co-actions">
        <van-button block @click="closeCheckoutError">取消</van-button>
        <van-button v-if="checkoutMaxedOut" block type="danger" @click="showDialog({ title: '联系客服', message: '客服热线：400-888-XXXX' }); closeCheckoutError()">联系客服</van-button>
        <van-button v-else block type="danger" @click="retryCheckout">重试</van-button>
      </div>
    </van-dialog>
    <!-- Confetti host (彩纸) for lucky draw winners -->
    <div id="tm-confetti" class="tm-confetti-host"></div>
  </div>
</template>

<style scoped>
.cart-page { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.invert-btn { font-size: 14px; color: #ff0036; margin-left: 12px; cursor: pointer; }
.group-btn { font-size: 13px; color: #fff; background: linear-gradient(135deg, #ff0036, #ff5a5f); padding: 5px 12px; margin-left: 10px; border-radius: 999px; cursor: pointer; white-space: nowrap; box-shadow: 0 2px 6px rgba(255, 0, 54, 0.35); }
/* 摇一摇 (shake to undo) trigger button */
.shake-btn { font-size: 13px; color: #ff0036; background: #fff5f6; border: 1px solid #ffc6cf; padding: 4px 12px; margin-left: 10px; border-radius: 999px; cursor: pointer; white-space: nowrap; }
.selected-count { font-size: 12px; color: #969799; margin-left: 12px; }
.cart-item { display: flex; align-items: center; gap: 10px; padding: 12px; background: #fff; border-bottom: 1px solid #f5f5f5; }
/* 购物车滑动操作 (swipe actions) */
.swipe-actions { display: flex; height: 100%; }
.swipe-btn { display: flex; align-items: center; justify-content: center; width: 60px; color: #fff; font-size: 14px; cursor: pointer; }
.swipe-fav { background: #ff976a; }
.swipe-del { background: #ee0a24; }
.ci-info { flex: 1; }
.ci-name { font-size: 13px; line-height: 18px; height: 36px; }
.ci-bottom { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.ci-bottom .price { font-size: 16px; flex: 1; }
.discount-line { color: #ff0036; font-size: 13px; text-align: right; padding: 6px 16px; background: #fff; }
/* 感谢快递员小费 (checkout courier tip) */
.tip-row { display: flex; align-items: center; gap: 10px; padding: 10px 16px; background: #fff; margin-top: 8px; }
.tip-label { font-size: 13px; color: #333; flex-shrink: 0; }
.tip-btns { display: flex; gap: 8px; }
.tip-btn { font-size: 13px; font-weight: bold; color: #ff0036; background: #fff5f6; border: 1px solid #ffd6df; border-radius: 999px; padding: 5px 16px; cursor: pointer; user-select: none; transition: all 0.15s; }
.tip-btn:active { transform: scale(0.94); }
.tip-btn.active { background: #ff0036; color: #fff; border-color: #ff0036; box-shadow: 0 2px 6px rgba(255, 0, 54, 0.3); }
.tier-banner { display: flex; align-items: center; gap: 8px; background: linear-gradient(90deg, #fff0f3, #fff6e6); color: #ff0036; font-size: 12px; padding: 8px 12px; margin: 8px 0; border-radius: 8px; line-height: 18px; }
.tier-icon { font-size: 14px; }
.tier-summary { flex: 1; font-weight: bold; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tier-next { color: #996600; flex-shrink: 0; }
.tier-next b { color: #ff0036; }
.tier-met { color: #07c160; font-weight: bold; flex-shrink: 0; }
/* Smart quantity (智能凑单) */
.smart-met { display: flex; align-items: center; gap: 6px; margin: 0 16px 8px; padding: 8px 12px; background: #e6f9ee; border: 1px solid #b6e8c9; border-radius: 8px; }
.sm-check { color: #07c160; font-size: 16px; font-weight: bold; }
.sm-text { font-size: 13px; color: #07c160; font-weight: bold; }
.smart-topup { margin: 0 0 8px; background: linear-gradient(90deg, #fff8e6, #fff0f3); padding: 10px 0 12px; }
.st-head { font-size: 13px; color: #996600; padding: 0 16px 8px; }
.st-head b { color: #ff0036; }
.st-scroll { display: flex; gap: 10px; overflow-x: auto; padding: 0 16px; -webkit-overflow-scrolling: touch; }
.st-scroll::-webkit-scrollbar { display: none; }
.st-card { flex: 0 0 auto; width: 78px; text-align: center; }
.st-name { font-size: 11px; color: #333; margin-top: 4px; }
.st-price { font-size: 12px; color: #ff0036; font-weight: bold; }
.st-add { font-size: 11px; color: #fff; background: linear-gradient(135deg, #ff0036, #ff5a5f); border-radius: 10px; padding: 3px 0; margin-top: 4px; }
/* 预估重量 (cart weight estimator) */
.weight-bar { display: flex; align-items: center; gap: 8px; margin: 0 16px 8px; padding: 8px 12px; background: #f0f7ff; border: 1px solid #cfe3f7; border-radius: 8px; font-size: 13px; color: #1989fa; }
.weight-bar.heavy { background: #fff5f6; border-color: #ffd6df; color: #ff0036; }
.wb-text { font-weight: bold; }
.wb-warn { margin-left: auto; font-size: 12px; }
.topup-hints { padding: 0 16px 8px; }
.th-item { background: #fff8e6; color: #996600; font-size: 12px; padding: 8px 12px; border-radius: 8px; margin-bottom: 6px; line-height: 18px; }
.th-item b { color: #ff0036; }
.th-go { color: #ff0036; font-weight: bold; }
.coupon-picker { padding: 16px; }
.cp-head { text-align: center; font-size: 15px; font-weight: bold; margin-bottom: 12px; }
.cp-empty { text-align: center; color: #999; padding: 30px; }
.cp-item { display: flex; align-items: center; gap: 12px; padding: 12px; border: 1px solid #eee; border-radius: 8px; margin-bottom: 8px; }
.cp-item.active { border-color: #ff0036; background: #fff0f3; }
.cp-value { color: #ff0036; font-size: 22px; font-weight: bold; min-width: 60px; text-align: center; }
.cp-info { flex: 1; }
.cp-title { font-size: 14px; }
.cp-sub { font-size: 12px; color: #999; margin-top: 2px; }
.addr-picker { padding: 16px; }
.addr-item { padding: 12px; border: 1px solid #eee; border-radius: 8px; margin-bottom: 8px; }
.addr-item.active { border-color: #ff0036; background: #fff0f3; }
.addr-top { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.addr-name { font-size: 14px; font-weight: bold; }
.addr-phone { font-size: 13px; color: #666; }
.addr-detail { font-size: 12px; color: #999; line-height: 18px; }
/* 地址地图预览 (checkout map preview): CSS gradient faux map with pin */
.addr-map { position: relative; margin: 8px 0; height: 120px; border-radius: 10px; overflow: hidden; cursor: pointer; }
.am-canvas { position: absolute; inset: 0; background:
  linear-gradient(115deg, #e8f4e8 0%, #e3f0f7 45%, #f0eadc 100%); }
/* Faint block grid to read like a city map. */
.am-grid { position: absolute; inset: 0; background-image:
  linear-gradient(rgba(120,140,120,0.12) 1px, transparent 1px),
  linear-gradient(90deg, rgba(120,140,120,0.12) 1px, transparent 1px);
  background-size: 26px 26px; }
/* Roads drawn as light-gray ribbons crossing the map. */
.am-road { position: absolute; background: rgba(255,255,255,0.85); box-shadow: 0 0 0 1px rgba(0,0,0,0.04); }
.am-road-h { left: -10%; right: -10%; top: 46%; height: 14px; transform: rotate(-6deg); }
.am-road-v { top: -10%; bottom: -10%; left: 62%; width: 12px; transform: rotate(8deg); }
/* A blue river ribbon for variety. */
.am-river { position: absolute; left: -10%; right: -10%; top: 18%; height: 10px; background: linear-gradient(90deg, #a9d6f0, #cfe8f7); transform: rotate(-4deg); opacity: 0.8; }
.am-pin { position: absolute; left: 58%; top: 40%; font-size: 30px; transform: translate(-50%, -100%); z-index: 3; line-height: 1; filter: drop-shadow(0 2px 3px rgba(0,0,0,0.25)); }
.am-pin-pulse { position: absolute; left: 50%; bottom: -4px; width: 14px; height: 14px; margin-left: -7px; border-radius: 50%; background: rgba(255,0,54,0.35); animation: am-pulse 1.6s ease-out infinite; }
@keyframes am-pulse { 0% { transform: scale(0.6); opacity: 0.8; } 100% { transform: scale(2.6); opacity: 0; } }
.am-overlay { position: absolute; left: 10px; right: 10px; bottom: 10px; z-index: 4; background: rgba(255,255,255,0.94); border-radius: 8px; padding: 7px 10px; box-shadow: 0 2px 8px rgba(0,0,0,0.12); }
.am-line { display: flex; align-items: center; gap: 8px; }
.am-name { font-size: 13px; font-weight: bold; color: #333; }
.am-phone { font-size: 12px; color: #999; }
.am-addr { font-size: 11px; color: #666; margin-top: 2px; }
.am-bigbtn { position: absolute; right: 10px; bottom: 8px; font-size: 11px; color: #ff0036; font-weight: bold; }
.invoice-form { padding: 16px 0; }
.inv-actions { display: flex; gap: 12px; padding: 16px; }
/* Recommendations (猜你喜欢) */
.rec-section { background: #fff; margin: 0 0 8px; padding: 10px 0 12px; }
.rec-head { font-size: 15px; font-weight: bold; color: #333; padding: 0 12px 8px; }
.rec-scroll { display: flex; gap: 10px; overflow-x: auto; padding: 0 12px; -webkit-overflow-scrolling: touch; }
.rec-scroll::-webkit-scrollbar { display: none; }
.rec-card { flex: 0 0 auto; width: 110px; }
.rec-name { font-size: 12px; color: #333; line-height: 16px; height: 32px; margin-top: 6px; }
.rec-bottom { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; margin-top: 4px; }
.rec-bottom .price { color: #ff0036; font-size: 14px; font-weight: bold; }
/* 搭配购买 (cart companion suggestions) */
.companion-section { background: #fff; margin: 8px 0; padding: 12px; }
.comp-head { font-size: 15px; font-weight: bold; color: #333; margin-bottom: 10px; }
.comp-item { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-top: 1px solid #f5f5f5; }
.comp-item:first-of-type { border-top: none; }
.comp-anchor { display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; width: 60px; }
.comp-anchor-name { font-size: 10px; color: #999; max-width: 60px; text-align: center; }
.comp-arrow { color: #ccc; font-size: 16px; flex-shrink: 0; }
.comp-pick { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0; }
.comp-pick-info { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.comp-pick-name { font-size: 11px; color: #ff0036; }
.comp-pick-title { font-size: 12px; color: #333; line-height: 16px; }
.comp-pick-price { font-size: 14px; color: #ff0036; font-weight: bold; }
/* Group buy invite popup (好友拼单邀请) */
.gb-popup { padding: 0 0 18px; overflow: hidden; }
.gb-hero { background: linear-gradient(135deg, #ff0036 0%, #ff5577 50%, #ffb347 100%); color: #fff; padding: 26px 18px 20px; text-align: center; }
.gb-title { font-size: 19px; font-weight: bold; letter-spacing: 0.5px; }
.gb-sub { font-size: 13px; opacity: 0.95; margin-top: 6px; }
.gb-counter { text-align: center; color: #ff0036; font-size: 14px; margin: 14px 0 4px; }
.gb-counter b { font-size: 18px; }
.gb-card { margin: 10px 18px; padding: 12px 14px; background: #fff5f6; border: 1px dashed #ffb3c0; border-radius: 10px; }
.gb-card-line { font-size: 14px; color: #333; line-height: 24px; }
.gb-invite-text { margin: 4px 18px 12px; padding: 12px; background: #f7f8fa; border-radius: 8px; font-size: 13px; color: #555; line-height: 20px; white-space: pre-wrap; word-break: break-all; }
.gb-actions { display: flex; gap: 12px; padding: 0 18px; }
.gb-demo { text-align: center; font-size: 12px; color: #ff0036; margin-top: 14px; cursor: pointer; }
/* 结算错误恢复 (checkout error recovery) */
.co-error { padding: 8px 20px 4px; text-align: center; }
.co-msg { font-size: 15px; color: #333; line-height: 22px; }
.co-countdown { font-size: 13px; color: #ff0036; margin-top: 8px; }
.co-countdown.co-contact { font-size: 15px; font-weight: bold; }
.co-retries { font-size: 12px; color: #999; margin-top: 6px; }
.co-actions { display: flex; gap: 12px; padding: 16px 20px 20px; }
/* 下单抽奖 (order lucky draw) */
.ld-popup { padding: 0 0 18px; overflow: hidden; }
.ld-hero { background: linear-gradient(135deg, #ff0036 0%, #ff5577 50%, #ffb347 100%); color: #fff; padding: 24px 18px 18px; text-align: center; }
.ld-title { font-size: 20px; font-weight: bold; letter-spacing: 0.5px; }
.ld-sub { font-size: 13px; opacity: 0.95; margin-top: 6px; }
.ld-machine { display: flex; gap: 8px; justify-content: center; margin: 20px 16px; background: #222; border-radius: 12px; padding: 10px; box-shadow: inset 0 2px 8px rgba(0,0,0,0.5); }
.ld-col { width: 64px; height: 64px; overflow: hidden; border-radius: 8px; background: #fff; position: relative; }
.ld-symbol { height: 64px; line-height: 64px; text-align: center; font-size: 38px; }
/* The strip is one symbol tall via the column's overflow; we translate by whole
   symbol heights (slotOffsets * 100% of a 64px symbol). */
.ld-strip { will-change: transform; }
/* While spinning we cycle rapidly; once a column stops we ease into place. */
.ld-strip.ld-spinning { animation: ld-spin 0.12s steps(1) infinite; }
.ld-strip.ld-stopped { transition: transform 0.5s cubic-bezier(0.15, 0.85, 0.25, 1); }
/* Scroll one symbol (64px) per step for a crisp slot-machine blur effect. */
@keyframes ld-spin { 0% { transform: translateY(0); } 100% { transform: translateY(-64px); } }
.ld-points { text-align: center; font-size: 13px; color: #ff0036; margin: -6px 0 10px; }
.ld-result { text-align: center; font-size: 16px; font-weight: bold; color: #666; padding: 6px 0 14px; }
.ld-result.win { color: #ff0036; font-size: 18px; animation: ld-pop 0.4s ease; }
@keyframes ld-pop { 0% { transform: scale(0.6); opacity: 0; } 60% { transform: scale(1.15); } 100% { transform: scale(1); opacity: 1; } }
.ld-actions { display: flex; flex-direction: column; gap: 10px; padding: 0 18px; }
.ld-rules { text-align: center; font-size: 11px; color: #999; margin-top: 12px; line-height: 18px; }
/* Confetti host (彩纸) — fixed full-screen, pointer-events-none. */
.tm-confetti-host { position: fixed; inset: 0; pointer-events: none; z-index: 10000; overflow: hidden; }
.tm-confetti-chip { position: absolute; top: -12px; width: 9px; height: 14px; border-radius: 2px; opacity: 0; animation: tm-confetti-fall 2.2s ease-in forwards; }
@keyframes tm-confetti-fall {
  0% { opacity: 1; transform: translateY(0) rotate(0deg); }
  100% { opacity: 0; transform: translateY(105vh) rotate(720deg); }
}
/* 购物车空状态 (cart empty state): animated emoji bag + recommendations */
.cart-empty { display: flex; flex-direction: column; align-items: center; padding: 48px 16px 16px; background: linear-gradient(180deg, #fff5f6 0%, #fff 220px); }
.ce-bag-wrap { position: relative; width: 120px; height: 120px; display: flex; align-items: center; justify-content: center; }
.ce-bag { font-size: 88px; line-height: 1; display: inline-block; transform-origin: bottom center; animation: ce-bag-bounce 2.2s ease-in-out infinite; filter: drop-shadow(0 6px 8px rgba(255, 0, 54, 0.18)); }
.ce-bag-shadow { position: absolute; bottom: -8px; left: 50%; transform: translateX(-50%); width: 70px; height: 10px; background: rgba(0, 0, 0, 0.1); border-radius: 50%; animation: ce-shadow 2.2s ease-in-out infinite; }
@keyframes ce-bag-bounce { 0%, 100% { transform: translateY(0) rotate(-3deg); } 50% { transform: translateY(-14px) rotate(3deg); } }
@keyframes ce-shadow { 0%, 100% { width: 70px; opacity: 0.18; } 50% { width: 46px; opacity: 0.08; } }
.ce-title { font-size: 17px; font-weight: bold; color: #333; margin-top: 24px; }
.ce-sub { font-size: 13px; color: #999; margin-top: 8px; text-align: center; line-height: 20px; }
.ce-go-btn { margin-top: 18px; padding: 0 28px; height: 38px; font-size: 15px; }
/* 为你推荐 (recommendations when empty) */
.ce-recommend { width: 100%; margin-top: 32px; }
.ce-rec-head { display: flex; align-items: center; justify-content: center; gap: 12px; margin-bottom: 14px; }
.ce-rec-line { width: 32px; height: 1px; background: #ddd; }
.ce-rec-title { font-size: 15px; font-weight: bold; color: #333; }
.ce-rec-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.ce-rec-card { background: #fff; border-radius: 10px; overflow: hidden; padding-bottom: 6px; box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05); cursor: pointer; transition: transform 0.15s; }
.ce-rec-card:active { transform: scale(0.98); }
.ce-rec-name { font-size: 13px; line-height: 18px; height: 36px; padding: 6px 8px 0; color: #333; }
.ce-rec-bottom { display: flex; align-items: center; justify-content: space-between; padding: 6px 8px 2px; }
.ce-rec-price { color: #ff0036; font-size: 16px; font-weight: bold; }
</style>
