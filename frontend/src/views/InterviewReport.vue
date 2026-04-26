<template>
  <main class="report-page">
    <section class="report-shell">
      <header class="report-header">
        <div>
          <p class="eyebrow">OfferPilot Interview Report</p>
          <h1>面试评价报告</h1>
          <p class="subtitle">会话 {{ sessionId }}</p>
        </div>
        <div class="actions">
          <el-button @click="goBack">返回面试</el-button>
          <el-button type="primary" :loading="isLoading" @click="retryGenerate">
            重新生成
          </el-button>
        </div>
      </header>

      <el-alert
        v-if="loadError"
        type="error"
        :closable="false"
        show-icon
        :title="loadError"
      />

      <div v-if="isLoading" class="loading">
        <el-skeleton :rows="9" animated />
      </div>

      <template v-else-if="report">
        <el-alert
          v-if="report.fallback"
          class="fallback-alert"
          type="warning"
          :closable="false"
          show-icon
          title="模型评分暂不可用，当前展示本地兜底评估；可稍后点击重新生成。"
        />

        <section class="summary-band">
          <div class="overall-score" :style="overallScoreStyle">
            <span class="score-number">{{ overallScore }}</span>
            <span class="score-label">综合评分</span>
          </div>
          <div class="summary-copy">
            <h2>{{ report.summary || "已生成面试评估报告" }}</h2>
            <p>
              {{ report.recommendation || "建议结合更多追问结果综合判断。" }}
            </p>
          </div>
          <div class="meta-list">
            <div>
              <span class="meta-k">有效作答</span>
              <strong>{{ report.evidenceCount || 0 }} 轮</strong>
            </div>
            <div>
              <span class="meta-k">生成时间</span>
              <strong>{{ generatedAt }}</strong>
            </div>
          </div>
        </section>

        <section class="ability-section">
          <div class="radar-panel">
            <h2>能力雷达图</h2>
            <svg class="radar-chart" viewBox="0 0 360 320" role="img">
              <g class="radar-grid">
                <polygon
                  v-for="level in radarLevels"
                  :key="level"
                  :points="radarGridPoints(level)"
                />
                <line
                  v-for="axis in radarAxisPoints"
                  :key="axis.key"
                  :x1="radarCenter.x"
                  :y1="radarCenter.y"
                  :x2="axis.x"
                  :y2="axis.y"
                />
              </g>
              <polygon class="radar-area" :points="radarValuePoints" />
              <polyline class="radar-line" :points="radarValuePoints" />
              <circle
                v-for="point in radarDataPoints"
                :key="point.key"
                :cx="point.x"
                :cy="point.y"
                r="4"
                class="radar-point"
              />
              <text
                v-for="label in radarLabels"
                :key="label.key"
                class="radar-label"
                :x="label.x"
                :y="label.y"
                :text-anchor="label.anchor"
              >
                {{ label.name }}
              </text>
            </svg>
          </div>

          <div class="score-list">
            <div v-for="item in scoreItems" :key="item.key" class="score-row">
              <div class="score-row-head">
                <span>{{ item.label }}</span>
                <strong>{{ normalizeScore(item.value) }}</strong>
              </div>
              <el-progress
                :percentage="normalizeScore(item.value)"
                :stroke-width="9"
                :color="progressColor"
              />
            </div>
          </div>
        </section>

        <section class="insight-grid">
          <article class="insight-card">
            <h2>亮点</h2>
            <ul>
              <li v-for="item in listOrDefault(report.strengths)" :key="item">
                {{ item }}
              </li>
            </ul>
          </article>
          <article class="insight-card">
            <h2>风险</h2>
            <ul>
              <li v-for="item in listOrDefault(report.risks)" :key="item">
                {{ item }}
              </li>
            </ul>
          </article>
          <article class="insight-card">
            <h2>下一步</h2>
            <ul>
              <li v-for="item in listOrDefault(report.actionItems)" :key="item">
                {{ item }}
              </li>
            </ul>
          </article>
        </section>

        <section class="detail-section">
          <h2>详细反馈</h2>
          <p>{{ report.detail || report.feedback || "暂无详细反馈。" }}</p>
        </section>
      </template>

      <section v-else class="empty">
        <el-empty description="暂无报告数据" />
      </section>
    </section>
  </main>
</template>

