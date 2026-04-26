<template>
  <div class="ai-chat-container">
    <div :class="['session-list', { 'is-collapsed': !isSidebarOpen }]">
      <div class="session-list-header">
        <span>会话列表</span>
        <button class="new-chat-btn" @click="createNewSession">
          ＋ 开始新面试
        </button>
      </div>
      <ul class="session-list-ul">
        <li
          v-for="session in sessions"
          :key="session.id"
          :class="['session-item', { active: currentSessionId === session.id }]"
          @click="switchSession(session.id)"
        >
          {{ session.name || `会话 ${session.id}` }}
        </li>
      </ul>
    </div>

    <div class="chat-section">
      <div class="top-bar">
        <button
          class="sidebar-toggle-btn"
          @click="isSidebarOpen = !isSidebarOpen"
        >
          {{ isSidebarOpen ? "◀ 沉浸模式" : "▶ 展开列表" }}
        </button>

        <button class="back-btn" @click="$router.push('/menu')">← 返回</button>
        <button
          class="sync-btn"
          @click="syncHistory"
          :disabled="!currentSessionId || tempSession"
        >
          同步历史数据
        </button>
        <button
          class="report-btn"
          @click="goToReport"
          :disabled="!currentSessionId || tempSession || loading"
        >
          结束面试并生成报告
        </button>
        <label for="modelType">选择模型：</label>
        <select id="modelType" v-model="selectedModel" class="model-select">
          <option value="1">阿里百炼</option>
        </select>
        <label for="streamingMode" style="margin-left: 20px">
          <input type="checkbox" id="streamingMode" v-model="isStreaming" />
          流式响应
        </label>
        <label for="voiceMode" style="margin-left: 12px">
          <input
            type="checkbox"
            id="voiceMode"
            v-model="voiceMode"
            @change="onVoiceModeChange"
          />
          沉浸式面试 (自动朗读)
        </label>
      </div>

      <div class="chat-messages" ref="messagesRef">
        <div
          v-for="(message, index) in currentMessages"
          :key="index"
          :class="[
            'message',
            message.role === 'user' ? 'user-message' : 'ai-message',
          ]"
        >
          <div class="message-header">
            <b>{{ message.role === "user" ? "你" : "AI" }}:</b>
            <button
              v-if="message.role === 'assistant'"
              class="tts-btn"
              @click="playTTS(message.content)"
            >
              🔊
            </button>
            <span
              v-if="message.meta && message.meta.status === 'streaming'"
              class="streaming-indicator"
            >
              ··</span
            >
          </div>
          <div
            class="message-content"
            v-html="renderMarkdown(message.content)"
          ></div>
        </div>
      </div>

      <div
        v-if="voiceMode"
        class="voice-status-bar"
        :class="{
          'status-thinking': voiceStatus === 'thinking',
          'status-speaking': voiceStatus === 'speaking',
          'status-idle': voiceStatus === 'idle',
        }"
      >
        <span class="voice-status-dot"></span>
        <span class="voice-status-text">{{ voiceStatusText }}</span>
        <button
          v-if="voiceStatus === 'speaking'"
          type="button"
          class="stop-tts-btn"
          @click="stopTTS"
        >
          停止朗读
        </button>
      </div>

      <div class="chat-input">
        <textarea
          v-model="inputMessage"
          placeholder="请作答面试官的问题…（Enter 发送，Shift+Enter 换行）"
          @keydown.enter.exact.prevent="sendMessage"
          :disabled="loading"
          ref="messageInput"
          rows="1"
        ></textarea>

        <div class="input-actions-wrapper">
          <AudioRecorder
            @upload-success="handleAudioSuccess"
            @recording-start="stopTTS"
          />
          <button
            type="button"
            :disabled="!inputMessage.trim() || loading"
            @click="sendMessage"
            class="send-btn"
          >
            {{ loading ? "发送中..." : "发送" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, nextTick, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import api from "../utils/api";
// 新增：引入录音组件
import AudioRecorder from "../components/AudioRecorder.vue";

export default {
  name: "AIChat",
  components: {
    AudioRecorder, // 新增：注册组件
  },
  setup() {
    const router = useRouter();
    // 新增：侧边栏状态控制
    const isSidebarOpen = ref(true);

    const sessions = ref({});
    const currentSessionId = ref(null);
    const tempSession = ref(false);
    const currentMessages = ref([]);
    const inputMessage = ref("");
    const loading = ref(false);
    const messagesRef = ref(null);
    const messageInput = ref(null);
    const selectedModel = ref("1");
    const isStreaming = ref(false);
    const voiceMode = ref(false);
    const voiceStatus = ref("idle");
    const voiceStatusText = computed(() => {
      if (voiceStatus.value === "thinking") return "AI 正在思考...";
      if (voiceStatus.value === "speaking") return "AI 正在回答（语音已播放）";
      return "等待你按下录音开始作答";
    });
    let currentAudio = null;
    /** 与 currentAudio 配对，用于 stop 时 revoke，避免泄漏 */
    let currentTTSUrl = null;

    const stopTTS = () => {
      if (currentTTSUrl) {
        try {
          URL.revokeObjectURL(currentTTSUrl);
        } catch (_) {
          // ignore
        }
        currentTTSUrl = null;
      }
      if (currentAudio) {
        try {
          currentAudio.pause();
          currentAudio.removeAttribute("src");
          currentAudio.load();
        } catch (_) {
          // ignore
        }
        currentAudio = null;
      }
      if (voiceMode.value) voiceStatus.value = "idle";
    };

    const onVoiceModeChange = () => {
      if (voiceMode.value) {
        isStreaming.value = true;
        voiceStatus.value = "idle";
        ElMessage.success("已进入沉浸式面试：AI 回复将自动朗读");
      } else {
        stopTTS();
        voiceStatus.value = "idle";
      }
    };

    // 新增：处理语音上传成功的回调
    const handleAudioSuccess = async (data) => {
      // 兼容后端的字段名 text 或 Information
      const text = data.text || data.Information || data.data?.text;
      if (text) {
        inputMessage.value = text;
        ElMessage.success("语音解析成功");
        await sendMessage(); // 自动触发发送
      } else {
        ElMessage.warning("未能识别出语音内容");
      }
    };

    const getJDProfileText = () => {
      try {
        const raw = localStorage.getItem("jd_profile");
        if (!raw) return "";
        const profile = JSON.parse(raw);
        if (!profile || typeof profile !== "object") return "";
        const parts = [];
        if (profile.jobTitle) parts.push(`岗位: ${profile.jobTitle}`);
        if (Array.isArray(profile.skills) && profile.skills.length) {
          parts.push(`技能: ${profile.skills.join("、")}`);
        }
        if (profile.experience) parts.push(`经验: ${profile.experience}`);
        if (Array.isArray(profile.keywords) && profile.keywords.length) {
          parts.push(`关键词: ${profile.keywords.join("、")}`);
        }
        if (profile.summary) parts.push(`摘要: ${profile.summary}`);
        return parts.join("\n");
      } catch (e) {
        return "";
      }
    };

    const goToReport = () => {
      if (!currentSessionId.value || tempSession.value) {
        ElMessage.warning("请先完成至少一轮对话再生成报告");
        return;
      }

      router.push({
        name: "InterviewReport",
        params: { sessionId: String(currentSessionId.value) },
      });
    };

    const renderMarkdown = (text) => {
      if (!text && text !== "") return "";
      return String(text)
        .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
        .replace(/\*(.*?)\*/g, "<em>$1</em>")
        .replace(/`(.*?)`/g, "<code>$1</code>")
        .replace(/\n/g, "<br>");
    };

    const playTTS = async (text) => {
      const content = String(text || "").trim();
      if (!content) return;
      try {
        stopTTS();
        const response = await api.post(
          "/ai/tts/synthesize",
          { text: content },
          { responseType: "arraybuffer" }
        );
        const blob = new Blob([response.data], { type: "audio/mpeg" });
        const url = URL.createObjectURL(blob);
        currentTTSUrl = url;
        const audio = new Audio(url);
        currentAudio = audio;
        voiceStatus.value = "speaking";
        const endCleanup = (revoke) => {
          if (revoke && currentTTSUrl === url) {
            try {
              URL.revokeObjectURL(url);
            } catch (_) {
              // ignore
            }
            currentTTSUrl = null;
          }
          if (currentAudio === audio) currentAudio = null;
          if (voiceMode.value) voiceStatus.value = "idle";
        };
        audio.onended = () => endCleanup(true);
        audio.onerror = () => endCleanup(true);
        await audio.play();
      } catch (error) {
        console.error("TTS error:", error);
        if (voiceMode.value) voiceStatus.value = "idle";
        ElMessage.error("请求语音接口失败");
      }
    };

    const loadSessions = async () => {
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
              messages: [], // lazy load
            };
          });
          sessions.value = sessionMap;
        }
      } catch (error) {
        console.error("Load sessions error:", error);
      }
    };

    const createNewSession = async () => {
      currentSessionId.value = "temp";
      tempSession.value = true;
      currentMessages.value = [];
      await nextTick();
      if (messageInput.value) messageInput.value.focus();

      // 由 AI 作为面试官主动开场：发送一条 kickoff 用户消息，
      // 触发模型输出自我介绍 + 第一道面试题。
      inputMessage.value = "请开始对我的模拟面试。";
      await sendMessage();
    };

    const switchSession = async (sessionId) => {
      if (!sessionId) return;
      currentSessionId.value = String(sessionId);
      tempSession.value = false;

      // lazy load history if not present
      if (
        !sessions.value[sessionId].messages ||
        sessions.value[sessionId].messages.length === 0
      ) {
        try {
          const response = await api.post("/ai/chat/history", {
            sessionId: currentSessionId.value,
          });
          if (
            response.data &&
            response.data.status_code === 1000 &&
            Array.isArray(response.data.history)
          ) {
            const messages = response.data.history.map((item) => ({
              role: item.is_user ? "user" : "assistant",
              content: item.content,
            }));
            sessions.value[sessionId].messages = messages;
          }
        } catch (err) {
          console.error("Load history error:", err);
        }
      }

      currentMessages.value = [...(sessions.value[sessionId].messages || [])];
      await nextTick();
      scrollToBottom();
    };

    const syncHistory = async () => {
      if (!currentSessionId.value || tempSession.value) {
        ElMessage.warning("请选择已有会话进行同步");
        return;
      }
      try {
        const response = await api.post("/ai/chat/history", {
          sessionId: currentSessionId.value,
        });
        if (
          response.data &&
          response.data.status_code === 1000 &&
          Array.isArray(response.data.history)
        ) {
          const messages = response.data.history.map((item) => ({
            role: item.is_user ? "user" : "assistant",
            content: item.content,
          }));
          sessions.value[currentSessionId.value].messages = messages;
          currentMessages.value = [...messages];
          await nextTick();
          scrollToBottom();
        } else {
          ElMessage.error("无法获取历史数据");
        }
      } catch (err) {
        console.error("Sync history error:", err);
        ElMessage.error("请求历史数据失败");
      }
    };

    const sendMessage = async () => {
      if (!inputMessage.value || !inputMessage.value.trim()) {
        ElMessage.warning("请输入消息内容");
        return;
      }

      const userMessage = {
        role: "user",
        content: inputMessage.value,
      };
      const currentInput = inputMessage.value;
      inputMessage.value = "";

      currentMessages.value.push(userMessage);
      await nextTick();
      scrollToBottom();

      try {
        loading.value = true;
        if (voiceMode.value) voiceStatus.value = "thinking";
        if (isStreaming.value) {
          await handleStreaming(currentInput);
        } else {
          await handleNormal(currentInput);
        }
      } catch (err) {
        console.error("Send message error:", err);
        ElMessage.error("发送失败，请重试");

        if (
          !tempSession.value &&
          currentSessionId.value &&
          sessions.value[currentSessionId.value] &&
          sessions.value[currentSessionId.value].messages
        ) {
          const sessionArr = sessions.value[currentSessionId.value].messages;
          if (sessionArr && sessionArr.length) sessionArr.pop();
        }
        currentMessages.value.pop();
      } finally {
        if (!isStreaming.value) {
          loading.value = false;
        }
        await nextTick();
        scrollToBottom();
      }
    };

    async function handleStreaming(question) {
      const aiMessage = {
        role: "assistant",
        content: "",
        meta: { status: "streaming" }, // mark streaming
      };

      const aiMessageIndex = currentMessages.value.length;
      currentMessages.value.push(aiMessage);

      if (
        !tempSession.value &&
        currentSessionId.value &&
        sessions.value[currentSessionId.value]
      ) {
        if (!sessions.value[currentSessionId.value].messages)
          sessions.value[currentSessionId.value].messages = [];
        sessions.value[currentSessionId.value].messages.push({
          role: "assistant",
          content: "",
        });
      }

      const url = tempSession.value
        ? "/api/v1/ai/chat/send-stream-new-session"
        : "/api/v1/ai/chat/send-stream";

      const headers = {
        "Content-Type": "application/json",
        Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
      };

      const body = tempSession.value
        ? {
            question: question,
            modelType: selectedModel.value,
            jdProfile: getJDProfileText(),
          }
        : {
            question: question,
            modelType: selectedModel.value,
            sessionId: currentSessionId.value,
            jdProfile: getJDProfileText(),
          };

      try {
        // 创建 fetch 连接读取 SSE 流
        const response = await fetch(url, {
          method: "POST",
          headers,
          body: JSON.stringify(body),
        });

        if (!response.ok) {
          loading.value = false;
          throw new Error("Network response was not ok");
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        // 读取流数据
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          buffer += chunk;

          // 按行分割
          const lines = buffer.split("\n");
          buffer = lines.pop() || ""; // 保留未完成的行

          for (const line of lines) {
            const trimmedLine = line.trim();
            if (!trimmedLine) continue;

            // 处理 SSE 格式：data: <content>
            if (trimmedLine.startsWith("data:")) {
              const data = trimmedLine.slice(5).trim();
              console.log("[SSE] Received:", data); // 调试日志

              if (data === "[DONE]") {
                // 流结束
                console.log("[SSE] Stream done");
                loading.value = false;
                currentMessages.value[aiMessageIndex].meta = { status: "done" };
                currentMessages.value = [...currentMessages.value];
              } else if (data.startsWith("{")) {
                // 尝试解析 JSON（如 sessionId）
                try {
                  const parsed = JSON.parse(data);
                  if (parsed.sessionId) {
                    const newSid = String(parsed.sessionId);
                    console.log("[SSE] Session ID:", newSid);
                    if (tempSession.value) {
                      sessions.value[newSid] = {
                        id: newSid,
                        name: "新会话",
                        messages: [...currentMessages.value],
                      };
                      currentSessionId.value = newSid;
                      tempSession.value = false;
                    }
                  }
                } catch (e) {
                  // 不是 JSON，当作普通文本处理
                  currentMessages.value[aiMessageIndex].content += data;
                  console.log(
                    "[SSE] Content updated:",
                    currentMessages.value[aiMessageIndex].content.length
                  );
                }
              } else {
                // 普通文本数据，直接追加
                // 使用数组索引直接更新，强制 Vue 响应式系统检测变化
                currentMessages.value[aiMessageIndex].content += data;
                console.log(
                  "[SSE] Content updated:",
                  currentMessages.value[aiMessageIndex].content.length
                );
              }

              // 每收到一条数据就立即更新 DOM
              // 强制更新整个数组以触发响应式
              currentMessages.value = [...currentMessages.value];

              // 使用 requestAnimationFrame 强制浏览器重排
              await new Promise((resolve) => {
                requestAnimationFrame(() => {
                  scrollToBottom();
                  resolve();
                });
              });
            }
          }
        }

        // 流读取完成后的处理
        loading.value = false;
        currentMessages.value[aiMessageIndex].meta = { status: "done" };
        currentMessages.value = [...currentMessages.value];

        // 同步到 sessions 存储
        if (
          !tempSession.value &&
          currentSessionId.value &&
          sessions.value[currentSessionId.value]
        ) {
          const sessMsgs = sessions.value[currentSessionId.value].messages;
          if (Array.isArray(sessMsgs) && sessMsgs.length) {
            const lastIndex = sessMsgs.length - 1;
            if (
              sessMsgs[lastIndex] &&
              sessMsgs[lastIndex].role === "assistant"
            ) {
              sessMsgs[lastIndex].content =
                currentMessages.value[aiMessageIndex].content;
            }
          }
        }

        if (voiceMode.value) {
          const finalText = currentMessages.value[aiMessageIndex].content;
          playTTS(finalText);
        }
      } catch (err) {
        console.error("Stream error:", err);
        loading.value = false;
        currentMessages.value[aiMessageIndex].meta = { status: "error" };
        currentMessages.value = [...currentMessages.value];
        ElMessage.error("流式传输出错");
      }
    }

    async function handleNormal(question) {
      if (tempSession.value) {
        const response = await api.post("/ai/chat/send-new-session", {
          question: question,
          modelType: selectedModel.value,
          jdProfile: getJDProfileText(),
        });
        if (response.data && response.data.status_code === 1000) {
          const sessionId = String(response.data.sessionId);
          const aiMessage = {
            role: "assistant",
            content: response.data.Information || "",
          };

          sessions.value[sessionId] = {
            id: sessionId,
            name: "新会话",
            messages: [{ role: "user", content: question }, aiMessage],
          };
          currentSessionId.value = sessionId;
          tempSession.value = false;
          currentMessages.value = [...sessions.value[sessionId].messages];
          if (voiceMode.value) playTTS(aiMessage.content);
        } else {
          ElMessage.error(response.data?.status_msg || "发送失败");

          currentMessages.value.pop();
        }
      } else {
        const sessionMsgs = sessions.value[currentSessionId.value].messages;

        sessionMsgs.push({ role: "user", content: question });

        const response = await api.post("/ai/chat/send", {
          question: question,
          modelType: selectedModel.value,
          sessionId: currentSessionId.value,
          jdProfile: getJDProfileText(),
        });
        if (response.data && response.data.status_code === 1000) {
          const aiMessage = {
            role: "assistant",
            content: response.data.Information || "",
          };
          sessionMsgs.push(aiMessage);
          currentMessages.value = [...sessionMsgs];
          if (voiceMode.value) playTTS(aiMessage.content);
        } else {
          ElMessage.error(response.data?.status_msg || "发送失败");
          sessionMsgs.pop(); // rollback
          currentMessages.value.pop();
        }
      }
    }

    const scrollToBottom = () => {
      if (messagesRef.value) {
        try {
          messagesRef.value.scrollTop = messagesRef.value.scrollHeight;
        } catch (e) {
          // ignore
        }
      }
    };

    onMounted(() => {
      loadSessions();
    });

    // expose to template
    return {
      isSidebarOpen, // 新增
      handleAudioSuccess, // 新增
      sessions: computed(() => Object.values(sessions.value)),
      currentSessionId,
      tempSession,
      currentMessages,
      inputMessage,
      loading,
      messagesRef,
      messageInput,
      selectedModel,
      isStreaming,
      voiceMode,
      voiceStatus,
      voiceStatusText,
      onVoiceModeChange,
      renderMarkdown,
      playTTS,
      stopTTS,
      createNewSession,
      switchSession,
      syncHistory,
      goToReport,
      sendMessage,
    };
  },
};
</script>

