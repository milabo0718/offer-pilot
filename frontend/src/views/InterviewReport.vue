<template>
  <div class="container">
    <div class="content-wrapper">
      <div class="report-card">
        <div class="header">
          <div class="title">
            <h1>面试评价报告</h1>
            <p class="subtitle">会话：{{ sessionId }}</p>
          </div>
          <div class="actions">
            <el-button type="primary" plain @click="goBack">返回面试</el-button>
          </div>
        </div>

        <div class="body">
          <el-alert
            v-if="loadError"
            type="error"
            :closable="false"
            show-icon
            :title="loadError"
          />

          <div v-if="isLoading" class="loading">
            <el-skeleton :rows="6" animated />
          </div>

          <div v-else-if="report" class="report">
            <div class="section">
              <h2>总评</h2>
              <div class="kv">
                <span class="k">结论</span>
                <span class="v">{{
                  report.summary || report.conclusion || "-"
                }}</span>
              </div>
              <div class="kv">
                <span class="k">建议</span>
                <span class="v">{{
                  report.recommendation || report.suggestion || "-"
                }}</span>
              </div>
            </div>

            <div class="section">
              <h2>能力评分</h2>
              <div class="ability-layout">
                <div class="radar-wrap">
                  <div ref="radarRef" class="radar-chart" />
                </div>

                <div class="score-grid">
                  <div
                    v-for="item in scoreItems"
                    :key="item.key"
                    class="score-item"
                  >
                    <div class="score-title">
                      <span class="label">{{ item.label }}</span>
                      <span class="value">{{ item.value ?? "-" }}</span>
                    </div>
                    <el-progress
                      :percentage="normalizeScore(item.value)"
                      :stroke-width="10"
                      status="success"
                    />
                  </div>
                </div>
              </div>
            </div>

            <div class="section">
              <h2>详细反馈</h2>
              <el-input
                type="textarea"
                :rows="10"
                :model-value="report.detail || report.feedback || ''"
                readonly
              />
            </div>
          </div>

          <div v-else class="empty">
            <el-empty description="暂无报告数据" />
            <div class="empty-actions">
              <el-button type="primary" @click="retryGenerate"
                >重新生成</el-button
              >
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { useReportStore } from "../stores/report";
import { useInterviewStore } from "../stores/interview";
import { useJDProfileStore } from "../stores/jdProfile";
import * as echarts from "echarts";

const route = useRoute();
const router = useRouter();

const interviewStore = useInterviewStore();
const jdStore = useJDProfileStore();
const reportStore = useReportStore();
const { loadingBySession } = storeToRefs(reportStore);
const { selectedModel } = storeToRefs(interviewStore);

const sessionId = computed(() => String(route.params.sessionId || ""));
const loadError = ref("");

const radarRef = ref(null);
let radarChart = null;

const report = computed(() => reportStore.getReport(sessionId.value));
const isLoading = computed(() =>
  Boolean(loadingBySession.value[sessionId.value])
);

const scoreItems = computed(() => {
  const scores =
    report.value?.scores ||
    report.value?.score ||
    report.value?.abilityScores ||
    {};
  return [
    {
      key: "tech",
      label: "技术基础",
      value: scores.tech ?? scores.technicalBasics,
    },
    {
      key: "eng",
      label: "工程实践",
      value: scores.eng ?? scores.engineeringPractice,
    },
    { key: "ps", label: "问题解决", value: scores.ps ?? scores.problemSolving },
    {
      key: "comm",
      label: "沟通表达",
      value: scores.comm ?? scores.communication,
    },
    {
      key: "learn",
      label: "学习能力",
      value: scores.learn ?? scores.learningAbility,
    },
    { key: "fit", label: "岗位匹配度", value: scores.fit ?? scores.roleFit },
  ];
});

function normalizeScore(value) {
  const n = Number(value);
  if (Number.isNaN(n)) return 0;
  return Math.max(0, Math.min(100, Math.round(n)));
}

function ensureRadarChart() {
  if (!radarRef.value) return;
  if (radarChart) return;
  radarChart = echarts.init(radarRef.value);
}