<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { useReportStore } from "../stores/report";
import { useJDProfileStore } from "../stores/jdProfile";

const route = useRoute();
const router = useRouter();
const jdStore = useJDProfileStore();
const reportStore = useReportStore();

const selectedModel = "1";
const sessionId = computed(() => String(route.params.sessionId || ""));
const loadError = ref("");
const report = computed(() => reportStore.getReport(sessionId.value));
const isLoading = computed(() =>
  Boolean(reportStore.loadingBySession[sessionId.value])
);

const radarCenter = { x: 180, y: 158 };
const radarRadius = 102;
const radarLevels = [0.2, 0.4, 0.6, 0.8, 1];
const progressColor = [
  { color: "#e74c3c", percentage: 45 },
  { color: "#e6a23c", percentage: 65 },
  { color: "#409eff", percentage: 80 },
  { color: "#2fb67c", percentage: 100 },
];

const scoreItems = computed(() => {
  const scores =
    report.value?.scores ||
    report.value?.score ||
    report.value?.abilityScores ||
    {};
  return [
    { key: "tech", label: "技术基础", value: scores.tech },
    { key: "eng", label: "工程实践", value: scores.eng },
    { key: "ps", label: "问题解决", value: scores.ps },
    { key: "comm", label: "沟通表达", value: scores.comm },
    { key: "learn", label: "学习能力", value: scores.learn },
    { key: "fit", label: "岗位匹配度", value: scores.fit },
  ];
});

const overallScore = computed(() => {
  const values = scoreItems.value.map((item) => normalizeScore(item.value));
  if (!values.length) return 0;
  return Math.round(
    values.reduce((sum, value) => sum + value, 0) / values.length
  );
});

const overallScoreStyle = computed(() => ({
  "--score-angle": `${overallScore.value * 3.6}deg`,
}));

const generatedAt = computed(() => {
  if (!report.value?.generatedAt) return "-";
  try {
    return new Date(report.value.generatedAt).toLocaleString();
  } catch {
    return "-";
  }
});

function normalizeScore(value) {
  const n = Number(value);
  if (Number.isNaN(n)) return 0;
  return Math.max(0, Math.min(100, Math.round(n)));
}

function radarPoint(index, valueRatio = 1, radius = radarRadius) {
  const angle = -Math.PI / 2 + (2 * Math.PI * index) / scoreItems.value.length;
  const r = radius * valueRatio;
  return {
    x: radarCenter.x + Math.cos(angle) * r,
    y: radarCenter.y + Math.sin(angle) * r,
  };
}

function pointsToString(points) {
  return points.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ");
}

function radarGridPoints(level) {
  return pointsToString(
    scoreItems.value.map((_, index) => radarPoint(index, level))
  );
}

const radarAxisPoints = computed(() =>
  scoreItems.value.map((item, index) => ({
    key: item.key,
    ...radarPoint(index),
  }))
);

const radarDataPoints = computed(() =>
  scoreItems.value.map((item, index) => ({
    key: item.key,
    ...radarPoint(index, normalizeScore(item.value) / 100),
  }))
);

const radarValuePoints = computed(() => pointsToString(radarDataPoints.value));

const radarLabels = computed(() =>
  scoreItems.value.map((item, index) => {
    const point = radarPoint(index, 1, radarRadius + 25);
    let anchor = "middle";
    if (point.x < radarCenter.x - 8) anchor = "end";
    if (point.x > radarCenter.x + 8) anchor = "start";
    return {
      key: item.key,
      name: item.label,
      anchor,
      x: point.x,
      y: point.y + 4,
    };
  })
);

function listOrDefault(items) {
  return Array.isArray(items) && items.length ? items : ["暂无明确证据。"];
}

function goBack() {
  router.push({ name: "AIChat" });
}

async function retryGenerate() {
  loadError.value = "";
  if (!sessionId.value) {
    ElMessage.warning("缺少会话 ID");
    return;
  }

  try {
    await reportStore.generateReport({
      sessionId: sessionId.value,
      modelType: selectedModel,
      jdProfile: jdStore.profileText.value,
      force: true,
    });
  } catch (e) {
    loadError.value = e?.message || "生成报告失败";
  }
}

onMounted(async () => {
  if (!sessionId.value) {
    loadError.value = "缺少会话 ID";
    return;
  }

  if (report.value) return;

  try {
    await reportStore.generateReport({
      sessionId: sessionId.value,
      modelType: selectedModel,
      jdProfile: jdStore.profileText.value,
    });
  } catch (e) {
    loadError.value = e?.message || "生成报告失败";
  }
});

