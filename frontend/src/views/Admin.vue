<script setup>
import { ref, computed, onMounted } from 'vue'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProducts, adminCreateProduct, adminUpdateProduct, adminDeleteProduct, getCategories, getBrands, uploadImage, getOrders } from '../api'

const uploadingImg = ref(false)
async function onUploadMainImage(item) {
  uploadingImg.value = true
  try {
    const res = await uploadImage(item.file)
    form.value.image = res.url
    showToast('图片已上传')
  } catch (e) {
    showToast(e.response?.data?.error || '上传失败')
  } finally {
    uploadingImg.value = false
  }
}

const products = ref([])
const categories = ref([])
const brands = ref([])
const loading = ref(true)
const showForm = ref(false)
const editingId = ref(null)
const form = ref(emptyForm())
// Orders list for the dashboard stats (recent orders + revenue).
const orders = ref([])

function emptyForm() {
  return { name: '', subtitle: '', price: 0, original_price: 0, image: '', category: '', category_id: 0, brand_id: 0, brand_name: '', shop: '', stock: 999, sales: 0, description: '', tags: '', is_genuine: 1 }
}

// ---- 后台增强 (admin dashboard) ----
// Stats cards: total products, total orders, total revenue (paid+shipped+
// completed orders) and average order value. Recent 5 orders mini list and
// the top 5 products by sales round out the overview.
function parseItems(json) { try { return JSON.parse(json) } catch { return [] } }
const totalRevenue = computed(() =>
  (orders.value || [])
    .filter((o) => ['paid', 'shipped', 'completed'].includes(o.status))
    .reduce((s, o) => s + (Number(o.total) || 0), 0)
)
const paidOrderCount = computed(() =>
  (orders.value || []).filter((o) => ['paid', 'shipped', 'completed'].includes(o.status)).length
)
const avgOrderValue = computed(() => {
  const n = paidOrderCount.value
  return n > 0 ? totalRevenue.value / n : 0
})
const recentOrders = computed(() => (orders.value || []).slice(0, 5))
const topProducts = computed(() =>
  [...(products.value || [])]
    .sort((a, b) => (Number(b.sales) || 0) - (Number(a.sales) || 0))
    .slice(0, 5)
)
function orderStatusText(s) {
  return { pending: '待付款', paid: '已付款', shipped: '已发货', completed: '已完成', cancelled: '已取消' }[s] || s
}
function orderStatusLabel(s) {
  return ['paid', 'shipped', 'completed'].includes(s) ? 'paid' : (s === 'pending' ? 'pending' : 'other')
}
function maxTopSales() {
  const m = Math.max(...topProducts.value.map((p) => Number(p.sales) || 0), 1)
  return m
}

onMounted(async () => {
  await loadProducts()
  try { categories.value = await getCategories(); brands.value = await getBrands() } catch (e) {}
  try { orders.value = (await getOrders()) || [] } catch (e) { /* dashboard optional */ }
})

