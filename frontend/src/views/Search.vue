<script setup>
import { ref, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { showToast } from 'vant'
import { ftsSearch, ftsSuggest } from '../api'

const router = useRouter()
const route = useRoute()
const keyword = ref('')
const results = ref([])
const suggestions = ref([])
const searched = ref(false)
// Whether the search box is focused (and not yet searched): drives the
// "hot search + history" discovery panel.
const focused = ref(false)

const HISTORY_KEY = 'tm_search_history'
const history = ref(JSON.parse(localStorage.getItem(HISTORY_KEY) || '[]'))
const hotList = ['小棕瓶', 'SK-II', 'iPhone', 'Nike', '奶粉']

// Auto-run a search when navigated with a ?q= query param (e.g. from the
// Home hot-tags discovery feed).
if (route.query.q) {
  doSearch(String(route.query.q))
}

// ---- debounced real-time suggestions (300ms) ----
let suggestTimer = null
function scheduleSuggest(val) {
  if (suggestTimer) { clearTimeout(suggestTimer); suggestTimer = null }
  if (!val.trim()) { suggestions.value = []; return }
  suggestTimer = setTimeout(async () => {
    try { suggestions.value = await ftsSuggest(val) } catch (e) { suggestions.value = [] }
  }, 300)
}
function onInput(val) {
  scheduleSuggest(val)
}
function onFocus() {
  focused.value = true
}
function onBlur() {
  // Defer so a tap on a suggestion/history/hot tag can fire before we hide.
  setTimeout(() => { focused.value = false }, 150)
}
function onCancel() {
  focused.value = false
  router.back()
}
onUnmounted(() => { if (suggestTimer) clearTimeout(suggestTimer) })

async function doSearch(kw) {
  if (!kw.trim()) return
  keyword.value = kw
  focused.value = false
  // Save to history (dedupe, newest first, last 10).
  const h = history.value.filter((x) => x !== kw)
  h.unshift(kw)
  history.value = h.slice(0, 10)
  localStorage.setItem(HISTORY_KEY, JSON.stringify(history.value))
  try {
    const res = await ftsSearch(kw)
    results.value = res.data || []
    suggestions.value = []
    searched.value = true
  } catch (e) {
    showToast('搜索失败')
  }
}
function clearHistory() {
  history.value = []
  localStorage.removeItem(HISTORY_KEY)
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="search-page">
    <van-sticky>
      <van-search
        v-model="keyword"
        placeholder="搜索天猫商品(全文搜索)"
        shape="round"
        show-action
        autofocus
        @search="doSearch(keyword)"
        @update:model-value="onInput"
        @focus="onFocus"
        @blur="onBlur"
      >
        <template #action><span @click="onCancel">取消</span></template>
      </van-search>
      <!-- real-time suggestions dropdown -->
      <div v-if="suggestions.length" class="suggest-list">
        <div v-for="s in suggestions" :key="s" class="suggest-item" @click="doSearch(s)">
          <van-icon name="search" size="14" /><span class="suggest-text">{{ s }}</span><van-icon name="arrow" size="12" class="suggest-arrow" />
        </div>
      </div>
    </van-sticky>

    <!-- discovery panel: shown when focused (and no active suggestions dropdown) -->
    <div v-if="focused && !suggestions.length">
      <div v-if="history.length" class="history">
        <div class="h-head">
          <span>搜索历史</span>
          <van-icon name="delete-o" @click="clearHistory" />
        </div>
        <div class="h-tags">
          <van-tag v-for="h in history" :key="h" plain round size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
      <div class="hot">
        <div class="h-head"><span>热门搜索</span></div>
        <div class="h-tags">
          <van-tag v-for="h in hotList" :key="h" round type="primary" plain size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
    </div>

    <!-- search results -->
    <div v-if="searched && !focused">
      <van-empty v-if="!results.length" description="没有找到相关商品" />
      <div class="res-list">
        <div v-for="p in results" :key="p.id" class="res-item" @click="router.push('/product/' + p.id)">
          <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
          <div class="ri-info">
            <div class="ri-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="ri-price">¥{{ fmt(p.price) }}</div>
            <div class="ri-sales">{{ p.sales }}人付款</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-page { min-height: 100vh; }
.suggest-list { background: #fff; border-top: 1px solid #eee; }
.suggest-item { padding: 12px 16px; font-size: 14px; color: #333; display: flex; align-items: center; gap: 8px; border-bottom: 1px solid #f5f5f5; }
.suggest-text { flex: 1; }
.suggest-arrow { color: #ccc; }
.history, .hot { padding: 16px; background: #fff; margin-bottom: 8px; }
.h-head { font-size: 14px; color: #666; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; }
.h-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.res-item { display: flex; gap: 10px; padding: 10px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.ri-info { flex: 1; }
.ri-name { font-size: 13px; line-height: 18px; }
.ri-price { color: #ff0036; font-size: 16px; font-weight: bold; margin-top: 6px; }
.ri-sales { color: #999; font-size: 11px; }
</style>