<style scoped>
/* ==========================================================
   原汁原味的 CSS 代码（无任何删减）
========================================================== */
.ai-chat-container {
  height: 100vh;
  display: flex;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
    "Helvetica Neue", Arial;
  color: #222;
}

.ai-chat-container::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="20" cy="20" r="2" fill="rgba(255,255,255,0.08)"/><circle cx="80" cy="80" r="2" fill="rgba(255,255,255,0.08)"/><circle cx="40" cy="60" r="1" fill="rgba(255,255,255,0.06)"/><circle cx="60" cy="30" r="1.5" fill="rgba(255,255,255,0.06)"/></svg>');
  animation: float 20s ease-in-out infinite;
  opacity: 0.25;
}

@keyframes float {
  0%,
  100% {
    transform: translateY(0px) rotate(0deg);
  }
  50% {
    transform: translateY(-20px) rotate(180deg);
  }
}

.session-list {
  width: 280px;
  height: 100vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(15px);
  border-right: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 2px 0 20px rgba(0, 0, 0, 0.08);
  position: relative;
  z-index: 2;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.3s ease; /* 新增过度动画 */
}

/* 新增：侧边栏折叠后的样式 */
.session-list.is-collapsed {
  width: 0 !important;
  border-right: none;
  opacity: 0;
}

