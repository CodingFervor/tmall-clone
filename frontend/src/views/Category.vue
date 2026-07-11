<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getCategories, getProducts } from '../api'

const route = useRoute()
const router = useRouter()
const cats = ref([])
const products = ref([])
const activeId = ref(0)
const loading = ref(false)
const sortBy = ref('default')

// Product compare (商品对比): track up to 2 selected products.
const compareList = ref([])
const showCompare = ref(false)
const MAX_COMPARE = 2

onMounted(async () => {
  try { cats.value = await getCategories(); if (cats.value.length) activeId.value = route.query.id ? Number(route.query.id) : cats.value[0].id }
  catch (e) { showToast('加载失败') }
})

watch(activeId, async (id) => {
  loading.value = true
  try { const res = await getProducts({ category_id: id, page: 1, page_size: 50 }); products.value = res.data } catch (e) { showToast('加载失败') } finally { loading.value = false }
}, { immediate: true })

const filteredProducts = computed(() => {
  const list = [...products.value]
  if (sortBy.value === 'price_asc') return list.sort((a, b) => a.price - b.price)
  if (sortBy.value === 'price_desc') return list.sort((a, b) => b.price - a.price)
  if (sortBy.value === 'sales') return list.sort((a, b) => b.sales - a.sales)
  return list
})

function fmt(n) { return Number(n).toFixed(2) }

// New product (新品限时标签): products created within the last 7 days.
function isNew(p) {
  if (!p || !p.created_at) return false
  const created = new Date(p.created_at)
  if (isNaN(created.getTime())) return false
  return (Date.now() - created.getTime()) <= 7 * 24 * 3600 * 1000
}

function isInCompare(p) { return compareList.value.some(c => c.id === p.id) }
function toggleCompare(p) {
  const idx = compareList.value.findIndex(c => c.id === p.id)
  if (idx >= 0) {
    compareList.value.splice(idx, 1)
    return
  }
  if (compareList.value.length >= MAX_COMPARE) {
    showToast('最多对比2件商品')
    return
  }
  compareList.value.push(p)
}
function clearCompare() { compareList.value = [] }
function openCompare() { if (compareList.value.length === MAX_COMPARE) showCompare.value = true }
// Lower price wins -> highlight in red. -1 if incomparable / equal.
function cheaperIndex() {
  if (compareList.value.length !== 2) return -1
  const a = Number(compareList.value[0].price)
  const b = Number(compareList.value[1].price)
  if (isNaN(a) || isNaN(b) || a === b) return -1
  return a < b ? 0 : 1
}

// ---- 分类广告条 (category banner carousel) ----
// Three demo banners from picsum. Auto-rotates every 3s; tapping a banner
// shows a toast (tappable toast).
const banners = [
  { id: 0, img: 'https://picsum.photos/seed/tmall-cat-a/640/200', text: '限时满减 爆款直降' },
  { id: 1, img: 'https://picsum.photos/seed/tmall-cat-b/640/200', text: '品质好物 新品首发' },
  { id: 2, img: 'https://picsum.photos/seed/tmall-cat-c/640/200', text: '品牌特卖 限时抢购' },
]
function tapBanner(b) { showToast(b.text) }
</script>

