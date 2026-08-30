// 应用壳：hash 路由 + 左侧导航 + 视图分发
const App = {
  user: null, // {id, username, role}
  version: null, // 服务端版本（来自 /healthz，用于确认部署实例代码版本）
  _inited: false,

  async init() {
    // 幂等保护：防止 DOMContentLoaded 与手动调用重复初始化（重复注册 hashchange 会导致路由双渲染）
    if (this._inited) return;
    this._inited = true;
    window.addEventListener('hashchange', () => this.route());
    // 非阻塞获取服务端版本（登录页也显示）
    fetch('/healthz')
      .then((r) => r.json())
      .then((d) => {
        this.version = d.data && d.data.version ? d.data.version : null;
        const el = $('#side-ver');
        if (el) el.textContent = this.version || ''; // version 自带 v 前缀（如 v0.14）
      })
      .catch(() => { /* 版本获取失败不影响使用 */ });
    if (API.token) {
      try {
        this.user = await API.get('/auth/me');
      } catch (e) {
        // token 失效：API 已跳转登录页
      }
    }
    this.route();
  },

  isAdmin() { return !!this.user && this.user.role === 'admin'; },

  async route() {
    const hash = location.hash || '#/dashboard';
    const parts = hash.replace(/^#\//, '').split('/').filter(Boolean);
    const root = $('#app');

    if (!this.user) {
      if (parts[0] !== 'login') { location.hash = '#/login'; return; }
      root.innerHTML = views.login();
      views.bindLogin(root);
      return;
    }
    if (parts[0] === 'login') { location.hash = '#/dashboard'; return; }

    // 渲染外壳：左侧毛玻璃导航 + 主区
    root.innerHTML = `
      <div class="app">
        <aside class="sidebar">
          <a class="brand" href="#/dashboard"><span class="brand-logo"></span><span>数据中心</span></a>
          <nav class="side-nav">
            <a href="#/dashboard" class="nav-link" data-nav="dashboard">📊 仪表盘</a>
            <a href="#/query" class="nav-link" data-nav="query" id="nav-query">🔍 数据查询</a>
            <a href="#/help" class="nav-link" data-nav="help" id="nav-help">📖 DQL 语法</a>
            <a href="#/collections" class="nav-link" data-nav="collections" id="nav-collections">🗂 集合管理</a>
            <a href="#/scrape-tasks" class="nav-link" data-nav="scrape-tasks" id="nav-scrape">⚙ 刮削管理</a>
            <a href="#/users" class="nav-link" data-nav="users" id="nav-users" style="${this.isAdmin() ? '' : 'display:none'}">👥 权限管理</a>
            <a href="#/profile" class="nav-link" data-nav="profile" id="nav-profile">👤 个人设置</a>
          </nav>
          <div class="side-user">
            <div class="user-line">
              <span class="badge ${this.isAdmin() ? 'badge-blue' : 'badge-gray'}">${this.isAdmin() ? '管理员' : '用户'}</span>
              <span class="user-name">${esc(this.user.username)}</span>
            </div>
            <button class="btn btn-ghost btn-sm" id="btn-logout" style="justify-content:center">退出登录</button>
          </div>
          <div class="side-ver" id="side-ver">${this.version || ''}</div>
        </aside>
        <main class="main" id="main"></main>
      </div>`;
    // 退出：清空 token 与当前用户，路由回登录页
    $('#btn-logout').onclick = () => {
      API.logout();
      App.user = null;
    };
    const navEl = $(`.nav-link[data-nav="${parts[0] === 'items' ? 'collections' : parts[0]}"]`);
    if (navEl) navEl.classList.add('active');

    try {
      switch (parts[0]) {
        case 'users':
          if (!this.isAdmin()) { location.hash = '#/dashboard'; return; }
          await views.users($('#main'));
          break;
        case 'items':
          if (parts[1]) await views.itemDetail($('#main'), parts[1]);
          else { location.hash = '#/collections'; }
          break;
        case 'collections':
          if (parts[1]) await views.collectionDetail($('#main'), parts[1]);
          else await views.collections($('#main'));
          break;
        case 'scrape-tasks':
          await views.scrapeTasks($('#main'));
          break;
        case 'query':
          await views.query($('#main'));
          break;
        case 'help':
          await views.dqlHelp($('#main'));
          break;
        case 'profile':
          await views.profile($('#main'));
          break;
        default:
          await views.dashboard($('#main'));
      }
    } catch (e) {
      toast(e.message);
    }
  },
};

// ---------- 通用工具 ----------
const $ = (sel, root) => (root || document).querySelector(sel);
const $$ = (sel, root) => Array.from((root || document).querySelectorAll(sel));

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
}