.session-list-header {
  padding: 20px;
  text-align: center;
  font-weight: 600;
  background: linear-gradient(
    135deg,
    rgba(102, 126, 234, 0.06) 0%,
    rgba(103, 194, 58, 0.06) 100%
  );
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}

.new-chat-btn {
  width: 100%;
  padding: 12px 0;
  cursor: pointer;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.28);
  transition: all 0.25s ease;
  position: relative;
  overflow: hidden;
}

.new-chat-btn::before {
  content: "";
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.12),
    transparent
  );
  transition: left 0.5s;
}

.new-chat-btn:hover::before {
  left: 100%;
}

.new-chat-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.36);
}

.session-list-ul {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  overflow-y: auto;
}

.session-item {
  padding: 15px 20px;
  cursor: pointer;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
  transition: all 0.2s ease;
  position: relative;
  color: #2c3e50;
}

.session-item.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-weight: 600;
  box-shadow: inset 0 0 20px rgba(102, 126, 234, 0.2);
}

.session-item:hover {
  background: rgba(102, 126, 234, 0.06);
  transform: translateX(4px);
}

/* chat section */
.chat-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  z-index: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.top-bar {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  color: #2c3e50;
  display: flex;
  align-items: center;
  padding: 12px 24px;
  box-shadow: 0 2px 14px rgba(0, 0, 0, 0.06);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  gap: 12px;
}

