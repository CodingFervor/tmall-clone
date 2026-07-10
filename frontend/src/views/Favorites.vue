<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { listFavorites, toggleFavorite, addToCart } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)
const showShare = ref(false)

async function load() {
  loading.value = true
  try { list.value = await listFavorites() } catch (e) { showToast('加载失败') } finally { loading.value = false }
}
onMounted(load)

async function remove(f) {
  try {
    const res = await toggleFavorite(f.product_id)
    if (!res.favorited) { list.value = list.value.filter((x) => x.id !== f.id); showSuccessToast('已取消收藏') }
  } catch (e) { showToast('操作失败') }
}
async function addCart(f) {
  try { await addToCart(f.product_id, 1); showSuccessToast('已加入购物车') } catch (e) { showToast('请先登录') }
}
function fmt(n) { return Number(n).toFixed(2) }

// ---- Wishlist share card (收藏夹分享) ----
// Top 3 most-recently added favorites, plus the summed value of the whole list.
const topProducts = computed(() => list.value.slice(0, 3))
const totalValue = computed(() => list.value.reduce((s, f) => s + Number(f.price || 0), 0))
// Build a plain-text summary suitable for pasting into a chat.
function shareText() {
  const lines = topProducts.value.map((f, i) => `${i + 1}. ${f.product_name}  ¥${fmt(f.price)}`)
  return `我的天猫收藏夹（共${list.value.length}件，合计¥${fmt(totalValue.value)}）\n精选好物：\n${lines.join('\n')}`
}
async function copyShareText() {
  try { await navigator.clipboard.writeText(shareText()); showSuccessToast('已复制分享文本') }
  catch (e) { showToast('复制失败') }
}
</script>

<template>
  <div class="fav-page">
    <van-nav-bar title="我的收藏" left-arrow @click-left="router.back()" fixed placeholder>
      <template #right>
        <span class="share-entry" @click="showShare = true">分享收藏</span>
      </template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!list.length" description="还没有收藏商品" />
    <div v-else class="fav-list">
      <div v-for="f in list" :key="f.id" class="fav-card">
        <van-image width="90" height="90" radius="8" :src="f.product_image" fit="cover" @click="router.push('/product/' + f.product_id)" />
        <div class="fc-info">
          <div class="fc-name van-multi-ellipsis--l2" @click="router.push('/product/' + f.product_id)">{{ f.product_name }}</div>
          <div class="fc-price">¥{{ fmt(f.price) }}</div>
          <div class="fc-actions">
            <van-button size="small" plain round @click="remove(f)">取消收藏</van-button>
            <van-button size="small" type="danger" round @click="addCart(f)">加入购物车</van-button>
          </div>
        </div>
      </div>
    </div>

    <!-- Wishlist share card (收藏夹分享) -->
    <van-popup v-model:show="showShare" round closeable position="bottom" :style="{ width: '88%' }">
      <div class="share-wrap">
        <div class="share-card">
          <div class="sc-header">
            <div class="sc-title">我的收藏夹</div>
            <div class="sc-sub">共 <b>{{ list.length }}</b> 件好物</div>
          </div>
          <div class="sc-body">
            <div v-for="(f, i) in topProducts" :key="f.id" class="sc-item">
              <span class="sc-rank">{{ i + 1 }}</span>
              <van-image width="56" height="56" radius="6" :src="f.product_image" fit="cover" />
              <div class="sc-item-info">
                <div class="sc-item-name van-multi-ellipsis--l2">{{ f.product_name }}</div>
                <div class="sc-item-price">¥{{ fmt(f.price) }}</div>
              </div>
            </div>
            <div v-if="list.length > 3" class="sc-more">还有 {{ list.length - 3 }} 件好物…</div>
          </div>
          <div class="sc-footer">
            <div class="sc-total-label">收藏总价值</div>
            <div class="sc-total-value">¥{{ fmt(totalValue) }}</div>
          </div>
          <div class="sc-brand">天猫 TMALL</div>
        </div>
        <van-button block type="danger" round @click="copyShareText" style="margin-top: 14px">复制分享文本</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.fav-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.share-entry { font-size: 13px; color: #ff0036; }
.fav-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.fc-info { flex: 1; display: flex; flex-direction: column; }
.fc-name { font-size: 14px; line-height: 20px; flex: 1; }
.fc-price { color: #ff0036; font-size: 16px; font-weight: bold; margin: 8px 0; }
.fc-actions { display: flex; gap: 8px; }
/* Wishlist share card (收藏夹分享) */
.share-wrap { padding: 20px; }
.share-card { border-radius: 14px; overflow: hidden; box-shadow: 0 4px 16px rgba(255, 0, 54, 0.12); }
.sc-header { background: linear-gradient(135deg, #ff0036, #ff5e3a); color: #fff; padding: 20px 16px; }
.sc-title { font-size: 19px; font-weight: bold; }
.sc-sub { font-size: 12px; opacity: 0.9; margin-top: 4px; }
.sc-sub b { font-size: 15px; }
.sc-body { background: #fff; padding: 14px 16px; }
.sc-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid #f5f5f5; }
.sc-item:last-of-type { border-bottom: none; }
.sc-rank { width: 20px; height: 20px; flex-shrink: 0; background: #fff0f2; color: #ff0036; font-size: 12px; font-weight: bold; border-radius: 50%; display: flex; align-items: center; justify-content: center; }
.sc-item-info { flex: 1; display: flex; flex-direction: column; justify-content: center; }
.sc-item-name { font-size: 13px; line-height: 18px; color: #333; }
.sc-item-price { color: #ff0036; font-size: 14px; font-weight: bold; margin-top: 2px; }
.sc-more { font-size: 12px; color: #999; text-align: center; padding: 8px 0 2px; }
.sc-footer { background: #fff; padding: 14px 16px; display: flex; align-items: center; justify-content: space-between; border-top: 1px dashed #eee; }
.sc-total-label { font-size: 13px; color: #666; }
.sc-total-value { color: #ff0036; font-size: 22px; font-weight: bold; }
.sc-brand { background: #fff; color: #ff0036; font-size: 12px; font-weight: bold; text-align: center; padding-bottom: 14px; letter-spacing: 1px; }
</style>
