<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview, uploadImage, checkFavorite, toggleFavorite } from '../api'

const route = useRoute()
const router = useRouter()
const product = ref(null)
const reviews = ref([])
const skus = ref([])
const selectedSKU = ref(null)
const loading = ref(true)
const showReview = ref(false)
const reviewRating = ref(5)
const reviewContent = ref('')
const reviewImages = ref([])
const favorited = ref(false)
async function onUploadReviewImage(item) {
  try { const res = await uploadImage(item.file); reviewImages.value.push(res.url) } catch (e) { showToast('图片上传失败') }
}
function removeReviewImage(idx) { reviewImages.value.splice(idx, 1) }

onMounted(async () => {
  try {
    const res = await getProduct(route.params.id)
    product.value = res.data
    reviews.value = res.reviews || []
    skus.value = res.skus || []
    if (localStorage.getItem('tm_token')) {
      favorited.value = await checkFavorite(route.params.id)
    }
  } catch (e) {
    showToast('商品不存在')
  } finally {
    loading.value = false
  }
})

async function doFavorite() {
  if (!checkLogin()) return
  try {
    const res = await toggleFavorite(product.value.id)
    favorited.value = res.favorited
    showSuccessToast(res.favorited ? '已收藏' : '已取消收藏')
  } catch (e) { showToast('操作失败') }
}

