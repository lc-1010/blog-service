const pageSize = 10;
let currentUser = "";
let articlePage = 1;
let tagPage = 1;
let articlesTotal = 0;
let tagsTotal = 0;
let tags = [];
let noticeTimer;
let currentPrincipal = { permissions: [] };

const $ = (selector) => document.querySelector(selector);
const escapeText = (value) => String(value ?? "");
const escapeHTML = (value) => escapeText(value).replace(/[&<>"']/g, (character) => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
})[character]);
const hasPermission = (permission) => currentPrincipal.permissions?.includes("*") || currentPrincipal.permissions?.includes(permission);

async function request(path, options = {}) {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers: {
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers,
    },
  });
  const body = await response.json().catch(() => ({}));
  if (response.status === 401 && !path.endsWith("/login")) {
    showLogin();
    throw new Error("登录状态已失效，请重新登录");
  }
  if (!response.ok) throw new Error(body.message || body.msg || `请求失败 (${response.status})`);
  return body;
}

function showLogin() {
  $("#admin-view").hidden = true;
  $("#login-view").hidden = false;
  $("#login-form").elements.password.value = "";
}

function showAdmin(username, principal = {}) {
  currentUser = username;
  currentPrincipal = principal;
  $("#current-user").textContent = username;
  $(".avatar").textContent = (principal.display_name || username).slice(0, 1).toUpperCase();
  $("#admin-view").hidden = false;
  $("#login-view").hidden = true;
  setPlatformNavigation();
  const landingPage = hasPermission("content.manage")
    ? "dashboard"
    : pageAllowed("platform") ? "platform"
      : ["users", "roles", "settings", "sites"].find(pageAllowed) || "audit";
  showPage(landingPage);
}

function notify(message) {
  const node = $("#notice");
  node.textContent = message;
  node.hidden = false;
  clearTimeout(noticeTimer);
  noticeTimer = setTimeout(() => { node.hidden = true; }, 3200);
}

function showError(error) {
  notify(error.message || "操作失败，请稍后重试");
}

function showPage(page) {
  if (!pageAllowed(page)) {
    showError(new Error("当前账号没有访问该平台功能的权限"));
    return;
  }
  document.querySelectorAll(".page").forEach((node) => {
    node.hidden = node.id !== `page-${page}`;
  });
  document.querySelectorAll(".nav-item").forEach((node) => {
    node.classList.toggle("active", node.dataset.page === page);
  });
  $("#page-title").textContent = {
    dashboard: "数据概览", articles: "文章管理", tags: "标签管理", platform: "平台总览",
    users: "用户与权限", roles: "角色管理", settings: "系统配置", sites: "站点管理", audit: "操作日志",
  }[page];
  if (page === "dashboard") loadDashboard().catch(showError);
  if (page === "articles") {
    loadTags().then(loadArticles).catch(showError);
  }
  if (page === "tags") loadTagsPage().catch(showError);
  if (pageAllowed(page)) loadPlatformPage(page).catch(showError);
}

function pageAllowed(page) {
  if (page === "dashboard" || page === "articles" || page === "tags") return hasPermission("content.manage");
  const required = {
    platform: "audit.read",
    users: "users.manage",
    roles: "roles.manage",
    settings: "settings.manage",
    sites: "sites.manage",
    audit: "audit.read",
  }[page];
  return required ? hasPermission(required) : false;
}

function setPlatformNavigation() {
  document.querySelectorAll(".nav-item[data-page]").forEach((button) => {
    const page = button.dataset.page;
    button.hidden = !pageAllowed(page);
  });
}

async function loadDashboard() {
  const stats = await request("/api/v1/admin/dashboard");
  $("#stat-articles").textContent = stats.articles;
  $("#stat-published").textContent = stats.published;
  $("#stat-drafts").textContent = stats.drafts;
  $("#stat-tags").textContent = stats.tags;
}

async function loadTags() {
  const result = await request(`/api/v1/tags?all_states=true&page=1&page_size=100`);
  tags = result.list || [];
  const select = $("#article-form").elements.tag_id;
  const selected = select.value;
  select.replaceChildren(new Option("请选择标签", ""));
  tags.filter((tag) => tag.state === 1).forEach((tag) => {
    select.add(new Option(tag.name, tag.id));
  });
  if (selected) select.value = selected;
  return tags;
}

