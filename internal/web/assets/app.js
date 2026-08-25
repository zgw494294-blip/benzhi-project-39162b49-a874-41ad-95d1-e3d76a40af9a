"use strict";

const state = { plans: [], stateCounts: {}, current: null, editingRevision: false };
const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];
const labels = {
  DRAFT: "草拟待校核", CHECK_BLOCKED: "校核阻断", REHEARSAL_READY: "联排就绪",
  REMEDIATION_REQUIRED: "需要整改", REVIEW_PENDING: "等待独立评审", AUTHORIZED: "已启用"
};

document.addEventListener("DOMContentLoaded", () => {
  bindStaticEvents();
  resetEditors();
  syncEvidenceRequirement();
  loadPlans();
});

function bindStaticEvents() {
  $("#refresh-plans").addEventListener("click", loadPlans);
  $("#plan-filter").addEventListener("submit", event => { event.preventDefault(); loadPlans(); });
  $("#plan-filter").addEventListener("reset", () => window.setTimeout(loadPlans));
  $("#new-plan").addEventListener("click", openCreateDialog);
  $("#open-revision").addEventListener("click", openRevisionDialog);
  $("#open-rehearsal").addEventListener("click", () => $("#rehearsal-dialog").showModal());
  $("#open-review").addEventListener("click", () => $("#review-dialog").showModal());
  $("#run-checks").addEventListener("click", runChecks);
  $("#add-load").addEventListener("click", () => addLoadRow());
  $("#add-cue").addEventListener("click", () => addCueRow());
  $("#plan-form").addEventListener("submit", savePlan);
  $("#rehearsal-form").addEventListener("submit", saveRehearsal);
  $$('#rehearsal-form [name="outcome"]').forEach(input => input.addEventListener("change", syncEvidenceRequirement));
  $("#review-form").addEventListener("submit", saveReview);
  $("#verify-form").addEventListener("submit", verifyAuthorization);
  $$('[data-close]').forEach(button => button.addEventListener("click", () => button.closest("dialog").close()));
  $$('.tabs [role="tab"]').forEach(button => button.addEventListener("click", () => selectTab(button.dataset.tab)));
}

async function api(path, options = {}) {
  const config = { ...options, headers: { "Content-Type": "application/json", ...(options.headers || {}) } };
  const response = await fetch(path, config);
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const failure = new Error(data.error?.message || `请求失败 (${response.status})`);
    failure.fields = data.error?.fields || [];
    failure.code = data.error?.code;
    throw failure;
  }
  return data;
}

async function loadPlans() {
  setBusy($("#refresh-plans"), true);
  try {
    const parameters = new URLSearchParams(new FormData($("#plan-filter")));
    for (const [key, value] of [...parameters]) if (!value) parameters.delete(key);
    const data = await api(`/api/v1/rigging-plans${parameters.size ? `?${parameters}` : ""}`);
    state.plans = data.plans || [];
    state.stateCounts = data.stateCounts || {};
    renderStateCounts();
    renderPlanList();
    if (state.current) {
      const found = state.plans.find(plan => plan.id === state.current.id);
      if (found) await selectPlan(found.id);
    }
  } catch (error) { toast(error.message, true); }
  finally { setBusy($("#refresh-plans"), false); }
}

function renderStateCounts() {
  $("#state-counts").innerHTML = Object.entries(labels).map(([key, label]) => `<span title="${escapeHTML(label)}"><strong>${state.stateCounts[key] || 0}</strong>${escapeHTML(label)}</span>`).join("");
}

