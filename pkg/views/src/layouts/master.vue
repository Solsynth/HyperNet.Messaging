<template>
  <v-navigation-drawer v-model="drawerOpen" color="grey-lighten-5" floating>
    <div class="flex flex-col h-full">
      <v-list class="border-b border-opacity-15 h-[64px]" style="border-bottom-width: thin">
        <v-list-item :subtitle="username" :title="nickname">
          <template #prepend>
            <v-avatar icon="mdi-account-circle" :image="id.userinfo.data?.avatar" />
          </template>
          <template #append>
            <v-menu v-if="id.userinfo.isLoggedIn">
              <template #activator="{ props }">
                <v-btn v-bind="props" icon="mdi-menu-down" size="small" variant="text" />
              </template>

              <v-list density="compact">
                <v-list-item
                  title="Solarpass"
                  prepend-icon="mdi-passport-biometric"
                  target="_blank"
                  :href="passportUrl"
                />
              </v-list>
            </v-menu>

            <v-btn v-else icon="mdi-login-variant" size="small" variant="text" :href="signinUrl" />
          </template>
        </v-list-item>
      </v-list>

      <div class="flex-grow-1">
        <!-- TODO Channel list -->
      </div>
    </div>
  </v-navigation-drawer>

  <v-app-bar height="64" color="primary" scroll-behavior="elevate" flat>
    <div class="max-md:px-5 md:px-12 flex flex-grow-1 items-center">
      <v-app-bar-nav-icon variant="text" @click.stop="toggleDrawer" />

      <router-link :to="{ name: 'landing' }">
        <h2 class="ml-2 text-lg font-500">Solarecho</h2>
      </router-link>

      <v-spacer />

      <v-tooltip v-for="item in navigationMenu" :text="item.name" location="bottom">
        <template #activator="{ props }">
          <v-btn flat exact v-bind="props" :to="{ name: item.to }" size="small" :icon="item.icon" />
        </template>
      </v-tooltip>
    </div>
  </v-app-bar>

  <v-main>
    <router-view />
  </v-main>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import { useUserinfo } from "@/stores/userinfo"
import { useWellKnown } from "@/stores/wellKnown"

const id = useUserinfo()
const navigationMenu = [{ name: "Landing", icon: "mdi-home", to: "landing" }]

const username = computed(() => {
  if (id.userinfo.isLoggedIn) {
    return "@" + id.userinfo.data?.name
  } else {
    return "@vistor"
  }
})
const nickname = computed(() => {
  if (id.userinfo.isLoggedIn) {
    return id.userinfo.data?.nick
  } else {
    return "Anonymous"
  }
})

id.readProfiles()

const meta = useWellKnown()

const signinUrl = computed(() => {
  return meta.wellKnown?.components?.identity + `/auth/sign-in?redirect_uri=${encodeURIComponent(location.href)}`
})
const passportUrl = computed(() => {
  return meta.wellKnown?.components?.identity
})

meta.readWellKnown()

const drawerOpen = ref(true)

function toggleDrawer() {
  drawerOpen.value = !drawerOpen.value
}
</script>

<style scoped>
.editor-fab {
  position: fixed !important;
  bottom: 16px;
  right: 20px;
}
</style>