<template>
  <div class="cat-page">
    <van-sticky><van-search placeholder="搜索天猫商品" shape="round" readonly @click="router.push('/search')" /></van-sticky>
    <div class="cat-body">
      <div class="cat-sidebar">
        <div v-for="c in cats" :key="c.id" class="cat-side-item" :class="{ active: activeId === c.id }" @click="activeId = c.id">{{ c.name }}</div>
      </div>
      <div class="cat-content">
        <!-- 分类广告条 (category banner carousel) -->
        <van-swipe class="cat-banner" :autoplay="3000" indicator-color="#ff0036" :loop="true">
          <van-swipe-item v-for="b in banners" :key="b.id" @click="tapBanner(b)">
            <div class="cb-slide">
              <van-image class="cb-img" width="100%" height="120" fit="cover" radius="8" :src="b.img" />
              <span class="cb-text">{{ b.text }}</span>
            </div>
          </van-swipe-item>
        </van-swipe>
        <div class="sort-bar">
          <span :class="{ active: sortBy === 'default' }" @click="sortBy = 'default'">综合</span>
          <span :class="{ active: sortBy === 'sales' }" @click="sortBy = 'sales'">销量</span>
          <span :class="{ active: sortBy === 'price_asc' }" @click="sortBy = 'price_asc'">价格↑</span>
          <span :class="{ active: sortBy === 'price_desc' }" @click="sortBy = 'price_desc'">价格↓</span>
        </div>
        <div v-if="loading" class="loading"><van-loading /></div>
        <div v-else>
          <div v-for="p in filteredProducts" :key="p.id" class="prod-row" :class="{ selected: isInCompare(p) }" @click="router.push('/product/' + p.id)">
            <div class="prod-thumb">
              <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
              <div class="new-badge" v-if="isNew(p)">NEW</div>
            </div>
            <div class="prod-info">
              <div class="prod-name van-multi-ellipsis--l2">{{ p.name }}</div>
              <div class="prod-sub van-ellipsis">{{ p.subtitle }}</div>
              <div class="prod-bottom"><span class="prod-price">¥{{ fmt(p.price) }}</span><span class="prod-sales">{{ p.sales }}人付款</span>
                <span class="cmp-btn" :class="{ active: isInCompare(p) }" @click.stop="toggleCompare(p)">对比</span>
              </div>
            </div>
          </div>
          <van-empty v-if="!filteredProducts.length" description="暂无商品" />
        </div>
      </div>
    </div>
    <!-- Floating compare button: appears when 2 products are selected. -->
    <div v-if="compareList.length === MAX_COMPARE" class="cmp-fab" @click="openCompare">
      <span class="cmp-fab-icon">⇄</span>
      <span class="cmp-fab-text">对比({{ compareList.length }})</span>
    </div>
    <!-- Compare popup: side-by-side table highlighting the cheaper price. -->
    <van-popup v-model:show="showCompare" round position="bottom" :style="{ height: '70%' }" closeable>
      <div class="cmp-popup">
        <div class="cmp-popup-title">商品对比</div>
        <div v-if="compareList.length === 2" class="cmp-table">
          <div class="cmp-col cmp-label">
            <div class="cmp-cell cmp-cell-img"></div>
            <div class="cmp-cell">商品名称</div>
            <div class="cmp-cell">现价</div>
            <div class="cmp-cell">原价</div>
            <div class="cmp-cell">店铺</div>
            <div class="cmp-cell">销量</div>
          </div>
          <div v-for="(p, idx) in compareList" :key="p.id" class="cmp-col">
            <div class="cmp-cell cmp-cell-img"><van-image width="80" height="80" radius="6" :src="p.image" fit="cover" /></div>
            <div class="cmp-cell cmp-cell-name">{{ p.name }}</div>
            <div class="cmp-cell" :class="{ cheaper: cheaperIndex() === idx }">¥{{ fmt(p.price) }}</div>
            <div class="cmp-cell" :class="{ cheaper: cheaperIndex() === idx }">¥{{ fmt(p.original_price) }}</div>
            <div class="cmp-cell">{{ p.shop || '—' }}</div>
            <div class="cmp-cell">{{ p.sales }}人付款</div>
          </div>
        </div>
        <div class="cmp-actions">
          <van-button block round type="danger" @click="clearCompare(); showCompare = false">清空</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.cat-page { display: flex; flex-direction: column; height: 100vh; }
