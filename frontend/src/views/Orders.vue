<script setup>
import { ref, computed, onMounted, onUnmounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showLoadingToast, closeToast } from 'vant'
import { getOrders, payOrder, createRefund, confirmOrder, cancelOrder, addToCart } from '../api'
// Repurchase loading state (一键回购): prevent double-clicks and show per-button spinner.
const repurchasing = ref(false)

const router = useRouter()
const orders = ref([])
const loading = ref(true)

// ---- 订单状态筛选 (order status filter tabs) ----
// 全部|待付款|待发货|待收货|已完成. Counts are computed live from the order
// list; the active tab drives `filteredOrders`. The 待发货 (paid) and 待收货
// (shipped) bands map directly to backend statuses.
const activeStatus = ref('all')
const statusTabs = computed(() => [
  { key: 'all', label: '全部', count: orders.value.length },
  { key: 'pending', label: '待付款', count: orders.value.filter((o) => o.status === 'pending').length },
  { key: 'paid', label: '待发货', count: orders.value.filter((o) => o.status === 'paid').length },
  { key: 'shipped', label: '待收货', count: orders.value.filter((o) => o.status === 'shipped').length },
  { key: 'completed', label: '已完成', count: orders.value.filter((o) => o.status === 'completed').length },
])
const filteredOrders = computed(() => {
  if (activeStatus.value === 'all') return orders.value
  return orders.value.filter((o) => o.status === activeStatus.value)
})

// After-sale application (售后换货): user first picks a service type, then enters a reason.
const refundTypeActions = [
  { name: '仅退款', value: 'refund_only' },
  { name: '退货退款', value: 'return_refund' },
  { name: '换货', value: 'exchange' },
]
const showRefundSheet = ref(false)
let refundOrder = null

// Order timeout countdown (订单超时倒计时).
// Pending orders expire 30 minutes after created_at.
const ORDER_TIMEOUT_MS = 30 * 60 * 1000
const now = ref(Date.now())
let timer = null
function startTimer() {
  stopTimer()
  timer = setInterval(() => { now.value = Date.now() }, 1000)
}
function stopTimer() { if (timer) { clearInterval(timer); timer = null } }
// Remaining ms for a pending order; <=0 means timed out.
function remainingMs(o) {
  if (o.status !== 'pending' || !o.created_at) return 0
  const created = new Date(o.created_at).getTime()
  if (isNaN(created)) return 0
  return created + ORDER_TIMEOUT_MS - now.value
}
function countdown(o) {
  const ms = remainingMs(o)
  if (ms <= 0) return '已超时'
  const total = Math.floor(ms / 1000)
  const m = String(Math.floor(total / 60)).padStart(2, '0')
  const s = String(total % 60).padStart(2, '0')
  return `剩余 ${m}:${s}`
}
function isTimedOut(o) { return o.status === 'pending' && remainingMs(o) <= 0 }

onMounted(async () => {
  startTimer()
  try { orders.value = await getOrders() } catch (e) { showToast('加载失败') } finally { loading.value = false }
})
onActivated(startTimer)
onUnmounted(stopTimer)
function pay(o) { router.push({ name: 'pay', query: { order_id: o.id } }) }
async function confirm(o) {
  try { await confirmOrder(o.id); o.status = 'completed'; showSuccessToast('确认收货成功') }
  catch (e) { showToast(e.response?.data?.error || '操作失败') }
}
async function cancel(o) {
  try { await cancelOrder(o.id); o.status = 'cancelled'; showSuccessToast('订单已取消') }
  catch (e) { showToast(e.response?.data?.error || '操作失败') }
}
function viewLogistics(o) { router.push({ name: 'logistics', query: { order_id: o.id } }) }
// One-click repurchase (一键回购): re-add all items of a completed order to the cart.
// Parses items_json, extracts product_id + quantity, calls addToCart for each, then jumps to /cart.
async function repurchase(o) {
  const items = parseItems(o.items_json)
  // Only items carrying a product_id can be re-ordered.
  const buyable = items.filter((it) => it.product_id)
  if (!buyable.length) { showToast('无可回购商品'); return }
  if (repurchasing.value) return
  repurchasing.value = true
  showLoadingToast({ message: '加入购物车中...', forbidClick: true, duration: 0 })
  try {
    for (const it of buyable) {
      const qty = Number(it.quantity) > 0 ? Number(it.quantity) : 1
      await addToCart(it.product_id, qty)
    }
    closeToast()
    showSuccessToast('已加入购物车')
    router.push('/cart')
  } catch (e) {
    closeToast()
    showToast(e.response?.data?.error || '回购失败')
  } finally {
    repurchasing.value = false
  }
}
async function applyRefund(o) {
  refundOrder = o
  showRefundSheet.value = true
}
async function onSelectRefundType(action) {
  showRefundSheet.value = false
  const o = refundOrder
  refundOrder = null
  if (!o) return
  const reason = window.prompt('请输入' + action.name + '原因')
  if (!reason || !reason.trim()) return
  try { await createRefund(o.id, reason, action.value); showSuccessToast('申请已提交') }
  catch (e) { showToast(e.response?.data?.error || '申请失败') }
}
function statusText(s) { return { pending: '待付款', paid: '已付款', shipped: '已发货', completed: '已完成', cancelled: '已取消' }[s] || s }
function fmt(n) { return Number(n).toFixed(2) }
function parseItems(json) { try { return JSON.parse(json) } catch { return [] } }