async function loadArticles() {
  const params = new URLSearchParams({ page: String(articlePage), page_size: String(pageSize), all_states: "true" });
  const title = $("#article-search").value.trim();
  const state = $("#article-state").value;
  if (title) params.set("title", title);
  if (state !== "") {
    params.delete("all_states");
    params.set("state", state);
  }
  const [result, allTags] = await Promise.all([
    request(`/api/v1/articles?${params}`),
    Promise.resolve(tags),
  ]);
  const tagNames = new Map(allTags.map((tag) => [String(tag.id), tag.name]));
  articlesTotal = result.pager?.total_rows || 0;
  const body = $("#articles-body");
  body.replaceChildren();
  (result.list || []).forEach((article) => {
    const row = document.createElement("tr");
    const titleCell = document.createElement("td");
    const title = document.createElement("strong");
    title.className = "row-title";
    title.textContent = escapeText(article.title);
    const summary = document.createElement("span");
    summary.className = "row-subtitle";
    summary.textContent = escapeText(article.desc || "暂无摘要");
    titleCell.append(title, summary);
    const tagCell = document.createElement("td");
    tagCell.textContent = tagNames.get(String(article.tag_id)) || "未分类";
    const stateCell = document.createElement("td");
    stateCell.append(statusBadge(article.state === 1, article.state === 1 ? "已发布" : "草稿"));
    const updatedCell = document.createElement("td");
    updatedCell.textContent = formatDate(article.modified_on || article.created_on);
    const actions = document.createElement("td");
    actions.className = "row-actions";
    actions.append(
      actionButton("编辑", () => editArticle(article)),
      actionButton(article.state === 1 ? "转为草稿" : "发布", () => toggleArticleState(article)),
      actionButton("删除", () => deleteArticle(article), "danger-text"),
    );
    row.append(titleCell, tagCell, stateCell, updatedCell, actions);
    body.append(row);
  });
  $("#articles-empty").hidden = body.children.length !== 0;
  updatePagination("articles", articlePage, articlesTotal);
}

async function loadTagsPage() {
  const params = new URLSearchParams({ page: String(tagPage), page_size: String(pageSize), all_states: "true" });
  const name = $("#tag-search").value.trim();
  const state = $("#tag-state").value;
  if (name) params.set("name", name);
  if (state !== "") {
    params.delete("all_states");
    params.set("state", state);
  }
  const result = await request(`/api/v1/tags?${params}`);
  tagsTotal = result.pager?.total_rows || 0;
  const body = $("#tags-body");
  body.replaceChildren();
  (result.list || []).forEach((tag) => {
    const row = document.createElement("tr");
    const nameCell = document.createElement("td");
    const name = document.createElement("strong");
    name.className = "row-title";
    name.textContent = escapeText(tag.name);
    nameCell.append(name);
    const stateCell = document.createElement("td");
    stateCell.append(statusBadge(tag.state === 1, tag.state === 1 ? "启用" : "停用"));
    const createdCell = document.createElement("td");
    createdCell.textContent = formatDate(tag.created_on);
    const actions = document.createElement("td");
    actions.className = "row-actions";
    actions.append(
      actionButton("编辑", () => editTag(tag)),
      actionButton(tag.state === 1 ? "停用" : "启用", () => toggleTagState(tag)),
      actionButton("删除", () => deleteTag(tag), "danger-text"),
    );
    row.append(nameCell, stateCell, createdCell, actions);
    body.append(row);
  });
  $("#tags-empty").hidden = body.children.length !== 0;
  updatePagination("tags", tagPage, tagsTotal);
}

function updatePagination(type, page, total) {
  const count = $(`#${type}-count`);
  count.textContent = `共 ${total} 条 · 第 ${page} 页`;
  $(`#${type}-prev`).disabled = page <= 1;
  $(`#${type}-next`).disabled = page * pageSize >= total;
}

function statusBadge(active, text) {
  const badge = document.createElement("span");
  badge.className = `badge ${active ? "badge-green" : "badge-gray"}`;
  badge.textContent = text;
  return badge;
}

function actionButton(text, action, className = "") {
  const button = document.createElement("button");
  button.type = "button";
  button.className = `table-action ${className}`;
  button.textContent = text;
  button.addEventListener("click", () => action().catch(showError));
  return button;
}

function formatDate(timestamp) {
  if (!timestamp) return "—";
  return new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium" }).format(new Date(timestamp * 1000));
}

function openArticleDialog(article) {
  const form = $("#article-form");
  form.reset();
  $("#article-form-error").textContent = "";
  $("#article-dialog-title").textContent = article ? "编辑文章" : "新建文章";
  if (article) {
    for (const key of ["id", "title", "desc", "tag_id", "state", "cover_image_url", "content"]) {
      form.elements[key].value = article[key] ?? "";
    }
    form.dataset.originalState = String(article.state);
  } else {
    delete form.dataset.originalState;
    form.elements.state.value = "1";
  }
  $("#article-dialog").showModal();
}

async function editArticle(article) {
  await loadTags();
  const fullArticle = await request(`/api/v1/articles/${article.id}`);
  openArticleDialog(fullArticle);
}