async function doAddCart() {
  if (!checkLogin()) return
  try { await addToCart(product.value.id, 1); showSuccessToast('已加入购物车') } catch (e) { showToast('失败') }
}
async function buyNow() {
  if (!checkLogin()) return
  try { await createOrder({ items: [{ product_id: product.value.id, quantity: 1 }], address: '' }); showSuccessToast('下单成功'); router.push('/orders') } catch (e) { showToast('下单失败') }
}
function checkLogin() {
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return false }
  return true
}
async function submitReview() {
  if (!reviewContent.value.trim()) { showToast('请输入评价'); return }
  try {
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value })
    reviews.value.unshift(rv); showReview.value = false; reviewContent.value = ''; reviewImages.value = []; showSuccessToast('评价成功')
  } catch (e) { showToast('请先登录') }
}
function selectSKU(sku) { selectedSKU.value = sku }
function currentPrice() { return selectedSKU.value ? selectedSKU.value.price : (product.value ? product.value.price : 0) }
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <van-image width="100%" height="375" :src="product.image" fit="cover" />
    <div class="price-block">
      <span class="big-price">¥{{ fmt(currentPrice()) }}</span>
      <span class="origin">¥{{ fmt(product.original_price) }}</span>
    </div>
    <div v-if="skus.length" class="sku-block">
      <div class="sku-title">已选：<b>{{ selectedSKU ? selectedSKU.spec_text : '请选择规格' }}</b></div>
      <div class="sku-tags">
        <span v-for="s in skus" :key="s.id" class="sku-tag" :class="{ active: selectedSKU && selectedSKU.id === s.id }" @click="selectSKU(s)">{{ s.spec_text }} <small>¥{{ fmt(s.price) }}</small></span>
      </div>
    </div>
    <div class="title-block">
      <div v-if="product.is_genuine" class="genuine-row"><span class="genuine-tag">正品保障</span> <span class="brand-link" v-if="product.brand_id" @click="router.push('/brand/' + product.brand_id)">{{ product.brand_name }} ›</span></div>
      <h2 class="p-title">{{ product.name }}</h2>
      <p class="p-sub">{{ product.subtitle }}</p>
    </div>
    <van-cell-group inset>
      <van-cell title="店铺" :value="product.shop" is-link v-if="product.brand_id" @click="router.push('/brand/' + product.brand_id)" />
      <van-cell title="店铺" :value="product.shop" v-else />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="标签" :value="product.tags || '正品保障'" />
    </van-cell-group>
    <div v-if="product.description" class="desc"><h3>商品详情</h3><p>{{ product.description }}</p></div>
    <div class="reviews">
      <div class="rev-head"><span>商品评价 ({{ reviews.length }})</span><van-button size="mini" type="danger" plain @click="showReview = true">写评价</van-button></div>
      <div v-for="r in reviews" :key="r.id" class="rev-item">
        <div class="rev-user"><span>{{ r.username }}</span><van-rate v-model="r.rating" readonly size="12" /></div>
        <div class="rev-content">{{ r.content }}</div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
    </div>
    <van-action-bar>
      <van-action-bar-icon icon="chat-o" text="客服" @click="showToast('客服功能为演示')" />
      <van-action-bar-icon :icon="favorited ? 'star' : 'star-o'" :text="favorited ? '已收藏' : '收藏'" :color="favorited ? '#ff0036' : '#323233'" @click="doFavorite" />
      <van-action-bar-icon icon="cart-o" text="购物车" @click="router.push('/cart')" />
      <van-action-bar-button color="#ffa300" type="warning" text="加入购物车" @click="doAddCart" />
      <van-action-bar-button color="#ff0036" type="danger" text="立即购买" @click="buyNow" />
    </van-action-bar>
    <van-popup v-model:show="showReview" position="bottom" round closeable>
      <div class="rev-form">
        <h3>写评价</h3>
        <van-rate v-model="reviewRating" />
        <van-field v-model="reviewContent" type="textarea" placeholder="说说你的使用感受" rows="3" />
        <div class="rev-upload">
          <van-uploader :after-read="onUploadReviewImage" accept="image/*" multiple :preview-image="false">
            <van-button icon="photo-o" size="small" plain round>添加晒图</van-button>
          </van-uploader>
          <div v-if="reviewImages.length" class="rev-imgs">
            <div v-for="(img, i) in reviewImages" :key="i" class="rev-img-wrap">
              <van-image width="60" height="60" radius="6" :src="img" fit="cover" />
              <van-icon name="cross" class="rev-img-del" @click="removeReviewImage(i)" />
            </div>
          </div>
        </div>
        <van-button type="danger" block @click="submitReview">提交评价</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.detail { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.price-block { padding: 12px 16px; background: #fff; }
.big-price { color: #ff0036; font-size: 28px; font-weight: bold; }
.origin { color: #999; text-decoration: line-through; margin-left: 10px; font-size: 14px; }
.sku-block { padding: 12px 16px; background: #fff; border-top: 1px solid #f5f5f5; }
.sku-title { font-size: 13px; color: #666; margin-bottom: 8px; }
.sku-title b { color: #333; }
.sku-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.sku-tag { padding: 6px 12px; background: #f7f7f7; border: 1px solid #eee; border-radius: 16px; font-size: 13px; color: #333; }
.sku-tag.active { background: #fff5f6; border-color: #ff0036; color: #ff0036; }
.sku-tag small { color: #ff0036; margin-left: 4px; }
.title-block { padding: 0 16px 12px; background: #fff; }
.genuine-row { margin-bottom: 6px; display: flex; align-items: center; gap: 8px; }
.brand-link { color: #ff0036; font-size: 12px; }
.p-title { font-size: 17px; line-height: 24px; }
.p-sub { color: #999; font-size: 13px; margin-top: 4px; }
.desc, .reviews { background: #fff; margin-top: 8px; padding: 12px 16px; }
.desc h3, .rev-head { font-size: 15px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
.rev-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.rev-user { display: flex; gap: 8px; align-items: center; font-size: 13px; color: #666; }
.rev-content { font-size: 13px; margin-top: 4px; line-height: 18px; }
.rev-form { padding: 20px; }
.rev-form h3 { text-align: center; margin-bottom: 16px; }
.rev-form .van-field { margin: 12px 0; border: 1px solid #eee; }
.rev-upload { margin: 8px 0; }
.rev-imgs { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.rev-img-wrap { position: relative; }
.rev-img-del { position: absolute; top: -6px; right: -6px; background: #ff0036; color: #fff; border-radius: 50%; padding: 2px; font-size: 12px; }
</style>
