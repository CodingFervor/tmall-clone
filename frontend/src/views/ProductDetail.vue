<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview, uploadImage, checkFavorite, toggleFavorite, replyReview, getPriceHistory, checkRestock, subscribeRestock, unsubscribeRestock, getProductQA, askProductQA, markReviewUseful, checkPriceAlert, subscribePriceAlert, unsubscribePriceAlert } from '../api'

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
const priceAlertSubscribed = ref(false)
const priceAlertTarget = ref(0)
const showPriceAlert = ref(false)
const priceAlertInput = ref(0)
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
// ---- 360°展示 (3D rotate placeholder) ----
// Cycles through the gallery images every 1.5s with a rotateY 3D flip
// animation to simulate a 360° turntable view. Requires at least 2 images;
// a single-image product surfaces "暂无360°视图" instead.
const rotating360 = ref(false)
const rotateIdx = ref(0)
// A monotonically increasing key used to restart the CSS rotateY animation on
// each cycle (re-mounting the animated node via :key re-triggers the keyframes).
const rotateTick = ref(0)
let rotateTimer = null
const rotateImage = computed(() => gallery.value[rotateIdx.value] || '')
function startRotate360() {
  if (gallery.value.length < 2) { showToast('暂无360°视图'); return }
  rotating360.value = true
  rotateIdx.value = 0
  rotateTick.value = 0
  // Pause the swipe autoplay by leaving it mounted but hidden; our timer drives
  // the image cycling for the 360° turntable.
  rotateTimer = setInterval(() => {
    rotateIdx.value = (rotateIdx.value + 1) % gallery.value.length
    rotateTick.value += 1
  }, 1500)
}
function stopRotate360() {
  if (rotateTimer) { clearInterval(rotateTimer); rotateTimer = null }
  rotating360.value = false
}
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
// Review filter tabs (评价筛选标签): horizontal pills that filter the review
// list by rating band or by "has images". Counts are shown in parentheses.
// `photo` treats a review as having media when its comma-separated images
// field contains at least one non-empty entry.
const activeReviewFilter = ref('all')
const reviewFilters = computed(() => {
  const rs = reviews.value
  return [
    { key: 'all', label: '全部', count: rs.length },
    { key: 'good', label: '好评 4-5★', count: rs.filter((r) => r.rating >= 4).length },
    { key: 'mid', label: '中评 3★', count: rs.filter((r) => r.rating === 3).length },
    { key: 'bad', label: '差评 1-2★', count: rs.filter((r) => r.rating <= 2).length },
    { key: 'photo', label: '有图', count: rs.filter((r) => r.images && String(r.images).trim()).length },
  ]
})
const filteredReviews = computed(() => {
  const f = activeReviewFilter.value
  if (f === 'good') return reviews.value.filter((r) => r.rating >= 4)
  if (f === 'mid') return reviews.value.filter((r) => r.rating === 3)
  if (f === 'bad') return reviews.value.filter((r) => r.rating <= 2)
  if (f === 'photo') return reviews.value.filter((r) => r.images && String(r.images).trim())
  return reviews.value
})
// Stock pressure meter (库存紧张指示): prefers the selected SKU's stock and
// falls back to the product stock. As stock drops the bar fills more and the
// color warms from green → orange → red, blinking when nearly sold out.
const effectiveStock = computed(() => {
  if (selectedSKU.value && typeof selectedSKU.value.stock === 'number') return selectedSKU.value.stock
  return product.value ? Number(product.value.stock) || 0 : 0
})
const stockState = computed(() => {
  const s = effectiveStock.value
  if (s > 100) return { label: '库存充足', pct: 25, color: '#07c160', blink: false, fire: false }
  if (s >= 20) return { label: '有货', pct: 50, color: '#07c160', blink: false, fire: false }
  if (s >= 5) return { label: '库存紧张', pct: 80, color: '#ff9800', blink: false, fire: false }
  if (s >= 1) return { label: `仅剩${s}件`, pct: 95, color: '#ff0036', blink: true, fire: true }
  return { label: '缺货', pct: 100, color: '#999', blink: false, fire: false }
})
async function onUploadReviewImage(item) {
  try { const res = await uploadImage(item.file); reviewImages.value.push(res.url) } catch (e) { showToast('图片上传失败') }
}
function removeReviewImage(idx) { reviewImages.value.splice(idx, 1) }
// Video reviews (视频评价): a video URL entered by the user is stored with a
// "video:" prefix in the comma-separated images field alongside photos.
// Only one video per review; blank/whitespace-only input is ignored.
const reviewVideo = ref('')
const showVideoInput = ref(false)
function addReviewVideo() {
  const url = reviewVideo.value.trim()
  if (!url) { showToast('请输入视频链接'); return }
  if (!/^https?:\/\//i.test(url)) { showToast('请输入有效的视频链接'); return }
  reviewVideo.value = ''
  showSuccessToast('视频已添加')
}
function removeReviewVideo() { reviewVideo.value = '' }
// Parse a review's comma-separated images field into media entries.
// Entries prefixed with "video:" render as a <video> player; the rest are photos.
function reviewMedia(r) {
  if (!r.images) return []
  return r.images.split(',').map((s) => s.trim()).filter(Boolean).map((raw) => {
    if (raw.startsWith('video:')) return { type: 'video', url: raw.slice(6) }
    return { type: 'image', url: raw }
  })
}

// ---- 比价悬浮球 (price track floating ball) ----
// Remember the last viewed product in localStorage so the Home page can surface
// a floating price-tracking ball. We store the id, name, thumb and the price at
// the time of viewing; Home re-fetches the live price to compute the ↑↓ trend.
const PRICE_TRACK_KEY = 'tm_price_track'
function rememberPriceTrack(p) {
  if (!p || !p.id) return
  try {
    localStorage.setItem(PRICE_TRACK_KEY, JSON.stringify({
      id: p.id,
      name: p.name,
      image: p.image,
      price: Number(p.price),
      ts: Date.now(),
    }))
  } catch (_) { /* storage may be full / unavailable */ }
}

onMounted(async () => {
  try {
    const res = await getProduct(route.params.id)
    product.value = res.data
    rememberPriceTrack(product.value)
    reviews.value = res.reviews || []
    skus.value = res.skus || []
    if (res.recommended_sku) { selectedSKU.value = res.recommended_sku; recommendedSKU.value = res.recommended_sku }
    relatedProducts.value = res.related || []
    if (localStorage.getItem('tm_token')) {
      favorited.value = await checkFavorite(route.params.id)
    }
    getPriceHistory(route.params.id).then((d) => { priceHistory.value = d.data || []; priceStats.value = d.stats }).catch(() => {})
    if (localStorage.getItem('tm_token')) { checkRestock(route.params.id).then((s) => { restockSubscribed.value = s }).catch(() => {}) }
    if (localStorage.getItem('tm_token')) { checkPriceAlert(route.params.id).then((d) => { priceAlertSubscribed.value = !!d.subscribed; priceAlertTarget.value = d.target_price || 0 }).catch(() => {}) }
    getProductQA(route.params.id).then((d) => { qaList.value = d || [] }).catch(() => {})
  } catch (e) {
    showToast('商品不存在')
  } finally {
    loading.value = false
    // Wire up the scroll-spy observer for the detail tabs now that the DOM exists.
    nextTick(() => {
      // Attach the auto-play observer for the intro video, if present.
      setupVideoObserver()
      const sectionEls = ['product', 'reviews', 'qa', 'recommend']
        .map((k) => tabRefs[k].value)
        .filter(Boolean)
      if (!sectionEls.length) return
      tabObserver = new IntersectionObserver(
        (entries) => {
          // Pick the topmost intersecting section as the active tab.
          const visible = entries
            .filter((e) => e.isIntersecting)
            .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
          if (visible.length) {
            const el = visible[0].target
            const found = ['product', 'reviews', 'qa', 'recommend'].find((k) => tabRefs[k].value === el)
            if (found) activeTab.value = found
          }
        },
        // Trigger when a section's top crosses ~120px from the viewport top
        // (below the fixed nav + sticky tabs).
        { rootMargin: '-120px 0px -65% 0px', threshold: 0 }
      )
      sectionEls.forEach((el) => tabObserver.observe(el))
    })
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
    // Combine photos and the optional video (prefixed "video:") into one images field.
    const parts = reviewImages.value.slice()
    const vid = reviewVideo.value.trim()
    if (vid) parts.push('video:' + vid)
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value, images: parts.join(',') })
    reviews.value.unshift(rv); showReview.value = false; reviewContent.value = ''; reviewImages.value = []; reviewVideo.value = ''; showSuccessToast('评价成功')
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
// SKU spec matrix table (商品规格矩阵表): when a product has 2+ SKUs whose
// spec JSON carries 2+ dimensions (e.g. {"颜色":"黑色","版本":"256GB"}), build a
// table with one column per dimension plus price & stock. SKUs with unparseable
// specs are dropped; if fewer than 2 valid rows or a single dimension remain, the
// matrix is hidden.
const specMatrix = computed(() => {
  if (!skus.value || skus.value.length < 2) return null
  const dims = []
  const seen = new Set()
  const rows = []
  for (const s of skus.value) {
    let spec = null
    try { spec = JSON.parse(s.spec) } catch { spec = null }
    if (!spec || typeof spec !== 'object' || Array.isArray(spec)) continue
    const values = {}
    for (const [k, v] of Object.entries(spec)) {
      values[k] = String(v)
      if (!seen.has(k)) { seen.add(k); dims.push(k) }
    }
    if (!Object.keys(values).length) continue
    rows.push({ sku: s, values })
  }
  if (rows.length < 2 || dims.length < 2) return null
  return { dims, rows }
})
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
// 分享卡片 (product share card): "复制商品信息" copies a formatted summary
// (name, spec, price, shop, link) for easy pasting into chat. "生成长图" is a
// lightweight demo that confirms the long-image generation (no canvas dependency).
function shareSpec() {
  if (selectedSKU.value && selectedSKU.value.spec_text) return selectedSKU.value.spec_text
  if (specMatrix.value && specMatrix.value.dims.length) {
    return specMatrix.value.dims.map((d) => d).join('/')
  }
  return '默认规格'
}
async function copyProductInfo() {
  const p = product.value
  if (!p) return
  const text =
    `【天猫】${p.name}\n` +
    `规格：${shareSpec()}\n` +
    `价格：¥${fmt(currentPrice())}\n` +
    `店铺：${p.shop || '天猫商城'}\n` +
    `链接：${window.location.href}`
  try {
    await navigator.clipboard.writeText(text)
    showSuccessToast('商品信息已复制')
  } catch (e) {
    // Fallback for non-secure contexts / older browsers.
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try { document.execCommand('copy'); showSuccessToast('商品信息已复制') } catch (_) { showToast('复制失败，请手动复制') }
    document.body.removeChild(ta)
  }
}
const generatingLongImage = ref(false)
function generateLongImage() {
  // Demo: simulate generating a long share image (no server/canvas round-trip).
  generatingLongImage.value = true
  setTimeout(() => {
    generatingLongImage.value = false
    showSuccessToast('长图已生成，可保存分享')
  }, 700)
}
async function toggleRestock() {
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return }
  try {
    if (restockSubscribed.value) { await unsubscribeRestock(product.value.id); restockSubscribed.value = false; showSuccessToast('已取消到货通知') }
    else { await subscribeRestock(product.value.id); restockSubscribed.value = true; showSuccessToast('到货后将通知您') }
  } catch (e) { showToast('操作失败') }
}
// Open the price-drop alert popup. Default the target to 90% of the current price
// (or the previously saved target if already subscribed).
function openPriceAlert() {
  if (!localStorage.getItem('tm_token')) { showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login')); return }
  const base = priceAlertTarget.value > 0 ? priceAlertTarget.value : Math.round(currentPrice() * 0.9 * 100) / 100
  priceAlertInput.value = base
  showPriceAlert.value = true
}
async function submitPriceAlert() {
  const target = Number(priceAlertInput.value)
  if (!target || target <= 0) { showToast('请输入有效的目标价'); return }
  if (target >= currentPrice()) { showToast('目标价需低于当前价'); return }
  try {
    await subscribePriceAlert(product.value.id, target)
    priceAlertSubscribed.value = true; priceAlertTarget.value = target; showPriceAlert.value = false
    showSuccessToast('降价后将通知您')
  } catch (e) { showToast('订阅失败') }
}
async function cancelPriceAlert() {
  try {
    await unsubscribePriceAlert(product.value.id)
    priceAlertSubscribed.value = false; priceAlertTarget.value = 0; showSuccessToast('已取消降价提醒')
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
// Eco score (环保评分): deterministic 0-100 score derived from a hash of the
// product id, with a level label (优秀/良好/合格/一般) and a matching color class.
function idHash() {
  if (!product.value) return 0
  const id = String(product.value.id)
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = (hash * 31 + id.charCodeAt(i)) >>> 0
  }
  return hash
}
const ecoScore = computed(() => idHash() % 101)
const ecoLevel = computed(() => {
  const s = ecoScore.value
  if (s >= 90) return '优秀'
  if (s >= 70) return '良好'
  if (s >= 60) return '合格'
  return '一般'
})
const ecoClass = computed(() => {
  const s = ecoScore.value
  if (s >= 90) return 'eco-excellent'
  if (s >= 70) return 'eco-good'
  if (s >= 60) return 'eco-pass'
  return 'eco-normal'
})
// Brand story (品牌故事): collapsed by default. The narrative is templated from
// the shop name, and the 品牌理念 tags are 3 unique picks from a fixed pool,
// selected deterministically via a seeded shuffle of the product id hash.
const showBrandStory = ref(false)
const brandStoryText = computed(() => {
  const shop = (product.value && product.value.shop) || '本品牌'
  return `${shop}创立多年，始终坚持品质至上，致力于为消费者提供优质的产品与服务。秉承匠心精神，从选材到工艺层层把关，只为打造让用户放心的好物。${shop}，值得信赖的品牌。`
})
const brandStoryTags = computed(() => {
  const pool = ['品质保证', '正品行货', '售后无忧', '极速配送', '用户至上', '匠心工艺']
  const shuffled = [...pool]
  let seed = idHash()
  for (let i = shuffled.length - 1; i > 0; i--) {
    seed = (seed * 1103515245 + 12345) >>> 0
    const j = seed % (i + 1)
    ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  return shuffled.slice(0, 3)
})
// Product origin map (产地溯源): city is picked deterministically from the
// product id via `id % 10` over a fixed 10-city pool. Each city is paired with
// its province for the "省份·城市" display format; municipalities use the city
// name as their own province label.
const originCityPool = ['北京', '上海', '广州', '深圳', '杭州', '成都', '武汉', '西安', '南京', '重庆']
const originCityProvince = {
  '北京': '北京', '上海': '上海', '广州': '广东', '深圳': '广东', '杭州': '浙江',
  '成都': '四川', '武汉': '湖北', '西安': '陕西', '南京': '江苏', '重庆': '重庆',
}
const originCity = computed(() => {
  if (!product.value) return ''
  const idx = Number(product.value.id) % 10
  return originCityPool[idx] || originCityPool[0]
})
const originText = computed(() => {
  const city = originCity.value
  const prov = originCityProvince[city] || city
  return `${prov}·${city}`
})
const showOriginMap = ref(false)
function openOriginMap() { showOriginMap.value = true }

// Purchase note templates (购买须知): a fixed set of 3 notes shown under a
// collapsible header. Collapsed by default to mirror the brand story pattern.
const showPurchaseNote = ref(false)
const purchaseNotes = [
  '7天无理由退货，请保持商品完好',
  '因显示器差异，商品颜色可能与实物略有差异',
  '请确认商品规格（颜色/尺寸/版本）后下单',
]
// ---- 详情Tab导航 (sticky section tabs) ----
// 商品|评价|问答|推荐. Clicking a tab scrolls to the section; the active tab is
// tracked both on click and via an IntersectionObserver as the user scrolls.
const detailTabs = [
  { key: 'product', label: '商品' },
  { key: 'reviews', label: '评价' },
  { key: 'qa', label: '问答' },
  { key: 'recommend', label: '推荐' },
]
const activeTab = ref('product')
const tabRefs = { product: ref(null), reviews: ref(null), qa: ref(null), recommend: ref(null) }
const tabsBar = ref(null)
let tabObserver = null

// ---- 视频自动播放 (product video auto-play) ----
// Watch the intro video with an IntersectionObserver: when ≥50% of it is in the
// viewport it muted-plays automatically; scrolling away pauses it. The video
// stays muted until the user taps the "🔇点击取消静音" overlay, after which the
// overlay is hidden. Tapping the video body toggles play/pause.
const productVideoRef = ref(null)
let videoObserver = null
const videoMuted = ref(true)
const videoPlaying = ref(false)
// We remove the native controls so our overlay + tap-to-toggle drive playback.
function onVideoLoaded() {
  const v = productVideoRef.value
  if (!v) return
  v.muted = true
}
function onVideoPlaying() { videoPlaying.value = true }
function onVideoPause() { videoPlaying.value = false }
function toggleVideoPlay() {
  const v = productVideoRef.value
  if (!v) return
  if (v.paused) { v.play().catch(() => {}) }
  else { v.pause() }
}
function unmuteVideo() {
  const v = productVideoRef.value
  if (!v) return
  v.muted = false
  videoMuted.value = false
  // If it was paused (e.g. by a browser policy when leaving the viewport),
  // resume now that the user has engaged.
  if (v.paused) v.play().catch(() => {})
}
// (Re)attach the IntersectionObserver once the video element exists.
function setupVideoObserver() {
  const v = productVideoRef.value
  if (!v || videoObserver) return
  videoObserver = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) {
          // IntersectionObserver supports a threshold array, but some browsers
          // report fractional ratios; treat >=0.5 visibility as "in view".
          if (e.intersectionRatio >= 0.5) {
            v.play().catch(() => { /* autoplay can be blocked; ignore */ })
          } else {
            v.pause()
          }
        }
      }
    },
    { threshold: [0, 0.5, 1] }
  )
  videoObserver.observe(v)
}
function scrollToTab(key) {
  const el = tabRefs[key] && tabRefs[key].value
  if (!el) return
  activeTab.value = key
  // Offset for the fixed nav-bar + sticky tabs bar so the section heading
  // isn't hidden underneath.
  const offset = 96
  const top = el.getBoundingClientRect().top + window.scrollY - offset
  window.scrollTo({ top, behavior: 'smooth' })
}
onUnmounted(() => {
  if (tabObserver) { tabObserver.disconnect(); tabObserver = null }
  if (videoObserver) { videoObserver.disconnect(); videoObserver = null }
  // Stop the 360° turntable timer if it was running.
  stopRotate360()
})
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Product intro video (商品视频介绍) — auto-plays (muted) when ≥50% in view -->
    <div v-if="product.video_url" class="product-video">
      <video
        ref="productVideoRef"
        :src="product.video_url"
        preload="metadata"
        playsinline
        muted
        class="pv-player"
        @loadeddata="onVideoLoaded"
        @playing="onVideoPlaying"
        @pause="onVideoPause"
        @click="toggleVideoPlay"
      ></video>
      <!-- "🔇点击取消静音" overlay: shown while muted; tapping unmutes. -->
      <div v-if="videoMuted" class="pv-unmute" @click.stop="unmuteVideo">🔇 点击取消静音</div>
      <!-- Play/pause indicator (briefly shows when paused by the user) -->
      <div v-if="!videoPlaying" class="pv-paused-badge" @click.stop="toggleVideoPlay">▶</div>
    </div>
    <!-- 360°展示 (3D rotate placeholder): a turntable view that cycles the
         gallery images every 1.5s with a rotateY flip. "停止旋转" exits. -->
    <div v-if="rotating360" class="gallery-360">
      <van-image
        :key="rotateTick"
        width="100%"
        height="375"
        :src="rotateImage"
        fit="cover"
        class="g360-img"
      />
      <div class="g360-hint">🔄 360°展示中 · {{ rotateIdx + 1 }}/{{ gallery.length }}</div>
      <van-button class="g360-stop" size="small" round type="danger" @click="stopRotate360">停止旋转</van-button>
    </div>
    <template v-else>
      <van-swipe class="gallery" :autoplay="3000" indicator-color="#ff0036" v-if="gallery.length > 1">
        <van-swipe-item v-for="(img, i) in gallery" :key="i">
          <van-image width="100%" height="375" :src="img" fit="cover" />
        </van-swipe-item>
      </van-swipe>
      <van-image v-else width="100%" height="375" :src="product.image" fit="cover" />
      <!-- 360° entry button (3D rotate placeholder) -->
      <div class="gallery-360-btn" @click="startRotate360">🔄 360°</div>
    </template>
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
      <!-- SKU spec matrix table (商品规格矩阵表) -->
      <table v-if="specMatrix" class="spec-matrix">
        <thead>
          <tr>
            <th v-for="d in specMatrix.dims" :key="d">{{ d }}</th>
            <th>价格</th>
            <th>库存</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in specMatrix.rows" :key="r.sku.id" :class="{ 'row-active': selectedSKU && selectedSKU.id === r.sku.id }" @click="selectSKU(r.sku)">
            <td v-for="d in specMatrix.dims" :key="d">{{ r.values[d] || '-' }}</td>
            <td class="sm-price">¥{{ fmt(r.sku.price) }}</td>
            <td class="sm-stock" :class="{ 'stock-low': r.sku.stock <= 10 }">{{ r.sku.stock > 0 ? r.sku.stock : '缺货' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <!-- Stock pressure meter (库存紧张指示): reflects selected SKU stock or product stock -->
    <div class="stock-meter" :class="{ blink: stockState.blink }">
      <div class="sm-track">
        <div class="sm-fill" :style="{ width: stockState.pct + '%', background: stockState.color }"></div>
      </div>
      <span class="sm-label" :style="{ color: stockState.color }">
        <span v-if="stockState.fire" class="sm-fire">🔥</span>{{ stockState.label }}
      </span>
    </div>
    <div class="title-block" :ref="(el) => tabRefs.product.value = el">
      <div v-if="product.is_genuine" class="genuine-row"><span class="genuine-tag">正品保障</span> <span class="brand-link" v-if="product.brand_id" @click="router.push('/brand/' + product.brand_id)">{{ product.brand_name }} ›</span></div>
      <h2 class="p-title">{{ product.name }}</h2>
      <p class="p-sub">{{ product.subtitle }}</p>
    </div>
    <!-- Sticky detail section tabs (详情Tab导航): 商品|评价|问答|推荐 -->
    <div class="detail-tabs" ref="tabsBar">
      <span
        v-for="t in detailTabs"
        :key="t.key"
        class="detail-tab"
        :class="{ active: activeTab === t.key }"
        @click="scrollToTab(t.key)"
      >{{ t.label }}</span>
    </div>
    <van-cell-group inset>
      <van-cell title="店铺" :value="product.shop" is-link v-if="product.brand_id" @click="router.push('/brand/' + product.brand_id)" />
      <van-cell title="店铺" :value="product.shop" is-link v-else @click="router.push('/shop/' + encodeURIComponent(product.shop))" />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="🌱 环保评分">
        <template #value>
          <span class="eco-score" :class="ecoClass">{{ ecoScore }}分 {{ ecoLevel }}</span>
        </template>
      </van-cell>
      <van-cell title="标签" :value="product.tags || '正品保障'" />
      <van-cell title="预计送达">
        <template #value>
          <span class="delivery-est"><span class="delivery-icon">🚚</span> {{ deliveryEstimate() }}</span>
        </template>
      </van-cell>
      <!-- Product origin map (产地溯源): city chosen by product id % 10 -->
      <van-cell title="📍 产地溯源" is-link :value="originText" @click="openOriginMap" />
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
      <!-- Price drop alert (降价提醒) -->
      <div class="ph-alert">
        <template v-if="!priceAlertSubscribed">
          <span class="pha-icon">🔔</span>
          <span class="pha-text">心仪好价？设置降价提醒</span>
          <van-button size="mini" type="danger" round @click="openPriceAlert">降价提醒</van-button>
        </template>
        <template v-else>
          <span class="pha-icon">✅</span>
          <span class="pha-text">已订阅，降到 <b class="pha-target">¥{{ fmt(priceAlertTarget) }}</b> 将通知您</span>
          <van-button size="mini" plain round @click="openPriceAlert">修改</van-button>
          <van-button size="mini" plain round @click="cancelPriceAlert">取消</van-button>
        </template>
      </div>
    </div>
    <!-- Product Q&A (商品问答) -->
    <div class="qa-section" :ref="(el) => tabRefs.qa.value = el">
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
    <!-- Price drop alert popup (降价提醒) -->
    <van-popup v-model:show="showPriceAlert" position="bottom" round closeable>
      <div class="pa-form">
        <h3>降价提醒</h3>
        <p class="pa-tip">当商品价格降至目标价时，我们将第一时间通知您</p>
        <div class="pa-cur">当前价格：<b class="pa-cur-price">¥{{ fmt(currentPrice()) }}</b></div>
        <van-field v-model="priceAlertInput" type="number" label="目标价格" placeholder="设置期望价格" input-align="right">
          <template #left-icon><span class="pa-yuan">¥</span></template>
        </van-field>
        <van-button type="danger" block round @click="submitPriceAlert" style="margin-top:16px">{{ priceAlertSubscribed ? '更新提醒' : '开启提醒' }}</van-button>
        <van-button v-if="priceAlertSubscribed" block plain round @click="cancelPriceAlert(); showPriceAlert = false" style="margin-top:8px">取消提醒</van-button>
      </div>
    </van-popup>
    <!-- Brand story (品牌故事) -->
    <div class="brand-story">
      <div class="bs-head" @click="showBrandStory = !showBrandStory">
        <span class="bs-title">📖 品牌故事</span>
        <van-icon :name="showBrandStory ? 'arrow-up' : 'arrow-down'" class="bs-arrow" />
      </div>
      <div v-show="showBrandStory" class="bs-body">
        <p class="bs-text">{{ brandStoryText }}</p>
        <div class="bs-tags">
          <span class="bs-tag-label">品牌理念</span>
          <span v-for="t in brandStoryTags" :key="t" class="bs-tag">{{ t }}</span>
        </div>
      </div>
    </div>
    <!-- Purchase note templates (购买须知): collapsible, 3 fixed notes -->
    <div class="purchase-note">
      <div class="pn-head" @click="showPurchaseNote = !showPurchaseNote">
        <span class="pn-title">📋 购买须知</span>
        <van-icon :name="showPurchaseNote ? 'arrow-up' : 'arrow-down'" class="pn-arrow" />
      </div>
      <div v-show="showPurchaseNote" class="pn-body">
        <div v-for="(n, i) in purchaseNotes" :key="i" class="pn-item"><span class="pn-check">✓</span><span class="pn-text">{{ n }}</span></div>
      </div>
    </div>
    <div v-if="product.description" class="desc"><h3>商品详情</h3><p>{{ product.description }}</p></div>
    <div class="reviews" :ref="(el) => tabRefs.reviews.value = el">
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
      <!-- Review filter tabs (评价筛选标签) -->
      <div v-if="reviews.length" class="rev-filters">
        <span
          v-for="f in reviewFilters"
          :key="f.key"
          class="rev-filter-chip"
          :class="{ active: activeReviewFilter === f.key }"
          @click="activeReviewFilter = f.key"
        >{{ f.label }}({{ f.count }})</span>
      </div>
      <div v-for="r in filteredReviews" :key="r.id" class="rev-item">
        <div class="rev-user"><span>{{ r.username }}</span><van-rate v-model="r.rating" readonly size="12" /><span class="rev-reply-btn" @click="toggleReply(r)">回复</span></div>
        <div class="rev-content">{{ r.content }}</div>
        <div v-if="reviewMedia(r).length" class="rev-photos">
          <template v-for="(m, i) in reviewMedia(r)" :key="i">
            <video v-if="m.type === 'video'" :src="m.url" controls preload="metadata" class="rev-video"></video>
            <van-image v-else width="72" height="72" radius="6" :src="m.url" fit="cover" />
          </template>
        </div>
        <div class="rev-actions"><span class="rev-useful-btn" @click="doUseful(r)"><van-icon name="good-job-o" /> 有用 ({{ r.useful || 0 }})</span></div>
        <div v-if="r.reply" class="rev-reply"><span class="rev-reply-name">{{ r.reply.username }}：</span>{{ r.reply.content }}</div>
        <div v-if="replyingTo === r.id" class="rev-reply-box">
          <van-field v-model="replyText" placeholder="写下你的回复..." />
          <van-button size="small" type="danger" @click="submitReply(r)">发送</van-button>
        </div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
      <van-empty v-else-if="!filteredReviews.length" :description="activeReviewFilter === 'photo' ? '暂无带图评价' : '该筛选下暂无评价'" />
    </div>
    <!-- Related products (看了又看) -->
    <div v-if="relatedProducts.length" class="related-section" :ref="(el) => tabRefs.recommend.value = el">
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
          <!-- 规格行 (specs row) -->
          <div class="pc-spec"><span class="pc-spec-label">规格</span><span class="pc-spec-val">{{ shareSpec() }}</span></div>
          <div class="pc-price">¥{{ fmt(currentPrice()) }}</div>
          <div class="pc-qr">
            <div class="qr-box">
              <div class="qr-grid"><div v-for="n in 64" :key="n" class="qr-cell" :class="{ on: qrPattern(n) }"></div></div>
            </div>
            <div class="qr-text">扫码查看商品</div>
          </div>
          <div class="pc-brand">天猫 TMALL</div>
          <!-- 天猫水印 (Tmall watermark) -->
          <div class="pc-watermark">天猫</div>
        </div>
        <van-button block type="danger" round :loading="generatingLongImage" style="margin-top: 12px" @click="generateLongImage">📋 生成长图</van-button>
        <van-button block plain type="danger" round style="margin-top: 8px" @click="copyProductInfo">复制商品信息</van-button>
        <van-button block plain style="margin-top: 8px" @click="copyShareLink">复制分享链接</van-button>
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
        <!-- Video review (视频评价): paste a video URL; stored as "video:URL" in images. -->
        <div class="rev-video-upload">
          <van-button icon="video-o" size="small" plain round @click="showVideoInput = !showVideoInput">添加视频</van-button>
          <div v-if="showVideoInput" class="rev-video-input">
            <van-field v-model="reviewVideo" placeholder="粘贴视频链接 https://..." clearable />
            <van-button size="small" type="danger" @click="addReviewVideo">添加</van-button>
          </div>
          <div v-if="reviewVideo" class="rev-video-chip">
            <span class="rv-chip-text van-ellipsis">🎬 {{ reviewVideo }}</span>
            <van-icon name="cross" class="rv-chip-del" @click="removeReviewVideo" />
          </div>
        </div>
        <van-button type="danger" block @click="submitReview" style="margin-top:12px">提交评价</van-button>
      </div>
    </van-popup>
    <!-- Origin map popup (产地溯源): visual route 📍 origin → 🏠 you -->
    <van-popup v-model:show="showOriginMap" round closeable position="bottom" :style="{ width: '90%' }">
      <div class="origin-map">
        <div class="om-head">📍 产地溯源</div>
        <div class="om-route">
          <div class="om-point om-origin">
            <div class="om-icon">📍</div>
            <div class="om-label">产地</div>
            <div class="om-city">{{ originText }}</div>
          </div>
          <div class="om-line">
            <div class="om-dash"></div>
            <div class="om-truck">🚚</div>
          </div>
          <div class="om-point om-home">
            <div class="om-icon">🏠</div>
            <div class="om-label">您</div>
            <div class="om-city">送货上门</div>
          </div>
        </div>
        <div class="om-tip">商品由 {{ originText }} 发货，正品保障，全程可溯源</div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.detail { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.product-video { position: relative; background: #000; width: 100%; }
.pv-player { width: 100%; max-height: 280px; object-fit: contain; display: block; }
/* 视频自动播放 (product video auto-play) overlays */
.pv-unmute { position: absolute; right: 12px; bottom: 12px; background: rgba(0,0,0,0.6); color: #fff; font-size: 12px; padding: 5px 12px; border-radius: 14px; cursor: pointer; user-select: none; backdrop-filter: blur(2px); }
.pv-paused-badge { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 56px; height: 56px; border-radius: 50%; background: rgba(255,0,54,0.85); color: #fff; font-size: 24px; display: flex; align-items: center; justify-content: center; cursor: pointer; box-shadow: 0 2px 10px rgba(0,0,0,0.4); }
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
/* SKU spec matrix table (商品规格矩阵表) */
.spec-matrix { width: 100%; margin-top: 12px; border-collapse: collapse; font-size: 12px; }
.spec-matrix th, .spec-matrix td { border: 1px solid #f0f0f0; padding: 6px 4px; text-align: center; white-space: nowrap; }
.spec-matrix thead th { background: #fafafa; color: #999; font-weight: normal; }
.spec-matrix tbody tr { transition: background 0.15s; }
.spec-matrix tbody tr:active { background: #fff5f6; }
.spec-matrix tr.row-active { background: #fff5f6; }
.spec-matrix tr.row-active td { border-color: #ffc6cf; }
.sm-price { color: #ff0036; font-weight: bold; }
.sm-stock { color: #666; }
.sm-stock.stock-low { color: #ff9800; }
.title-block { padding: 0 16px 12px; background: #fff; }
/* Sticky detail section tabs (详情Tab导航) */
.detail-tabs { position: sticky; top: 46px; z-index: 10; display: flex; align-items: center; justify-content: space-around; background: #fff; border-bottom: 1px solid #f0f0f0; padding: 10px 0; margin-top: -1px; }
.detail-tab { position: relative; font-size: 14px; color: #666; padding: 4px 8px; cursor: pointer; transition: color 0.15s; }
.detail-tab.active { color: #ff0036; font-weight: bold; }
.detail-tab.active::after { content: ''; position: absolute; left: 50%; bottom: -4px; transform: translateX(-50%); width: 20px; height: 3px; border-radius: 2px; background: #ff0036; }
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
/* Price drop alert (降价提醒) */
.ph-alert { display: flex; align-items: center; gap: 8px; margin-top: 12px; padding: 10px 12px; background: #fff5f6; border: 1px solid #ffd6df; border-radius: 8px; }
.pha-icon { font-size: 16px; }
.pha-text { flex: 1; font-size: 13px; color: #333; line-height: 18px; }
.pha-text b.pha-target { color: #ff0036; }
.pa-form { padding: 20px; }
.pa-form h3 { text-align: center; margin-bottom: 8px; }
.pa-tip { font-size: 12px; color: #999; text-align: center; margin-bottom: 16px; }
.pa-cur { font-size: 13px; color: #666; margin-bottom: 12px; }
.pa-cur-price { color: #ff0036; font-size: 18px; }
.pa-yuan { color: #ff0036; font-weight: bold; }
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
/* Review filter tabs (评价筛选标签) */
.rev-filters { display: flex; gap: 8px; overflow-x: auto; padding: 10px 0; border-top: 1px solid #f5f5f5; -webkit-overflow-scrolling: touch; }
.rev-filters::-webkit-scrollbar { display: none; }
.rev-filter-chip { flex-shrink: 0; padding: 5px 14px; background: #f7f7f7; border: 1px solid #eee; border-radius: 14px; font-size: 13px; color: #333; white-space: nowrap; cursor: pointer; transition: all 0.15s; }
.rev-filter-chip.active { background: #ff0036; border-color: #ff0036; color: #fff; }
/* Stock pressure meter (库存紧张指示) */
.stock-meter { display: flex; align-items: center; gap: 10px; padding: 10px 16px; background: #fff; border-top: 1px solid #f5f5f5; }
.sm-track { flex: 1; height: 8px; background: #f0f0f0; border-radius: 4px; overflow: hidden; }
.sm-fill { height: 100%; border-radius: 4px; transition: width 0.3s, background 0.3s; }
.sm-label { font-size: 13px; font-weight: bold; white-space: nowrap; flex-shrink: 0; min-width: 70px; text-align: right; }
.sm-fire { margin-right: 2px; }
.stock-meter.blink .sm-fill { animation: stock-blink 1s ease-in-out infinite; }
.stock-meter.blink .sm-label { animation: stock-blink 1s ease-in-out infinite; }
@keyframes stock-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
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
/* Video review (视频评价) */
.rev-video { width: 200px; height: 200px; border-radius: 6px; background: #000; object-fit: cover; }
.rev-video-upload { margin: 8px 0; }
.rev-video-input { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.rev-video-input .van-field { flex: 1; border: 1px solid #eee; border-radius: 6px; }
.rev-video-chip { display: flex; align-items: center; gap: 6px; margin-top: 8px; padding: 6px 10px; background: #fff5f6; border: 1px solid #ffd6df; border-radius: 6px; }
.rv-chip-text { font-size: 12px; color: #ff0036; flex: 1; }
.rv-chip-del { color: #ff0036; font-size: 14px; cursor: pointer; }
.poster { padding: 20px; }
.poster-head { text-align: center; font-size: 16px; font-weight: bold; margin-bottom: 16px; }
.poster-card { position: relative; background: #fff; border: 1px solid #eee; border-radius: 12px; padding: 16px; text-align: center; overflow: hidden; }
.pc-name { font-size: 15px; line-height: 22px; margin: 12px 0 6px; text-align: left; }
.pc-price { color: #ff0036; font-size: 24px; font-weight: bold; text-align: left; }
.pc-qr { display: flex; flex-direction: column; align-items: center; margin-top: 16px; }
.qr-box { padding: 8px; border: 1px solid #eee; border-radius: 8px; }
.qr-grid { display: grid; grid-template-columns: repeat(8, 1fr); gap: 1px; width: 120px; height: 120px; }
.qr-cell { background: #fff; }
.qr-cell.on { background: #333; }
.qr-text { font-size: 11px; color: #999; margin-top: 6px; }
.pc-brand { color: #ff0036; font-size: 13px; font-weight: bold; margin-top: 12px; }
/* 规格行 (specs row) on the share card */
.pc-spec { display: flex; align-items: center; gap: 8px; text-align: left; margin-top: 8px; }
.pc-spec-label { font-size: 11px; color: #999; background: #f5f5f5; border-radius: 3px; padding: 1px 6px; flex-shrink: 0; }
.pc-spec-val { font-size: 13px; color: #333; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
/* 天猫水印 (Tmall watermark) — diagonal translucent brand stamp on the card */
.pc-watermark { position: absolute; top: 16px; right: 18px; font-size: 36px; font-weight: 900; color: rgba(255, 0, 54, 0.12); transform: rotate(-18deg); letter-spacing: 2px; pointer-events: none; user-select: none; z-index: 1; }
.related-section { background: #fff; margin-top: 8px; padding: 12px 16px; }
.rs-head { font-size: 15px; font-weight: bold; margin-bottom: 10px; }
.rs-scroll { display: flex; gap: 10px; overflow-x: auto; }
.rs-card { flex-shrink: 0; width: 110px; }
.rs-name { font-size: 12px; color: #333; line-height: 16px; margin-top: 4px; height: 32px; }
.rs-price { color: #ff0036; font-size: 14px; font-weight: bold; }
/* Eco score (环保评分) */
.eco-score { font-size: 13px; font-weight: bold; }
.eco-score.eco-excellent { color: #07c160; }
.eco-score.eco-good { color: #4caf50; }
.eco-score.eco-pass { color: #ff9800; }
.eco-score.eco-normal { color: #999; }
/* Brand story (品牌故事) */
.brand-story { background: #fff; margin-top: 8px; padding: 12px 16px; }
.bs-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; font-weight: bold; }
.bs-title { display: flex; align-items: center; gap: 4px; }
.bs-arrow { color: #999; font-size: 14px; }
.bs-body { padding-top: 10px; border-top: 1px solid #f5f5f5; margin-top: 10px; }
.bs-text { font-size: 13px; color: #666; line-height: 20px; margin: 0; }
.bs-tags { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 12px; }
.bs-tag-label { font-size: 13px; font-weight: bold; color: #333; }
.bs-tag { font-size: 12px; color: #ff0036; background: #fff5f6; border: 1px solid #ffd6df; padding: 3px 10px; border-radius: 12px; }
/* Purchase note templates (购买须知) */
.purchase-note { background: #fff; margin-top: 8px; padding: 12px 16px; }
.pn-head { display: flex; justify-content: space-between; align-items: center; font-size: 15px; font-weight: bold; }
.pn-title { display: flex; align-items: center; gap: 4px; }
.pn-arrow { color: #999; font-size: 14px; }
.pn-body { padding-top: 10px; border-top: 1px solid #f5f5f5; margin-top: 10px; }
.pn-item { display: flex; align-items: flex-start; gap: 8px; padding: 6px 0; }
.pn-check { color: #07c160; font-weight: bold; font-size: 14px; flex-shrink: 0; line-height: 20px; }
.pn-text { font-size: 13px; color: #666; line-height: 20px; flex: 1; }
/* Origin map popup (产地溯源) */
.origin-map { padding: 20px; }
.om-head { text-align: center; font-size: 16px; font-weight: bold; margin-bottom: 20px; }
.om-route { display: flex; align-items: center; justify-content: space-between; padding: 8px 4px 20px; }
.om-point { display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; }
.om-icon { font-size: 36px; }
.om-label { font-size: 12px; color: #999; }
.om-city { font-size: 13px; color: #ff0036; font-weight: bold; }
.om-origin .om-city { color: #ff9800; }
.om-line { flex: 1; display: flex; align-items: center; justify-content: center; position: relative; padding: 0 8px; }
.om-dash { width: 100%; height: 0; border-top: 3px dashed #ff0036; }
.om-truck { position: absolute; font-size: 24px; background: #fff; padding: 0 6px; }
.om-tip { text-align: center; font-size: 13px; color: #666; padding: 12px; background: #fff5f6; border: 1px solid #ffd6df; border-radius: 8px; line-height: 20px; }
/* 360°展示 (3D rotate placeholder) */
.gallery-360-btn { position: absolute; top: 56px; right: 14px; z-index: 6; background: rgba(0,0,0,0.55); color: #fff; font-size: 12px; padding: 6px 12px; border-radius: 16px; cursor: pointer; user-select: none; backdrop-filter: blur(2px); box-shadow: 0 2px 6px rgba(0,0,0,0.3); }
.gallery-360 { position: relative; background: #000; perspective: 1000px; }
.gallery-360 .g360-img { display: block; transform-origin: center center; animation: g360-rotatey 1.5s ease-in-out; }
@keyframes g360-rotatey { 0% { transform: rotateY(0deg); } 100% { transform: rotateY(360deg); } }
.g360-hint { position: absolute; left: 50%; bottom: 16px; transform: translateX(-50%); color: #fff; font-size: 12px; background: rgba(0,0,0,0.55); padding: 4px 12px; border-radius: 12px; }
.g360-stop { position: absolute; right: 14px; top: 14px; z-index: 7; }
</style>