// ==================== 全局 Toast 提示（v0.13：UI 提示整体重构） ====================
// 所有页面/弹窗的成功、失败、信息提示统一走右下角毛玻璃 Toast：
// 自动消失（成功 3.2s / 错误 5.2s）、点击立即关闭、悬浮在弹窗之上。
let toastSeq = 0;
function toast(text, kind) {
  if (!text) return;
  let box = document.getElementById('toast-container');
  if (!box) {
    box = document.createElement('div');
    box.id = 'toast-container';
    box.setAttribute('aria-live', 'polite');
    document.body.appendChild(box);
  }
  const k = kind === 'success' ? 'success' : kind === 'info' ? 'info' : 'error';
  const el = document.createElement('div');
  el.className = `toast toast-${k}`;
  el.innerHTML = `<span class="toast-ico">${k === 'success' ? '✓' : k === 'info' ? 'ℹ' : '✕'}</span><span class="toast-text">${esc(text)}</span>`;
  el.onclick = () => el.remove();
  box.appendChild(el);
  const id = ++toastSeq;
  const ms = k === 'success' ? 3200 : 5200;
  setTimeout(() => { const t = document.getElementById('toast-' + id); if (t) t.classList.add('toast-out'); }, ms);
  setTimeout(() => { const t = document.getElementById('toast-' + id); if (t) t.remove(); }, ms + 350);
  el.id = 'toast-' + id;
}

// 统一消息入口：历史调用（页面顶部/卡片内 msg 横幅）全部重定向为全局 Toast
function setMsg(_el, text, kind) {
  toast(text, kind);
}

function fmtTime(s) {
  if (!s) return '-';
  const d = new Date(s);
  return isNaN(d) ? String(s) : d.toLocaleString('zh-CN', { hour12: false });
}

function tagValueHtml(v) {
  if (v === null || v === undefined) return '-';
  if (typeof v === 'object') return `<code class="json">${esc(JSON.stringify(v))}</code>`;
  return esc(String(v));
}

// 复制文本到剪贴板（含降级方案）
function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).catch(() => { /* 忽略 */ });
    return;
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand('copy'); } catch (e) { /* 忽略 */ }
  ta.remove();
}

function openModal(title, body) {
  const mask = document.createElement('div');
  mask.className = 'modal-mask';
  mask.innerHTML = `
    <div class="modal">
      <div class="modal-title">${esc(title)}</div>
      <div class="modal-body"></div>
      <div class="modal-foot"><button class="btn" data-act="cancel">取消</button></div>
    </div>`;
  mask.querySelector('.modal-body').appendChild(body);
  mask.querySelector('[data-act="cancel"]').onclick = () => mask.remove();
  mask.addEventListener('click', (e) => { if (e.target === mask) mask.remove(); });
  document.body.appendChild(mask);
  return mask;
}

function confirmAction(msg) { return window.confirm(msg); }

const SCRAPE_BADGE = {
  none: ['badge-gray', '未刮削'], pending: ['badge-orange', '刮削中'], success: ['badge-green', '已刮削'], failed: ['badge-red', '失败'],
};
const SOURCE_BADGE = {
  manual: ['badge-blue', '手动'], scrape: ['badge-green', '刮削'], mixed: ['badge-orange', '混合'],
};
function statusBadge(map, key) {
  const [cls, label] = map[key] || ['badge-gray', String(key)];
  return `<span class="badge ${cls}">${label}</span>`;
}
function roleBadge(role) {
  if (role === 'collection_admin') return '<span class="badge badge-blue">集合管理员</span>';
  if (role === 'operator') return '<span class="badge badge-green">操作工</span>';
  return '<span class="badge badge-gray">非成员</span>';
}

// ---------- 标签类型相关 ----------
const TAG_TYPES = ['string', 'int', 'float', 'bool', 'date', 'enum', 'array', 'object'];
const BASE_TYPES = ['string', 'int', 'float', 'bool', 'date'];

// 按标签定义生成输入控件（用于添加/编辑数据项）
function tagInputHtml(tag, value) {
  const name = 'tag_' + tag.name;
  const v = value !== undefined && value !== null ? value : '';
  const ph = 'JSON，如 ["a","b"] 或 {"k":"v"}';
  switch (tag.type) {
    case 'enum': {
      const opts = (tag.enum_values || []).map((e) =>
        `<option value="${esc(e)}" ${String(e) === String(v) ? 'selected' : ''}>${esc(e)}</option>`).join('');
      return `<select name="${name}" class="input">${opts}</select>`;
    }
    case 'bool':
      return `<select name="${name}" class="input">
        <option value="" ${v === '' ? 'selected' : ''}>—</option>
        <option value="true" ${v === true || v === 'true' ? 'selected' : ''}>true</option>
        <option value="false" ${v === false || v === 'false' ? 'selected' : ''}>false</option>
      </select>`;
    case 'int':
    case 'float':
      return `<input name="${name}" class="input" type="number" step="${tag.type === 'int' ? '1' : 'any'}" value="${esc(v)}">`;
    case 'date': {
      const dv = typeof v === 'string' && v.length >= 16 ? v.slice(0, 16) : '';
      return `<input name="${name}" class="input" type="datetime-local" value="${esc(dv)}">`;
    }
    case 'array':
    case 'object':
      return `<input name="${name}" class="input" placeholder="${ph}" value="${esc(typeof v === 'object' ? JSON.stringify(v) : v)}">`;
    default:
      return `<input name="${name}" class="input" value="${esc(v)}">`;
  }
}

