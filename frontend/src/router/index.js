import { createRouter, createWebHistory } from "vue-router";
import Login from "../views/Login.vue";
import Register from "../views/Register.vue";
import Menu from "../views/Menu.vue";
import AIChat from "../views/AIChat.vue";
import JDParser from "../views/JDParser.vue";
import InterviewReport from "../views/InterviewReport.vue";

const routes = [
  {
    path: "/",
    redirect: "/login",
  },
  {
    path: "/login",
    name: "Login",
    component: Login,
  },
  {
    path: "/register",
    name: "Register",
    component: Register,
  },
  {
    path: "/menu",
    name: "Menu",
    component: Menu,
    meta: { requiresAuth: true },
  },
  {
    path: "/ai-chat",
    name: "AIChat",
    component: AIChat,
    meta: { requiresAuth: true },
  },
  {
    path: "/jd-parser",
    name: "JDParser",
    component: JDParser,
    meta: { requiresAuth: true },
  },
  {
    path: "/interview-report/:sessionId",
    name: "InterviewReport",
    component: InterviewReport,
    props: true,
    meta: { requiresAuth: true },
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem("token");
  if (to.matched.some((record) => record.meta.requiresAuth) && !token) {
    next("/login");
  } else {
    next();
  }
});

export default router;