function buildRadarOption() {
  const indicators = scoreItems.value.map((item) => ({
    name: item.label,
    max: 100,
  }));

  const values = scoreItems.value.map((item) => normalizeScore(item.value));

  return {
    backgroundColor: "transparent",
    tooltip: {
      trigger: "item",
    },
    radar: {
      indicator: indicators,
      radius: "68%",
      center: ["50%", "54%"],
      splitNumber: 5,
      axisName: {
        color: "rgba(255,255,255,0.85)",
        fontSize: 12,
      },
      axisLine: {
        lineStyle: {
          color: "rgba(255,255,255,0.22)",
        },
      },
      splitLine: {
        lineStyle: {
          color: "rgba(255,255,255,0.14)",
        },
      },
      splitArea: {
        areaStyle: {
          color: ["rgba(255,255,255,0.02)", "rgba(255,255,255,0.03)"],
        },
      },
    },
    series: [
      {
        type: "radar",
        data: [
          {
            value: values,
            name: "能力雷达",
            areaStyle: { opacity: 0.12 },
          },
        ],
        symbol: "circle",
        symbolSize: 6,
        lineStyle: {
          width: 2,
        },
        emphasis: {
          lineStyle: {
            width: 3,
          },
        },
      },
    ],
  };
}

async function renderRadar() {
  if (!report.value) return;
  await nextTick();
  ensureRadarChart();
  if (!radarChart) return;

  radarChart.setOption(buildRadarOption(), true);
  try {
    radarChart.resize();
  } catch {
    // ignore
  }
}

function handleResize() {
  if (!radarChart) return;
  try {
    radarChart.resize();
  } catch {
    // ignore
  }
}

function goBack() {
  router.push({ name: "AIChat" });
}

async function retryGenerate() {
  loadError.value = "";

  const sid = sessionId.value;
  if (!sid) {
    ElMessage.warning("缺少会话 ID");
    return;
  }

  try {
    await reportStore.generateReport({
      sessionId: sid,
      modelType: selectedModel.value,
      jdProfile: jdStore.profileText.value,
    });
  } catch (e) {
    loadError.value = e?.message || "生成报告失败";
  }
}

onMounted(async () => {
  window.addEventListener("resize", handleResize);

  if (!sessionId.value) {
    loadError.value = "缺少会话 ID";
    return;
  }

  if (report.value) {
    await renderRadar();
    return;
  }

  try {
    await reportStore.generateReport({
      sessionId: sessionId.value,
      modelType: selectedModel.value,
      jdProfile: jdStore.profileText.value,
    });
  } catch (e) {
    loadError.value = e?.message || "生成报告失败";
  } finally {
    await renderRadar();
  }
});

watch(
  () => report.value,
  async (nextReport) => {
    if (!nextReport) return;
    await renderRadar();
  },
  { immediate: true }
);

watch(
  () => scoreItems.value.map((x) => normalizeScore(x.value)).join(","),
  async () => {
    await renderRadar();
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize);
  if (radarChart) {
    try {
      radarChart.dispose();
    } catch {
      // ignore
    }
    radarChart = null;
  }
});
</script>

<style scoped>
.container {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  padding: 40px 20px;
}

.content-wrapper {
  max-width: 1100px;
  margin: 0 auto;
}

.report-card {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(15px);
  border-radius: 20px;
  padding: 32px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
}

.title h1 {
  margin: 0;
  font-size: 24px;
  color: #fff;
}

.subtitle {
  margin: 8px 0 0;
  color: rgba(255, 255, 255, 0.75);
}

.body {
  margin-top: 20px;
}

.section {
  margin-top: 20px;
  padding: 16px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.section h2 {
  margin: 0 0 12px;
  font-size: 16px;
  color: rgba(255, 255, 255, 0.9);
}

.kv {
  display: flex;
  gap: 12px;
  margin-top: 10px;
}

.k {
  min-width: 72px;
  color: rgba(255, 255, 255, 0.7);
}

.v {
  color: rgba(255, 255, 255, 0.9);
  white-space: pre-wrap;
}

.score-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.ability-layout {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 16px;
  align-items: stretch;
}

.radar-wrap {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 12px;
  min-height: 320px;
}

.radar-chart {
  width: 100%;
  height: 320px;
}

.score-item {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 12px;
}

.score-title {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  color: rgba(255, 255, 255, 0.9);
}

.hint {
  margin-top: 12px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
}

.empty {
  padding: 20px 0;
}

.empty-actions {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}

@media (max-width: 768px) {
  .report-card {
    padding: 20px;
  }

  .header {
    flex-direction: column;
    align-items: stretch;
  }

  .score-grid {
    grid-template-columns: 1fr;
  }

  .ability-layout {
    grid-template-columns: 1fr;
  }

  .radar-chart {
    height: 280px;
  }
}
</style>
