import { createRouter, createWebHistory } from "vue-router"
import MasterLayout from "@/layouts/master.vue"

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      component: MasterLayout,
      children: [
        {
          path: "/",
          name: "landing",
          component: () => import("@/views/landing.vue")
        },
      ]
    }
  ]
})

export default router