async function toggleArticleState(article) {
  const state = article.state === 1 ? 0 : 1;
  await request(`/api/v1/articles/${article.id}/state`, {
    method: "PATCH",
    body: JSON.stringify({ state, modified_by: currentUser }),
  });
  notify(state === 1 ? "文章已发布" : "文章已转为草稿");
  await Promise.all([loadArticles(), loadDashboard()]);
}

async function deleteArticle(article) {
  if (!window.confirm(`确定删除文章“${article.title}”吗？`)) return;
  await request(`/api/v1/articles/${article.id}`, { method: "DELETE" });
  notify("文章已删除");
  await Promise.all([loadArticles(), loadDashboard()]);
}

function openTagDialog(tag) {
  const form = $("#tag-form");
  form.reset();
  $("#tag-form-error").textContent = "";
  $("#tag-dialog-title").textContent = tag ? "编辑标签" : "新建标签";
  if (tag) {
    form.elements.id.value = tag.id;
    form.elements.name.value = tag.name;
    form.elements.state.value = tag.state;
  } else {
    form.elements.state.value = "1";
  }
  $("#tag-dialog").showModal();
}

function editTag(tag) {
  openTagDialog(tag);
  return Promise.resolve();
}

async function saveTag(tag, method, path) {
  const payload = {
    name: tag.name,
    state: Number(tag.state),
    ...(method === "POST" ? { created_by: currentUser } : { modified_by: currentUser }),
  };
  await request(path, { method, body: JSON.stringify(payload) });
  $("#tag-dialog").close();
  notify("标签已保存");
  await Promise.all([loadTagsPage(), loadTags(), loadDashboard()]);
}

async function toggleTagState(tag) {
  await saveTag({ ...tag, state: tag.state === 1 ? 0 : 1 }, "PUT", `/api/v1/tags/${tag.id}`);
}

async function deleteTag(tag) {
  if (!window.confirm(`确定删除标签“${tag.name}”吗？`)) return;
  await request(`/api/v1/tags/${tag.id}`, { method: "DELETE" });
  notify("标签已删除");
  await Promise.all([loadTagsPage(), loadTags(), loadDashboard()]);
}

const platformRequest = (path, options) => request(`/api/v1/admin/platform${path}`, options);
const permissionLabels = {
  "content.manage": "文章与标签管理",
  "users.manage": "用户管理",
  "roles.manage": "角色与权限配置",
  "settings.manage": "系统配置",
  "sites.manage": "站点管理",
  "audit.read": "操作日志查看",
};

function platformPageHeader(eyebrow, title, description, action = "") {
  return `<div class="page-heading"><div><p class="eyebrow">${escapeHTML(eyebrow)}</p><h1>${escapeHTML(title)}</h1><p class="muted">${escapeHTML(description)}</p></div>${action}</div>`;
}

async function loadPlatformPage(page) {
  const host = $(`#page-${page}`);
  if (!host) return;
  if (page === "platform") return loadPlatformDashboard(host);
  if (page === "users") return loadPlatformUsers(host);
  if (page === "roles") return loadPlatformRoles(host);
  if (page === "settings") return loadPlatformSettings(host);
  if (page === "sites") return loadPlatformSites(host);
  if (page === "audit") return loadPlatformAudit(host);
}

async function loadPlatformDashboard(host) {
  host.innerHTML = `${platformPageHeader("PLATFORM", "平台概览", "查看平台运行状态与近期管理活动。")}<div id="platform-overview-error"></div><div class="stat-grid" id="platform-stats"></div><div class="platform-grid"><section class="panel platform-panel"><div class="platform-panel-heading"><div><p class="eyebrow">PLATFORM HEALTH</p><h2>平台运行状态</h2><p class="muted">基础服务信息与当前站点状态。</p></div><span class="health-pill">● 管理服务在线</span></div><div class="health-grid"><article><span>API 服务</span><strong>运行中</strong><small>博客管理 API</small></article><article><span>数据库连接</span><strong>已连接</strong><small>MySQL</small></article><article><span>会话认证</span><strong>已启用</strong><small>HttpOnly Cookie</small></article></div></section><section class="panel platform-panel"><div class="platform-panel-heading"><div><p class="eyebrow">QUICK ACTIONS</p><h2>快捷入口</h2></div></div><div class="quick-actions">${[
    ["users", "♙", "管理平台用户"],
    ["roles", "◇", "配置角色权限"],
    ["settings", "⚙", "调整系统配置"],
    ["sites", "▣", "管理站点"],
  ].filter(([target]) => pageAllowed(target)).map(([target, icon, title]) => `<button class="quick-action" data-go="${target}"><span>${icon}</span>${title}<b>→</b></button>`).join("")}</div></section></div><section class="panel platform-panel audit-preview"><div class="platform-panel-heading"><div><h2>最近操作</h2><p class="muted">平台用户最近执行的管理操作。</p></div><button class="link-button" data-go="audit">查看全部日志 →</button></div><div class="table-wrap"><table><thead><tr><th>操作人</th><th>操作内容</th><th>资源</th><th>结果</th><th>时间</th></tr></thead><tbody id="platform-recent-audit"></tbody></table></div></section>`;
  host.querySelectorAll("[data-go]").forEach((button) => button.addEventListener("click", () => showPage(button.dataset.go)));
  try {
    const data = await platformRequest("/dashboard");
    $("#platform-stats").innerHTML = [
      ["平台用户", data.users, "活跃用户", "♙"],
      ["角色权限", data.roles, "平台角色", "◇"],
      ["托管站点", data.sites, "启用站点", "▣"],
      ["管理操作", data.operations, `近 7 天 · ${data.failures} 项异常`, "◷"],
    ].map(([label, value, note, icon]) => `<article class="stat-card platform-stat"><span class="stat-icon green">${icon}</span><p>${label}</p><strong>${value}</strong><small>${note}</small></article>`).join("");
    const logs = await platformRequest("/audit-logs?page=1&page_size=5");
    renderAuditRows($("#platform-recent-audit"), logs.list || []);
  } catch (error) {
    $("#platform-overview-error").innerHTML = `<p class="error platform-error">${escapeHTML(error.message)}。如为首次部署，请先应用平台管理数据库迁移。</p>`;
  }
}