async function loadProducts() {
  loading.value = true
  try { const res = await getProducts({ page: 1, page_size: 100 }); products.value = res.data } catch (e) { showToast('加载失败') } finally { loading.value = false }
}
function openCreate() { editingId.value = null; form.value = emptyForm(); showForm.value = true }
function openEdit(p) {
  editingId.value = p.id
  form.value = { name: p.name, subtitle: p.subtitle, price: p.price, original_price: p.original_price, image: p.image, category: p.category, category_id: p.category_id, brand_id: p.brand_id, brand_name: p.brand_name, shop: p.shop, stock: p.stock, sales: p.sales, description: p.description, tags: p.tags, is_genuine: p.is_genuine }
  showForm.value = true
}
async function save() {
  if (!form.value.name || !form.value.price) { showToast('商品名和价格必填'); return }
  try {
    if (editingId.value) { await adminUpdateProduct(editingId.value, form.value); showSuccessToast('已更新') }
    else { await adminCreateProduct(form.value); showSuccessToast('已创建') }
    showForm.value = false; await loadProducts()
  } catch (e) { showToast('保存失败') }
}
async function remove(p) {
  try { await showDialog({ title: '确认删除', message: '删除商品「' + p.name + '」？' }); await adminDeleteProduct(p.id); showSuccessToast('已删除'); await loadProducts() } catch (e) {}
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="admin">
    <van-nav-bar title="商品管理后台" left-arrow @click-left="$router.back()" fixed placeholder><template #right><van-icon name="plus" size="20" @click="openCreate" /></template></van-nav-bar>
    <!-- 后台增强 (admin dashboard): stat cards + recent orders + top products -->
    <div v-if="!loading && products.length" class="dashboard">
      <div class="dash-stats">
        <div class="ds-card">
          <div class="ds-num">{{ products.length }}</div>
          <div class="ds-label">商品数</div>
        </div>
        <div class="ds-card">
          <div class="ds-num">{{ orders.length }}</div>
          <div class="ds-label">订单数</div>
        </div>
        <div class="ds-card">
          <div class="ds-num">¥{{ fmt(totalRevenue) }}</div>
          <div class="ds-label">总营收</div>
        </div>
        <div class="ds-card">
          <div class="ds-num">¥{{ fmt(avgOrderValue) }}</div>
          <div class="ds-label">客单价</div>
        </div>
      </div>
      <!-- 最近 5 笔订单 (recent 5 orders) -->
      <div class="dash-section">
        <div class="dash-head">最近订单</div>
        <div v-if="!recentOrders.length" class="dash-empty">暂无订单</div>
        <div v-for="o in recentOrders" :key="o.id" class="ro-item">
          <div class="ro-info">
            <div class="ro-no van-ellipsis">{{ o.order_no }}</div>
            <div class="ro-meta">{{ parseItems(o.items_json).length }} 件 · {{ String(o.created_at).slice(0, 16).replace('T', ' ') }}</div>
          </div>
          <div class="ro-right">
            <span class="ro-total">¥{{ fmt(o.total) }}</span>
            <span class="ro-status" :class="orderStatusLabel(o.status)">{{ orderStatusText(o.status) }}</span>
          </div>
        </div>
      </div>
      <!-- 销量 Top 5 商品 (top 5 products by sales) -->
      <div class="dash-section">
        <div class="dash-head">热销 Top 5</div>
        <div v-for="(p, i) in topProducts" :key="p.id" class="tp-item" @click="openEdit(p)">
          <span class="tp-rank" :class="{ gold: i === 0, silver: i === 1, bronze: i === 2 }">{{ i + 1 }}</span>
          <van-image width="40" height="40" radius="6" :src="p.image" fit="cover" />
          <div class="tp-info">
            <div class="tp-name van-ellipsis">{{ p.name }}</div>
            <div class="tp-track"><div class="tp-fill" :style="{ width: Math.round(((Number(p.sales) || 0) / maxTopSales()) * 100) + '%' }"></div></div>
          </div>
          <div class="tp-sales">{{ Number(p.sales) || 0 }}件</div>
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!products.length" description="暂无商品"><van-button type="danger" round @click="openCreate">添加商品</van-button></van-empty>
    <van-cell-group v-else inset>
      <van-swipe-cell v-for="p in products" :key="p.id">
        <van-cell @click="openEdit(p)">
          <template #title>
            <div class="acell"><van-image width="50" height="50" radius="4" :src="p.image" fit="cover" /><div class="ac-info"><div class="van-ellipsis">{{ p.name }}</div><div class="ac-price">¥{{ fmt(p.price) }} <small>库存{{ p.stock }}</small></div></div></div>
          </template>
          <template #right><van-button square type="primary" text="编辑" @click="openEdit(p)" /><van-button square type="danger" text="删除" @click="remove(p)" /></template>
        </van-cell>
      </van-swipe-cell>
    </van-cell-group>
    <van-popup v-model:show="showForm" position="bottom" round :style="{ height: '80%' }" closeable>
      <div class="form">
        <h3>{{ editingId ? '编辑商品' : '新增商品' }}</h3>
        <van-cell-group inset>
          <van-field v-model="form.name" label="名称" placeholder="商品名称" />
          <van-field v-model="form.subtitle" label="副标题" placeholder="卖点" />
          <van-field v-model="form.price" type="number" label="价格" placeholder="0.00" />
          <van-field v-model="form.original_price" type="number" label="原价" placeholder="0.00" />
          <van-field label="商品主图" :loading="uploadingImg">
            <template #input>
              <van-uploader :after-read="onUploadMainImage" accept="image/*" max-count="1" :preview-image="false">
                <van-button icon="photo-o" size="small" round color="#ff0036">上传图片</van-button>
              </van-uploader>
              <van-image v-if="form.image" width="60" height="60" radius="6" :src="form.image" fit="cover" style="margin-left: 8px" />
            </template>
          </van-field>
          <van-field v-model="form.shop" label="店铺" placeholder="旗舰店名称" />
          <van-field v-model="form.brand_name" label="品牌名" placeholder="品牌旗舰店" />
          <van-field v-model="form.stock" type="digit" label="库存" placeholder="999" />
          <van-field v-model="form.tags" label="标签" placeholder="正品保障" />
          <van-field v-model="form.description" type="textarea" label="描述" rows="2" />
          <van-cell title="正品保障"><template #right-icon><van-switch v-model="form.is_genuine" :active-value="1" :inactive-value="0" /></template></van-cell>
        </van-cell-group>
        <div style="margin: 16px"><van-button type="danger" block round @click="save">保 存</van-button></div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.admin { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.acell { display: flex; gap: 10px; align-items: center; }
.ac-info { flex: 1; font-size: 13px; }
.ac-price { color: #ff0036; margin-top: 4px; }
.ac-price small { color: #999; font-weight: normal; }
.form { padding: 16px 0; }
.form h3 { text-align: center; padding: 12px; }
/* 后台增强 (admin dashboard) */
.dashboard { margin: 10px 12px; }
.dash-stats { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; margin-bottom: 10px; }
.ds-card { background: #fff; border-radius: 10px; padding: 14px 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); position: relative; overflow: hidden; }
.ds-card::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 4px; background: #ff0036; }
.ds-num { font-size: 18px; font-weight: bold; color: #ff0036; line-height: 1.2; font-variant-numeric: tabular-nums; }
.ds-label { font-size: 12px; color: #999; margin-top: 4px; }
.dash-section { background: #fff; border-radius: 10px; padding: 12px; margin-bottom: 10px; }
.dash-head { font-size: 15px; font-weight: bold; color: #333; margin-bottom: 10px; }
.dash-empty { font-size: 13px; color: #999; text-align: center; padding: 16px 0; }
/* recent orders */
.ro-item { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 8px 0; border-top: 1px solid #f5f5f5; }
.ro-item:first-of-type { border-top: none; }
.ro-info { flex: 1; min-width: 0; }
.ro-no { font-size: 13px; color: #333; }
.ro-meta { font-size: 11px; color: #999; margin-top: 2px; }
.ro-right { display: flex; flex-direction: column; align-items: flex-end; gap: 3px; flex-shrink: 0; }
.ro-total { font-size: 14px; font-weight: bold; color: #ff0036; }
.ro-status { font-size: 10px; padding: 1px 6px; border-radius: 8px; }
.ro-status.paid { color: #07c160; background: #e6f9ee; }
.ro-status.pending { color: #ff9800; background: #fff6e6; }
.ro-status.other { color: #999; background: #f5f5f5; }
/* top products */
.tp-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-top: 1px solid #f5f5f5; }
.tp-item:first-of-type { border-top: none; }
.tp-rank { width: 22px; height: 22px; line-height: 22px; text-align: center; border-radius: 50%; background: #eee; color: #666; font-size: 12px; font-weight: bold; flex-shrink: 0; }
.tp-rank.gold { background: linear-gradient(135deg, #ffd700, #ffb300); color: #fff; }
.tp-rank.silver { background: linear-gradient(135deg, #c0c0c0, #999); color: #fff; }
.tp-rank.bronze { background: linear-gradient(135deg, #cd7f32, #a85f1f); color: #fff; }
.tp-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.tp-name { font-size: 13px; color: #333; }
.tp-track { height: 5px; background: #f0f0f0; border-radius: 3px; overflow: hidden; }
.tp-fill { height: 100%; background: linear-gradient(90deg, #ff5577, #ff0036); border-radius: 3px; }
.tp-sales { font-size: 12px; color: #ff0036; font-weight: bold; flex-shrink: 0; }
</style>
