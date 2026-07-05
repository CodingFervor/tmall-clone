<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getMyCoupons, getAddresses, requestInvoice } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
const coupons = ref([])
const selectedCouponId = ref(null)
const showCouponPicker = ref(false)
const remark = ref('')
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
    const [cartRes, myCoupons, addrList] = await Promise.all([
      getCart(),
      getMyCoupons().catch(() => []),
      getAddresses().catch(() => []),
    ])
    items.value = cartRes.data || []
    selectedTotal.value = cartRes.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
    coupons.value = myCoupons || []
    addresses.value = addrList || []
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
async function toggleAll() {
  const t = allSelected.value ? 0 : 1
  for (const it of items.value) { if (it.selected !== t) await updateCart(it.id, it.quantity, t) }
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
.addr-picker { padding: 16px; }
.addr-item { padding: 12px; border: 1px solid #eee; border-radius: 8px; margin-bottom: 8px; }
.addr-item.active { border-color: #ff0036; background: #fff0f3; }
.addr-top { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.addr-name { font-size: 14px; font-weight: bold; }
.addr-phone { font-size: 13px; color: #666; }
.addr-detail { font-size: 12px; color: #999; line-height: 18px; }
.invoice-form { padding: 16px 0; }
.inv-actions { display: flex; gap: 12px; padding: 16px; }
</style>
