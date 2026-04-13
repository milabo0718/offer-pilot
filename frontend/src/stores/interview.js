import { defineStore } from "pinia";
import { computed, ref } from "vue";
import api from "../utils/api";

export const useInterviewStore = defineStore("interview", () => {
  const sessions = ref({});
  const currentSessionId = ref(null);
  const tempSession = ref(false);
  const currentMessages = ref([]);

  // UI preferences that should survive route changes
  const selectedModel = ref("1");
  const isStreaming = ref(false);

  const sessionList = computed(() => Object.values(sessions.value));

  async function loadSessions() {
    try {
      const response = await api.get("/ai/chat/sessions");
      if (
        response.data &&
        response.data.status_code === 1000 &&
        Array.isArray(response.data.sessions)
      ) {
        const sessionMap = {};
        response.data.sessions.forEach((s) => {
          const sid = String(s.sessionId);
          sessionMap[sid] = {
            id: sid,
            name: s.name || `会话 ${sid}`,
            messages: [], // lazy
          };
        });
        sessions.value = sessionMap;
      }
    } catch (e) {
      // ignore; UI can still work with temp session
    }
  }

  function createTempSession() {
    currentSessionId.value = "temp";
    tempSession.value = true;
    currentMessages.value = [];
  }

  async function switchSession(sessionId) {
    if (!sessionId) return;
    const sid = String(sessionId);
    if (!sessions.value[sid]) return;

    currentSessionId.value = sid;
    tempSession.value = false;

    // lazy load history
    if (
      !Array.isArray(sessions.value[sid].messages) ||
      sessions.value[sid].messages.length === 0
    ) {
      try {
        const response = await api.post("/ai/chat/history", { sessionId: sid });
        if (
          response.data &&
          response.data.status_code === 1000 &&
          Array.isArray(response.data.history)
        ) {
          sessions.value[sid].messages = response.data.history.map((item) => ({
            role: item.is_user ? "user" : "assistant",
            content: item.content,
          }));
        }
      } catch {
        // ignore
      }
    }

    currentMessages.value = [...(sessions.value[sid].messages || [])];
  }

  async function syncHistory() {
    if (!currentSessionId.value || tempSession.value) return;
    const sid = String(currentSessionId.value);
    try {
      const response = await api.post("/ai/chat/history", { sessionId: sid });
      if (
        response.data &&
        response.data.status_code === 1000 &&
        Array.isArray(response.data.history)
      ) {
        const messages = response.data.history.map((item) => ({
          role: item.is_user ? "user" : "assistant",
          content: item.content,
        }));
        sessions.value[sid].messages = messages;
        currentMessages.value = [...messages];
      }
    } catch {
      // ignore
    }
  }

  function upsertSessionFromTemp(sessionId, name) {
    const sid = String(sessionId);
    sessions.value[sid] = {
      id: sid,
      name: name || "新会话",
      messages: [...currentMessages.value],
    };
    currentSessionId.value = sid;
    tempSession.value = false;
  }

  async function renameSession(sessionId, title) {
    const sid = String(sessionId || "");
    const nextTitle = String(title || "").trim();
    if (!sid || !nextTitle) throw new Error("会话名称不能为空");

    const response = await api.post("/ai/chat/rename", {
      sessionId: sid,
      title: nextTitle,
    });

    if (!response.data || response.data.status_code !== 1000) {
      throw new Error(response.data?.status_msg || "重命名失败");
    }

    if (sessions.value[sid]) {
      sessions.value[sid].name = nextTitle;
      sessions.value = { ...sessions.value };
    }
  }

  return {
    // state
    sessions,
    sessionList,
    currentSessionId,
    tempSession,
    currentMessages,
    selectedModel,
    isStreaming,

    // actions
    loadSessions,
    createTempSession,
    switchSession,
    syncHistory,
    upsertSessionFromTemp,
    renameSession,
  };
});
