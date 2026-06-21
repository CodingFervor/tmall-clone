<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getBrands } from '../api'

const router = useRouter()
const brands = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    brands.value = await getBrands()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="brands-page">
    <van-nav-bar title="品牌馆" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else class="brand-list">
      <div v-for="b in brands" :key="b.id" class="brand-item" @click="router.push('/brand/' + b.id)">
        <div class="bi-logo">{{ b.logo }}</div>
        <div class="bi-info">
          <div class="bi-name">{{ b.name }}</div>
          <div class="bi-desc van-ellipsis">{{ b.description }}</div>
          <div class="bi-fans">{{ (b.followers / 10000).toFixed(0) }}万粉丝关注</div>
        </div>
        <van-button size="small" type="danger" round plain>进店</van-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.brands-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.brand-item { display: flex; align-items: center; gap: 12px; padding: 14px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.bi-logo { width: 56px; height: 56px; line-height: 56px; text-align: center; background: #fff5f6; color: #ff0036; border: 1px solid #ffd6dd; border-radius: 50%; font-weight: bold; flex-shrink: 0; }
.bi-info { flex: 1; min-width: 0; }
.bi-name { font-size: 15px; font-weight: bold; }
.bi-desc { font-size: 12px; color: #999; margin-top: 3px; }
.bi-fans { font-size: 11px; color: #ff0036; margin-top: 3px; }
</style>
