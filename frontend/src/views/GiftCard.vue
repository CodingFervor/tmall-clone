<script setup>
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { generateGiftCard, redeemGiftCard, getGiftCards } from '../api'

const router = useRouter()
const code = ref('')
const cards = ref([])
const total = ref('0.00')
const loading = ref(true)
const busy = ref(false)

// Preset demo denominations for the "生成演示卡" quick actions.
const presets = [50, 100, 200, 500]

async function load() {
  if (!localStorage.getItem('tm_token')) { router.replace({ name: 'login', query: { redirect: '/gift-card' } }); return }
  loading.value = true
  try {
    const res = await getGiftCards()
    cards.value = res.data || []
    total.value = res.total || '0.00'
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)
onActivated(load)

async function doRedeem() {
  if (!code.value.trim()) { showToast('请输入礼品卡卡号'); return }
  if (busy.value) return
  busy.value = true
  try {
    const res = await redeemGiftCard(code.value.trim())
    cards.value = res.cards || []
    code.value = ''
    showSuccessToast(res.message || '兑换成功')
  } catch (e) {
    showToast(e.response?.data?.error || '兑换失败')
  } finally {
    busy.value = false
  }
}

async function doGenerate(amount) {
  if (busy.value) return
  busy.value = true
  try {
    const res = await generateGiftCard(amount)
    code.value = res.data.code
    showSuccessToast('演示卡已生成，点击兑换即可绑定')
  } catch (e) {
    showToast(e.response?.data?.error || '生成失败')
  } finally {
    busy.value = false
  }
}

function fmt(n) { return Number(n || 0).toFixed(2) }
function copyCode(c) {
  const ta = document.createElement('textarea')
  ta.value = c
  document.body.appendChild(ta)
  ta.select()
  try { document.execCommand('copy'); showSuccessToast('卡号已复制') } catch { showToast('复制失败') }
  document.body.removeChild(ta)
}
</script>

<template>
  <div class="gc-page">
    <van-nav-bar title="天猫礼品卡" left-arrow @click-left="router.back()" fixed placeholder />

    <!-- balance banner -->
    <div class="gc-banner">
      <div class="gc-banner-label">礼品卡余额 (元)</div>
      <div class="gc-banner-value">{{ total }}</div>
    </div>

    <!-- redeem input -->
    <div class="gc-card">
      <div class="gc-section-title">兑换礼品卡</div>
      <van-field v-model="code" center clearable placeholder="请输入礼品卡卡号" class="gc-input">
        <template #button>
          <van-button type="danger" size="small" round :loading="busy" @click="doRedeem">兑换</van-button>
        </template>
      </van-field>

      <!-- demo generate -->
      <div class="gc-demo">
        <div class="gc-demo-tip">没有卡号？点击生成演示礼品卡：</div>
        <div class="gc-presets">
          <van-button v-for="p in presets" :key="p" plain type="danger" size="small" round @click="doGenerate(p)">¥{{ p }}</van-button>
        </div>
      </div>
    </div>

    <!-- redeemed list -->
    <div class="gc-card">
      <div class="gc-section-title">我的礼品卡</div>
      <div v-if="loading" class="gc-loading"><van-loading /></div>
      <van-empty v-else-if="!cards.length" description="还没有礼品卡，去生成一张试试吧" />
      <div v-else class="gc-list">
        <div v-for="gc in cards" :key="gc.id" class="gc-item" :class="{ used: gc.status === 'used' }">
          <div class="gi-left">
            <div class="gi-amount">¥{{ fmt(gc.amount) }}</div>
            <div class="gi-status">{{ gc.status === 'used' ? '已使用' : '未使用' }}</div>
          </div>
          <div class="gi-right">
            <div class="gi-code" @click="copyCode(gc.code)">{{ gc.code }} <van-icon name="description" size="12" /></div>
            <div class="gi-date">{{ gc.created_at }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gc-page { min-height: 100vh; }
.gc-banner {
  margin: 12px;
  border-radius: 12px;
  padding: 24px 20px;
  color: #fff;
  background: linear-gradient(135deg, #ff0036, #ff5577);
  box-shadow: 0 4px 12px rgba(255, 0, 54, 0.2);
}
.gc-banner-label { font-size: 13px; opacity: 0.9; }
.gc-banner-value { font-size: 32px; font-weight: bold; margin-top: 6px; font-variant-numeric: tabular-nums; }

.gc-card { margin: 12px; background: #fff; border-radius: 12px; padding: 16px; }
.gc-section-title { font-size: 15px; font-weight: bold; color: #333; margin-bottom: 12px; }
.gc-input { border: 1px solid #eee; border-radius: 8px; }

.gc-demo { margin-top: 16px; padding-top: 14px; border-top: 1px dashed #eee; }
.gc-demo-tip { font-size: 12px; color: #999; margin-bottom: 10px; }
.gc-presets { display: flex; flex-wrap: wrap; gap: 10px; }

.gc-loading { text-align: center; padding: 30px; }
.gc-list { display: flex; flex-direction: column; gap: 10px; }
.gc-item {
  display: flex;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #f0f0f0;
}
.gc-item.used { opacity: 0.65; }
.gi-left {
  width: 100px;
  background: linear-gradient(135deg, #ff0036, #ff5577);
  color: #fff;
  padding: 16px 10px;
  text-align: center;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.gi-amount { font-size: 20px; font-weight: bold; }
.gi-status { font-size: 11px; opacity: 0.9; margin-top: 4px; }
.gi-right { flex: 1; padding: 14px 16px; display: flex; flex-direction: column; justify-content: center; gap: 6px; }
.gi-code { font-size: 15px; font-weight: bold; color: #333; letter-spacing: 1px; cursor: pointer; }
.gi-date { font-size: 12px; color: #999; }
</style>