watch(
  () => report.value,
  (nextReport) => {
    if (nextReport) loadError.value = "";
  },
  { immediate: true }
);
</script>

<style scoped>
.report-page {
  min-height: 100vh;
  background: #f4f7fb;
  color: #1f2d3d;
  padding: 28px;
}

.report-shell {
  max-width: 1160px;
  margin: 0 auto;
}

.report-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 20px;
  margin-bottom: 20px;
}

.eyebrow {
  margin: 0 0 6px;
  color: #409eff;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.report-header h1 {
  margin: 0;
  font-size: 28px;
}

.subtitle {
  margin: 8px 0 0;
  color: #667085;
}

.actions {
  display: flex;
  gap: 10px;
}

.loading,
.empty {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 24px;
}

.fallback-alert {
  margin-bottom: 16px;
}

.summary-band,
.ability-section,
.insight-card,
.detail-section {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(31, 45, 61, 0.06);
}

.summary-band {
  display: grid;
  grid-template-columns: 150px 1fr 240px;
  gap: 24px;
  align-items: center;
  padding: 24px;
  margin-bottom: 18px;
}

.overall-score {
  width: 126px;
  height: 126px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: conic-gradient(#2fb67c var(--score-angle), #e8edf3 0);
  position: relative;
}

.overall-score::before {
  content: "";
  position: absolute;
  width: 98px;
  height: 98px;
  border-radius: 50%;
  background: #fff;
}

.score-number,
.score-label {
  position: relative;
  z-index: 1;
}

.score-number {
  font-size: 34px;
  font-weight: 800;
}

.score-label {
  color: #667085;
  font-size: 13px;
}

.summary-copy h2 {
  margin: 0 0 10px;
  font-size: 20px;
}

.summary-copy p {
  margin: 0;
  color: #475467;
  line-height: 1.7;
}

.meta-list {
  display: grid;
  gap: 12px;
}

.meta-list div {
  padding: 12px;
  background: #f7f9fc;
  border: 1px solid #edf0f5;
  border-radius: 8px;
}

.meta-k {
  display: block;
  color: #667085;
  font-size: 12px;
  margin-bottom: 4px;
}

.ability-section {
  display: grid;
  grid-template-columns: minmax(320px, 1fr) 1fr;
  gap: 22px;
  padding: 24px;
  margin-bottom: 18px;
}

.radar-panel h2,
.insight-card h2,
.detail-section h2 {
  margin: 0 0 14px;
  font-size: 17px;
}

.radar-chart {
  width: 100%;
  height: 320px;
  display: block;
}

.radar-grid polygon {
  fill: #f7f9fc;
  stroke: #d8dee8;
  stroke-width: 1;
}

.radar-grid line {
  stroke: #d8dee8;
  stroke-width: 1;
}

.radar-area {
  fill: rgba(47, 182, 124, 0.2);
}

.radar-line {
  fill: none;
  stroke: #2fb67c;
  stroke-width: 3;
  stroke-linejoin: round;
}

.radar-point {
  fill: #2fb67c;
  stroke: #fff;
  stroke-width: 2;
}

.radar-label {
  fill: #344054;
  font-size: 12px;
  font-weight: 700;
}

.score-list {
  display: grid;
  gap: 12px;
}

.score-row {
  padding: 14px;
  background: #f7f9fc;
  border: 1px solid #edf0f5;
  border-radius: 8px;
}

.score-row-head {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.score-row-head strong {
  color: #2fb67c;
}

.insight-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 18px;
  margin-bottom: 18px;
}

.insight-card {
  padding: 20px;
}

.insight-card ul {
  margin: 0;
  padding-left: 18px;
  color: #475467;
  line-height: 1.8;
}

.detail-section {
  padding: 22px;
}

.detail-section p {
  margin: 0;
  color: #475467;
  line-height: 1.8;
  white-space: pre-wrap;
}

@media (max-width: 900px) {
  .report-header,
  .summary-band,
  .ability-section,
  .insight-grid {
    grid-template-columns: 1fr;
  }

  .report-header {
    align-items: stretch;
  }

  .actions {
    justify-content: flex-start;
  }
}
</style>
