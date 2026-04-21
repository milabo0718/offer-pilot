<template>
  <div class="audio-recorder">
    <div class="visualizer" v-if="isRecording">
      <span class="bar"></span><span class="bar"></span
      ><span class="bar"></span>
      <p class="recording-text">正在聆听... {{ recordingTime }}s</p>
    </div>

    <button
      class="record-btn"
      :class="{ 'is-recording': isRecording }"
      @click="toggleRecording"
      :disabled="isUploading"
      title="按住说话"
    >
      <el-icon class="icon"
        ><Microphone v-if="!isRecording" /><VideoPause v-else
      /></el-icon>
      <span>{{ recordButtonText }}</span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, onBeforeUnmount } from "vue";
import { ElMessage } from "element-plus";
import { Microphone, VideoPause } from "@element-plus/icons-vue";
import api from "../utils/api";

const emit = defineEmits(["upload-success", "upload-error"]);

const isRecording = ref(false);
const isUploading = ref(false);
const recordingTime = ref(0);
let mediaRecorder = null;
let audioChunks = [];
let timer = null;

const recordButtonText = computed(() => {
  if (isUploading.value) return "语音解析中...";
  return isRecording.value ? "点击发送" : "点击语音回答";
});

// 开启录音
const startRecording = async () => {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    mediaRecorder = new MediaRecorder(stream);
    audioChunks = [];

    mediaRecorder.ondataavailable = (event) => {
      if (event.data.size > 0) audioChunks.push(event.data);
    };

    mediaRecorder.onstop = async () => {
      // 停止录音后将数据转为 Blob 并上传
      const audioBlob = new Blob(audioChunks, { type: "audio/webm" });
      await uploadAudio(audioBlob);
      // 释放麦克风
      stream.getTracks().forEach((track) => track.stop());
    };

    mediaRecorder.start();
    isRecording.value = true;
    recordingTime.value = 0;

    timer = setInterval(() => {
      recordingTime.value++;
    }, 1000);
  } catch (error) {
    console.error("麦克风权限错误:", error);
    ElMessage.error("无法访问麦克风，请允许浏览器权限");
  }
};

// 停止录音
const stopRecording = () => {
  if (mediaRecorder && mediaRecorder.state !== "inactive") {
    mediaRecorder.stop();
    isRecording.value = false;
    clearInterval(timer);
  }
};

const toggleRecording = () => {
  if (isRecording.value) {
    stopRecording();
  } else {
    startRecording();
  }
};

// 上传音频
const uploadAudio = async (blob) => {
  isUploading.value = true;
  const formData = new FormData();
  formData.append("audio", blob, `interview_${Date.now()}.webm`);

  try {
    // 调用后端的语音上传接口（需确保后端有对应路由处理 multipart/form-data）
    const response = await api.post("/chat/audio-upload", formData);

    // 假设 status_code 1000 为成功
    if (response.data && response.data.status_code === 1000) {
      emit("upload-success", response.data);
    } else {
      throw new Error(response.data?.status_msg || "解析失败");
    }
  } catch (error) {
    console.error("音频上传报错:", error);
    ElMessage.error("语音解析失败，请重试");
    emit("upload-error", error);
  } finally {
    isUploading.value = false;
  }
};

onBeforeUnmount(() => {
  if (timer) clearInterval(timer);
  if (mediaRecorder && mediaRecorder.state !== "inactive") {
    mediaRecorder.stop();
  }
});
</script>

<style scoped>
.audio-recorder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.record-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: 30px;
  border: none;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  transition: all 0.3s ease;
  min-width: 160px;
}

.record-btn .icon {
  font-size: 18px;
}

.record-btn.is-recording {
  background: #ff4757;
  box-shadow: 0 0 20px rgba(255, 71, 87, 0.6);
  animation: pulse 1.5s infinite;
}

.record-btn:disabled {
  background: #a4b0be;
  cursor: not-allowed;
  box-shadow: none;
  animation: none;
}

.visualizer {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 10px;
  color: #ff4757;
  font-size: 13px;
  font-weight: 600;
}

.bar {
  width: 3px;
  height: 12px;
  background-color: #ff4757;
  border-radius: 2px;
  animation: wave 1s infinite ease-in-out;
}
.bar:nth-child(2) {
  animation-delay: 0.2s;
}
.bar:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.03);
  }
  100% {
    transform: scale(1);
  }
}

@keyframes wave {
  0%,
  100% {
    height: 8px;
  }
  50% {
    height: 20px;
  }
}
</style>
