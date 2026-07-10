<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { login, register, getCoupons, claimCoupon } from '../api'

const route = useRoute()
const router = useRouter()
const mode = ref('login')
const username = ref('admin')
const password = ref('admin123')
const nickname = ref('')

// ---- New user welcome pack (新人专享礼包) ----
const showWelcome = ref(false)
const welcomeCoupons = ref([])
const welcomeClaiming = ref(false)
const WELCOME_FLAG = 'tm_welcome_claimed'

function couponValue(c) { return c.coupon_type === 'discount' ? (c.value * 10).toFixed(1) + '折' : '¥' + c.value }

async function gotoNext() {
  router.replace(route.query.redirect || '/mine')
}

// On login/register success, show the welcome pack if not yet claimed.
async function tryShowWelcome() {
  if (localStorage.getItem(WELCOME_FLAG)) { await gotoNext(); return }
  try {
    const list = await getCoupons()
    // First 3 still-available coupons.
    welcomeCoupons.value = (list || []).filter((c) => !c.is_claimed).slice(0, 3)
  } catch (e) { /* ignore coupon fetch errors */ }
  if (welcomeCoupons.value.length) {
    showWelcome.value = true
  } else {
    localStorage.setItem(WELCOME_FLAG, '1')
    await gotoNext()
  }
}

// Claim the 3 welcome coupons, mark as claimed, then continue.
async function claimWelcome() {
  if (welcomeClaiming.value) return
  welcomeClaiming.value = true
  let ok = 0
  for (const c of welcomeCoupons.value) {
    try { await claimCoupon(c.id); ok++ } catch (e) { /* already claimed / unavailable */ }
  }
  localStorage.setItem(WELCOME_FLAG, '1')
  showWelcome.value = false
  if (ok > 0) showSuccessToast(`已领取 ${ok} 张优惠券`)
  await gotoNext()
}

// Dismiss the popup: still mark claimed so it won't reappear, then continue.
async function closeWelcome() {
  localStorage.setItem(WELCOME_FLAG, '1')
  showWelcome.value = false
  await gotoNext()
}

async function submit() {
  if (!username.value || !password.value) { showToast('请输入用户名和密码'); return }
  try {
    const res = mode.value === 'login' ? await login(username.value, password.value) : await register({ username: username.value, password: password.value, nickname: nickname.value })
    localStorage.setItem('tm_token', res.token)
    localStorage.setItem('tm_user', JSON.stringify(res.user))
    showSuccessToast(mode.value === 'login' ? '登录成功' : '注册成功')
    await tryShowWelcome()
  } catch (e) { showToast(e.response?.data?.error || '操作失败') }
}
</script>

<template>
  <div class="login-page">
    <van-nav-bar left-arrow @click-left="router.back()" />
    <div class="logo-area"><div class="tm-logo">天猫</div><p class="welcome">{{ mode === 'login' ? '欢迎登录天猫' : '注册天猫账号' }}</p></div>
    <div class="form">
      <van-cell-group inset>
        <van-field v-model="username" label="用户名" placeholder="请输入用户名" clearable />
        <van-field v-model="password" type="password" label="密码" placeholder="请输入密码" clearable />
        <van-field v-if="mode === 'register'" v-model="nickname" label="昵称" placeholder="选填" clearable />
      </van-cell-group>
      <div style="margin: 16px"><van-button type="danger" block round @click="submit">{{ mode === 'login' ? '登 录' : '注 册' }}</van-button></div>
      <div class="switch" @click="mode = mode === 'login' ? 'register' : 'login'">{{ mode === 'login' ? '没有账号？去注册' : '已有账号？去登录' }}</div>
      <div class="hint">演示账号: admin / admin123</div>
    </div>

    <!-- New user welcome pack popup (新人专享礼包) -->
    <van-popup v-model:show="showWelcome" round closeable close-icon-position="top-right" :style="{ width: '86%', maxHeight: '85vh' }" :close-on-click-overlay="false" @close="closeWelcome">
      <div class="welcome-pack">
        <div class="wp-header">
          <div class="wp-confetti">🎉</div>
          <div class="wp-title">新人专享礼包</div>
          <div class="wp-sub">欢迎来到天猫 · 3 重好礼等你拿</div>
        </div>
        <div class="wp-coupons">
          <div v-for="c in welcomeCoupons" :key="c.id" class="wp-coupon">
            <div class="wp-c-left">
              <div class="wp-c-value">{{ couponValue(c) }}</div>
              <div class="wp-c-threshold" v-if="c.threshold > 0">满{{ c.threshold }}可用</div>
              <div class="wp-c-threshold" v-else>无门槛</div>
            </div>
            <div class="wp-c-right"><div class="wp-c-title van-ellipsis">{{ c.title }}</div><div class="wp-c-date">{{ c.end_date }} 到期</div></div>
          </div>
          <div v-if="!welcomeCoupons.length" class="wp-empty">暂无可领取的优惠券</div>
        </div>
        <van-button type="danger" block round class="wp-btn" :loading="welcomeClaiming" @click="claimWelcome">立即领取</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.login-page { min-height: 100vh; background: #fff; }
.logo-area { text-align: center; padding: 30px 0 20px; }
.tm-logo { display: inline-block; background: #ff0036; color: #fff; font-size: 28px; font-weight: bold; width: 80px; height: 80px; line-height: 80px; border-radius: 12px; }
.welcome { margin-top: 14px; font-size: 18px; color: #333; }
.switch { text-align: center; color: #ff0036; font-size: 14px; }
.hint { text-align: center; color: #999; font-size: 12px; margin-top: 16px; }
/* New user welcome pack popup (新人专享礼包) */
.welcome-pack { padding: 36px 20px 24px; }
.wp-header { text-align: center; margin-bottom: 18px; }
.wp-confetti { font-size: 40px; line-height: 1; }
.wp-title { font-size: 22px; font-weight: bold; color: #ff0036; margin-top: 8px; }
.wp-sub { font-size: 13px; color: #999; margin-top: 4px; }
.wp-coupons { display: flex; flex-direction: column; gap: 10px; margin-bottom: 20px; }
.wp-coupon { display: flex; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 6px rgba(255, 0, 54, 0.15); }
.wp-c-left { width: 96px; background: linear-gradient(135deg, #ff0036, #ff5577); color: #fff; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 10px 6px; }
.wp-c-value { font-size: 22px; font-weight: bold; }
.wp-c-threshold { font-size: 11px; opacity: 0.9; margin-top: 2px; }
.wp-c-right { flex: 1; background: #fff; padding: 10px 14px; display: flex; flex-direction: column; justify-content: center; gap: 4px; }
.wp-c-title { font-size: 14px; font-weight: bold; color: #333; }
.wp-c-date { font-size: 11px; color: #999; }
.wp-empty { text-align: center; color: #999; font-size: 13px; padding: 30px 0; }
.wp-btn { font-size: 16px; }
</style>