/* 新增：侧边栏折叠按钮专用样式 */
.sidebar-toggle-btn {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  border: none;
  color: white;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
  box-shadow: 0 4px 12px rgba(79, 172, 254, 0.2);
}

.sidebar-toggle-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(79, 172, 254, 0.3);
}

.back-btn {
  background: rgba(255, 255, 255, 0.22);
  border: 1px solid rgba(0, 0, 0, 0.06);
  color: #2c3e50;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.back-btn:hover {
  background: rgba(255, 255, 255, 0.32);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.08);
}

.sync-btn,
.report-btn {
  background: linear-gradient(135deg, #67c23a 0%, #409eff 100%);
  color: white;
  padding: 8px 14px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.2);
  transition: all 0.2s ease;
}

.report-btn {
  background: linear-gradient(135deg, #409eff 0%, #67c23a 100%);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.2);
}

.sync-btn:disabled,
.report-btn:disabled {
  background: #ccc;
  box-shadow: none;
  cursor: not-allowed;
}

.model-select {
  margin-left: 6px;
  padding: 6px 10px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  background: white;
  color: #2c3e50;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 30px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  position: relative;
  z-index: 1;
}

/* scrollbar */
.chat-messages::-webkit-scrollbar {
  width: 8px;
}
.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.12);
  border-radius: 8px;
}
.chat-messages::-webkit-scrollbar-track {
  background: transparent;
}

