import { defineStore } from "pinia";
import { ref } from "vue";
import api from "../utils/api";

export const useReportStore = defineStore("report", () => {
  const reports = ref({}); // sessionId -> report
  const loadingBySession = ref({});

  function getReport(sessionId) {
    const sid = String(sessionId || "");
    if (!sid) return null;
    return reports.value[sid] || null;
  }

  async function generateReport({ sessionId, modelType, jdProfile }) {
    const sid = String(sessionId || "");
    if (!sid) throw new Error("缺少 sessionId");

    if (reports.value[sid]) return reports.value[sid];

    loadingBySession.value[sid] = true;
    try {
      const response = await api.post("/ai/interview/report", {
        sessionId: sid,
        modelType: modelType || "1",
        jdProfile: jdProfile || "",
      });

      if (!response.data || response.data.status_code !== 1000) {
        throw new Error(response.data?.status_msg || "生成报告失败");
      }

      reports.value[sid] = response.data.data || response.data.report || null;
      return reports.value[sid];
    } finally {
      loadingBySession.value[sid] = false;
    }
  }

  return {
    reports,
    loadingBySession,
    getReport,
    generateReport,
  };
});
