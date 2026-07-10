<script setup>
import { ref, onMounted, onUnmounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showLoadingToast, closeToast } from 'vant'
import { getOrders, payOrder, createRefund, confirmOrder, cancelOrder, addToCart } from '../api'
// Repurchase loading state (一键回购): prevent double-clicks and show per-button spinner.
const repurchasing = ref(false)

const router = useRouter()
const orders = ref([])
const loading = ref(true)

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
</script>

<template>
  <div class="orders-page">
    <van-nav-bar title="我的订单" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!orders.length" description="暂无订单" />
    <div v-else>
      <div v-for="o in orders" :key="o.id" class="order-card">
        <div class="o-head"><span class="o-no">订单号: {{ o.order_no }}</span><span class="o-status">{{ statusText(o.status) }}</span></div>
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
.loading { text-align: center; padding: 80px; }
.order-card { background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.o-head { display: flex; justify-content: space-between; font-size: 12px; color: #999; margin-bottom: 8px; }
.o-status { color: #ff0036; }
.o-countdown { color: #ff0036; font-size: 13px; font-weight: bold; margin-bottom: 8px; }
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