function renderPlanList() {
  const list = $("#plan-list");
  if (!state.plans.length) {
    list.innerHTML = '<p class="eyebrow">当前没有方案</p>';
    return;
  }
  list.innerHTML = state.plans.map(plan => `
    <button class="plan-card ${state.current?.id === plan.id ? "active" : ""}" data-id="${escapeHTML(plan.id)}" type="button">
      <strong>${escapeHTML(plan.title)}</strong>
      <span>${escapeHTML(plan.venue)} · ${escapeHTML(plan.performanceDate)}</span>
      <span class="mini-state">${escapeHTML(labels[plan.state] || plan.state)} / R${plan.currentRevision}</span>
    </button>`).join("");
  $$(".plan-card", list).forEach(button => button.addEventListener("click", () => selectPlan(button.dataset.id)));
}

async function selectPlan(id) {
  try {
    state.current = await api(`/api/v1/rigging-plans/${encodeURIComponent(id)}`);
    renderPlanList();
    renderCurrentPlan();
  } catch (error) { toast(error.message, true); }
}

function renderCurrentPlan() {
  const plan = state.current;
  $("#empty-state").hidden = true;
  $("#plan-view").hidden = false;
  $("#plan-title").textContent = plan.title;
  $("#plan-context").textContent = `${plan.venue} · ${plan.performanceDate} · 技术负责人 ${plan.owner}`;
  $("#plan-state").textContent = labels[plan.state] || plan.state;
  $("#plan-state").className = `state-pill ${stateClass(plan.state)}`;
  $("#plan-revision").textContent = `REVISION ${plan.currentRevision}`;
  $("#plan-version").textContent = `v${plan.version}`;
  const revision = currentRevision();
  const latestCheck = [...plan.checkRuns].reverse().find(run => run.revisionId === revision.id);
  const blockers = latestCheck ? latestCheck.findings.filter(item => item.severity === "BLOCKER").length : 0;
  $("#metrics").innerHTML = metric("吊点", revision.loadPoints.length, "个") + metric("动作", revision.cues.length, "条") + metric("当前阻断", blockers, "项") + metric("审计事件", plan.timeline.length, "条");
  renderActions();
  renderRevision(revision);
  renderChecks(latestCheck);
  renderRecords();
  renderTimeline();
}

function renderActions() {
  const plan = state.current;
  const actionMap = {
    DRAFT: "执行确定性安全校核", CHECK_BLOCKED: "依据阻断项提交替代修订",
    REHEARSAL_READY: "完成当前修订的技术联排", REMEDIATION_REQUIRED: "提交替代修订并重新校核",
    REVIEW_PENDING: "由非提交者执行独立安全评审", AUTHORIZED: "启用单已冻结，可使用授权码验证"
  };
  $("#next-action").textContent = actionMap[plan.state];
  $("#run-checks").hidden = plan.state !== "DRAFT";
  $("#open-revision").hidden = !["CHECK_BLOCKED", "REMEDIATION_REQUIRED"].includes(plan.state);
  $("#open-rehearsal").hidden = plan.state !== "REHEARSAL_READY";
  $("#open-review").hidden = plan.state !== "REVIEW_PENDING";
}

function renderRevision(revision) {
  $("#load-table").innerHTML = revision.loadPoints.map(point => `<tr><td><strong>${escapeHTML(point.name)}</strong></td><td>${escapeHTML(point.position)}</td><td>${number(point.ratedCapacityKg)} kg</td><td>${number(point.plannedLoadKg)} kg</td><td>${number(point.angleDeg)}°</td><td>${number(point.safetyFactor, 2)}</td></tr>`).join("");
  $("#cue-table").innerHTML = revision.cues.sort((a,b) => a.cueNo-b.cueNo).map(cue => `<tr><td><strong>Q${cue.cueNo}</strong> ${escapeHTML(cue.label)}</td><td>${cue.startOffsetMs}–${cue.startOffsetMs + cue.durationMs} ms</td><td>${cue.movingPoints.map(escapeHTML).join("、")}</td><td>${number(cue.clearanceCm)} cm</td><td>${escapeHTML(cue.operator || "未指定")}</td></tr>`).join("");
  const diffs = new Map((state.current.revisionDiffs || []).map(diff => [diff.toRevisionId, diff]));
  $("#revision-history").innerHTML = [...state.current.revisions].reverse().map(item => {
    const diff = diffs.get(item.id);
    return `<div class="history-item"><strong>R${item.revisionNo}</strong><div>${escapeHTML(item.changeReason)}<br><span>${escapeHTML(item.submittedBy)}</span>${diff ? renderRevisionDiff(diff) : ""}</div><span>${formatTime(item.submittedAt)}${item.supersedesId ? " · 替代前版" : ""}</span></div>`;
  }).join("");
}