.message {
  max-width: 70%;
  padding: 14px 18px;
  border-radius: 18px;
  line-height: 1.6;
  word-wrap: break-word;
  position: relative;
  animation: messageSlideIn 0.28s ease-out;
  font-size: 15px;
  box-sizing: border-box;
}

@keyframes messageSlideIn {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.user-message {
  align-self: flex-end;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.16);
}

.user-message::after {
  content: "";
  position: absolute;
  bottom: -6px;
  right: 18px;
  width: 0;
  height: 0;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-top: 8px solid #764ba2;
}

.ai-message {
  align-self: flex-start;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(4px);
  color: #2c3e50;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.ai-message::after {
  content: "";
  position: absolute;
  bottom: -6px;
  left: 18px;
  width: 0;
  height: 0;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-top: 8px solid rgba(255, 255, 255, 0.95);
}

.message-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.message-header b {
  font-weight: 600;
}

.tts-btn {
  padding: 6px 10px;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  background: linear-gradient(135deg, #67c23a 0%, #409eff 100%);
  color: white;
  border: none;
  transition: all 0.18s ease;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.18);
}

.tts-btn:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.25);
}

.streaming-indicator {
  color: #999;
  font-weight: 600;
  margin-left: 6px;
}

.voice-status-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px 16px;
  margin: 12px 20px 0;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.85);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
  font-size: 14px;
  font-weight: 600;
  color: #34495e;
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.stop-tts-btn {
  margin-left: 8px;
  padding: 6px 12px;
  font-size: 13px;
  font-weight: 600;
  color: #c0392b;
  background: #fff5f4;
  border: 1px solid rgba(192, 57, 43, 0.35);
  border-radius: 8px;
  cursor: pointer;
}
.stop-tts-btn:hover {
  background: #ffe8e5;
}