async function loadPlatformUsers(host) {
  host.innerHTML = `${platformPageHeader("PLATFORM ACCESS", "用户与权限", "创建平台账号、分配角色并控制账号状态。", '<button class="button primary" id="new-platform-user">＋ 新建用户</button>')}<div class="platform-error" id="users-error"></div><section class="panel"><div class="toolbar"><label class="search-field"><span>⌕</span><input id="user-filter" placeholder="搜索用户名或姓名"></label><button class="button outline" id="user-filter-button">查询</button></div><div class="table-wrap"><table><thead><tr><th>用户</th><th>角色</th><th>状态</th><th>创建时间</th><th class="actions-head">操作</th></tr></thead><tbody id="platform-users"></tbody></table></div><div class="pagination"><span id="user-count"></span></div></section><dialog id="platform-user-dialog" class="editor-dialog compact-dialog"><form id="platform-user-form"><div class="dialog-heading"><div><p class="eyebrow">PLATFORM USER</p><h2 id="platform-user-heading">新建平台用户</h2></div><button type="button" class="icon-button" data-close="platform-user-dialog">×</button></div><input type="hidden" name="id"><label>登录账号<input name="username" required minlength="3" maxlength="100" pattern="[A-Za-z0-9._-]+" placeholder="例如：editor01"></label><label>显示名称<input name="display_name" required maxlength="100" placeholder="姓名或团队名称"></label><label>初始密码<input name="password" type="password" minlength="12" maxlength="72" autocomplete="new-password" placeholder="至少 12 位；编辑时留空表示不修改"></label><label>角色<select name="role_id" required></select></label><label>账号状态<select name="state"><option value="1">启用</option><option value="0">停用</option></select></label><p class="error" id="platform-user-error"></p><div class="dialog-actions"><button type="button" class="button outline" data-close="platform-user-dialog">取消</button><button class="button primary" type="submit">保存用户</button></div></form></dialog>`;
  host.querySelector("#new-platform-user").addEventListener("click", () => openPlatformUser());
  host.querySelector("#user-filter-button").addEventListener("click", () => renderPlatformUsers());
  host.querySelector("#user-filter").addEventListener("keydown", (event) => {
    if (event.key === "Enter") renderPlatformUsers();
  });
  host.querySelectorAll("[data-close]").forEach((button) => button.addEventListener("click", () => host.querySelector("#platform-user-dialog").close()));
  host.querySelector("#platform-user-form").addEventListener("submit", savePlatformUser);
  try {
    const roles = await platformRequest("/roles");
    const assignableRoles = (roles.list || []).filter((role) => !role.permissions.includes("*"));
    host.querySelector('[name="role_id"]').innerHTML = assignableRoles.map((role) => `<option value="${role.id}">${escapeHTML(role.name)}</option>`).join("");
    await renderPlatformUsers();
  } catch (error) {
    host.querySelector("#users-error").textContent = `无法加载平台用户：${error.message}`;
  }
}

