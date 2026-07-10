<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getMyCoupons, getAddresses, requestInvoice, getTieredDiscounts, getProducts, addToCart, toggleFavorite } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
const coupons = ref([])
const selectedCouponId = ref(null)
const showCouponPicker = ref(false)
const remark = ref('')
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
onMounted(load); onActivated(load)

async function toggleSelect(item) {
  const ns = item.selected === 1 ? 0 : 1
  try { await updateCart(item.id, item.quantity, ns); item.selected = ns; await load() } catch (e) {}
}
async function changeQty(item, qty) {
  if (qty < 1) return
  try { await updateCart(item.id, qty, item.selected); item.quantity = qty; await load() } catch (e) {}
}
async function removeItem(item) { try { await deleteCart(item.id); await load(); showSuccessToast('已删除') } catch (e) {} }
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
async function checkout() {
  if (selectedTotal.value <= 0) { showToast('请选择商品'); return }
  if (!selectedAddress.value) { showToast('请选择收货地址'); return }
  const sel = items.value.filter((i) => i.selected === 1).map((i) => ({ product_id: i.product_id, quantity: i.quantity }))
  const a = selectedAddress.value
  const addressText = `${a.name} ${a.phone} ${a.detail}`
  try {
    const order = await createOrder({ items: sel, address: addressText, user_coupon_id: selectedCouponId.value || undefined, remark: remark.value })
    // Optionally request an invoice for the new order.
    if (invoiceEnabled.value) {
      await requestInvoice(order.id, {
        invoice_type: invoiceType.value,
        title: invoiceTitle.value,
        tax_no: invoiceTaxNo.value,
        email: invoiceEmail.value,
      }).catch(() => {})
    }
    showSuccessToast('下单成功'); router.push('/orders')
  } catch (e) { showToast(e.response?.data?.error || '下单失败') }
}
function couponLabel(uc) {
  if (!uc.coupon) return ''
  const c = uc.coupon
  if (c.coupon_type === 'discount') return `${(c.value * 10).toFixed(1)}折券 · 满${c.threshold}可用`
  return `满${c.threshold}减${c.value}`
}
function fmt(n) { return Number(n).toFixed(2) }
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
    <van-empty v-else-if="!items.length" description="购物车是空的"><van-button type="danger" round @click="router.push('/home')">去逛逛</van-button></van-empty>
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
      <!-- Coupon selector -->
      <van-cell
        title="优惠券"
        :value="selectedCouponId ? couponLabel(usableCoupons.find((c) => c.id === selectedCouponId)) : (usableCoupons.length ? usableCoupons.length + '张可用' : '暂无可用')"
        is-link
        @click="showCouponPicker = true"
        style="margin-top: 8px"
      />
      <van-field v-model="remark" label="订单备注" placeholder="选填，如送货时间、发票等" style="margin-top: 8px" />
      <!-- Address picker (收货地址) -->
      <van-cell
        title="收货地址"
        :value="addressLabel()"
        is-link
        @click="showAddressPicker = true"
        style="margin-top: 8px"
      />
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
      <van-submit-bar :price="finalTotal * 100" :button-text="'结算 (' + selectedCount + '件)'" @submit="checkout">
        <van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox>
        <span class="invert-btn" @click="invertSelection">反选</span>
        <span class="group-btn" @click="openGroupBuy">👥 邀请拼单</span>
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
  </div>
</template>

<style scoped>
.cart-page { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.invert-btn { font-size: 14px; color: #ff0036; margin-left: 12px; cursor: pointer; }
.group-btn { font-size: 13px; color: #fff; background: linear-gradient(135deg, #ff0036, #ff5a5f); padding: 5px 12px; margin-left: 10px; border-radius: 999px; cursor: pointer; white-space: nowrap; box-shadow: 0 2px 6px rgba(255, 0, 54, 0.35); }
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
</style>
