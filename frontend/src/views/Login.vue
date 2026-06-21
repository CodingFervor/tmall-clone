<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { login, register } from '../api'

const route = useRoute()
const router = useRouter()
const mode = ref('login')
const username = ref('admin')
const password = ref('admin123')
const nickname = ref('')

async function submit() {
  if (!username.value || !password.value) { showToast('请输入用户名和密码'); return }
  try {
    const res = mode.value === 'login' ? await login(username.value, password.value) : await register({ username: username.value, password: password.value, nickname: nickname.value })
    localStorage.setItem('tm_token', res.token)
    localStorage.setItem('tm_user', JSON.stringify(res.user))
    showSuccessToast(mode.value === 'login' ? '登录成功' : '注册成功')
    router.replace(route.query.redirect || '/mine')
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
  </div>
</template>

<style scoped>
.login-page { min-height: 100vh; background: #fff; }
.logo-area { text-align: center; padding: 30px 0 20px; }
.tm-logo { display: inline-block; background: #ff0036; color: #fff; font-size: 28px; font-weight: bold; width: 80px; height: 80px; line-height: 80px; border-radius: 12px; }
.welcome { margin-top: 14px; font-size: 18px; color: #333; }
.switch { text-align: center; color: #ff0036; font-size: 14px; }
.hint { text-align: center; color: #999; font-size: 12px; margin-top: 16px; }
</style>
