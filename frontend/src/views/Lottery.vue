<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getPrizes, spinLottery } from '../api'

const router = useRouter()
const prizes = ref([])
const points = ref(0)
const loading = ref(true)
const spinning = ref(false)
// Current wheel rotation in degrees (accumulates across spins).
const rotation = ref(0)
// Recent draw results, newest first.
const records = ref([])

// Palette for wheel segments — repeats if there are more prizes than colors.
const palette = ['#ff0036', '#ff8a00', '#ffc107', '#07c160', '#1989fa', '#7232dd', '#ff5577', '#00bcd4']

// The number of degrees each segment occupies.
const segAngle = computed(() => (prizes.value.length ? 360 / prizes.value.length : 0))

// Build the conic-gradient background for the wheel. Segment i spans
// [i*segAngle, (i+1)*segAngle); we paint alternating palette colors.
const wheelBg = computed(() => {
  if (!prizes.value.length) return '#fff'
  const stops = []
  for (let i = 0; i < prizes.value.length; i++) {
    const from = (i * segAngle.value).toFixed(2)
    const to = ((i + 1) * segAngle.value).toFixed(2)
    stops.push(`${palette[i % palette.length]} ${from}deg ${to}deg`)
  }
  return `conic-gradient(${stops.join(',')})`
})