function renderRevisionDiff(diff) {
  const entries = diff.entries.length ? diff.entries.map(change => `<li><strong>${escapeHTML(diffKind(change.kind))}</strong> ${escapeHTML(diffSubject(change.subject))} ${escapeHTML(change.identifier)}${change.field ? ` / ${escapeHTML(change.field)}` : ""}${change.kind === "CHANGED" ? `：${escapeHTML(formatDiffValue(change.oldValue))} → ${escapeHTML(formatDiffValue(change.newValue))}` : ""}</li>`).join("") : "<li>业务字段无变化</li>";
  const closure = diff.closure;
  const status = `${closure.currentRechecked ? "已重新校核" : "待重新校核"} / ${closure.currentRehearsalPassed ? "已通过复验" : "待通过复验"}`;
  const oldIssue = closure.blockingFindings.length || closure.oldRehearsalOutcome ? `前版阻断 ${closure.blockingFindings.length} 项${closure.oldRehearsalOutcome ? `，联排 ${closure.oldRehearsalOutcome}` : ""}` : "前版无阻断记录";
  const findings = closure.blockingFindings.map(finding => `${finding.code} ${finding.subject}：${finding.description}`).join("；");
  const context = [findings, closure.oldObservations, (closure.oldEvidenceRefs || []).join("、")].filter(Boolean).map(value => `<p>${escapeHTML(value)}</p>`).join("");
  return `<div class="revision-diff"><p>${escapeHTML(oldIssue)} · ${escapeHTML(status)}</p>${context}<ul>${entries}</ul></div>`;
}

function formatDiffValue(value) { return typeof value === "object" ? JSON.stringify(value) : String(value ?? ""); }
function diffKind(value) { return ({ ADDED: "新增", DELETED: "删除", CHANGED: "变更" }[value] || value); }
function diffSubject(value) { return ({ LOAD_POINT: "吊点", CUE: "动作" }[value] || value); }

function renderChecks(run) {
  const list = $("#finding-list");
  if (!run) {
    $("#check-summary").textContent = "尚未校核";
    list.innerHTML = '<div class="clean-result">当前修订尚未生成校核结果。</div>';
    return;
  }
  $("#check-summary").textContent = `${formatTime(run.checkedAt)} · ${run.passed ? "通过" : "阻断"}`;
  if (!run.findings.length) {
    list.innerHTML = '<div class="clean-result">未发现阻断项或警告，确定性规则全部通过。</div>';
    return;
  }
  list.innerHTML = run.findings.map(finding => `<article class="finding ${finding.severity === "WARNING" ? "warning" : ""}"><div class="finding-code">${escapeHTML(finding.code)}<br><small>${escapeHTML(finding.severity)}</small></div><div class="finding-body"><strong>${escapeHTML(finding.subject)}</strong><span>${escapeHTML(finding.description)}</span></div></article>`).join("");
}

