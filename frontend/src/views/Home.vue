<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProducts, getBrands, getCategories } from '../api'

const router = useRouter()
const products = ref([])
const brands = ref([])
const categories = ref([])
const loading = ref(false)

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
  }
})
function fmt(n) { return Number(n).toFixed(2) }
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

    <div class="cat-grid">
      <div v-for="c in categories" :key="c.id" class="cat-item" @click="router.push({ name: 'category', query: { id: c.id } })">
        <div class="cat-icon">{{ c.icon }}</div>
        <div class="cat-name">{{ c.name }}</div>
      </div>
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

    <div class="section">
      <div class="section-head"><span>为你推荐</span></div>
      <div class="product-grid">
        <div v-for="p in products" :key="p.id" class="product-card" @click="router.push('/product/' + p.id)">
          <van-image width="100%" height="170" :src="p.image" fit="cover" radius="6" />
          <div class="p-tag" v-if="p.is_genuine"><span class="genuine-tag">正品保障</span></div>
          <div class="p-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="p-shop van-ellipsis">{{ p.shop }}</div>
          <div class="p-bottom">
            <span class="price">¥{{ fmt(p.price) }}</span>
            <span class="sales">{{ p.sales }}人付款</span>
          </div>
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading /></div>
  </div>
</template>

<style scoped>
.topbar { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: #fff; }
.logo { color: #ff0036; font-weight: bold; font-size: 18px; }
.search { flex: 1; padding: 0; }
.banner { margin: 8px; border-radius: 8px; overflow: hidden; }
.cat-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px 0; padding: 16px 8px; background: #fff; margin: 0 8px 8px; border-radius: 8px; }
.cat-item { text-align: center; }
.cat-icon { font-size: 28px; }
.cat-name { font-size: 12px; color: #666; margin-top: 4px; }
.section { background: #fff; margin: 0 8px 8px; border-radius: 8px; padding: 12px; }
.section-head { display: flex; justify-content: space-between; align-items: center; font-size: 16px; font-weight: bold; margin-bottom: 10px; }
.more { font-size: 12px; color: #999; font-weight: normal; }
.brand-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; }
.brand-card { flex-shrink: 0; width: 76px; text-align: center; }
.brand-logo { width: 56px; height: 56px; line-height: 56px; margin: 0 auto; background: #fff5f6; color: #ff0036; border: 1px solid #ffd6dd; border-radius: 50%; font-weight: bold; font-size: 14px; }
.brand-name { font-size: 12px; margin-top: 6px; }
.brand-fans { font-size: 10px; color: #999; }
.product-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.product-card { background: #fafafa; border-radius: 8px; overflow: hidden; padding-bottom: 6px; }
.p-tag { padding: 2px 6px; }
.p-name { font-size: 13px; line-height: 18px; padding: 0 6px; height: 36px; }
.p-shop { font-size: 11px; color: #999; padding: 0 6px; }
.p-bottom { display: flex; align-items: baseline; justify-content: space-between; padding: 2px 6px; }
.p-bottom .price { font-size: 16px; }
.sales { font-size: 11px; color: #999; }
.loading { text-align: center; padding: 20px; }
</style>