// Position each prize label at the center of its segment, radiating outward.
// The wheel's 0deg points right (3 o'clock); we rotate labels so they sit
// upright along the radius. The pointer sits at the top (12 o'clock / -90deg).
function labelStyle(i) {
  const center = i * segAngle.value + segAngle.value / 2
  return {
    transform: `rotate(${center}deg) translate(-50%, -50%)`,
    // Push the label outward along the rotated x-axis.
    '--push': `${Math.min(58, 30 + segAngle.value * 0.4)}%`,
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getPrizes()
    prizes.value = res.data || []
    points.value = res.points || 0
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const spinCost = computed(() => (prizes.value.length ? prizes.value[0].points_cost || 50 : 50))

async function doSpin() {
  if (!localStorage.getItem('tm_token')) {
    showToast('请先登录')
    router.push({ name: 'login', query: { redirect: '/lottery' } })
    return
  }
  if (spinning.value) return
  if (points.value < spinCost.value) {
    showToast('积分不足，去签到赚积分')
    return
  }
  spinning.value = true
  try {
    const res = await spinLottery()
    const winner = res.data
    points.value = res.points
    // Find the winning segment index to aim the pointer at its center.
    const idx = Math.max(0, prizes.value.findIndex((p) => p.id === winner.id))
    const seg = 360 / prizes.value.length
    const segCenter = idx * seg + seg / 2
    // The pointer is at the top (270deg in our 0deg=right conic frame, i.e. -90).
    // To bring segCenter to the top we must rotate the wheel by (270 - segCenter),
    // then add full turns so it always spins forward.
    const baseTurns = 6 * 360
    const current = ((rotation.value % 360) + 360) % 360
    const targetOffset = (270 - segCenter - current + 360 * 4) % 360
    rotation.value += baseTurns + targetOffset
    // Wait for the CSS transition to finish before announcing the result.
    setTimeout(() => {
      records.value.unshift({
        name: winner.name,
        prize: winner.prize,
        icon: winner.icon,
        time: new Date().toLocaleString('zh-CN', { hour12: false }),
      })
      if (records.value.length > 10) records.value.pop()
      if (winner.prize === '下次再来' || winner.name === '谢谢参与') {
        showToast('很遗憾，未中奖')
      } else {
        showSuccessToast(`恭喜中奖：${winner.prize}`)
      }
      spinning.value = false
    }, 4200)
  } catch (e) {
    spinning.value = false
    showToast(e.response?.data?.error || '抽奖失败')
  }
}
</script>

<template>
  <div class="lottery-page">
    <van-nav-bar title="积分大转盘" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="hero">
      <div class="hero-coin">{{ points }}</div>
      <div class="hero-label">我的可用积分</div>
      <van-button size="mini" plain round color="#fff" @click="router.push('/checkin')">去签到赚积分</van-button>
    </div>

    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else>
      <div class="wheel-wrap">
        <!-- Fixed pointer at the top of the wheel -->
        <div class="pointer">▼</div>
        <div class="wheel-outer">
          <div
            class="wheel"
            :style="{ background: wheelBg, transform: `rotate(${rotation}deg)`, transition: spinning ? 'transform 4s cubic-bezier(0.17, 0.67, 0.21, 0.99)' : 'none' }"
          >
            <div
              v-for="(p, i) in prizes"
              :key="p.id"
              class="seg-label"
              :style="labelStyle(i)"
            >
              <span class="seg-icon">{{ p.icon }}</span>
              <span class="seg-name">{{ p.name }}</span>
            </div>
          </div>
          <button class="hub" :disabled="spinning" @click="doSpin">
            {{ spinning ? '转动中' : '抽奖' }}
          </button>
        </div>
        <div class="cost-tip">每次抽奖消耗 {{ spinCost }} 积分</div>
      </div>

      <div class="rules">
        <div class="rules-title">活动说明</div>
        <ul>
          <li>每日可多次抽奖，积分充足即可参与。</li>
          <li>中奖结果以服务端为准，奖品将发放至账户。</li>
          <li>88VIP月卡、美妆小样等高价值奖品概率较低，祝您好运。</li>
          <li>抽奖消耗的积分将从可用积分中实时扣除。</li>
        </ul>
      </div>

      <div v-if="records.length" class="records">
        <div class="records-title">中奖记录</div>
        <div v-for="(r, i) in records" :key="i" class="record-item">
          <span class="r-icon">{{ r.icon }}</span>
          <span class="r-name">{{ r.name }}</span>
          <span class="r-time">{{ r.time }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.lottery-page { min-height: 100vh; padding-bottom: 30px; background: #fff5f6; }
.loading { text-align: center; padding: 80px; }
.hero { background: linear-gradient(135deg, #ff0036, #ff5577); padding: 22px 20px; text-align: center; color: #fff; }
.hero-coin { font-size: 36px; font-weight: bold; line-height: 1.1; }
.hero-label { font-size: 13px; opacity: 0.9; margin: 4px 0 10px; }

.wheel-wrap { display: flex; flex-direction: column; align-items: center; padding: 30px 0 10px; }
.pointer { color: #ff0036; font-size: 30px; line-height: 1; z-index: 3; margin-bottom: -8px; text-shadow: 0 2px 4px rgba(0,0,0,0.2); }
.wheel-outer { position: relative; width: 300px; height: 300px; border-radius: 50%; padding: 8px; background: #ffd700; box-shadow: 0 8px 24px rgba(255,0,54,0.25); }
.wheel { position: relative; width: 100%; height: 100%; border-radius: 50%; overflow: hidden; }
.seg-label { position: absolute; left: 50%; top: 50%; width: 0; height: 0; transform-origin: 0 0; display: flex; flex-direction: column; align-items: center; }
.seg-label .seg-icon { font-size: 20px; transform: translateX(var(--push)) rotate(90deg); }
.seg-label .seg-name { font-size: 11px; color: #fff; font-weight: bold; text-shadow: 0 1px 2px rgba(0,0,0,0.3); white-space: nowrap; transform: translateX(var(--push)) rotate(90deg); margin-top: 2px; }
.hub { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 76px; height: 76px; border-radius: 50%; border: 4px solid #fff; background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; font-size: 18px; font-weight: bold; cursor: pointer; box-shadow: 0 4px 12px rgba(255,0,54,0.4); z-index: 2; }
.hub:disabled { opacity: 0.7; cursor: not-allowed; }
.cost-tip { margin-top: 14px; font-size: 12px; color: #999; }

.rules { margin: 16px 14px 0; background: #fff; border-radius: 12px; padding: 14px 16px; }
.rules-title { font-size: 15px; font-weight: bold; margin-bottom: 8px; color: #333; }
.rules ul { margin: 0; padding-left: 18px; }
.rules li { font-size: 12px; color: #888; line-height: 20px; }

.records { margin: 16px 14px 0; background: #fff; border-radius: 12px; padding: 12px 16px; }
.records-title { font-size: 15px; font-weight: bold; margin-bottom: 6px; color: #333; }
.record-item { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-top: 1px solid #f5f5f5; font-size: 13px; }
.record-item:first-of-type { border-top: none; }
.r-icon { font-size: 18px; }
.r-name { color: #333; flex: 1; }
.r-time { color: #bbb; font-size: 11px; }
</style>