.voice-status-bar .voice-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #bdc3c7;
  box-shadow: 0 0 0 rgba(189, 195, 199, 0.6);
}

.voice-status-bar.status-thinking .voice-status-dot {
  background-color: #f39c12;
  animation: pulseDot 1.2s infinite;
}

.voice-status-bar.status-speaking .voice-status-dot {
  background-color: #27ae60;
  animation: pulseDot 1.2s infinite;
}

.voice-status-bar.status-idle .voice-status-dot {
  background-color: #3498db;
}

@keyframes pulseDot {
  0% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(0, 0, 0, 0.2);
  }
  70% {
    transform: scale(1.1);
    box-shadow: 0 0 0 8px rgba(0, 0, 0, 0);
  }
  100% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(0, 0, 0, 0);
  }
}

/* message content */
.message-content {
  white-space: pre-wrap;
  word-break: break-word;
}

/* input area */
.chat-input {
  padding: 24px;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(8px);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 1;
}

.chat-input textarea {
  width: 100%;
  resize: none;
  border: 2px solid rgba(0, 0, 0, 0.06);
  border-radius: 12px;
  /* 调整右侧 padding 为录音按钮和发送按钮留出足够空间 */
  padding: 14px 220px 14px 16px;
  font-size: 15px;
  outline: none;
  background: rgba(255, 255, 255, 0.96);
  color: #2c3e50;
  transition: all 0.18s ease;
  min-height: 20px;
  max-height: 160px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
}

.chat-input textarea:focus {
  border-color: #409eff;
  box-shadow: 0 8px 30px rgba(64, 158, 255, 0.06);
  transform: translateY(-1px);
}

/* 新增：输入操作区包装器 */
.input-actions-wrapper {
  position: absolute;
  right: 36px;
  bottom: 28px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.send-btn {
  /* 移除原有的 absolute 设置，让其在 flex 容器内自然排列 */
  position: static !important;
  padding: 12px 22px;
  border: none;
  border-radius: 50px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.18);
  transition: all 0.18s ease;
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-3px) scale(1.02);
}

.send-btn:disabled {
  background: #ccc;
  box-shadow: none;
  cursor: not-allowed;
}
</style>
