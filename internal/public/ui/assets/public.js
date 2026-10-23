const root = document.querySelector("#public-app");
const pageSize = 9;
const view = { page: 1, title: "", tagId: "" };
const escapeHTML = (value = "") => String(value).replace(/[&<>"']/g, (char) => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
})[char]);

async function api(path) {
  const response = await fetch(`/api/v1/public${path}`, { credentials: "same-origin" });
  const raw = await response.text();
  let payload = {};
  if (raw) {
    try { payload = JSON.parse(raw); } catch { throw new Error(`服务返回了无法识别的内容（${response.status}）`); }
  }
  if (!response.ok) throw new Error(payload.msg || payload.message || `请求失败（${response.status}）`);
  return payload;
}

function date(value) {
  if (!value) return "纸间编辑部";
  const number = Number(value);
  const parsed = new Date(number > 1e12 ? number : number * 1000);
  return Number.isNaN(parsed.getTime()) ? "纸间编辑部" : new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" }).format(parsed);
}

function brand() {
  return `<a class="brand" href="/" aria-label="纸间首页"><span class="brand-mark">紙</span><span><span class="brand-name">纸间</span><span class="brand-subtitle">WRITE A LITTLE, LIVE A LOT</span></span></a>`;
}

function header() {
  return `<header class="site-header">${brand()}<nav class="site-nav" aria-label="主导航"><a href="/" aria-current="page">推荐</a><a href="/#articles">文章</a><a href="/#topics">专题</a><a href="/#tags">标签</a><a href="/#about">关于</a><form class="search-form" id="search-form"><input id="search" aria-label="搜索文章" placeholder="搜索文章…" value="${escapeHTML(view.title)}"><button aria-label="搜索">⌕</button></form></nav></header>`;
}

function footer() {
  return `<footer class="site-footer"><span>© 2026 纸间　·　用文字记录生活</span></footer>`;
}

function shell(content) {
  return `<div class="wrap">${header()}${content}${footer()}</div>`;
}

function storyCard(article, index, tags) {
  const tag = tags.find((item) => Number(item.id) === Number(article.tag_id));
  const art = ["", "warm", "cool"][index % 3];
  const excerpt = article.desc || (article.content || "").slice(0, 72);
  const cover = safeImage(article.cover_image_url);
  return `<a class="story" href="/?article=${encodeURIComponent(article.id)}"><div class="story-cover ${art}" aria-hidden="true">${cover ? `<img src="${escapeHTML(cover)}" alt="" loading="lazy">` : ""}</div><div class="story-body"><span class="topic">${escapeHTML(tag?.name || "纸间故事")}</span><h3>${escapeHTML(article.title)}</h3><p>${escapeHTML(excerpt)}</p><span class="story-meta">${escapeHTML(article.created_by || "纸间编辑部")}　·　${date(article.created_on)}</span></div></a>`;
}

function safeImage(value) {
  try {
    const url = new URL(value, location.origin);
    return ["http:", "https:"].includes(url.protocol) ? url.href : "";
  } catch {
    return "";
  }
}

function hero() {
  return `<section class="hero"><div class="hero-copy"><span class="kicker">本周精选　·　ISSUE 01</span><h1>重新发现生活里的<br>那些微小灵感</h1><p>从日常出发，收集值得分享的想法与故事。</p><div class="hero-meta">纸间编辑部　·　专题策划</div><a class="button" href="#articles">开始阅读　→</a></div><div class="hero-art" aria-hidden="true"></div></section>`;
}

function pagination(total) {
  const pageCount = Math.ceil(total / pageSize);
  if (pageCount < 2) return "";
  return `<nav class="pagination" aria-label="文章分页"><span>共 ${total} 篇 · 第 ${view.page} / ${pageCount} 页</span><span class="pagination-controls"><button data-page="${Math.max(1, view.page - 1)}" ${view.page <= 1 ? "disabled" : ""}>上一页</button><button class="current">${view.page}</button><button data-page="${Math.min(pageCount, view.page + 1)}" ${view.page >= pageCount ? "disabled" : ""}>下一页</button></span></nav>`;
}

async function renderHome() {
  root.innerHTML = shell(`${hero()}<div id="message"></div><div class="layout"><main id="articles"><div class="section-title"><div><h2>${view.tagId ? "标签文章" : "编辑推荐"}</h2><p>每一篇文章，都是生活留下的小小注脚。</p></div><a href="/#articles">浏览最新　↗</a></div><div class="feed" id="feed"><div class="loading">正在整理新文章…</div></div><div id="pages"></div></main><aside class="sidebar"><section class="side-card" id="tags"><h3>按标签发现</h3><div class="tag-list" id="tag-list"><span class="side-copy">正在加载标签…</span></div></section><section class="side-card" id="topics"><h3>本周专题</h3><p class="side-copy">读者来信 · 秋日阅读清单<br>下一期专题：重新整理生活</p><a class="side-copy" href="/#articles">浏览专题文章　→</a></section><section class="side-card" id="about"><h3>关于纸间</h3><p class="side-copy">在这里分享值得停留的想法与故事。慢一点，也没关系。</p></section><section class="side-card"><h3>最近阅读</h3><ul class="side-links" id="recent"><li class="side-copy">文章发布后会出现在这里。</li></ul></section></aside></div>`);
  document.querySelector("#search-form").addEventListener("submit", (event) => {
    event.preventDefault();
    view.title = document.querySelector("#search").value.trim();
    view.page = 1;
    loadHomeData();
  });
  await loadHomeData();
}

