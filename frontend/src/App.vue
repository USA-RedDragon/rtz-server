<template>
  <AppHeader />
  <RouterView />
  <AppFooter />
  <ThemeConfig />
</template>

<script>
import { RouterView } from 'vue-router';
import AppFooter from './components/AppFooter.vue';
import AppHeader from './components/AppHeader.vue';
import ThemeConfig from './components/ThemeConfig.vue';

import { mapStores } from 'pinia';
import { useUserStore, useSettingsStore } from '@/store';

import { getWebsocketURI } from '@/services/util';
import ws from '@/services/ws';

export default {
  name: 'App',
  components: {
    RouterView,
    AppHeader,
    AppFooter,
    ThemeConfig,
  },
  data() {
    return {
      socket: null,
    };
  },
  created() {},
  mounted() {
    this.socket = ws.connect(getWebsocketURI() + '/events', this.onWebsocketMessage);
  },
  unmounted() {
    if (this.socket) {
      this.socket.close();
    }
  },
  methods: {
    onWebsocketMessage(event) {
      const data = JSON.parse(event.data);
      this.$EventBus.emit(data.type, data.data);
    },
  },
  computed: {
    ...mapStores(useUserStore, useSettingsStore),
  },
};
</script>

<style scoped></style>