// ---- 评价提醒 (review reminder) ----
// Completed orders that the user hasn't yet reviewed surface a pulsing
// "📝写评价赚积分" badge. The backend doesn't link reviews to orders, so we
// treat a completed order as "needs review" until the reminder is dismissed
// or the user opens the review flow. Dismissed order ids are remembered in
// localStorage so the nag stays quiet within and across sessions.
const REVIEW_DISMISS_KEY = 'tm_review_dismissed'
const reviewedDismissed = ref(loadDismissed())
function loadDismissed() {
  try { return new Set(JSON.parse(localStorage.getItem(REVIEW_DISMISS_KEY) || '[]')) } catch { return new Set() }
}
function saveDismissed() {
  try { localStorage.setItem(REVIEW_DISMISS_KEY, JSON.stringify([...reviewedDismissed.value])) } catch { /* ignore */ }
}
// An order needs a review reminder when it's completed and not dismissed.
function needsReview(o) {
  return o.status === 'completed' && !reviewedDismissed.value.has(o.id)
}
// "稍后" dismiss: remember the order id so the badge stays gone.
function dismissReview(o) {
  reviewedDismissed.value.add(o.id)
  reviewedDismissed.value = new Set(reviewedDismissed.value)
  saveDismissed()
  showToast('稍后提醒')
}
// "+5积分" incentive: jump to the first product's detail to write a review,
// and mark this order as handled so the badge clears.
function goReview(o) {
  reviewedDismissed.value.add(o.id)
  reviewedDismissed.value = new Set(reviewedDismissed.value)
  saveDismissed()
  const items = parseItems(o.items_json)
  const pid = items.length ? items[0].product_id : null
  if (pid) router.push('/product/' + pid)
  else showSuccessToast('感谢支持，+5积分已到账')
}