async function loadHomeData() {
  const feed = document.querySelector("#feed");
  const message = document.querySelector("#message");
  if (!feed) return;
  feed.innerHTML = `<div class="loading">正在整理新文章…</div>`;
  try {
    const [tagData, articleData] = await Promise.all([
      api("/tags?page=1&page_size=100"),
      api(`/articles?page=${view.page}&page_size=${pageSize}${view.title ? `&title=${encodeURIComponent(view.title)}` : ""}${view.tagId ? `&tag_id=${encodeURIComponent(view.tagId)}` : ""}`),
    ]);
    const tags = tagData.list || [];
    const articles = articleData.list || [];
    document.querySelector("#tag-list").innerHTML = `<button data-tag="">全部</button>${tags.length ? tags.map((tag) => `<button class="${String(tag.id) === String(view.tagId) ? "selected" : ""}" data-tag="${escapeHTML(tag.id)}">${escapeHTML(tag.name)}</button>`).join("") : `<span class="side-copy">还没有启用的标签。</span>`}`;
    document.querySelectorAll("#tag-list [data-tag]").forEach((button) => button.addEventListener("click", () => {
      view.tagId = button.dataset.tag;
      view.page = 1;
      loadHomeData();
    }));
    feed.innerHTML = articles.length
      ? articles.map((article, index) => storyCard(article, index, tags)).join("")
      : `<div class="empty">暂时没有匹配的文章，试试其他关键词或标签。</div>`;
    document.querySelector("#pages").innerHTML = pagination(articleData.pager?.total_rows || 0);
    document.querySelectorAll("#pages [data-page]").forEach((button) => button.addEventListener("click", () => {
      view.page = Number(button.dataset.page);
      loadHomeData();
    }));
    document.querySelector("#recent").innerHTML = articles.slice(0, 3).map((article) => `<li><a href="/?article=${encodeURIComponent(article.id)}">${escapeHTML(article.title)}</a><small>${date(article.created_on)}</small></li>`).join("") || `<li class="side-copy">文章发布后会出现在这里。</li>`;
    message.innerHTML = "";
  } catch (error) {
    feed.innerHTML = `<div class="empty">文章暂时无法加载。</div>`;
    document.querySelector("#tag-list").innerHTML = `<span class="side-copy">标签暂时无法加载。</span>`;
    message.innerHTML = `<div class="alert">内容加载失败：${escapeHTML(error.message)}。请确认服务端已启动且数据库连接正常，再刷新页面。</div>`;
  }
}

async function renderArticle(id) {
  root.innerHTML = shell(`<main class="article-page"><a class="back" href="/">← 返回推荐</a><div class="loading">正在打开文章…</div></main>`);
  try {
    const article = await api(`/articles/${encodeURIComponent(id)}`);
    const tagsData = await api("/tags?page=1&page_size=100");
    const tag = (tagsData.list || []).find((item) => Number(item.id) === Number(article.tag_id));
    const cover = safeImage(article.cover_image_url);
    document.querySelector(".article-page").innerHTML = `<a class="back" href="/">← 返回推荐</a><span class="topic">${escapeHTML(tag?.name || "纸间故事")}</span><h1>${escapeHTML(article.title)}</h1><p class="article-lead">${escapeHTML(article.desc || "")}</p><div class="article-byline">${escapeHTML(article.created_by || "纸间编辑部")}　·　${date(article.created_on)}</div><div class="article-visual">${cover ? `<img src="${escapeHTML(cover)}" alt="">` : "紙間"}</div><div class="article-copy">${escapeHTML(article.content || "")}</div>`;
  } catch (error) {
    document.querySelector(".article-page").innerHTML = `<a class="back" href="/">← 返回推荐</a><div class="alert">文章无法打开：${escapeHTML(error.message)}</div>`;
  }
}

async function start() {
  const articleId = new URLSearchParams(location.search).get("article");
  if (articleId) await renderArticle(articleId);
  else await renderHome();
}

start().catch((error) => {
  root.innerHTML = `<main class="wrap"><div class="alert">页面加载失败：${escapeHTML(error.message)}</div></main>`;
});