.cat-body { display: flex; flex: 1; overflow: hidden; }
.cat-sidebar { width: 90px; background: #f5f5f5; overflow-y: auto; }
.cat-side-item { padding: 16px 8px; text-align: center; font-size: 13px; color: #333; position: relative; }
.cat-side-item.active { background: #fff; color: #ff0036; font-weight: bold; }
.cat-side-item.active::before { content: ''; position: absolute; left: 0; top: 50%; transform: translateY(-50%); width: 3px; height: 18px; background: #ff0036; }
.cat-content { flex: 1; background: #fff; overflow-y: auto; }
/* 分类广告条 (category banner carousel) */
.cat-banner { margin: 8px; border-radius: 8px; overflow: hidden; }
.cb-slide { position: relative; cursor: pointer; }
.cb-img { display: block; }
.cb-text {
  position: absolute;
  left: 12px; bottom: 10px;
  color: #fff;
  font-size: 14px;
  font-weight: bold;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.55);
  letter-spacing: 0.5px;
}
.sort-bar { display: flex; background: #fff; border-bottom: 1px solid #f0f0f0; position: sticky; top: 0; z-index: 5; }
.sort-bar span { flex: 1; text-align: center; padding: 10px 0; font-size: 13px; color: #666; }
.sort-bar span.active { color: #ff0036; font-weight: bold; }
.prod-row { display: flex; gap: 10px; padding: 10px; border-bottom: 1px solid #f5f5f5; }
.prod-thumb { position: relative; flex-shrink: 0; }
.new-badge { position: absolute; top: 4px; left: 4px; z-index: 5; background: #ff0036; color: #fff; font-size: 11px; font-weight: bold; line-height: 1; padding: 3px 7px; border-radius: 999px; box-shadow: 0 1px 4px rgba(255, 0, 54, 0.45); letter-spacing: 0.5px; }
.prod-info { flex: 1; display: flex; flex-direction: column; justify-content: space-between; }
.prod-name { font-size: 13px; line-height: 18px; }
.prod-sub { font-size: 11px; color: #999; }
.prod-bottom { display: flex; align-items: baseline; gap: 8px; }
.prod-price { color: #ff0036; font-weight: bold; font-size: 16px; }
.prod-sales { font-size: 11px; color: #999; }
.loading { text-align: center; padding: 40px; }
/* Product compare (商品对比) */
.prod-row.selected { background: #fff0f3; }
.cmp-btn { margin-left: auto; padding: 2px 10px; font-size: 11px; color: #ff0036; border: 1px solid #ff0036; border-radius: 10px; line-height: 16px; cursor: pointer; }
.cmp-btn.active { background: #ff0036; color: #fff; }
.cmp-fab { position: fixed; right: 16px; bottom: 80px; z-index: 20; display: flex; flex-direction: column; align-items: center; justify-content: center; width: 56px; height: 56px; border-radius: 50%; background: #ff0036; color: #fff; box-shadow: 0 2px 10px rgba(255, 0, 54, 0.4); cursor: pointer; }
.cmp-fab-icon { font-size: 20px; line-height: 1; }
.cmp-fab-text { font-size: 10px; margin-top: 2px; }
.cmp-popup { padding: 16px; padding-top: 28px; height: 100%; display: flex; flex-direction: column; }
.cmp-popup-title { font-size: 16px; font-weight: bold; text-align: center; margin-bottom: 12px; }
.cmp-table { display: flex; flex: 1; overflow-y: auto; }
.cmp-col { flex: 1; min-width: 0; display: flex; flex-direction: column; align-items: center; text-align: center; }
.cmp-col.cmp-label { flex: 0 0 60px; align-items: stretch; text-align: left; }
.cmp-col.cmp-label .cmp-cell { color: #999; font-size: 12px; align-items: center; justify-content: flex-end; text-align: right; padding-right: 8px; }
.cmp-cell { min-height: 84px; padding: 8px 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #333; border-bottom: 1px solid #f5f5f5; width: 100%; }
.cmp-cell-img { min-height: 92px; }
.cmp-cell-name { font-size: 13px; }
.cmp-cell.cheaper { color: #ff0036; font-weight: bold; }
.cmp-actions { padding-top: 12px; }
</style>