async function renderPlatformUsers() {
  const host = $("#platform-users");
  if (!host) return;
  try {
    const result = await platformRequest("/users");
    const filter = $("#user-filter").value.trim().toLowerCase();
    const users = (result.list || []).filter((user) => `${user.username} ${user.display_name}`.toLowerCase().includes(filter));
    host.innerHTML = users.map((user) => `<tr><td><strong class="row-title">${escapeHTML(user.display_name || user.username)}</strong><span class="row-subtitle">${escapeHTML(user.username)}</span></td><td>${escapeHTML(user.role_name || "未分配")}</td><td><span class="badge ${user.state === 1 ? "badge-green" : "badge-gray"}">${user.state === 1 ? "启用" : "停用"}</span></td><td>${formatDate(user.created_on)}</td><td class="row-actions">${user.id === currentPrincipal.id ? '<span class="row-subtitle">当前账号</span>' : `<button class="table-action" data-edit-user="${user.id}">编辑</button>`}</td></tr>`).join("");
    $("#user-count").textContent = `共 ${users.length} 个用户`;
    host.querySelectorAll("[data-edit-user]").forEach((button) => button.addEventListener("click", () => {
      const user = users.find((item) => item.id === Number(button.dataset.editUser));
      if (user) openPlatformUser(user);
    }));
  } catch (error) {
    $("#users-error").textContent = `用户列表加载失败：${error.message}`;
  }
}

function openPlatformUser(user = null) {
  const form = $("#platform-user-form");
  form.reset();
  form.elements.id.value = user?.id || "";
  form.elements.username.value = user?.username || "";
  form.elements.username.readOnly = Boolean(user);
  form.elements.display_name.value = user?.display_name || "";
  form.elements.role_id.value = user?.role_id || form.elements.role_id.options[0]?.value || "";
  form.elements.state.value = String(user?.state ?? 1);
  form.elements.password.required = !user;
  form.elements.password.value = "";
  $("#platform-user-heading").textContent = user ? "编辑平台用户" : "新建平台用户";
  $("#platform-user-error").textContent = "";
  $("#platform-user-dialog").showModal();
}