// 从表单/任意容器收集标签值（按定义做基础类型转换）。
// 通过 [name^="tag_"] 查找，兼容 <form> 与普通 div 容器（弹窗）。
function collectTagValues(root, schema) {
  const tags = {};
  const schemaByName = {};
  for (const tag of schema) schemaByName[tag.name] = tag;
  const els = root.querySelectorAll('[name^="tag_"]');
  for (const el of els) {
    const name = el.getAttribute('name').slice('tag_'.length);
    const tag = schemaByName[name];
    if (!tag) continue;
    const raw = String(el.value).trim();
    if (raw === '') continue;
    switch (tag.type) {
      case 'int': {
        const n = parseInt(raw, 10);
        if (isNaN(n)) throw new Error('标签「' + tag.name + '」应为整数');
        tags[tag.name] = n;
        break;
      }
      case 'float': {
        const n = parseFloat(raw);
        if (isNaN(n)) throw new Error('标签「' + tag.name + '」应为数值');
        tags[tag.name] = n;
        break;
      }
      case 'bool': tags[tag.name] = raw === 'true'; break;
      case 'date': tags[tag.name] = new Date(raw).toISOString(); break;
      case 'array':
      case 'object':
        try { tags[tag.name] = JSON.parse(raw); } catch (e) { throw new Error('标签「' + tag.name + '」需要合法 JSON'); }
        break;
      default: tags[tag.name] = raw;
    }
  }
  return tags;
}

// 标签定义表单（新增/编辑标签用）
function tagDefFormHtml() {
  const typeOpts = TAG_TYPES.map((t) => `<option value="${t}">${t}</option>`).join('');
  const baseOpts = BASE_TYPES.map((t) => `<option value="${t}">${t}</option>`).join('');
  return `
    <label class="field"><span>标签名</span><input class="input" id="td-name" placeholder="如 model_name"></label>
    <label class="field"><span>类型</span><select class="input" id="td-type">${typeOpts}</select></label>
    <div id="td-enum-wrap" style="display:none">
      <label class="field"><span>枚举值（逗号分隔）</span><input class="input" id="td-enum" placeholder="dev,test,prod"></label>
    </div>
    <div id="td-array-wrap" style="display:none">
      <label class="field"><span>元素类型</span><select class="input" id="td-element">${baseOpts}</select></label>
    </div>
    <div id="td-object-wrap" style="display:none">
      <label class="field"><span>子字段（JSON 数组，可递归）</span><textarea class="input" id="td-fields" placeholder='[{"name":"version","type":"string","required":true}]'></textarea></label>
    </div>
    <label class="check-line"><input type="checkbox" id="td-required"> 必填标签</label>`;
}

// 解析标签定义表单
function readTagDefForm() {
  const def = {
    name: $('#td-name').value.trim(),
    type: $('#td-type').value,
    required: $('#td-required').checked,
  };
  if (!def.name) throw new Error('标签名不能为空');
  if (def.type === 'enum') {
    def.enum_values = $('#td-enum').value.split(',').map((s) => s.trim()).filter(Boolean);
    if (!def.enum_values.length) throw new Error('枚举标签需要至少一个枚举值');
  }
  if (def.type === 'array') {
    def.element_type = $('#td-element').value;
  }
  if (def.type === 'object') {
    const raw = $('#td-fields').value.trim();
    if (!raw) throw new Error('object 标签需要子字段定义');
    try { def.fields = JSON.parse(raw); } catch (e) { throw new Error('子字段定义需要合法 JSON'); }
  }
  return def;
}

// 数据项标签摘要（表格内展示前 N 个）
function tagsSummary(tags, n) {
  const keys = Object.keys(tags || {});
  if (!keys.length) return '<span class="badge badge-gray">无标签</span>';
  const shown = keys.slice(0, n);
  const html = shown.map((k) => `<span class="badge badge-gray" style="margin-right:4px">${esc(k)}=${tagValueHtml(tags[k])}</span>`).join('');
  return html + (keys.length > n ? `<span class="badge badge-gray">+${keys.length - n}</span>` : '');
}

document.addEventListener('DOMContentLoaded', () => App.init());
