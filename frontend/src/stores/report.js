import { reactive } from "vue";
import api from "../utils/api";

const reports = reactive({});
const loadingBySession = reactive({});

export function useReportStore() {
  function getReport(sessionId) {
    const sid = String(sessionId || "");
    if (!sid) return null;
    return reports[sid] || null;
  }

  async function generateReport({ sessionId, modelType, jdProfile, force }) {
    const sid = String(sessionId || "");
    if (!sid) throw new Error("缺少 sessionId");

    if (reports[sid] && !force) return reports[sid];

    loadingBySession[sid] = true;
    try {
      const response = await api.post("/ai/interview/report", {
        sessionId: sid,
        modelType: modelType || "1",
        jdProfile: jdProfile || "",
        force: Boolean(force),
      });

      if (!response.data || response.data.status_code !== 1000) {
        throw new Error(response.data?.status_msg || "生成报告失败");
      }

      reports[sid] = response.data.data || response.data.report || null;
      if (!reports[sid]) {
        throw new Error("报告数据为空");
      }
      return reports[sid];
    } finally {
      loadingBySession[sid] = false;
    }
  }

  return {
    reports,
    loadingBySession,
    getReport,
    generateReport,
  };
}
