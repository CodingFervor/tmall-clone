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

// ---- 第三方登录 & 忘记密码 (social login + forgot password, visual only) ----
function socialLogin(provider) {
  const names = { wechat: '微信', alipay: '支付宝', qq: 'QQ' }
  showToast(`${names[provider] || '第三方'}登录暂未开通`)
}
function forgotPassword() {
  showToast('请联系客服重置密码')
}
</script>

<template>
  <div class="login-page">
    <!-- Gradient brand banner with wordmark -->
    <div class="brand-banner">
      <span class="back-btn" @click="router.back()">‹</span>
      <div class="bb-wordmark">天猫</div>
      <p class="bb-welcome">{{ mode === 'login' ? '欢迎回来👋' : '加入天猫👋' }}</p>
      <p class="bb-sub">{{ mode === 'login' ? '理想生活上天猫' : '开启理想生活' }}</p>
      <div class="bb-bubble bb-bubble-1"></div>
      <div class="bb-bubble bb-bubble-2"></div>
    </div>

    <!-- Slide-up form card -->
    <div class="form-card slide-up">
      <div class="form-title">{{ mode === 'login' ? '账号登录' : '注册账号' }}</div>
      <van-cell-group inset>
        <van-field v-model="username" label="用户名" placeholder="请输入用户名" clearable />
        <van-field v-model="password" type="password" label="密码" placeholder="请输入密码" clearable />
        <van-field v-if="mode === 'register'" v-model="nickname" label="昵称" placeholder="选填" clearable />
      </van-cell-group>
      <div class="forgot-row" v-if="mode === 'login'"><span class="forgot-link" @click="forgotPassword">忘记密码</span></div>
      <div style="margin: 16px"><van-button type="danger" block round @click="submit">{{ mode === 'login' ? '登 录' : '注 册' }}</van-button></div>
      <div class="switch" @click="mode = mode === 'login' ? 'register' : 'login'">{{ mode === 'login' ? '没有账号？去注册' : '已有账号？去登录' }}</div>

      <!-- Social login (visual only, toast on click) -->
      <div class="social">
        <div class="social-divider"><span>其他登录方式</span></div>
        <div class="social-btns">
          <div class="social-btn sb-wechat" @click="socialLogin('wechat')">
            <span class="sb-icon">💬</span><span class="sb-label">微信</span>
          </div>
          <div class="social-btn sb-alipay" @click="socialLogin('alipay')">
            <span class="sb-icon">💰</span><span class="sb-label">支付宝</span>
          </div>
          <div class="social-btn sb-qq" @click="socialLogin('qq')">
            <span class="sb-icon">🐧</span><span class="sb-label">QQ</span>
          </div>
        </div>
      </div>
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
/* Gradient brand banner with wordmark */
.brand-banner {
  position: relative;
  padding: 50px 24px 60px;
  background: linear-gradient(135deg, #ff0036 0%, #ff4d6d 55%, #ff7a18 100%);
  color: #fff;
  overflow: hidden;
  border-bottom-left-radius: 28px;
  border-bottom-right-radius: 28px;
}
.back-btn {
  position: absolute; top: 16px; left: 14px;
  color: #fff; font-size: 30px; line-height: 1; cursor: pointer;
  width: 32px; height: 32px; display: flex; align-items: center; justify-content: center;
}
.bb-wordmark {
  font-size: 40px; font-weight: 900; letter-spacing: 4px;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.18);
}
.bb-welcome { margin-top: 14px; font-size: 22px; font-weight: bold; }
.bb-sub { margin-top: 6px; font-size: 14px; opacity: 0.9; }
/* Decorative translucent bubbles */
.bb-bubble { position: absolute; border-radius: 50%; background: rgba(255, 255, 255, 0.16); }
.bb-bubble-1 { width: 120px; height: 120px; top: -30px; right: -20px; }
.bb-bubble-2 { width: 70px; height: 70px; bottom: -20px; right: 60px; background: rgba(255, 255, 255, 0.12); }

/* Slide-up form card */
.form-card {
  position: relative;
  margin: -32px 12px 0;
  background: #fff;
  border-radius: 18px;
  padding: 22px 4px 16px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.08);
  z-index: 2;
}
.slide-up { animation: slideUp 0.5s cubic-bezier(0.22, 1, 0.36, 1) both; }
@keyframes slideUp {
  from { opacity: 0; transform: translateY(36px); }
  to { opacity: 1; transform: translateY(0); }
}
.form-title { font-size: 18px; font-weight: bold; color: #333; padding: 0 20px 14px; }
.forgot-row { display: flex; justify-content: flex-end; padding: 6px 28px 0; }
.forgot-link { font-size: 13px; color: #999; cursor: pointer; }
.forgot-link:active { color: #ff0036; }
.switch { text-align: center; color: #ff0036; font-size: 14px; }
.hint { text-align: center; color: #999; font-size: 12px; margin-top: 18px; }

/* Social login buttons */
.social { margin-top: 24px; }
.social-divider {
  display: flex; align-items: center; justify-content: center;
  color: #bbb; font-size: 12px; margin-bottom: 18px;
}
.social-divider::before, .social-divider::after {
  content: ''; flex: 0 0 60px; height: 1px; background: #eee; margin: 0 12px;
}
.social-btns { display: flex; justify-content: center; gap: 34px; }
.social-btn {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  cursor: pointer; user-select: none; -webkit-tap-highlight-color: transparent;
  transition: transform 0.15s ease;
}
.social-btn:active { transform: scale(0.9); }
.sb-icon {
  width: 48px; height: 48px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 24px; color: #fff;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.12);
}
.sb-wechat .sb-icon { background: linear-gradient(135deg, #07c160, #2ecc71); }
.sb-alipay .sb-icon { background: linear-gradient(135deg, #1677ff, #4d9bff); }
.sb-qq .sb-icon { background: linear-gradient(135deg, #12b7f5, #3ecbff); }
.sb-label { font-size: 12px; color: #666; }
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
