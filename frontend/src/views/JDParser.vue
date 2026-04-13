<template>
  <div class="jd-parser-container">
    <div class="parser-card">
      <div class="card-header">
        <h2>JD 解析</h2>
        <p>输入岗位描述后，系统会提取技能标签并自动进入 AI 面试</p>
      </div>

      <div class="form-section">
        <label class="label">岗位描述（文本优先）</label>
        <textarea
          v-model="jdText"
          class="jd-textarea"
          placeholder="请粘贴岗位 JD，例如：负责 Go 后端开发，熟悉 Redis、MySQL、K8s..."
        ></textarea>
      </div>

      <div class="form-section">
        <label class="label">或上传 JD 截图（可选）</label>
        <input
          type="file"
          accept="image/*"
          @change="handleFileChange"
          class="file-input"
        />
        <p v-if="fileName" class="file-name">已选择：{{ fileName }}</p>
      </div>

      <div class="actions">
        <button class="back-btn" @click="$router.push('/menu')">返回</button>
        <button class="parse-btn" :disabled="loading" @click="parseAndStart">
          {{ loading ? "解析中..." : "解析并开始面试" }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import api from "../utils/api";
import { useJDProfileStore } from "../stores/jdProfile";

export default {
  name: "JDParser",
  setup() {
    const router = useRouter();
    const jdStore = useJDProfileStore();
    const loading = ref(false);
    const jdText = ref("");
    const fileName = ref("");

    const handleFileChange = (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      fileName.value = file.name;
    };

    const parseAndStart = async () => {
      if (!jdText.value.trim()) {
        ElMessage.warning("请先输入 JD 文本");
        return;
      }

      loading.value = true;
      try {
        const response = await api.post("/ai/jd/parse", {
          jdText: jdText.value.trim(),
          modelType: "1",
        });
        if (!response.data || response.data.status_code !== 1000) {
          throw new Error(response.data?.status_msg || "JD 解析失败");
        }

        jdStore.setProfile(response.data.data || {});
        ElMessage.success("JD 解析完成，正在进入 AI 面试");
        router.push("/ai-chat");
      } catch (error) {
        console.error("JD parse failed:", error);
        ElMessage.error("JD 解析失败，请重试");
      } finally {
        loading.value = false;
      }
    };

    return {
      loading,
      jdText,
      fileName,
      handleFileChange,
      parseAndStart,
    };
  },
};
</script>

<style scoped>
.jd-parser-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 24px;
}

.parser-card {
  width: 860px;
  max-width: 100%;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 20px;
  padding: 28px;
  box-shadow: 0 20px 48px rgba(0, 0, 0, 0.16);
}

.card-header h2 {
  margin: 0;
  color: #2c3e50;
}

.card-header p {
  margin-top: 8px;
  color: #7f8c8d;
}

.form-section {
  margin-top: 18px;
}

.label {
  display: block;
  margin-bottom: 8px;
  font-weight: 600;
  color: #2c3e50;
}

.jd-textarea {
  width: 100%;
  min-height: 180px;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid #dcdfe6;
  outline: none;
  resize: vertical;
  font-size: 14px;
}

.jd-textarea:focus {
  border-color: #409eff;
}

.file-input {
  width: 100%;
}

.file-name {
  margin-top: 8px;
  color: #606266;
}

.actions {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.back-btn,
.parse-btn {
  border: none;
  border-radius: 10px;
  padding: 10px 16px;
  cursor: pointer;
  font-weight: 600;
}

.back-btn {
  background: #ecf5ff;
  color: #409eff;
}

.parse-btn {
  background: linear-gradient(135deg, #409eff 0%, #67c23a 100%);
  color: #fff;
}

.parse-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}
</style>