function renderRecords() {
  const plan = state.current;
  $("#rehearsal-list").innerHTML = plan.rehearsals.length ? [...plan.rehearsals].reverse().map(record => `<article class="record"><header><strong>${escapeHTML(record.observer)}</strong><span class="outcome ${record.outcome === "BLOCKED" ? "blocked" : ""}">${escapeHTML(record.outcome)}</span></header><p>${escapeHTML(record.observations || "无异常观察")}</p><small>${record.evidenceRefs.map(escapeHTML).join("、") || "无外部证据引用"} · ${formatTime(record.completedAt)}</small></article>`).join("") : '<p class="eyebrow">暂无联排记录</p>';
  $("#review-list").innerHTML = plan.reviews.length ? [...plan.reviews].reverse().map(review => `<article class="record"><header><strong>${escapeHTML(review.reviewer)}</strong><span class="outcome ${review.decision === "REJECTED" ? "blocked" : ""}">${escapeHTML(review.decision)}</span></header><p>${escapeHTML(review.comment || "评审通过")}</p><small>${formatTime(review.decidedAt)}</small></article>`).join("") : '<p class="eyebrow">暂无评审记录</p>';
  const panel = $("#authorization-panel");
  panel.hidden = plan.state !== "AUTHORIZED";
  if (!panel.hidden) panel.innerHTML = `<p class="eyebrow">FROZEN AUTHORIZATION</p><h3>演出启用单已冻结</h3><div class="authorization-code">${escapeHTML(plan.authorizationCode)}</div><p>SHA-256：${escapeHTML(plan.frozenDigest)}</p>`;
}

function renderTimeline() {
  $("#timeline").innerHTML = [...state.current.timeline].sort((a,b) => new Date(b.occurredAt)-new Date(a.occurredAt)).map(event => `<li><strong>${escapeHTML(event.summary)}</strong><span>${escapeHTML(event.actor)} · ${escapeHTML(labels[event.state] || event.state)} · ${formatTime(event.occurredAt)}</span></li>`).join("");
}

function openCreateDialog() {
  state.editingRevision = false;
  $("#form-mode").textContent = "NEW PLAN";
  $("#form-title").textContent = "建立吊挂方案";
  $("#save-plan").textContent = "提交方案";
  $("#plan-fields").hidden = false;
  $("#plan-form").reset();
  resetEditors();
  $("#plan-dialog").showModal();
}

function openRevisionDialog() {
  const revision = currentRevision();
  state.editingRevision = true;
  $("#form-mode").textContent = "REMEDIATION REVISION";
  $("#form-title").textContent = `提交第 ${state.current.currentRevision + 1} 版`;
  $("#save-plan").textContent = "提交替代修订";
  $("#plan-fields").hidden = true;
  $("#plan-form").reset();
  $("#load-editor").innerHTML = "";
  $("#cue-editor").innerHTML = "";
  revision.loadPoints.forEach(addLoadRow);
  revision.cues.forEach(addCueRow);
  $("#plan-dialog").showModal();
}

function resetEditors() {
  $("#load-editor").innerHTML = "";
  $("#cue-editor").innerHTML = "";
  addLoadRow({ name: "LX-01", position: "舞台左", ratedCapacityKg: 1000, plannedLoadKg: 400, angleDeg: 0, safetyFactor: 1.5 });
  addCueRow({ cueNo: 1, label: "飞景升起", startOffsetMs: 0, durationMs: 5000, movingPoints: ["LX-01"], clearanceCm: 50, operator: "机械操作员" });
}

function addLoadRow(value = {}) {
  const row = document.createElement("div"); row.className = "edit-row load";
  row.innerHTML = field("名称*", "name", value.name) + field("位置", "position", value.position) + numberField("额定 kg*", "ratedCapacityKg", value.ratedCapacityKg, 0.1) + numberField("载荷 kg*", "plannedLoadKg", value.plannedLoadKg, 0.1) + numberField("角度°", "angleDeg", value.angleDeg ?? 0, 0.1) + numberField("安全系数*", "safetyFactor", value.safetyFactor ?? 1.5, 0.01) + '<button class="remove-row" type="button" aria-label="删除吊点">×</button>';
  row.querySelector("button").addEventListener("click", () => row.remove());
  $("#load-editor").append(row);
}