async function savePlatformUser(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const values = new FormData(form);
  const id = values.get("id");
  const payload = {
    username: String(values.get("username")).trim(),
    display_name: String(values.get("display_name")).trim(),
    role_id: Number(values.get("role_id")),
    state: Number(values.get("state")),
  };
  if (values.get("password")) payload.password = String(values.get("password"));
  try {
    await platformRequest(id ? `/users/${id}` : "/users", {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    $("#platform-user-dialog").close();
    notify(id ? "平台用户已更新" : "平台用户已创建");
    await renderPlatformUsers();
  } catch (error) {
    $("#platform-user-error").textContent = error.message;
  }
}

async function loadPlatformRoles(host) {
  host.innerHTML = `${platformPageHeader("ACCESS CONTROL", "角色管理", "为不同工作职责配置最小必要权限。")}<div class="platform-error" id="roles-error"></div><div class="platform-grid role-layout"><section class="panel platform-panel"><div class="platform-panel-heading"><div><h2>角色列表</h2><p class="muted">点击角色可编辑权限。</p></div></div><div id="platform-role-list"></div></section><section class="panel platform-panel"><div class="platform-panel-heading"><div><h2 id="role-form-title">创建角色</h2><p class="muted">内置超级管理员角色不可修改或删除。</p></div></div><form id="platform-role-form"><input type="hidden" name="id"><label>角色名称<input name="name" required maxlength="80"></label><label>角色说明<input name="description" maxlength="255"></label><fieldset class="permission-fieldset"><legend>功能权限</legend><div id="permission-options" class="permission-options"></div></fieldset><p class="error" id="platform-role-error"></p><div class="dialog-actions"><button class="button outline" type="reset" id="role-reset">清空</button><button class="button primary" type="submit">保存角色</button></div></form></section></div>`;
  const form = host.querySelector("#platform-role-form");
  form.addEventListener("reset", () => {
    setTimeout(() => {
      form.elements.id.value = "";
      $("#role-form-title").textContent = "创建角色";
      form.querySelectorAll("[name=permissions]").forEach((input) => { input.checked = false; input.disabled = false; });
    });
  });
  form.addEventListener("submit", savePlatformRole);
  try {
    const result = await platformRequest("/roles");
    host.querySelector("#permission-options").innerHTML = result.available_permissions.map((permission) => `<label class="permission-option"><input type="checkbox" name="permissions" value="${escapeHTML(permission)}"><span><strong>${escapeHTML(permissionLabels[permission] || permission)}</strong><small>${escapeHTML(permission)}</small></span></label>`).join("");
    const list = host.querySelector("#platform-role-list");
    list.innerHTML = result.list.map((role) => `<article class="role-card"><div><strong>${escapeHTML(role.name)}</strong><p>${escapeHTML(role.description)}</p><div class="role-tags">${role.permissions.map((permission) => `<span>${escapeHTML(permissionLabels[permission] || permission)}</span>`).join("")}</div></div><div class="role-actions"><button class="table-action" data-edit-role="${role.id}">编辑</button>${role.permissions.includes("*") ? "" : `<button class="table-action danger-text" data-delete-role="${role.id}">删除</button>`}</div></article>`).join("") || `<div class="empty">暂无角色，请先创建角色。</div>`;
    list.querySelectorAll("[data-edit-role]").forEach((button) => button.addEventListener("click", () => {
      const role = result.list.find((item) => item.id === Number(button.dataset.editRole));
      form.elements.id.value = role.id;
      form.elements.name.value = role.name;
      form.elements.description.value = role.description;
      form.querySelectorAll("[name=permissions]").forEach((input) => { input.checked = role.permissions.includes(input.value); });
      form.querySelectorAll("[name=permissions]").forEach((input) => { input.disabled = role.permissions.includes("*"); });
      $("#role-form-title").textContent = "编辑角色";
    }));
    list.querySelectorAll("[data-delete-role]").forEach((button) => button.addEventListener("click", () => deletePlatformRole(button.dataset.deleteRole)));
  } catch (error) {
    host.querySelector("#roles-error").textContent = `角色列表加载失败：${error.message}`;
  }
}

async function savePlatformRole(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const values = new FormData(form);
  const permissions = [...form.querySelectorAll("[name=permissions]:checked")].map((input) => input.value);
  const id = values.get("id");
  const payload = {
    name: String(values.get("name")).trim(),
    description: String(values.get("description")).trim(),
    permissions,
  };
  try {
    await platformRequest(id ? `/roles/${id}` : "/roles", {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    notify("角色已保存");
    await loadPlatformRoles($("#page-roles"));
  } catch (error) {
    $("#platform-role-error").textContent = error.message;
  }
}

async function deletePlatformRole(id) {
  if (!window.confirm("确定删除该角色吗？已分配用户的角色需要先调整。")) return;
  try {
    await platformRequest(`/roles/${id}`, { method: "DELETE" });
    notify("角色已删除");
    await loadPlatformRoles($("#page-roles"));
  } catch (error) {
    $("#roles-error").textContent = error.message;
  }
}

async function loadPlatformSettings(host) {
  host.innerHTML = `${platformPageHeader("PLATFORM SETTINGS", "系统配置", "维护公开站点信息和平台运行开关。")}<div class="platform-error" id="settings-error"></div><section class="panel platform-panel settings-panel"><form id="platform-settings-form"><div class="platform-section-title"><h2>站点基础信息</h2><p>以下信息用于站点展示，不包含数据库凭据或密钥。</p></div><div class="platform-form-grid"><label>站点名称<input name="site_name" maxlength="120"></label><label>站点网址<input name="site_url" type="url" placeholder="https://blog.example.com"></label><label class="wide-field">站点描述<textarea name="site_description" rows="3" maxlength="500"></textarea></label><label>默认时区<input name="timezone" maxlength="80" placeholder="Asia/Shanghai"></label><label>联系邮箱<input name="support_email" type="email" maxlength="254"></label></div><div class="platform-section-title separated"><h2>平台运行策略</h2><p>变更会立即保存至数据库。</p></div><div class="platform-form-grid"><label>开放注册<select name="allow_registration"><option value="false">关闭</option><option value="true">开启</option></select></label><label>维护模式<select name="maintenance_mode"><option value="false">关闭</option><option value="true">开启</option></select></label></div><div class="dialog-actions"><button class="button primary" type="submit">保存系统配置</button></div></form></section>`;
  const form = host.querySelector("#platform-settings-form");
  try {
    const result = await platformRequest("/settings");
    Object.entries(result.settings || {}).forEach(([key, value]) => {
      if (form.elements[key]) form.elements[key].value = value;
    });
  } catch (error) {
    host.querySelector("#settings-error").textContent = `配置加载失败：${error.message}`;
  }
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = Object.fromEntries(new FormData(form).entries());
    try {
      await platformRequest("/settings", { method: "PUT", body: JSON.stringify(payload) });
      notify("系统配置已保存");
    } catch (error) {
      $("#settings-error").textContent = `保存配置失败：${error.message}`;
    }
  });
}

async function loadPlatformSites(host) {
  host.innerHTML = `${platformPageHeader("SITE MANAGEMENT", "站点管理", "登记和维护由当前平台管理的博客站点。")}<div class="platform-error" id="sites-error"></div><section class="panel platform-panel"><form id="site-form" class="site-form"><label>站点名称<input name="name" required maxlength="120" placeholder="例如：纸间博客"></label><label>域名<input name="domain" required maxlength="255" placeholder="blog.example.com"></label><label>站点说明<input name="description" maxlength="255" placeholder="简短描述"></label><label>状态<select name="state"><option value="1">启用</option><option value="0">停用</option></select></label><button class="button primary" type="submit">＋ 添加站点</button></form><div class="table-wrap"><table><thead><tr><th>站点</th><th>域名</th><th>说明</th><th>状态</th><th>添加时间</th><th class="actions-head">操作</th></tr></thead><tbody id="platform-sites"></tbody></table></div></section>`;
  host.querySelector("#site-form").addEventListener("submit", createPlatformSite);
  await renderPlatformSites();
}

async function renderPlatformSites() {
  const body = $("#platform-sites");
  if (!body) return;
  try {
    const result = await platformRequest("/sites");
    body.innerHTML = (result.list || []).map((site) => `<tr><td><strong class="row-title">${escapeHTML(site.name)}</strong></td><td>${escapeHTML(site.domain)}</td><td>${escapeHTML(site.description || "—")}</td><td><span class="badge ${site.state === 1 ? "badge-green" : "badge-gray"}">${site.state === 1 ? "运行中" : "已停用"}</span></td><td>${formatDate(site.created_on)}</td><td class="row-actions"><button class="table-action" data-toggle-site="${site.id}" data-state="${site.state}">${site.state === 1 ? "停用" : "启用"}</button><button class="table-action danger-text" data-delete-site="${site.id}">删除</button></td></tr>`).join("") || `<tr><td colspan="6"><div class="empty">暂无站点，请添加站点信息。</div></td></tr>`;
    body.querySelectorAll("[data-toggle-site]").forEach((button) => button.addEventListener("click", () => updatePlatformSiteState(button.dataset.toggleSite, Number(button.dataset.state) ? 0 : 1)));
    body.querySelectorAll("[data-delete-site]").forEach((button) => button.addEventListener("click", () => deletePlatformSite(button.dataset.deleteSite)));
  } catch (error) {
    $("#sites-error").textContent = `站点加载失败：${error.message}`;
  }
}

async function createPlatformSite(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const payload = Object.fromEntries(new FormData(form).entries());
  payload.state = Number(payload.state);
  try {
    await platformRequest("/sites", { method: "POST", body: JSON.stringify(payload) });
    form.reset();
    notify("站点已添加");
    await renderPlatformSites();
  } catch (error) {
    $("#sites-error").textContent = `添加站点失败：${error.message}`;
  }
}

async function updatePlatformSiteState(id, state) {
  try {
    const result = await platformRequest("/sites");
    const site = result.list.find((item) => item.id === Number(id));
    if (!site) throw new Error("站点不存在");
    await platformRequest(`/sites/${id}`, {
      method: "PUT",
      body: JSON.stringify({ name: site.name, domain: site.domain, description: site.description, state }),
    });
    await renderPlatformSites();
  } catch (error) {
    $("#sites-error").textContent = `更新站点失败：${error.message}`;
  }
}

async function deletePlatformSite(id) {
  if (!window.confirm("确定删除该站点记录吗？")) return;
  try {
    await platformRequest(`/sites/${id}`, { method: "DELETE" });
    notify("站点记录已删除");
    await renderPlatformSites();
  } catch (error) {
    $("#sites-error").textContent = `删除站点失败：${error.message}`;
  }
}

async function loadPlatformAudit(host) {
  host.innerHTML = `${platformPageHeader("AUDIT TRAIL", "操作日志", "追踪平台用户对内容、账号和系统设置的管理行为。")}<div class="platform-error" id="audit-error"></div><section class="panel"><div class="toolbar"><label class="search-field"><span>⌕</span><input id="audit-user-filter" placeholder="按操作账号筛选"></label><label class="search-field"><span>⌕</span><input id="audit-action-filter" placeholder="按操作内容筛选"></label><button id="audit-search" class="button outline">筛选</button></div><div class="table-wrap"><table><thead><tr><th>操作人</th><th>操作内容</th><th>资源</th><th>方式</th><th>结果</th><th>IP 地址</th><th>时间</th></tr></thead><tbody id="platform-audit-rows"></tbody></table></div><div class="pagination"><span id="audit-count"></span><div><button class="button outline small-button" id="audit-prev">上一页</button><button class="button outline small-button" id="audit-next">下一页</button></div></div></section>`;
  host.querySelector("#audit-search").addEventListener("click", () => loadAuditRows(1));
  host.querySelector("#audit-prev").addEventListener("click", () => loadAuditRows(Math.max(1, (window.auditPage || 1) - 1)));
  host.querySelector("#audit-next").addEventListener("click", () => loadAuditRows((window.auditPage || 1) + 1));
  await loadAuditRows(1);
}

async function loadAuditRows(page) {
  const rows = $("#platform-audit-rows");
  if (!rows) return;
  const query = new URLSearchParams({ page: String(page), page_size: "20" });
  const username = $("#audit-user-filter").value.trim();
  const action = $("#audit-action-filter").value.trim();
  if (username) query.set("username", username);
  if (action) query.set("action", action);
  try {
    const result = await platformRequest(`/audit-logs?${query}`);
    window.auditPage = page;
    renderAuditRows(rows, result.list || []);
    const total = result.pager?.total_rows || 0;
    $("#audit-count").textContent = `共 ${total} 条 · 第 ${page} 页`;
    $("#audit-prev").disabled = page <= 1;
    $("#audit-next").disabled = page * 20 >= total;
  } catch (error) {
    $("#audit-error").textContent = `操作日志加载失败：${error.message}`;
  }
}

function renderAuditRows(host, logs) {
  if (!host) return;
  host.innerHTML = logs.map((log) => `<tr><td>${escapeHTML(log.username)}</td><td>${escapeHTML(log.action)}</td><td>${escapeHTML(log.resource)}</td><td>${escapeHTML(log.method)}</td><td><span class="badge ${log.status < 400 ? "badge-green" : "badge-gray"}">${log.status < 400 ? "成功" : `失败 ${log.status}`}</span></td><td>${escapeHTML(log.remote_ip)}</td><td>${formatDate(log.created_on)}</td></tr>`).join("") || `<tr><td colspan="7"><div class="empty">暂无操作记录。</div></td></tr>`;
}

$("#login-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  $("#login-error").textContent = "";
  const form = event.currentTarget;
  try {
    const result = await request("/api/v1/admin/login", {
      method: "POST",
      body: JSON.stringify({ username: form.elements.username.value, password: form.elements.password.value }),
    });
    form.elements.password.value = "";
    const principal = await request("/api/v1/admin/me");
    showAdmin(result.username, principal);
  } catch (error) {
    $("#login-error").textContent = error.message;
  }
});

$("#logout").addEventListener("click", async () => {
  try {
    await request("/api/v1/admin/logout", { method: "POST" });
    showLogin();
  } catch (error) {
    showError(error);
  }
});

document.querySelectorAll(".nav-item").forEach((button) => {
  button.addEventListener("click", () => showPage(button.dataset.page));
});
document.querySelectorAll("[data-go]").forEach((button) => {
  button.addEventListener("click", () => showPage(button.dataset.go));
});
document.querySelectorAll("[data-close]").forEach((button) => {
  button.addEventListener("click", () => $(`#${button.dataset.close}`).close());
});
$("#new-article").addEventListener("click", async () => {
  try {
    await loadTags();
    if (!tags.some((tag) => tag.state === 1)) {
      notify("请先创建或启用一个标签");
      return;
    }
    openArticleDialog();
  } catch (error) { showError(error); }
});
$("#article-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const id = form.elements.id.value;
  const payload = {
    title: form.elements.title.value,
    desc: form.elements.desc.value,
    tag_id: Number(form.elements.tag_id.value),
    state: Number(form.elements.state.value),
    cover_image_url: form.elements.cover_image_url.value,
    content: form.elements.content.value,
    ...(id ? { modified_by: currentUser } : { created_by: currentUser }),
  };
  try {
    await request(id ? `/api/v1/articles/${id}` : "/api/v1/articles", {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    if (id && Number(form.dataset.originalState) !== payload.state) {
      await request(`/api/v1/articles/${id}/state`, {
        method: "PATCH",
        body: JSON.stringify({ state: payload.state, modified_by: currentUser }),
      });
    }
    $("#article-dialog").close();
    notify("文章已保存");
    await Promise.all([loadArticles(), loadDashboard()]);
  } catch (error) { $("#article-form-error").textContent = error.message; }
});
$("#new-tag").addEventListener("click", () => openTagDialog());
$("#tag-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  try {
    await saveTag(
      { name: form.elements.name.value.trim(), state: form.elements.state.value },
      form.elements.id.value ? "PUT" : "POST",
      form.elements.id.value ? `/api/v1/tags/${form.elements.id.value}` : "/api/v1/tags",
    );
  } catch (error) { $("#tag-form-error").textContent = error.message; }
});
$("#search-articles").addEventListener("click", () => { articlePage = 1; loadArticles().catch(showError); });
$("#article-search").addEventListener("keydown", (event) => {
  if (event.key === "Enter") { event.preventDefault(); articlePage = 1; loadArticles().catch(showError); }
});
$("#article-state").addEventListener("change", () => { articlePage = 1; loadArticles().catch(showError); });
$("#search-tags").addEventListener("click", () => { tagPage = 1; loadTagsPage().catch(showError); });
$("#tag-search").addEventListener("keydown", (event) => {
  if (event.key === "Enter") { event.preventDefault(); tagPage = 1; loadTagsPage().catch(showError); }
});
$("#tag-state").addEventListener("change", () => { tagPage = 1; loadTagsPage().catch(showError); });
$("#articles-prev").addEventListener("click", () => { articlePage -= 1; loadArticles().catch(showError); });
$("#articles-next").addEventListener("click", () => { articlePage += 1; loadArticles().catch(showError); });
$("#tags-prev").addEventListener("click", () => { tagPage -= 1; loadTagsPage().catch(showError); });
$("#tags-next").addEventListener("click", () => { tagPage += 1; loadTagsPage().catch(showError); });

request("/api/v1/admin/me")
  .then((user) => showAdmin(user.username, user))
  .catch(() => showLogin());
