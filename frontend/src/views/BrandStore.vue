<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getBrand } from '../api'

const route = useRoute()
const router = useRouter()
const brand = ref(null)
const products = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await getBrand(route.params.id)
    brand.value = res.brand
    products.value = res.products || []
  } catch (e) {
    showToast('品牌不存在')
  } finally {
    loading.value = false
  }
})
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="store-page">
    <van-nav-bar left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else-if="brand">
      <div class="store-header">
        <div class="sh-logo">{{ brand.logo }}</div>
        <div class="sh-info">
          <div class="sh-name">{{ brand.name }}</div>
          <div class="sh-desc">{{ brand.description }}</div>
          <div class="sh-fans">{{ (brand.followers / 10000).toFixed(0) }}万粉丝关注</div>
        </div>
        <van-button size="small" type="danger" round>关注</van-button>
      </div>
      <div class="genuine-bar"><van-icon name="certificate" color="#ff0036" /> 该店铺为品牌官方旗舰店，正品保障</div>
      <div class="store-products">
        <div class="sp-head">店铺商品 ({{ products.length }})</div>
        <div class="product-grid">
          <div v-for="p in products" :key="p.id" class="product-card" @click="router.push('/product/' + p.id)">
            <van-image width="100%" height="160" :src="p.image" fit="cover" radius="6" />
            <div class="p-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="p-bottom">
              <span class="price">¥{{ fmt(p.price) }}</span>
              <span class="sales">{{ p.sales }}付款</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.store-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.store-header { display: flex; align-items: center; gap: 12px; padding: 16px; background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; }
.sh-logo { width: 60px; height: 60px; line-height: 60px; text-align: center; background: rgba(255,255,255,0.2); border-radius: 50%; font-weight: bold; flex-shrink: 0; }
.sh-info { flex: 1; }
.sh-name { font-size: 17px; font-weight: bold; }
.sh-desc { font-size: 12px; opacity: 0.9; margin-top: 3px; }
.sh-fans { font-size: 11px; opacity: 0.8; margin-top: 3px; }
.genuine-bar { background: #fff5f6; color: #ff0036; font-size: 12px; padding: 8px 16px; display: flex; align-items: center; gap: 4px; }
.store-products { padding: 12px; }
.sp-head { font-size: 15px; font-weight: bold; margin-bottom: 10px; }
.product-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.product-card { background: #fff; border-radius: 8px; overflow: hidden; padding-bottom: 6px; }
.p-name { font-size: 13px; line-height: 18px; padding: 4px 6px; height: 36px; }
.p-bottom { display: flex; align-items: baseline; justify-content: space-between; padding: 2px 6px; }
.p-bottom .price { font-size: 16px; }
.sales { font-size: 11px; color: #999; }
</style>
