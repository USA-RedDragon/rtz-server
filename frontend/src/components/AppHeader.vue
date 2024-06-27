<template>
  <header>
    <h1>
      <RouterLink to="/">Comma Connect</RouterLink>
    </h1>
    <div class="wrapper">
      <nav>
        <RouterLink to="/">Home</RouterLink>
        <RouterLink v-if="!this.userStore.loggedIn" to="/login">Login</RouterLink>
        <a v-else href="#" @click="logout()">Logout</a>
      </nav>
    </div>
  </header>
</template>

<script>
import Menu from 'primevue/menu';
import API from '@/services/API';

import { mapStores } from 'pinia';
import { useUserStore } from '@/store';

export default {
  components: {
    PVMenu: Menu,
  },
  data: function () {
    return {
    };
  },
  mounted() { },
  methods: {
    logout() {
      API.get('/auth/logout')
        .then((_res) => {
          this.userStore.loggedIn = false;
          this.$router.push('/login');
        })
        .catch((err) => {
          console.error(err);
        });
    },
  },
  computed: {
    ...mapStores(useUserStore),
  },
};
</script>

<style scoped>
header {
  text-align: center;
  max-height: 100vh;
}

header a,
.adminNavLink {
  text-decoration: none;
}

header a,
header a:visited,
header a:link,
.adminNavLink:visited,
.adminNavLink:link {
  color: var(--primary-text-color);
}

header nav .router-link-active,
.adminNavLink.router-link-active {
  color: var(--cyan-300) !important;
}

header nav a:active,
.adminNavLink:active {
  color: var(--cyan-500) !important;
}

header h1 a,
header h1 a:hover,
header h1 a:visited {
  color: var(--primary-text-color);
  background-color: inherit;
}

nav {
  width: 100%;
  font-size: 1rem;
  text-align: center;
  margin-top: 0.5rem;
  margin-bottom: 1rem;
}

nav a {
  display: inline-block;
  padding: 0 1rem;
  border-left: 2px solid #444;
}

nav a:first-of-type {
  border: 0;
}
</style>