// Order lifecycle timeline (订单全流程时间线): per-order expanded state keyed by order id.
const expanded = ref({})
function toggleProgress(o) { expanded.value[o.id] = !expanded.value[o.id] }
// Format an ISO timestamp to MM-dd HH:mm for timeline display.
function fmtTime(t) {
  if (!t) return ''
  const d = new Date(t)
  if (isNaN(d.getTime())) return ''
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}
// Build the lifecycle stages for an order. The backend model only stores
// created_at, so the reached stage is derived from the status field:
// pending→0, paid→1, shipped→2, completed→3. Only created_at has a real
// timestamp; reached later stages reuse created_at as an approximate marker
// since per-stage timestamps aren't persisted.
function timelineStages(o) {
  const statusOrder = ['pending', 'paid', 'shipped', 'completed']
  const reached = statusOrder.indexOf(o.status)
  const created = fmtTime(o.created_at)
  return [
    { icon: '📝', label: '下单', time: created, reached: reached >= 0, current: o.status !== 'cancelled' && reached === 0 },
    { icon: '💰', label: '已付', time: reached >= 1 ? created : '', reached: reached >= 1, current: reached === 1 },
    { icon: '📦', label: '已发', time: reached >= 2 ? created : '', reached: reached >= 2, current: reached === 2 },
    { icon: '✅', label: '完成', time: reached >= 3 ? created : '', reached: reached >= 3, current: reached === 3 },
  ]
}
// Group an order's items into shipment packages (拆包发货) by shop.
// Returns an array of { shop, items }. Items without a shop fall into a single group.
function packagesOf(o) {
  const items = parseItems(o.items_json)
  const order = []
  const map = {}
  for (const it of items) {
    const key = it.shop || ''
    if (!(key in map)) { map[key] = []; order.push(key) }
    map[key].push(it)
  }
  return order.map(k => ({ shop: k, items: map[k] }))
}
// Order export (订单导出): build a plain-text summary of all orders and copy
// it to the clipboard. Each order lists its number, status, total, created time
// and itemized lines. Uses navigator.clipboard with a textarea fallback so it
// works in non-secure contexts too.
function exportOrders() {
  if (!orders.value.length) { showToast('暂无订单可导出'); return }
  const lines = []
  lines.push('=========== 订单导出 ===========')
  lines.push('导出时间：' + new Date().toLocaleString('zh-CN'))
  lines.push('订单总数：' + orders.value.length + ' 笔')
  lines.push('================================')
  orders.value.forEach((o, i) => {
    lines.push('')
    lines.push(`【订单 ${i + 1}】`)
    lines.push(`订单号：${o.order_no || o.id}`)
    lines.push(`状态：${statusText(o.status)}`)
    lines.push(`合计：¥${fmt(o.total)}`)
    if (o.created_at) lines.push(`下单时间：${fmtFullTime(o.created_at)}`)
    const items = parseItems(o.items_json)
    if (items.length) {
      lines.push('商品明细：')
      items.forEach((it, j) => {
        const qty = Number(it.quantity) > 0 ? Number(it.quantity) : 1
        const price = Number(it.price) || 0
        lines.push(`  ${j + 1}. ${it.name || '-'}  ¥${price.toFixed(2)} × ${qty} = ¥${(price * qty).toFixed(2)}`)
      })
    }
    lines.push('--------------------------------')
  })
  const text = lines.join('\n')
  copyToClipboard(text)
}
function fmtFullTime(t) {
  if (!t) return ''
  const d = new Date(t)
  if (isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${mm}-${dd} ${hh}:${mi}`
}
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    showSuccessToast('订单已导出到剪贴板')
  } catch (e) {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    let ok = false
    try { ok = document.execCommand('copy') } catch { ok = false }
    document.body.removeChild(ta)
    if (ok) showSuccessToast('订单已导出到剪贴板')
    else showToast('复制失败，请重试')
  }
}
</script>

<template>
  <div class="orders-page">
    <van-nav-bar title="我的订单" left-arrow @click-left="router.back()" fixed placeholder>
      <template #right>
        <span class="export-btn" @click="exportOrders">📋 导出</span>
      </template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!orders.length" description="暂无订单" />
    <div v-else>
      <!-- 订单状态筛选 (order status filter tabs with count badges) -->
      <div class="status-tabs">
        <span
          v-for="t in statusTabs"
          :key="t.key"
          class="status-tab"
          :class="{ active: activeStatus === t.key }"
          @click="activeStatus = t.key"
        >{{ t.label }}<span v-if="t.count" class="st-count">{{ t.count }}</span></span>
      </div>
      <van-empty v-if="!filteredOrders.length" :description="'暂无' + (statusTabs.find((t) => t.key === activeStatus)?.label || '') + '订单'" />
      <div v-for="o in filteredOrders" :key="o.id" class="order-card">
        <div class="o-head"><span class="o-no">订单号: {{ o.order_no }}</span><span class="o-status">{{ statusText(o.status) }}</span></div>
        <!-- 评价提醒 (review reminder): pulsing badge for unreviewed completed orders -->
        <div v-if="needsReview(o)" class="review-reminder">
          <span class="rr-badge">📝写评价赚积分</span>
          <span class="rr-incentive">+5积分</span>
          <span class="rr-dismiss" @click="dismissReview(o)">稍后</span>
          <van-button size="mini" type="danger" round class="rr-go" @click="goReview(o)">去评价</van-button>
        </div>
        <div v-if="o.status === 'pending'" class="o-countdown" :class="{ 'timed-out': isTimedOut(o) }">{{ countdown(o) }}</div>
        <!-- Multi-package shipment (拆包发货): when an order spans 2+ shops, show per-package groups. -->
        <template v-if="packagesOf(o).length >= 2">
          <div v-for="(pkg, pi) in packagesOf(o)" :key="pi" class="pkg-block">
            <div class="pkg-head"><span class="pkg-title">📦 包裹{{ pi + 1 }}</span><span v-if="pkg.shop" class="pkg-shop van-ellipsis">{{ pkg.shop }}</span></div>
            <div v-for="(it, i) in pkg.items" :key="i" class="o-item">
              <van-image width="60" height="60" radius="6" :src="it.image" fit="cover" />
              <div class="oi-info"><div class="oi-name van-ellipsis">{{ it.name }}</div><div class="oi-price">¥{{ fmt(it.price) }} × {{ it.quantity }}</div></div>
            </div>
          </div>
        </template>
        <!-- Single shop (or no shop info): normal flat list. -->
        <template v-else>
          <div v-for="(it, i) in parseItems(o.items_json)" :key="i" class="o-item">
            <van-image width="60" height="60" radius="6" :src="it.image" fit="cover" />
            <div class="oi-info"><div class="oi-name van-ellipsis">{{ it.name }}</div><div class="oi-price">¥{{ fmt(it.price) }} × {{ it.quantity }}</div></div>
          </div>
        </template>
        <div class="o-foot">
          <span>共 {{ parseItems(o.items_json).length }} 件 合计: <b class="price">¥{{ fmt(o.total) }}</b></span>
          <div class="o-actions">
            <van-button v-if="o.status === 'pending'" size="small" type="danger" round @click="pay(o)">去支付</van-button>
            <van-button v-if="o.status === 'pending'" size="small" plain round @click="cancel(o)">取消订单</van-button>
            <van-button v-if="['shipped','completed'].includes(o.status)" size="small" type="danger" round @click="confirm(o)">确认收货</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain type="danger" round @click="viewLogistics(o)">查看物流</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain @click="applyRefund(o)">申请售后</van-button>
            <van-button v-if="o.status === 'completed'" size="small" type="danger" round :loading="repurchasing" @click="repurchase(o)">再次购买</van-button>
            <!-- Order lifecycle timeline toggle (订单全流程时间线) -->
            <van-button size="small" plain round @click="toggleProgress(o)">{{ expanded[o.id] ? '收起进度' : '查看进度' }}</van-button>
          </div>
        </div>
        <!-- Vertical lifecycle timeline; collapsed by default, current stage highlighted red. -->
        <div v-if="expanded[o.id]" class="o-timeline">
          <div v-for="(st, si) in timelineStages(o)" :key="si" class="tl-stage" :class="{ 'tl-current': st.current, 'tl-done': st.reached && !st.current }">
            <div class="tl-rail">
              <span class="tl-dot">{{ st.icon }}</span>
              <span v-if="si < timelineStages(o).length - 1" class="tl-line"></span>
            </div>
            <div class="tl-body">
              <div class="tl-label">{{ st.label }}</div>
              <div v-if="st.time" class="tl-time">{{ st.time }}</div>
              <div v-else class="tl-time tl-pending">待处理</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <van-action-sheet v-model:show="showRefundSheet" :actions="refundTypeActions" cancel-text="取消" close-on-click-action @select="onSelectRefundType" />
  </div>
</template>

<style scoped>
.orders-page { min-height: 100vh; }
/* 订单状态筛选 (order status filter tabs) */
.status-tabs { position: sticky; top: 46px; z-index: 10; display: flex; align-items: center; background: #fff; border-bottom: 1px solid #f0f0f0; overflow-x: auto; -webkit-overflow-scrolling: touch; }
.status-tabs::-webkit-scrollbar { display: none; }
.status-tab { flex-shrink: 0; position: relative; display: inline-flex; align-items: center; gap: 3px; padding: 12px 14px; font-size: 14px; color: #333; white-space: nowrap; cursor: pointer; transition: color 0.15s; }
.status-tab.active { color: #ff0036; font-weight: bold; }
.status-tab.active::after { content: ''; position: absolute; left: 50%; bottom: 6px; transform: translateX(-50%); width: 22px; height: 3px; border-radius: 2px; background: #ff0036; }
.st-count { font-size: 11px; color: #fff; background: #ff0036; border-radius: 9px; padding: 0 6px; min-width: 16px; line-height: 16px; text-align: center; }
.status-tab.active .st-count { background: #ff0036; }
/* Order export button (订单导出) */
.export-btn { font-size: 13px; color: #ff0036; font-weight: bold; padding: 4px 8px; }
.loading { text-align: center; padding: 80px; }
.order-card { background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.o-head { display: flex; justify-content: space-between; font-size: 12px; color: #999; margin-bottom: 8px; }
.o-status { color: #ff0036; }
.o-countdown { color: #ff0036; font-size: 13px; font-weight: bold; margin-bottom: 8px; }
/* 评价提醒 (review reminder): pulsing badge for unreviewed completed orders */
.review-reminder { display: flex; align-items: center; gap: 8px; margin: 8px 0; padding: 8px 12px; background: linear-gradient(90deg, #fff5f6, #ffeef0); border: 1px solid #ffd6df; border-radius: 8px; flex-wrap: wrap; }
.rr-badge { font-size: 13px; font-weight: bold; color: #ff0036; animation: rr-pulse 1.4s ease-in-out infinite; }
@keyframes rr-pulse { 0%, 100% { transform: scale(1); opacity: 1; } 50% { transform: scale(1.06); opacity: 0.8; } }
.rr-incentive { font-size: 11px; font-weight: bold; color: #fff; background: #ff9800; padding: 1px 8px; border-radius: 10px; }
.rr-dismiss { margin-left: auto; font-size: 12px; color: #999; cursor: pointer; padding: 2px 6px; }
.rr-go { }
.o-countdown.timed-out { color: #999; }
.o-item { display: flex; gap: 10px; padding: 6px 0; }
.pkg-block { border: 1px solid #ffe0e8; border-radius: 8px; padding: 8px; margin-bottom: 8px; background: #fff8fa; }
.pkg-head { display: flex; align-items: center; gap: 8px; padding-bottom: 6px; margin-bottom: 4px; border-bottom: 1px dashed #ffd6df; }
.pkg-title { font-size: 13px; font-weight: bold; color: #ff0036; }
.pkg-shop { font-size: 12px; color: #666; flex: 1; }
.oi-info { flex: 1; }
.oi-name { font-size: 13px; }
.oi-price { color: #999; font-size: 12px; margin-top: 4px; }
.o-foot { display: flex; justify-content: space-between; align-items: center; padding-top: 8px; border-top: 1px solid #f5f5f5; margin-top: 8px; font-size: 13px; }
.o-actions { display: flex; gap: 8px; }
.o-actions { display: flex; gap: 8px; }
/* Order lifecycle timeline (订单全流程时间线) */
.o-timeline { margin-top: 10px; padding: 10px 12px; background: #fafafa; border-radius: 8px; border: 1px solid #f0f0f0; }
.tl-stage { display: flex; align-items: flex-start; }
.tl-rail { display: flex; flex-direction: column; align-items: center; width: 22px; flex-shrink: 0; }
.tl-dot { width: 22px; height: 22px; border-radius: 50%; background: #fff; border: 2px solid #ddd; display: flex; align-items: center; justify-content: center; font-size: 11px; z-index: 1; }
.tl-line { flex: 1; width: 2px; min-height: 22px; background: #ddd; margin-top: 2px; }
.tl-body { padding: 0 0 14px 10px; flex: 1; }
.tl-stage:last-child .tl-body { padding-bottom: 0; }
.tl-label { font-size: 13px; color: #999; }
.tl-time { font-size: 11px; color: #bbb; margin-top: 2px; }
.tl-time.tl-pending { color: #ccc; }
.tl-done .tl-dot { border-color: #ff0036; background: #fff5f6; }
.tl-done .tl-label { color: #666; }
.tl-current .tl-dot { border-color: #ff0036; background: #ff0036; box-shadow: 0 0 0 4px rgba(255, 0, 54, 0.12); }
.tl-current .tl-label { color: #ff0036; font-weight: bold; }
.tl-current .tl-time { color: #ff0036; }
</style>