function addCueRow(value = {}) {
  const row = document.createElement("div"); row.className = "edit-row cue";
  row.innerHTML = numberField("编号*", "cueNo", value.cueNo ?? $$(".edit-row.cue").length + 1, 1) + field("动作名称*", "label", value.label) + numberField("开始 ms*", "startOffsetMs", value.startOffsetMs ?? 0, 1) + numberField("持续 ms*", "durationMs", value.durationMs ?? 1000, 1) + field("吊点（逗号）*", "movingPoints", (value.movingPoints || []).join(",")) + numberField("净空 cm*", "clearanceCm", value.clearanceCm ?? 30, .1) + field("操作员", "operator", value.operator) + '<button class="remove-row" type="button" aria-label="删除动作">×</button>';
  row.querySelector("button").addEventListener("click", () => row.remove());
  $("#cue-editor").append(row);
}

async function savePlan(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const button = $("#save-plan");
  setBusy(button, true);
  clearFeedback("#plan-feedback");
  try {
    const data = Object.fromEntries(new FormData(form));
    const payload = { ...data, requestKey: crypto.randomUUID(), loadPoints: collectLoads(), cues: collectCues() };
    let plan;
    if (state.editingRevision) {
      payload.version = state.current.version;
      plan = await api(`/api/v1/rigging-plans/${state.current.id}/revisions`, { method: "POST", body: JSON.stringify(payload) });
    } else {
      plan = await api("/api/v1/rigging-plans", { method: "POST", body: JSON.stringify(payload) });
    }
    state.current = plan;
    form.closest("dialog").close();
    await loadPlans();
    toast(state.editingRevision ? "整改修订已提交" : "吊挂方案已建立");
  } catch (error) { showFeedback("#plan-feedback", error); }
  finally { setBusy(button, false); }
}

async function runChecks() {
  const button = $("#run-checks"); setBusy(button, true);
  try {
    state.current = await api(`/api/v1/rigging-plans/${state.current.id}/checks`, { method: "POST", body: JSON.stringify({ version: state.current.version, actor: state.current.owner, requestKey: crypto.randomUUID() }) });
    await loadPlans(); selectTab("checks"); toast("安全校核已完成");
  } catch (error) { toast(error.message, true); }
  finally { setBusy(button, false); }
}

async function saveRehearsal(event) {
  event.preventDefault(); const form = event.currentTarget; const button = $('button[type="submit"]', form); setBusy(button, true); clearFeedback("#rehearsal-feedback");
  try {
    const data = Object.fromEntries(new FormData(form));
    const payload = { ...data, revisionId: currentRevision().id, version: state.current.version, requestKey: crypto.randomUUID(), evidenceRefs: data.evidenceRefs === "" ? [] : data.evidenceRefs.split(",") };
    state.current = await api(`/api/v1/rigging-plans/${state.current.id}/rehearsals`, { method: "POST", body: JSON.stringify(payload) });
    form.closest("dialog").close(); form.reset(); await loadPlans(); selectTab("rehearsal"); toast("联排记录已进入审计时间线");
  } catch (error) { showFeedback("#rehearsal-feedback", error); }
  finally { setBusy(button, false); }
}

async function saveReview(event) {
  event.preventDefault(); const form = event.currentTarget; const button = $('button[type="submit"]', form); setBusy(button, true); clearFeedback("#review-feedback");
  try {
    const data = Object.fromEntries(new FormData(form));
    const payload = { ...data, version: state.current.version, requestKey: crypto.randomUUID() };
    state.current = await api(`/api/v1/rigging-plans/${state.current.id}/reviews`, { method: "POST", body: JSON.stringify(payload) });
    form.closest("dialog").close(); form.reset(); await loadPlans(); selectTab("rehearsal"); toast(payload.decision === "APPROVED" ? "评审通过，启用单已冻结" : "评审已退回整改");
  } catch (error) { showFeedback("#review-feedback", error); }
  finally { setBusy(button, false); }
}

