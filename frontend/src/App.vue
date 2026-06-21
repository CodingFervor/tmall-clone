<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const showTabbar = computed(() => route.meta.tab !== undefined)
const active = computed(() => route.meta.tab ?? 0)
const tabs = [
  { name: 'home', icon: 'wap-home-o', label: '首页' },
  { name: 'brands', icon: 'medal-o', label: '品牌' },
  { name: 'category', icon: 'apps-o', label: '分类' },
  { name: 'cart', icon: 'shopping-cart-o', label: '购物车' },
  { name: 'mine', icon: 'contact', label: '我的' },
]
</script>

<template>
  <div class="app-wrap">
    <router-view v-slot="{ Component }">
      <keep-alive include="Home,Brands">
        <component :is="Component" />
      </keep-alive>
    </router-view>
    <van-tabbar v-if="showTabbar" v-model="active" route active-color="#ff0036" inactive-color="#7d7e80">
      <van-tabbar-item v-for="t in tabs" :key="t.name" :to="{ name: t.name }" :icon="t.icon">{{ t.label }}</van-tabbar-item>
    </van-tabbar>
  </div>
</template>
