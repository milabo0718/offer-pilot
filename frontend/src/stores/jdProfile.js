import { defineStore } from "pinia";
import { computed, ref } from "vue";

const STORAGE_KEY = "jd_profile";

export const useJDProfileStore = defineStore("jdProfile", () => {
  const profile = ref(loadFromStorage());

  function loadFromStorage() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return null;
      const parsed = JSON.parse(raw);
      if (!parsed || typeof parsed !== "object") return null;
      return parsed;
    } catch {
      return null;
    }
  }

  function setProfile(nextProfile) {
    profile.value =
      nextProfile && typeof nextProfile === "object" ? nextProfile : null;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(profile.value || {}));
    } catch {
      // ignore
    }
  }

  const profileText = computed(() => {
    const p = profile.value;
    if (!p || typeof p !== "object") return "";
    const parts = [];
    if (p.jobTitle) parts.push(`岗位: ${p.jobTitle}`);
    if (Array.isArray(p.skills) && p.skills.length) {
      parts.push(`技能: ${p.skills.join("、")}`);
    }
    if (p.experience) parts.push(`经验: ${p.experience}`);
    if (Array.isArray(p.keywords) && p.keywords.length) {
      parts.push(`关键词: ${p.keywords.join("、")}`);
    }
    if (p.summary) parts.push(`摘要: ${p.summary}`);
    return parts.join("\n");
  });

  const interviewerPromptText = computed(() => {
    const jdText = profileText.value;
    return [
      "你现在是一名资深面试官。请严格按照面试官的口吻与流程进行提问与追问。",
      "请围绕以下 6 个能力维度进行评估并在对话中收集证据：技术基础、工程实践、问题解决、沟通表达、学习能力、岗位匹配度。",
      "若岗位画像为空，请先用 1-2 个问题确认候选人的目标岗位/方向与年限，再继续。",
      "面试过程中：一次只问一个问题；必要时追问；避免一次性给出完整答案；最后给出简短小结。",
      "\n【岗位画像】\n" + (jdText || "（空）"),
    ].join("\n");
  });

  return {
    profile,
    profileText,
    interviewerPromptText,
    setProfile,
    loadFromStorage,
  };
});