async function verifyAuthorization(event) {
  event.preventDefault(); const output = $("#verify-result"); const code = new FormData(event.currentTarget).get("code").trim();
  if (!code) return;
  output.className = ""; output.textContent = "正在比对冻结摘要…";
  try {
    const result = await api(`/api/v1/authorizations/${encodeURIComponent(code)}`);
    output.className = result.valid ? "valid" : "invalid";
    output.textContent = `${result.valid ? "验证通过" : "验证失败"} [${result.reason}]：${result.message}${result.frozenRevisionDigest ? ` · ${result.frozenRevisionDigest}` : ""}`;
    if (result.valid) await selectPlan(result.planId);
  } catch (error) { output.className = "invalid"; output.textContent = error.message; }
}

function syncEvidenceRequirement() {
  const passed = $('#rehearsal-form [name="outcome"]:checked').value === "PASSED";
  $('#rehearsal-form [name="evidenceRefs"]').required = passed;
}

function collectLoads() { return $$(".edit-row.load").map(row => ({ name: value(row,"name"), position: value(row,"position"), ratedCapacityKg: numeric(row,"ratedCapacityKg"), plannedLoadKg: numeric(row,"plannedLoadKg"), angleDeg: numeric(row,"angleDeg"), safetyFactor: numeric(row,"safetyFactor") })); }
function collectCues() { return $$(".edit-row.cue").map(row => ({ cueNo: numeric(row,"cueNo"), label: value(row,"label"), startOffsetMs: numeric(row,"startOffsetMs"), durationMs: numeric(row,"durationMs"), movingPoints: value(row,"movingPoints").split(",").map(item => item.trim()).filter(Boolean), clearanceCm: numeric(row,"clearanceCm"), operator: value(row,"operator") })); }
function currentRevision() { return state.current.revisions.find(item => item.revisionNo === state.current.currentRevision); }
function selectTab(name) { $$('.tabs [role="tab"]').forEach(tab => tab.setAttribute("aria-selected", String(tab.dataset.tab === name))); $$(".tab-panel").forEach(panel => panel.hidden = panel.dataset.panel !== name); }
function metric(label, value, suffix) { return `<div class="metric"><span>${label}</span><strong>${value} ${suffix}</strong></div>`; }
function stateClass(value) { if (["CHECK_BLOCKED","REMEDIATION_REQUIRED"].includes(value)) return "blocked"; if (value === "AUTHORIZED") return "authorized"; if (["REHEARSAL_READY","REVIEW_PENDING"].includes(value)) return "ready"; return ""; }
function field(label, name, value = "") { return `<label>${label}<input name="${name}" value="${escapeHTML(String(value ?? ""))}" ${label.includes("*") ? "required" : ""}></label>`; }
function numberField(label, name, value = 0, step = 1) { return `<label>${label}<input name="${name}" type="number" step="${step}" value="${Number(value)}" ${label.includes("*") ? "required" : ""}></label>`; }
function value(row,name) { return $(`[name="${name}"]`,row).value.trim(); } function numeric(row,name) { return Number(value(row,name)); }
function number(value, digits = 1) { return Number(value).toLocaleString("zh-CN", { maximumFractionDigits: digits }); }
function formatTime(value) { return new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value)); }
function escapeHTML(value) { return String(value ?? "").replace(/[&<>'"]/g, char => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"})[char]); }
function setBusy(button,busy) { button.disabled = busy; button.setAttribute("aria-busy", String(busy)); }
function clearFeedback(selector) { $(selector).textContent = ""; }
function showFeedback(selector,error) { $(selector).textContent = error.fields?.length ? error.fields.map(item => `${item.field}：${item.message}`).join("；") : error.message; }
function toast(message,isError=false) { const element=$("#toast"); element.textContent=message; element.className=`toast show ${isError?"error":""}`; window.setTimeout(()=>element.classList.remove("show"),3000); }
