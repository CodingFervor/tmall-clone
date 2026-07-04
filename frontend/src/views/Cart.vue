<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getMyCoupons } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
const coupons = ref([])
const selectedCouponId = ref(null)
const showCouponPicker = ref(false)
const remark = ref('')

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

async function load() {
  loading.value = true
  try {
    const [cartRes, myCoupons] = await Promise.all([getCart(), getMyCoupons().catch(() => [])])
    items.value = cartRes.data || []
    selectedTotal.value = cartRes.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
    coupons.value = myCoupons || []
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
async function toggleAll() {
  const t = allSelected.value ? 0 : 1
  for (const it of items.value) { if (it.selected !== t) await updateCart(it.id, it.quantity, t) }
  await load()
}
async function checkout() {
  if (selectedTotal.value <= 0) { showToast('请选择商品'); return }
  const sel = items.value.filter((i) => i.selected === 1).map((i) => ({ product_id: i.product_id, quantity: i.quantity }))
  try { await createOrder({ items: sel, address: '', user_coupon_id: selectedCouponId.value || undefined, remark: remark.value }); showSuccessToast('下单成功'); router.push('/orders') } catch (e) { showToast(e.response?.data?.error || '下单失败') }
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
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!items.length" description="购物车是空的"><van-button type="danger" round @click="router.push('/home')">去逛逛</van-button></van-empty>
    <div v-else>
      <div v-for="it in items" :key="it.id" class="cart-item">
        <van-checkbox :model-value="it.selected === 1" @click="toggleSelect(it)" />
        <van-image width="80" height="80" radius="6" :src="it.product_image" fit="cover" @click="router.push('/product/' + it.product_id)" />
        <div class="ci-info">
          <div class="ci-name van-multi-ellipsis--l2">{{ it.product_name }}</div>
          <div class="ci-bottom"><span class="price">¥{{ fmt(it.price) }}</span><van-stepper v-model="it.quantity" :min="1" :max="it.stock" @change="(v) => changeQty(it, v)" /><van-icon name="delete-o" size="20" @click="removeItem(it)" /></div>
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
      <!-- Top-up hints (凑单提示) -->
      <div v-if="topupHints.length" class="topup-hints">
        <div v-for="h in topupHints.slice(0, 2)" :key="h.id" class="th-item">💡 再买 <b>¥{{ h.diff }}</b> 可使用 {{ h.label }}，<span class="th-go" @click="router.push('/home')">去凑单 ›</span></div>
      </div>
      <div v-if="discount > 0" class="discount-line">优惠券抵扣 -¥{{ fmt(discount) }}</div>
      <van-submit-bar :price="finalTotal * 100" :button-text="'结算 (' + selectedCount + '件)'" @submit="checkout"><van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox></van-submit-bar>

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
    </div>
  </div>
</template>

<style scoped>
.cart-page { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.cart-item { display: flex; align-items: center; gap: 10px; padding: 12px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.ci-info { flex: 1; }
.ci-name { font-size: 13px; line-height: 18px; height: 36px; }
.ci-bottom { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.ci-bottom .price { font-size: 16px; flex: 1; }
.discount-line { color: #ff0036; font-size: 13px; text-align: right; padding: 6px 16px; background: #fff; }
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
</style>
