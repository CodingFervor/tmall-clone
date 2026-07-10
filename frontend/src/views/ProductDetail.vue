<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview, uploadImage, checkFavorite, toggleFavorite, replyReview, getPriceHistory, checkRestock, subscribeRestock, unsubscribeRestock, getProductQA, askProductQA, markReviewUseful } from '../api'

const route = useRoute()
const router = useRouter()
const product = ref(null)
const reviews = ref([])
const skus = ref([])
const selectedSKU = ref(null)
const recommendedSKU = ref(null)
const priceHistory = ref([])
const priceStats = ref(null)
const showPoster = ref(false)
const restockSubscribed = ref(false)
const qaList = ref([])
const showQA = ref(false)
const qaQuestion = ref('')
const relatedProducts = ref([])
const loading = ref(true)
const showReview = ref(false)
const reviewRating = ref(5)
const reviewContent = ref('')
const reviewImages = ref([])
const favorited = ref(false)
// Build the gallery list from the product's images field (comma-separated),
// falling back to the single main image.
const gallery = computed(() => {
  if (!product.value) return []
  const imgs = (product.value.images || '').split(',').map((s) => s.trim()).filter(Boolean)
  if (imgs.length) return imgs
  return product.value.image ? [product.value.image] : []
})
// Review summary stats (评价概览统计): average rating, good-rate (4-5★),
// and per-star distribution counts (index 0 = 5★ ... index 4 = 1★).
const reviewStats = computed(() => {
  const rs = reviews.value
  if (!rs.length) return { avg: 0, goodRate: 0, dist: [0, 0, 0, 0, 0] }
  let sum = 0
  const dist = [0, 0, 0, 0, 0]
  let good = 0
  for (const r of rs) {
    sum += r.rating
    const idx = 5 - r.rating
    if (idx >= 0 && idx < 5) dist[idx]++
    if (r.rating >= 4) good++
  }
  return {
    avg: Math.round((sum / rs.length) * 10) / 10,
    goodRate: Math.round((good / rs.length) * 100),
    dist,
  }
})
function starBarPct(count) {
  return reviews.value.length ? (count / reviews.value.length) * 100 : 0
}
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
    if (res.recommended_sku) { selectedSKU.value = res.recommended_sku; recommendedSKU.value = res.recommended_sku }
    relatedProducts.value = res.related || []
    if (localStorage.getItem('tm_token')) {
      favorited.value = await checkFavorite(route.params.id)
    }
    getPriceHistory(route.params.id).then((d) => { priceHistory.value = d.data || []; priceStats.value = d.stats }).catch(() => {})
    if (localStorage.getItem('tm_token')) { checkRestock(route.params.id).then((s) => { restockSubscribed.value = s }).catch(() => {}) }
    getProductQA(route.params.id).then((d) => { qaList.value = d || [] }).catch(() => {})
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
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value, images: reviewImages.value.join(',') })
    reviews.value.unshift(rv); showReview.value = false; reviewContent.value = ''; reviewImages.value = []; showSuccessToast('评价成功')
  } catch (e) { showToast('请先登录') }
}
const replyingTo = ref(null)
const replyText = ref('')
function toggleReply(r) { replyingTo.value = replyingTo.value === r.id ? null : r.id; replyText.value = '' }
async function submitReply(r) {
  if (!replyText.value.trim()) { showToast('请输入回复内容'); return }
  try {
    const rep = await replyReview(r.id, replyText.value)
    r.reply = rep; replyingTo.value = null; replyText.value = ''; showSuccessToast('回复成功')
  } catch (e) { showToast('请先登录') }
}
function selectSKU(sku) { selectedSKU.value = sku }
function currentPrice() { return selectedSKU.value ? selectedSKU.value.price : (product.value ? product.value.price : 0) }
function fmt(n) { return Number(n).toFixed(2) }
function goProduct(id) { router.replace('/product/' + id); setTimeout(() => window.location.reload(), 50) }
async function doUseful(r) {
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return }
  try { await markReviewUseful(r.id); r.useful = (r.useful || 0) + 1; showSuccessToast('已标记有用') }
  catch (e) { showToast('操作失败') }
}
function qrPattern(n) { const row = Math.floor((n - 1) / 8); const col = (n - 1) % 8; const corner = (row < 2 || row > 5) && (col < 2 || col > 5); return corner || ((row * 7 + col * 3 + n) % 3 === 0) }
async function copyShareLink() { try { await navigator.clipboard.writeText(window.location.href); showSuccessToast('链接已复制') } catch (e) { showToast('复制失败') } }
async function toggleRestock() {
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return }
  try {
    if (restockSubscribed.value) { await unsubscribeRestock(product.value.id); restockSubscribed.value = false; showSuccessToast('已取消到货通知') }
    else { await subscribeRestock(product.value.id); restockSubscribed.value = true; showSuccessToast('到货后将通知您') }
  } catch (e) { showToast('操作失败') }
}
async function submitQA() {
  if (!qaQuestion.value.trim()) { showToast('请输入问题'); return }
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return }
  try { const qa = await askProductQA(product.value.id, qaQuestion.value); qaList.value.unshift(qa); showQA.value = false; qaQuestion.value = ''; showSuccessToast('提问成功') }
  catch (e) { showToast('请先登录') }
}
function priceBars() {
  if (!priceHistory.value.length) return []
  const prices = priceHistory.value.map((p) => p.price)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const range = max - min || 1
  return priceHistory.value.map((p) => ({ price: p.price, height: 30 + Math.round(((p.price - min) / range) * 70), date: String(p.recorded_at).slice(5, 10) }))
}
function priceTrend() {
  if (!priceHistory.value.length || priceHistory.value.length < 2) return 'flat'
  const first = priceHistory.value[0].price
  const last = priceHistory.value[priceHistory.value.length - 1].price
  if (last < first) return 'down'
  if (last > first) return 'up'
  return 'flat'
}
// Delivery estimation (预计送达): order before 11:00 → 今日达, before 17:00 → 明日达, else 后天达.
function deliveryEstimate() {
  const h = new Date().getHours()
  if (h < 11) return '今日达'
  if (h < 17) return '明日达'
  return '后天达'
}
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Product intro video (商品视频介绍) -->
    <div v-if="product.video_url" class="product-video"><video :src="product.video_url" controls preload="metadata" class="pv-player"></video></div>
    <van-swipe class="gallery" :autoplay="3000" indicator-color="#ff0036" v-if="gallery.length > 1">
      <van-swipe-item v-for="(img, i) in gallery" :key="i">
        <van-image width="100%" height="375" :src="img" fit="cover" />
      </van-swipe-item>
    </van-swipe>
    <van-image v-else width="100%" height="375" :src="product.image" fit="cover" />
    <div class="price-block">
      <span class="big-price">¥{{ fmt(currentPrice()) }}</span>
      <span class="origin">¥{{ fmt(product.original_price) }}</span>
      <span v-if="product.vip_price > 0 && product.vip_price < currentPrice()" class="vip-price"><van-icon name="diamond-o" /> 88VIP ¥{{ fmt(product.vip_price) }}</span>
    </div>
    <div v-if="skus.length" class="sku-block">
      <div class="sku-title">已选：<b>{{ selectedSKU ? selectedSKU.spec_text : '请选择规格' }}</b><van-tag v-if="recommendedSKU && selectedSKU && selectedSKU.id === recommendedSKU.id" type="danger" round size="mini" style="margin-left:6px">AI推荐·性价比</van-tag></div>
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
      <van-cell title="店铺" :value="product.shop" is-link v-else @click="router.push('/shop/' + encodeURIComponent(product.shop))" />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="标签" :value="product.tags || '正品保障'" />
      <van-cell title="预计送达">
        <template #value>
          <span class="delivery-est"><span class="delivery-icon">🚚</span> {{ deliveryEstimate() }}</span>
        </template>
      </van-cell>
      <van-cell :title="restockSubscribed ? '到货通知已开启' : '到货通知'" @click="toggleRestock">
        <template #right-icon><van-switch :model-value="restockSubscribed" size="20" @click.stop="toggleRestock" active-color="#ff0036" /></template>
      </van-cell>
    </van-cell-group>
    <!-- Price history (比价历史) -->
    <div v-if="priceHistory.length" class="price-history">
      <div class="ph-head">
        <span>📈 比价历史</span>
        <span v-if="priceStats" class="ph-stats">最低 <b class="green">¥{{ fmt(priceStats.lowest) }}</b><span v-if="priceTrend() === 'down'" class="trend down">↓降价</span><span v-else-if="priceTrend() === 'up'" class="trend up">↑涨价</span><span v-else class="trend flat">→平稳</span></span>
      </div>
      <div class="ph-chart">
        <div v-for="(b, i) in priceBars()" :key="i" class="ph-bar-col">
          <div class="ph-bar" :style="{ height: b.height + '%' }"></div>
          <span class="ph-date">{{ b.date }}</span>
        </div>
      </div>
    </div>
    <!-- Product Q&A (商品问答) -->
    <div class="qa-section">
      <div class="qa-head"><span>商品问答 ({{ qaList.length }})</span><van-button size="mini" type="danger" plain @click="showQA = true">提问</van-button></div>
      <div v-if="!qaList.length" class="qa-empty">暂无问答</div>
      <div v-for="qa in qaList.slice(0, 5)" :key="qa.id" class="qa-item">
        <div class="qa-q"><span class="qa-tag-q">问</span> {{ qa.question }}</div>
        <div v-if="qa.answer" class="qa-a"><span class="qa-tag-a">答</span> {{ qa.answer }} <small class="qa-answerer">{{ qa.answerer }}</small></div>
      </div>
    </div>
    <van-popup v-model:show="showQA" position="bottom" round closeable>
      <div class="qa-form"><h3>我要提问</h3><van-field v-model="qaQuestion" type="textarea" placeholder="说说你想了解的问题" rows="3" /><van-button type="danger" block @click="submitQA" style="margin-top:12px">提交问题</van-button></div>
    </van-popup>
    <div v-if="product.description" class="desc"><h3>商品详情</h3><p>{{ product.description }}</p></div>
    <div class="reviews">
      <div class="rev-head"><span>商品评价 ({{ reviews.length }})</span><van-button size="mini" type="danger" plain @click="showReview = true">写评价</van-button></div>
      <!-- Review summary stats (评价概览统计) -->
      <div v-if="reviews.length" class="rev-summary">
        <div class="rs-left">
          <div class="rs-avg">{{ reviewStats.avg.toFixed(1) }}</div>
          <van-rate :model-value="reviewStats.avg" allow-half readonly size="12" color="#ff0036" />
          <div class="rs-goodrate">好评率 <b>{{ reviewStats.goodRate }}%</b></div>
        </div>
        <div class="rs-right">
          <div v-for="(c, i) in reviewStats.dist" :key="i" class="rs-star-row">
            <span class="rs-star-label">{{ 5 - i }}星</span>
            <div class="rs-star-track"><div class="rs-star-fill" :style="{ width: starBarPct(c) + '%' }"></div></div>
            <span class="rs-star-count">{{ c }}</span>
          </div>
        </div>
      </div>
      <div v-for="r in reviews" :key="r.id" class="rev-item">
        <div class="rev-user"><span>{{ r.username }}</span><van-rate v-model="r.rating" readonly size="12" /><span class="rev-reply-btn" @click="toggleReply(r)">回复</span></div>
        <div class="rev-content">{{ r.content }}</div>
        <div v-if="r.images" class="rev-photos">
          <van-image v-for="(img, i) in r.images.split(',')" :key="i" width="72" height="72" radius="6" :src="img" fit="cover" />
        </div>
        <div class="rev-actions"><span class="rev-useful-btn" @click="doUseful(r)"><van-icon name="good-job-o" /> 有用 ({{ r.useful || 0 }})</span></div>
        <div v-if="r.reply" class="rev-reply"><span class="rev-reply-name">{{ r.reply.username }}：</span>{{ r.reply.content }}</div>
        <div v-if="replyingTo === r.id" class="rev-reply-box">
          <van-field v-model="replyText" placeholder="写下你的回复..." />
          <van-button size="small" type="danger" @click="submitReply(r)">发送</van-button>
        </div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
    </div>
    <!-- Related products (看了又看) -->
    <div v-if="relatedProducts.length" class="related-section">
      <div class="rs-head">看了又看</div>
      <div class="rs-scroll">
        <div v-for="rp in relatedProducts" :key="rp.id" class="rs-card" @click="goProduct(rp.id)">
          <van-image width="100" height="100" radius="6" :src="rp.image" fit="cover" />
          <div class="rs-name van-multi-ellipsis--l2">{{ rp.name }}</div>
          <div class="rs-price">¥{{ fmt(rp.price) }}</div>
        </div>
      </div>
    </div>
    <van-action-bar>
      <van-action-bar-icon icon="chat-o" text="客服" @click="showToast('客服功能为演示')" />
      <van-action-bar-icon icon="share-o" text="分享" @click="showPoster = true" />
      <van-action-bar-icon :icon="favorited ? 'star' : 'star-o'" :text="favorited ? '已收藏' : '收藏'" :color="favorited ? '#ff0036' : '#323233'" @click="doFavorite" />
      <van-action-bar-icon icon="cart-o" text="购物车" @click="router.push('/cart')" />
      <van-action-bar-button color="#ffa300" type="warning" text="加入购物车" @click="doAddCart" />
      <van-action-bar-button color="#ff0036" type="danger" text="立即购买" @click="buyNow" />
    </van-action-bar>

    <!-- Share poster popup -->
    <van-popup v-model:show="showPoster" round closeable position="bottom" :style="{ width: '85%' }">
      <div class="poster">
        <div class="poster-head">分享给好友</div>
        <div class="poster-card">
          <van-image width="100%" height="200" radius="8" :src="product.image" fit="cover" />
          <div class="pc-name van-multi-ellipsis--l2">{{ product.name }}</div>
          <div class="pc-price">¥{{ fmt(currentPrice()) }}</div>
          <div class="pc-qr">
            <div class="qr-box">
              <div class="qr-grid"><div v-for="n in 64" :key="n" class="qr-cell" :class="{ on: qrPattern(n) }"></div></div>
            </div>
            <div class="qr-text">扫码查看商品</div>
          </div>
          <div class="pc-brand">天猫 TMALL</div>
        </div>
        <van-button block type="danger" round style="margin-top: 12px" @click="copyShareLink">复制分享链接</van-button>
      </div>
    </van-popup>
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
.product-video { background: #000; width: 100%; }
.pv-player { width: 100%; max-height: 280px; object-fit: contain; display: block; }
.price-block { padding: 12px 16px; background: #fff; }
.vip-price { margin-left: 12px; color: #333; font-size: 13px; background: linear-gradient(90deg, #ffd700, #ffaa00); padding: 2px 10px; border-radius: 12px; }
.qa-section { background: #fff; margin-top: 8px; padding: 12px 16px; }
.qa-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; font-weight: bold; }
.qa-empty { color: #999; font-size: 13px; padding: 12px 0; }
.qa-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.qa-q { font-size: 14px; line-height: 20px; }
.qa-tag-q { background: #ff0036; color: #fff; font-size: 11px; padding: 1px 6px; border-radius: 4px; margin-right: 6px; }
.qa-a { font-size: 13px; color: #666; margin-top: 6px; line-height: 18px; }
.qa-tag-a { background: #07c160; color: #fff; font-size: 11px; padding: 1px 6px; border-radius: 4px; margin-right: 6px; }
.qa-answerer { color: #999; margin-left: 6px; }
.qa-form { padding: 20px; }
.qa-form h3 { text-align: center; margin-bottom: 16px; }
.qa-form .van-field { border: 1px solid #eee; }
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
.price-history { background: #fff; margin-top: 8px; padding: 12px 16px; }
.ph-head { display: flex; justify-content: space-between; align-items: center; font-size: 14px; font-weight: bold; margin-bottom: 10px; }
.ph-stats { font-size: 12px; color: #666; font-weight: normal; }
.ph-stats b.green { color: #07c160; }
.trend { margin-left: 6px; font-size: 11px; }
.trend.down { color: #07c160; }
.trend.up { color: #ff0036; }
.trend.flat { color: #999; }
.delivery-est { color: #ff0036; font-size: 13px; font-weight: bold; }
.delivery-icon { margin-right: 2px; }
.ph-chart { display: flex; align-items: flex-end; gap: 6px; height: 80px; }
.ph-bar-col { flex: 1; display: flex; flex-direction: column; align-items: center; height: 100%; justify-content: flex-end; }
.ph-bar { width: 60%; background: linear-gradient(180deg, #ff9800, #ff0036); border-radius: 3px 3px 0 0; min-height: 8px; }
.ph-date { font-size: 9px; color: #999; margin-top: 4px; }
.desc h3, .rev-head { font-size: 15px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
/* Review summary stats (评价概览统计) */
.rev-summary { display: flex; gap: 16px; padding: 12px 0; border-top: 1px solid #f5f5f5; }
.rs-left { flex: 0 0 90px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; }
.rs-avg { font-size: 32px; font-weight: bold; color: #ff0036; line-height: 1.1; }
.rs-goodrate { font-size: 12px; color: #666; }
.rs-goodrate b { color: #ff0036; }
.rs-right { flex: 1; display: flex; flex-direction: column; gap: 5px; justify-content: center; }
.rs-star-row { display: flex; align-items: center; gap: 6px; }
.rs-star-label { font-size: 11px; color: #999; width: 26px; flex-shrink: 0; }
.rs-star-track { flex: 1; height: 7px; background: #f0f0f0; border-radius: 4px; overflow: hidden; }
.rs-star-fill { height: 100%; background: #ff0036; border-radius: 4px; }
.rs-star-count { font-size: 11px; color: #999; width: 24px; text-align: right; flex-shrink: 0; }
.rev-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.rev-user { display: flex; gap: 8px; align-items: center; font-size: 13px; color: #666; }
.rev-content { font-size: 13px; margin-top: 4px; line-height: 18px; }
.rev-photos { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.rev-actions { margin-top: 6px; }
.rev-useful-btn { color: #999; font-size: 12px; cursor: pointer; }
.rev-useful-btn:active { color: #ff0036; }
.rev-reply-btn { margin-left: auto; color: #ff0036; font-size: 12px; }
.rev-reply { background: #f7f7f7; border-radius: 6px; padding: 6px 10px; margin-top: 6px; font-size: 12px; color: #666; line-height: 18px; }
.rev-reply-name { color: #ff0036; }
.rev-reply-box { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.rev-reply-box .van-field { flex: 1; border: 1px solid #eee; border-radius: 6px; }
.rev-form { padding: 20px; }
.rev-form h3 { text-align: center; margin-bottom: 16px; }
.rev-form .van-field { margin: 12px 0; border: 1px solid #eee; }
.rev-upload { margin: 8px 0; }
.rev-imgs { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.rev-img-wrap { position: relative; }
.rev-img-del { position: absolute; top: -6px; right: -6px; background: #ff0036; color: #fff; border-radius: 50%; padding: 2px; font-size: 12px; }
.poster { padding: 20px; }
.poster-head { text-align: center; font-size: 16px; font-weight: bold; margin-bottom: 16px; }
.poster-card { background: #fff; border: 1px solid #eee; border-radius: 12px; padding: 16px; text-align: center; }
.pc-name { font-size: 15px; line-height: 22px; margin: 12px 0 6px; text-align: left; }
.pc-price { color: #ff0036; font-size: 24px; font-weight: bold; text-align: left; }
.pc-qr { display: flex; flex-direction: column; align-items: center; margin-top: 16px; }
.qr-box { padding: 8px; border: 1px solid #eee; border-radius: 8px; }
.qr-grid { display: grid; grid-template-columns: repeat(8, 1fr); gap: 1px; width: 120px; height: 120px; }
.qr-cell { background: #fff; }
.qr-cell.on { background: #333; }
.qr-text { font-size: 11px; color: #999; margin-top: 6px; }
.pc-brand { color: #ff0036; font-size: 13px; font-weight: bold; margin-top: 12px; }
.related-section { background: #fff; margin-top: 8px; padding: 12px 16px; }
.rs-head { font-size: 15px; font-weight: bold; margin-bottom: 10px; }
.rs-scroll { display: flex; gap: 10px; overflow-x: auto; }
.rs-card { flex-shrink: 0; width: 110px; }
.rs-name { font-size: 12px; color: #333; line-height: 16px; margin-top: 4px; height: 32px; }
.rs-price { color: #ff0036; font-size: 14px; font-weight: bold; }
</style>
