// 视图层：仪表盘 / 数据查询 / DQL 语法帮助 / 集合管理 / 刮削管理 / 权限管理 / 个人设置 / 数据项详情
const views = { login, bindLogin, dashboard, query, dqlHelp, collections, collectionDetail, itemDetail, scrapeTasks, profile, users };

// DQL 示例语句（「示例」按钮随机填入）
const DQL_EXAMPLES = [
  'model_name = "demo-model"',
  'collection = "光刻模型库" AND age >= 3',
  'age >= 3 AND stage IN ("dev", "test")',
  'name LIKE "demo" OR version = "1.0"',
  'config EXISTS true AND collection != "光刻模型库"',
  '(stage = "dev" OR stage = "test") AND age EXISTS true',
];

// DQL 提示示例（按场景分类，点击自动填入并查询）
const DQL_EXAMPLE_HINTS = [
  { title: '指定集合中按字段精确查询', dql: 'collection = "光刻模型库" AND model_name = "demo-model"' },
  { title: '集合内字段范围查询', dql: 'collection = "光刻模型库" AND age >= 3' },
  { title: '跨全部可访问集合查询字段值', dql: 'model_name = "demo-model"' },
  { title: '枚举 IN + 模糊 LIKE 组合', dql: 'stage IN ("dev", "test") AND model_name LIKE "demo"' },
  { title: '字段存在性 + 排除集合', dql: 'age EXISTS true AND collection != "光刻模型库"' },
  { title: '括号组合（AND 优先于 OR）', dql: '(model_name = "a" OR model_name = "b") AND version = "1.2"' },
  { title: '按关联查询直接子节点', dql: 'parent = "数据项ID" AND stage = "test"' },
  { title: '按关联查询子树内节点', dql: 'ancestor = "数据项ID"' },
  { title: '结果按数值字段排序（系统优化 1.2）', dql: 'collection = "model" AND status = "released" ORDER BY accuracy_rms DESC' },
];

// ==================== 登录 ====================
function login() {
  return `
    <div class="login-wrap">
      <div class="login-card">
        <div class="login-brand"><span class="brand-logo"></span><span>数据中心</span></div>
        <p class="login-sub">数据存储 · 集合管理 · 标签刮削</p>
        <div id="login-msg"></div>
        <label class="field"><span>用户名</span><input id="login-username" class="input" placeholder="请输入用户名" autocomplete="username"></label>
        <label class="field"><span>密码</span><input id="login-password" class="input" type="password" placeholder="请输入密码" autocomplete="current-password"></label>
        <button class="btn btn-primary btn-block" id="btn-login">登 录</button>
        <p class="login-hint">默认管理员：admin / admin123</p>
      </div>
    </div>`;
}

function bindLogin(root) {
  const doLogin = async () => {
    const username = $('#login-username').value.trim();
    const password = $('#login-password').value;
    if (!username || !password) { setMsg($('#login-msg'), '请输入用户名和密码'); return; }
    try {
      const data = await API.post('/auth/login', { username, password });
      API.token = data.token;
      localStorage.setItem('dc_token', data.token);
      App.user = data.user;
      location.hash = '#/dashboard';
    } catch (e) {
      setMsg($('#login-msg'), e.message);
    }
  };
  $('#btn-login').onclick = doLogin;
  $('#login-password').addEventListener('keydown', (e) => { if (e.key === 'Enter') doLogin(); });
}

// ==================== 集合列表 ====================
async function collections(root) {
  let page = 1;
  const pageSize = 12;

  root.innerHTML = `
    <div class="page-head">
      <h1>集合</h1>
      <button class="btn btn-primary" id="btn-new-col" style="${App.isAdmin() ? '' : 'display:none'}">新建集合</button>
    </div>
    <div id="col-msg"></div>
    <div class="grid-cols" id="col-grid"></div>
    <div class="pager" id="col-pager"></div>`;

  const load = async (p) => {
    try {
      const data = await API.get(`/collections?page=${p}&page_size=${pageSize}`);
      page = p;
      const grid = $('#col-grid');
      if (!data.items.length) {
        grid.innerHTML = '<div class="empty" style="grid-column:1/-1">暂无集合</div>';
      } else {
        grid.innerHTML = data.items.map((c) => `
          <div class="col-card" data-id="${c.id}">
            <h3>${esc(c.name)}</h3>
            <div class="desc">${esc(c.description || '暂无描述')}</div>
            <div class="meta">
              <span class="badge badge-blue">${c.members.length} 名成员</span>
              <span>${fmtTime(c.created_at)}</span>
            </div>
          </div>`).join('');
        $$('.col-card', grid).forEach((el) => {
          el.onclick = () => { location.hash = '#/collections/' + el.dataset.id; };
        });
      }
      const totalPage = Math.max(1, Math.ceil(data.total / pageSize));
      $('#col-pager').innerHTML = `
        <button class="btn btn-sm" id="pg-prev" ${page <= 1 ? 'disabled' : ''}>上一页</button>
        <span>第 ${page} / ${totalPage} 页 · 共 ${data.total} 个</span>
        <button class="btn btn-sm" id="pg-next" ${page >= totalPage ? 'disabled' : ''}>下一页</button>`;
      $('#pg-prev').onclick = () => { if (page > 1) load(page - 1); };
      $('#pg-next').onclick = () => { if (page < totalPage) load(page + 1); };
    } catch (e) {
      setMsg($('#col-msg'), e.message);
    }
  };

  $('#btn-new-col').onclick = async () => {
    let users = [];
    try { users = (await API.get('/users?page=1&page_size=200')).items; } catch (e) { /* 忽略 */ }
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="modal-msg"></div>
      <label class="field"><span>集合名称</span><input class="input" id="nc-name" placeholder="如 光刻模型库"></label>
      <label class="field"><span>描述</span><textarea class="input" id="nc-desc" placeholder="可选"></textarea></label>
      <label class="field"><span>初始集合管理员</span><select class="input" id="nc-admin">
        <option value="">请选择用户</option>
        ${users.map((u) => `<option value="${u.id}">${esc(u.username)}</option>`).join('')}
      </select></label>`;
    const mask = openModal('新建集合', body);
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '创建';
    mask.querySelector('.modal-foot').prepend(saveBtn);
    saveBtn.onclick = async () => {
      try {
        const name = $('#nc-name').value.trim();
        const initialAdminId = $('#nc-admin').value;
        if (!name) throw new Error('集合名称不能为空');
        if (!initialAdminId) throw new Error('请选择初始集合管理员');
        await API.post('/collections', { name, description: $('#nc-desc').value.trim(), initial_admin_id: initialAdminId });
        mask.remove();
        load(page);
      } catch (e) {
        setMsg($('#modal-msg'), e.message);
      }
    };
  };

  await load(1);
}

// ==================== 集合详情 ====================
async function collectionDetail(root, id) {
  const col = await API.get('/collections/' + id);
  // 兼容旧数据/缺省字段：后端可能返回 null
  col.tag_schema = col.tag_schema || [];
  col.members = col.members || [];
  const me = App.user;
  const myMember = (col.members || []).find((m) => m.user_id === me.id);
  // admin 拥有最高权限：视为每个集合的集合管理员（显示全部管理操作）
  const myRole = App.isAdmin() ? 'collection_admin' : (myMember ? myMember.role : null);
  const isColAdmin = myRole === 'collection_admin';

  root.innerHTML = `
    <a class="back" href="#/collections">← 返回集合列表</a>
    <div class="page-head">
      <h1>${esc(col.name)} ${roleBadge(myRole)}</h1>
      <div>
        <button class="btn btn-sm" id="btn-refresh">刷新</button>
        ${App.isAdmin() ? '<button class="btn btn-sm btn-danger" id="btn-del-col">删除集合</button>' : ''}
      </div>
    </div>
    <div id="cd-msg"></div>

    <div class="card">
      <div class="card-title">概览 <span class="sub">创建于 ${fmtTime(col.created_at)} · 创建者 ${esc(col.created_by)}</span></div>
      <div class="detail-line"><span class="k">描述</span><span class="v" id="cd-desc">${esc(col.description || '暂无描述')}</span></div>
      <div class="detail-line"><span class="k">刮削脚本</span><span class="v" id="cd-script">${col.scrape_script ? `<code class="json">${esc(col.scrape_script.path)}</code>` : '<span class="badge badge-gray">未配置</span>'}</span></div>
      ${App.isAdmin() || isColAdmin ? `
        <div style="margin-top:14px;display:flex;gap:10px;flex-wrap:wrap">
          ${isColAdmin ? '<button class="btn btn-sm" id="btn-edit-desc">编辑描述</button>' : ''}
          ${App.isAdmin() ? '<button class="btn btn-sm" id="btn-assign-admin">更换集合管理员</button>' : ''}
        </div>` : ''}
      ${isColAdmin ? '<button class="btn btn-sm" id="btn-set-script" style="margin-top:10px">配置刮削脚本</button>' : ''}
    </div>

    <div class="card">
      <div class="card-title">标签定义 <span class="sub">共 ${col.tag_schema.length} 个</span>
        ${isColAdmin ? '<button class="btn btn-sm btn-primary" id="btn-add-tag">添加标签</button>' : ''}
      </div>
      <table class="table" id="tag-table"></table>
      ${isColAdmin ? '<p style="color:var(--text-2);font-size:12.5px;margin-top:8px">修改类型或必填性：先删除再添加；已有数据项按 §5.3/Q8 策略处理。</p>' : ''}
    </div>

    ${isColAdmin ? `
    <div class="card">
      <div class="card-title">删除策略 <span class="sub">数据项删除时的关联处理（默认：均拒绝）</span></div>
      <div class="detail-line"><span class="k">有子节点时</span><span class="v">
        <label class="check-line" style="margin-bottom:4px"><input type="radio" name="dp-children" value="deny"> 拒绝删除（可显式级联）</label>
        <label class="check-line" style="margin-bottom:4px"><input type="radio" name="dp-children" value="cascade"> 自动级联删除子树</label>
        <label class="check-line"><input type="radio" name="dp-children" value="detach"> 自动解除父子边</label>
      </span></div>
      <div class="detail-line"><span class="k">被引用时</span><span class="v">
        <label class="check-line" style="margin-bottom:4px"><input type="radio" name="dp-incoming" value="deny"> 拒绝删除（可强制解除）</label>
        <label class="check-line"><input type="radio" name="dp-incoming" value="detach"> 自动解除引用</label>
      </span></div>
      <div style="margin-top:10px;display:flex;gap:10px">
        <button class="btn btn-sm btn-primary" id="btn-save-policy">保存策略</button>
        <button class="btn btn-sm" id="btn-reset-policy">恢复默认</button>
      </div>
    </div>` : ''}

    <div class="card">
      <div class="card-title">成员 <span class="sub">${col.members.length} 人</span>
        ${isColAdmin ? '<button class="btn btn-sm btn-primary" id="btn-grant">授权操作工</button>' : ''}
      </div>
      <table class="table" id="member-table"></table>
    </div>
    <p style="color:var(--text-2);font-size:12.5px">数据项请前往 <a href="#/query">数据查询</a> 页面管理。</p>`;

  const msg = $('#cd-msg');
  const setMsgBox = (t, k) => setMsg(msg, t, k);

  // 标签定义表
  const renderTags = () => {
    $('#tag-table').innerHTML = `
      <thead><tr><th>标签名</th><th>类型</th><th>必填</th><th>约束</th>${isColAdmin ? '<th></th>' : ''}</tr></thead>
      <tbody>${col.tag_schema.map((t) => `
        <tr>
          <td>${esc(t.name)}</td>
          <td><span class="badge badge-blue">${esc(t.type)}</span></td>
          <td>${t.required ? '是' : '否'}</td>
          <td class="wide">${esc(tagConstraintText(t))}</td>
          ${isColAdmin ? `<td class="actions"><button class="btn btn-sm btn-danger" data-del-tag="${esc(t.name)}">删除</button></td>` : ''}
        </tr>`).join('')}</tbody>`;
    if (isColAdmin) {
      $$('[data-del-tag]', $('#tag-table')).forEach((btn) => {
        btn.onclick = async () => {
          const name = btn.dataset.delTag;
          if (!confirmAction('确定删除标签「' + name + '」？已有数据项中的该标签将被移除。')) return;
          try {
            col.tag_schema = col.tag_schema.filter((t) => t.name !== name);
            await API.put(`/collections/${id}/tags`, { tags: col.tag_schema });
            setMsgBox('标签已删除', 'success');
            renderTags();
          } catch (e) { setMsgBox(e.message); }
        };
      });
    }
  };

  // 成员表
  const renderMembers = async () => {
    let users = [];
    try { users = (await API.get('/users?page=1&page_size=200')).items; } catch (e) { /* 非 admin 无权限，忽略 */ }
    const nameOf = (uid) => { const u = users.find((x) => x.id === uid); return u ? u.username : uid; };
    $('#member-table').innerHTML = `
      <thead><tr><th>用户</th><th>角色</th>${isColAdmin ? '<th></th>' : ''}</tr></thead>
      <tbody>${col.members.map((m) => `
        <tr>
          <td>${esc(nameOf(m.user_id))}</td>
          <td>${roleBadge(m.role)}</td>
          ${isColAdmin ? `<td class="actions">${m.role === 'operator' ? `<button class="btn btn-sm btn-danger" data-rm-member="${m.user_id}">移除</button>` : '<span class="badge badge-gray">不可移除</span>'}</td>` : ''}
        </tr>`).join('')}</tbody>`;
    if (isColAdmin) {
      $$('[data-rm-member]', $('#member-table')).forEach((btn) => {
        btn.onclick = async () => {
          if (!confirmAction('确定移除该操作工？')) return;
          try {
            await API.del(`/collections/${id}/members/${btn.dataset.rmMember}`);
            col.members = col.members.filter((m) => m.user_id !== btn.dataset.rmMember);
            renderMembers();
            setMsgBox('成员已移除', 'success');
          } catch (e) { setMsgBox(e.message); }
        };
      });
    }
  };

  // ---------- 事件绑定 ----------
  $('#btn-refresh').onclick = () => { location.reload(); };

  if (App.isAdmin()) {
    $('#btn-del-col').onclick = async () => {
      if (!confirmAction('确定删除该集合？将级联删除其数据项与刮削任务元数据（不触碰 NFS 文件）。')) return;
      try {
        await API.del('/collections/' + id);
        location.hash = '#/collections';
      } catch (e) { setMsgBox(e.message); }
    };
    $('#btn-edit-desc').onclick = () => {
      const body = document.createElement('div');
      body.innerHTML = '<div id="ed-msg"></div><label class="field"><span>描述</span><textarea class="input" id="ed-desc"></textarea></label>';
      $('#ed-desc', body).value = col.description || ''; // 弹窗未挂载前须在 body 内查找
      const mask = openModal('编辑描述', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '保存';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          await API.patch(`/collections/${id}`, { description: $('#ed-desc').value.trim() });
          col.description = $('#ed-desc').value.trim();
          $('#cd-desc').textContent = col.description || '暂无描述';
          mask.remove();
          setMsgBox('已保存', 'success');
        } catch (e) { setMsg($('#ed-msg'), e.message); }
      };
    };
    $('#btn-assign-admin').onclick = async () => {
      let users = [];
      try { users = (await API.get('/users?page=1&page_size=200')).items; } catch (e) { setMsgBox('获取用户列表失败：' + e.message); return; }
      const body = document.createElement('div');
      body.innerHTML = '<div id="aa-msg"></div><label class="field"><span>新的集合管理员</span><select class="input" id="aa-user"><option value="">请选择用户</option>' +
        users.map((u) => `<option value="${u.id}">${esc(u.username)}</option>`).join('') + '</select></label>';
      const mask = openModal('更换集合管理员', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '更换';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          const uid = $('#aa-user').value;
          if (!uid) throw new Error('请选择用户');
          await API.put(`/collections/${id}/admin`, { user_id: uid });
          mask.remove();
          setMsgBox('集合管理员已更换', 'success');
          location.reload();
        } catch (e) { setMsg($('#aa-msg'), e.message); }
      };
    };
  }

  if (isColAdmin) {
    $('#btn-set-script').onclick = () => {
      const body = document.createElement('div');
      body.innerHTML = `
        <div id="ss-msg"></div>
        <label class="field"><span>刮削脚本绝对路径（NFS 上存在的可执行文件）</span><input class="input" id="ss-path" placeholder="/nfs/scripts/model_scraper.sh"></label>
        <p style="color:var(--text-2);font-size:12.5px">约定：脚本接收一个数据路径入参，stdout 输出 JSON 标签对象；每个集合仅一个脚本。</p>`;
      $('#ss-path', body).value = col.scrape_script ? col.scrape_script.path : ''; // 弹窗未挂载前须在 body 内查找
      const mask = openModal('配置刮削脚本', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '保存';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          const path = $('#ss-path').value.trim();
          if (!path) throw new Error('路径不能为空');
          await API.put(`/collections/${id}/script`, { path });
          col.scrape_script = { path };
          $('#cd-script').innerHTML = `<code class="json">${esc(path)}</code>`;
          mask.remove();
          setMsgBox('刮削脚本已配置', 'success');
        } catch (e) { setMsg($('#ss-msg'), e.message); }
      };
    };
    $('#btn-add-tag').onclick = () => {
      const body = document.createElement('div');
      body.innerHTML = '<div id="at-msg"></div>' + tagDefFormHtml();
      const typeSel = $('#td-type', body);
      typeSel.onchange = () => {
        $('#td-enum-wrap', body).style.display = typeSel.value === 'enum' ? '' : 'none';
        $('#td-array-wrap', body).style.display = typeSel.value === 'array' ? '' : 'none';
        $('#td-object-wrap', body).style.display = typeSel.value === 'object' ? '' : 'none';
      };
      const mask = openModal('添加标签', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '添加';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          const def = readTagDefForm();
          col.tag_schema.push(def);
          await API.put(`/collections/${id}/tags`, { tags: col.tag_schema });
          mask.remove();
          setMsgBox('标签已添加', 'success');
          renderTags();
        } catch (e) { setMsg($('#at-msg'), e.message); }
      };
    };
    $('#btn-grant').onclick = async () => {
      let users = [];
      try { users = (await API.get('/users?page=1&page_size=200')).items; } catch (e) { setMsgBox('获取用户列表失败：' + e.message); return; }
      const memberIds = new Set(col.members.map((m) => m.user_id));
      const candidates = users.filter((u) => !memberIds.has(u.id));
      if (!candidates.length) { setMsgBox('没有可授权的用户（所有用户都已是成员）'); return; }
      const body = document.createElement('div');
      body.innerHTML = '<div id="gr-msg"></div><label class="field"><span>选择用户授权为操作工</span><select class="input" id="gr-user"><option value="">请选择用户</option>' +
        candidates.map((u) => `<option value="${u.id}">${esc(u.username)}</option>`).join('') + '</select></label>';
      const mask = openModal('授权操作工', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '授权';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          const uid = $('#gr-user').value;
          if (!uid) throw new Error('请选择用户');
          await API.post(`/collections/${id}/members`, { user_id: uid });
          col.members.push({ user_id: uid, role: 'operator' });
          mask.remove();
          setMsgBox('授权成功', 'success');
          renderMembers();
        } catch (e) { setMsg($('#gr-msg'), e.message); }
      };
    };
  }

  renderTags();
  await renderMembers();

  // 删除策略配置（集合管理员）
  if (isColAdmin) {
    const policy = col.delete_policy || { children: 'deny', incoming: 'deny' };
    const radio = (name, val) => { const el = document.querySelector(`input[name="${name}"][value="${val}"]`); if (el) el.checked = true; };
    radio('dp-children', policy.children);
    radio('dp-incoming', policy.incoming);
    $('#btn-save-policy').onclick = async () => {
      try {
        const children = document.querySelector('input[name="dp-children"]:checked').value;
        const incoming = document.querySelector('input[name="dp-incoming"]:checked').value;
        await API.put(`/collections/${id}/delete-policy`, { children, incoming });
        col.delete_policy = { children, incoming };
        setMsgBox('删除策略已保存', 'success');
      } catch (e) { setMsgBox(e.message); }
    };
    $('#btn-reset-policy').onclick = async () => {
      try {
        await API.put(`/collections/${id}/delete-policy`, { children: 'deny', incoming: 'deny' });
        col.delete_policy = { children: 'deny', incoming: 'deny' };
        radio('dp-children', 'deny');
        radio('dp-incoming', 'deny');
        setMsgBox('已恢复默认策略（均拒绝）', 'success');
      } catch (e) { setMsgBox(e.message); }
    };
  }
}

function tagConstraintText(t) {
  if (t.type === 'enum') return '枚举值：' + (t.enum_values || []).join(' / ');
  if (t.type === 'array') return '元素类型：' + t.element_type;
  if (t.type === 'object') return '子字段：' + (t.fields || []).map((f) => f.name + (f.required ? '*' : '')).join(', ');
  return '';
}

// ==================== 数据项详情 ====================
async function itemDetail(root, itemId) {
  const item = await API.get('/items/' + itemId);
  const col = await API.get('/collections/' + item.collection_id);
  col.tag_schema = col.tag_schema || [];
  const me = App.user;
  const myMember = (col.members || []).find((m) => m.user_id === me.id);
  // admin 拥有最高权限：视为集合管理员/操作工，可操作数据项（编辑/手动刮削/添加关联）
  const canOperate = !!myMember || App.isAdmin();

  root.innerHTML = `
    <a class="back" href="#/query">← 返回数据查询</a>
    <div class="page-head"><h1>数据项详情</h1>
      <div>
        <button class="btn btn-sm" id="btn-refresh">刷新</button>
        ${canOperate ? '<button class="btn btn-sm" id="btn-edit">编辑</button>' : ''}
        ${canOperate ? '<button class="btn btn-sm btn-primary" id="btn-scrape">手动刮削</button>' : ''}
      </div>
    </div>
    <div id="id-msg"></div>

    <div class="card">
      <div class="card-title">基本信息</div>
      <div class="detail-line"><span class="k">路径</span><span class="v" id="id-path">${esc(item.path)}</span></div>
      <div class="detail-line"><span class="k">标签来源</span><span class="v">${statusBadge(SOURCE_BADGE, item.tag_source)}</span></div>
      <div class="detail-line"><span class="k">刮削状态</span><span class="v">${statusBadge(SCRAPE_BADGE, item.scrape_status)}</span></div>
      <div class="detail-line"><span class="k">上次刮削</span><span class="v">${fmtTime(item.last_scraped_at)}</span></div>
      <div class="detail-line"><span class="k">创建信息</span><span class="v">${fmtTime(item.created_at)}</span></div>
    </div>

    <div class="card">
      <div class="card-title">标签 <span class="sub">${Object.keys(item.tags || {}).length} 个</span></div>
      <table class="table" id="tags-table"></table>
    </div>

    <div class="card">
      <div class="card-title">关联关系
        ${canOperate ? '<button class="btn btn-sm btn-primary" id="btn-add-rel">添加关联</button>' : ''}
        <button class="btn btn-sm" id="btn-tree-view">树视图</button>
      </div>
      <div class="card-title" style="margin-bottom:8px">出边 <span class="sub" id="rel-out-count"></span></div>
      <table class="table" id="rel-out-table"></table>
      <div class="card-title" style="margin-top:14px;margin-bottom:8px">入边 <span class="sub" id="rel-in-count"></span></div>
      <table class="table" id="rel-in-table"></table>
    </div>

    <div class="card">
      <div class="card-title">刮削历史</div>
      <table class="table" id="task-table"></table>
      <div class="pager" id="task-pager"></div>
    </div>`;

  const msg = $('#id-msg');
  const setMsgBox = (t, k) => setMsg(msg, t, k);

  const renderTagsTable = () => {
    const tags = item.tags || {};
    const keys = Object.keys(tags);
    $('#tags-table').innerHTML = keys.length ? `
      <thead><tr><th>标签</th><th>值</th></tr></thead>
      <tbody>${keys.map((k) => `<tr><td>${esc(k)}</td><td>${tagValueHtml(tags[k])}</td></tr>`).join('')}</tbody>`
      : '<tr><td class="empty">暂无标签</td></tr>';
  };

  // ---------- 关联关系 ----------
  const REL_TYPE_BADGE = {
    parent_child: ['badge-blue', '父子'],
    reference: ['badge-green', '引用'],
    call: ['badge-orange', '调用'],
  };
  const relTypeBadge = (t) => { const [c, l] = REL_TYPE_BADGE[t] || ['badge-gray', t]; return `<span class="badge ${c}">${l}</span>`; };
  const relMetaText = (m) => {
    if (!m || !Object.keys(m).length) return '-';
    return `<code class="json">${esc(JSON.stringify(m))}</code>`;
  };

  const loadRels = async () => {
    try {
      const [outD, inD] = await Promise.all([
        API.get(`/items/${itemId}/relations?direction=out&page=1&page_size=20`),
        API.get(`/items/${itemId}/relations?direction=in&page=1&page_size=20`),
      ]);
      $('#rel-out-count').textContent = `共 ${outD.total} 条`;
      $('#rel-in-count').textContent = `共 ${inD.total} 条`;
      const renderRows = (dir, data) => {
        const table = dir === 'out' ? $('#rel-out-table') : $('#rel-in-table');
        if (!data.items.length) { table.innerHTML = '<tr><td class="empty">暂无</td></tr>'; return; }
        table.innerHTML = `
          <thead><tr><th>类型</th><th>${dir === 'out' ? '目标' : '来源'}</th><th>属性</th><th></th></tr></thead>
          <tbody>${data.items.map((r) => `
            <tr>
              <td>${relTypeBadge(r.type)}</td>
              <td class="wide">${esc(r.target.path || r.target.item_id)}<span class="sub" style="margin-left:6px">${esc(r.target.collection_name || '')}</span></td>
              <td>${relMetaText(r.meta)}</td>
              <td class="actions">
                <button class="btn btn-sm" data-rel-jump="${r.target.item_id}">跳转</button>
                ${canOperate ? `<button class="btn btn-sm btn-danger" data-rel-del="${r.id}">删除</button>` : ''}
              </td>
            </tr>`).join('')}</tbody>`;
        $$('[data-rel-jump]', table).forEach((b) => { b.onclick = () => { location.hash = '#/items/' + b.dataset.relJump; }; });
        if (canOperate) {
          $$('[data-rel-del]', table).forEach((b) => {
            b.onclick = async () => {
              if (!confirmAction('确定删除该关联关系？仅删除关系，不影响数据项。')) return;
              try { await API.del('/relations/' + b.dataset.relDel); loadRels(); setMsgBox('关联已删除', 'success'); }
              catch (e) { setMsgBox(e.message); }
            };
          });
        }
      };
      renderRows('out', outD);
      renderRows('in', inD);
    } catch (e) { setMsgBox(e.message); }
  };

  // 添加关联弹窗（支持多选批量，父子快捷字段）
  const openAddRel = () => {
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="ar-msg"></div>
      <label class="field"><span>关联类型</span>
        <label class="check-line" style="margin-bottom:6px"><input type="radio" name="ar-type" value="parent_child" checked> 父子/包含（如 版图 → 层）</label>
        <label class="check-line" style="margin-bottom:6px"><input type="radio" name="ar-type" value="reference"> 引用（如 用例 → model）</label>
        <label class="check-line"><input type="radio" name="ar-type" value="call"> 调用（运行时依赖）</label>
      </label>
      <label class="field"><span>搜索目标数据项（路径包含）</span>
        <div style="display:flex;gap:8px"><input class="input" id="ar-search" placeholder="如 opc_model 或 /nfs/..."><button type="button" class="btn btn-sm" id="ar-search-btn">搜索</button></div>
      </label>
      <div id="ar-results" style="max-height:200px;overflow:auto;border:1px solid rgba(31,38,135,.12);border-radius:8px;margin-bottom:12px"></div>
      <div id="ar-quick" style="display:none">
        <div class="form-row">
          <label class="field"><span>层序 layer_order</span><input class="input" id="ar-order" type="number" step="1"></label>
          <label class="field"><span>角色 role</span><input class="input" id="ar-role" placeholder="如 metal / opc"></label>
        </div>
      </div>
      <label class="field"><span>其他属性（JSON，可选）</span><textarea class="input" id="ar-meta" placeholder='{"param": "--layer 2"}'></textarea></label>`;
    const mask = openModal('添加关联', body);
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '添加';
    mask.querySelector('.modal-foot').prepend(saveBtn);

    const typeSel = () => body.querySelector('input[name="ar-type"]:checked');
    const onTypeChange = () => {
      $('#ar-quick').style.display = typeSel() && typeSel().value === 'parent_child' ? '' : 'none';
    };
    $$('input[name="ar-type"]', body).forEach((r) => { r.onchange = onTypeChange; });
    onTypeChange();

    let selected = [];
    const doSearch = async () => {
      try {
        const kw = $('#ar-search').value.trim();
        const data = await API.get(`/items/search?keyword=${encodeURIComponent(kw)}&page=1&page_size=20`);
        selected = [];
        $('#ar-results').innerHTML = data.items.length ? data.items.map((it) => `
          <label class="list-row" style="cursor:pointer">
            <input type="checkbox" class="ar-check" value="${it.id}" data-path="${esc(it.path)}">
            <div class="row-main"><div class="row-title">${esc(it.path)}</div></div>
          </label>`).join('') : '<div class="empty">无结果</div>';
        $$('.ar-check', $('#ar-results')).forEach((cb) => {
          cb.onchange = () => {
            if (cb.checked) selected.push({ id: cb.value, path: cb.dataset.path });
            else selected = selected.filter((s) => s.id !== cb.value);
          };
        });
      } catch (e) { setMsg($('#ar-msg'), e.message); }
    };
    $('#ar-search-btn').onclick = doSearch;
    $('#ar-search').addEventListener('keydown', (e) => { if (e.key === 'Enter') doSearch(); });

    saveBtn.onclick = async () => {
      try {
        const type = typeSel().value;
        if (!selected.length) throw new Error('请先搜索并勾选目标数据项（可多选批量添加）');
        const meta = {};
        if ($('#ar-meta').value.trim()) {
          try { Object.assign(meta, JSON.parse($('#ar-meta').value)); } catch (e) { throw new Error('属性需要合法 JSON'); }
        }
        if (type === 'parent_child') {
          const order = $('#ar-order').value;
          const role = $('#ar-role').value.trim();
          if (order !== '') meta.layer_order = parseInt(order, 10);
          if (role) meta.role = role;
        }
        const relations = selected.map((s) => ({ type, to_item_id: s.id, meta: Object.keys(meta).length ? meta : undefined }));
        const result = await API.post(`/items/${itemId}/relations/batch`, { relations });
        mask.remove();
        const fail = (result.failed || []).length;
        setMsgBox(fail ? `已添加 ${result.success.length} 条，失败 ${fail} 条（单父/环等约束）` : `已添加 ${result.success.length} 条关联`, 'success');
        loadRels();
      } catch (e) { setMsg($('#ar-msg'), e.message); }
    };
  };

  // 树视图（按深度重新加载）
  const openTree = async () => {
    try {
      const tree = await API.get(`/items/${itemId}/tree?direction=desc&depth=3`);
      const body = document.createElement('div');
      const renderTreeHtml = (node) => {
        if (!node) return '<div class="empty">无层级</div>';
        const children = (node.children || []).map((c) => `<li style="list-style:none">${renderTreeHtml(c)}</li>`).join('');
        return `<div class="tree-node">
          <div class="tree-row" style="display:flex;align-items:center;gap:8px;padding:4px 0">
            <span>${children ? '▼' : '·'}</span>
            <a href="#/items/${node.item.item_id}">${esc(node.item.path)}</a>
            ${node.meta && node.meta.layer_order !== undefined ? `<span class="badge badge-gray">order=${esc(node.meta.layer_order)}</span>` : ''}
          </div>
          ${children ? `<ul style="padding-left:22px">${children}</ul>` : ''}
        </div>`;
      };
      body.innerHTML = `<div id="tree-root">${renderTreeHtml(tree)}</div>`;
      openModal('层级树（父子关系 · 深度 3，点击节点跳转）', body);
    } catch (e) { setMsgBox(e.message); }
  };

  const tasksState = { page: 1, pageSize: 10 };
  const loadTasks = async () => {
    try {
      const data = await API.get(`/items/${itemId}/scrape-tasks?page=${tasksState.page}&page_size=${tasksState.pageSize}`);
      const table = $('#task-table');
      if (!data.items.length) {
        table.innerHTML = '<tr><td class="empty">暂无刮削任务</td></tr>';
      } else {
        table.innerHTML = `
          <thead><tr><th>状态</th><th>触发</th><th>脚本</th><th>错误信息</th><th>时间</th></tr></thead>
          <tbody>${data.items.map((t) => `
            <tr>
              <td>${statusBadge(SCRAPE_BADGE, t.status)}</td>
              <td>${esc(t.trigger_by === 'auto' ? '自动' : t.trigger_by)}</td>
              <td class="wide">${esc(t.script_path)}</td>
              <td class="wide">${esc(t.error || (t.result_tags ? '产出 ' + Object.keys(t.result_tags).length + ' 个标签' : '-'))}</td>
              <td>${fmtTime(t.created_at)}</td>
            </tr>`).join('')}</tbody>`;
      }
      const totalPage = Math.max(1, Math.ceil(data.total / tasksState.pageSize));
      $('#task-pager').innerHTML = `
        <button class="btn btn-sm" id="tp-prev" ${tasksState.page <= 1 ? 'disabled' : ''}>上一页</button>
        <span>第 ${tasksState.page} / ${totalPage} 页</span>
        <button class="btn btn-sm" id="tp-next" ${tasksState.page >= totalPage ? 'disabled' : ''}>下一页</button>`;
      $('#tp-prev').onclick = () => { if (tasksState.page > 1) { tasksState.page--; loadTasks(); } };
      $('#tp-next').onclick = () => { if (tasksState.page < totalPage) { tasksState.page++; loadTasks(); } };
    } catch (e) { setMsgBox(e.message); }
  };

  $('#btn-refresh').onclick = () => { location.reload(); };
  if (canOperate) {
    $('#btn-scrape').onclick = async () => {
      try {
        await API.post(`/items/${itemId}/scrape`);
        setMsgBox('已触发刮削，请稍后刷新查看', 'success');
        loadTasks();
      } catch (e) { setMsgBox(e.message); }
    };
    $('#btn-edit').onclick = () => {
      const body = document.createElement('div');
      body.innerHTML = `
        <div id="ed-msg"></div>
        <label class="field"><span>路径</span><input class="input" id="ed-path"></label>
        <div class="card-title" style="margin-bottom:10px">标签（全量覆盖）</div>
        ${col.tag_schema.map((t) => `
          <label class="field"><span>${esc(t.name)}${t.required ? ' *' : ''}（${esc(t.type)}）</span>${tagInputHtml(t, item.tags ? item.tags[t.name] : undefined)}</label>`).join('')}`;
      $('#ed-path', body).value = item.path; // 弹窗未挂载前须在 body 内查找
      const mask = openModal('编辑数据项', body);
      const saveBtn = document.createElement('button');
      saveBtn.className = 'btn btn-primary';
      saveBtn.textContent = '保存';
      mask.querySelector('.modal-foot').prepend(saveBtn);
      saveBtn.onclick = async () => {
        try {
          const payload = { path: $('#ed-path').value.trim() };
          const tags = collectTagValues(body, col.tag_schema);
          if (Object.keys(tags).length) payload.tags = tags;
          const updated = await API.patch('/items/' + itemId, payload);
          Object.assign(item, updated);
          mask.remove();
          $('#id-path').textContent = item.path;
          setMsgBox('已保存', 'success');
          renderTagsTable();
        } catch (e) { setMsg($('#ed-msg'), e.message); }
      };
    };
  }

  renderTagsTable();
  await loadTasks();
  $('#btn-tree-view').onclick = openTree;
  if (canOperate) $('#btn-add-rel').onclick = openAddRel;
  await loadRels();
}

// ==================== 用户管理（admin） ====================
async function users(root) {
  let page = 1;
  const pageSize = 20;

  root.innerHTML = `
    <div class="page-head">
      <h1>用户管理</h1>
      <button class="btn btn-primary" id="btn-new-user">创建用户</button>
    </div>
    <div id="u-msg"></div>
    <div class="card">
      <table class="table" id="user-table"></table>
      <div class="pager" id="user-pager"></div>
    </div>`;

  const msg = $('#u-msg');
  const setMsgBox = (t, k) => setMsg(msg, t, k);

  const load = async (p) => {
    try {
      const data = await API.get(`/users?page=${p}&page_size=${pageSize}`);
      page = p;
      const table = $('#user-table');
      table.innerHTML = `
        <thead><tr><th>用户名</th><th>角色</th><th>状态</th><th>创建时间</th><th></th></tr></thead>
        <tbody>${data.items.map((u) => `
          <tr>
            <td>${esc(u.username)}</td>
            <td>${u.role === 'admin' ? '<span class="badge badge-blue">管理员</span>' : '<span class="badge badge-gray">用户</span>'}</td>
            <td>${u.status === 'active' ? '<span class="badge badge-green">启用</span>' : '<span class="badge badge-red">禁用</span>'}</td>
            <td>${fmtTime(u.created_at)}</td>
            <td class="actions">
              ${u.username === App.user.username ? '<span class="badge badge-gray">当前账号</span>' : `
                <button class="btn btn-sm" data-user-status="${u.id}" data-status="${u.status}">${u.status === 'active' ? '禁用' : '启用'}</button>
                <button class="btn btn-sm btn-danger" data-user-del="${u.id}">删除</button>`}
            </td>
          </tr>`).join('')}</tbody>`;
      $$('[data-user-status]', table).forEach((b) => {
        b.onclick = async () => {
          const status = b.dataset.status === 'active' ? 'disabled' : 'active';
          try {
            await API.patch('/users/' + b.dataset.userStatus, { status });
            load(page);
          } catch (e) { setMsgBox(e.message); }
        };
      });
      $$('[data-user-del]', table).forEach((b) => {
        b.onclick = async () => {
          if (!confirmAction('确定删除该用户？')) return;
          try { await API.del('/users/' + b.dataset.userDel); load(page); } catch (e) { setMsgBox(e.message); }
        };
      });
      const totalPage = Math.max(1, Math.ceil(data.total / pageSize));
      $('#user-pager').innerHTML = `
        <button class="btn btn-sm" id="up-prev" ${page <= 1 ? 'disabled' : ''}>上一页</button>
        <span>第 ${page} / ${totalPage} 页 · 共 ${data.total} 人</span>
        <button class="btn btn-sm" id="up-next" ${page >= totalPage ? 'disabled' : ''}>下一页</button>`;
      $('#up-prev').onclick = () => { if (page > 1) load(page - 1); };
      $('#up-next').onclick = () => { if (page < totalPage) load(page + 1); };
    } catch (e) { setMsgBox(e.message); }
  };

  $('#btn-new-user').onclick = () => {
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="nu-msg"></div>
      <label class="field"><span>用户名</span><input class="input" id="nu-name"></label>
      <label class="field"><span>密码（至少 6 位）</span><input class="input" type="password" id="nu-pass"></label>
      <label class="field"><span>角色</span><select class="input" id="nu-role">
        <option value="user">user（普通用户）</option>
        <option value="admin">admin（系统管理员）</option>
      </select></label>`;
    const mask = openModal('创建用户', body);
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '创建';
    mask.querySelector('.modal-foot').prepend(saveBtn);
    saveBtn.onclick = async () => {
      try {
        const username = $('#nu-name').value.trim();
        const password = $('#nu-pass').value;
        if (!username || !password) throw new Error('用户名和密码不能为空');
        if (password.length < 6) throw new Error('密码至少 6 位');
        await API.post('/users', { username, password, role: $('#nu-role').value });
        mask.remove();
        load(page);
        setMsgBox('用户已创建', 'success');
      } catch (e) { setMsg($('#nu-msg'), e.message); }
    };
  };

  await load(1);
}

// ==================== 仪表盘 ====================
async function dashboard(root) {
  root.innerHTML = `
    <div class="page-head">
      <h1>仪表盘</h1>
      <span class="sub">数据概览 · 普通用户仅统计自己参与的集合</span>
    </div>
    <div id="db-msg"></div>
    <div class="stat-grid">
      <div class="stat-card glass"><div class="stat-num" id="st-collections">-</div><div class="stat-label">集合</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-items">-</div><div class="stat-label">数据项</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-tasks">-</div><div class="stat-label">刮削任务</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-pending">-</div><div class="stat-label">待处理</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-success">-</div><div class="stat-label">成功</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-failed">-</div><div class="stat-label">失败</div></div>
      <div class="stat-card glass"><div class="stat-num" id="st-relations">-</div><div class="stat-label">关联关系</div></div>
    </div>
    <div class="card glass">
      <div class="card-title">最近集合 <a href="#/collections">查看全部 →</a></div>
      <div id="db-cols"></div>
    </div>
    <div class="card glass">
      <div class="card-title">最近刮削任务 <a href="#/scrape-tasks">查看全部 →</a></div>
      <div id="db-tasks"></div>
    </div>`;

  const msg = $('#db-msg');
  const setMsgBox = (t) => setMsg(msg, t);

  try {
    const ov = await API.get('/stats/overview');
    $('#st-collections').textContent = ov.collections;
    $('#st-items').textContent = ov.items;
    $('#st-tasks').textContent = ov.tasks.total;
    $('#st-pending').textContent = ov.tasks.pending;
    $('#st-success').textContent = ov.tasks.success;
    $('#st-failed').textContent = ov.tasks.failed;
    $('#st-relations').textContent = ov.relations ? ov.relations.total : 0;
    $('#st-relations').title = ov.relations
      ? `父子 ${ov.relations.parent_child} · 引用 ${ov.relations.reference} · 调用 ${ov.relations.call}`
      : '';
  } catch (e) { setMsgBox(e.message); }

  try {
    const cols = await API.get('/collections?page=1&page_size=5');
    $('#db-cols').innerHTML = cols.items.length ? cols.items.map((c) => `
      <div class="list-row" data-href="#/collections/${c.id}">
        <div class="row-main">
          <div class="row-title">${esc(c.name)}</div>
          <div class="row-sub">${c.members.length} 名成员 · ${fmtTime(c.created_at)}</div>
        </div>
        <span class="badge badge-blue">${c.tag_schema.length} 字段</span>
      </div>`).join('') : '<div class="empty">暂无集合</div>';
    $$('.list-row[data-href]', $('#db-cols')).forEach((el) => {
      el.onclick = () => { location.hash = el.dataset.href; };
    });
  } catch (e) { setMsgBox(e.message); }

  try {
    const tasks = await API.get('/scrape-tasks?page=1&page_size=5');
    $('#db-tasks').innerHTML = tasks.items.length ? tasks.items.map((t) => `
      <div class="list-row">
        <div class="row-main">
          <div class="row-title">${esc(t.data_path)}</div>
          <div class="row-sub">${esc(t.script_path)} · ${fmtTime(t.created_at)}</div>
        </div>
        ${statusBadge(SCRAPE_BADGE, t.status)}
      </div>`).join('') : '<div class="empty">暂无刮削任务</div>';
  } catch (e) { setMsgBox(e.message); }
}

// ==================== 刮削管理 ====================
async function scrapeTasks(root) {
  const pageSize = 15;
  let page = 1;

  root.innerHTML = `
    <div class="page-head">
      <h1>刮削管理</h1>
      <span class="sub">全局刮削任务列表（普通用户仅可见自己参与的集合）</span>
    </div>
    <div id="gt-msg"></div>
    <div class="card glass">
      <div class="filter-bar">
        <select class="input" id="tf-status">
          <option value="">全部状态</option>
          <option value="pending">pending · 待处理</option>
          <option value="running">running · 执行中</option>
          <option value="success">success · 成功</option>
          <option value="failed">failed · 失败</option>
        </select>
        <button class="btn btn-primary btn-sm" id="tf-search">查询</button>
      </div>
      <table class="table">
        <thead><tr><th>状态</th><th>数据路径</th><th>刮削脚本</th><th>触发</th><th>创建时间</th><th>错误信息 / 结果</th></tr></thead>
        <tbody id="gtask-body"></tbody>
      </table>
      <div class="pager" id="gtask-pager"></div>
    </div>`;

  const msg = $('#gt-msg');
  const setMsgBox = (t) => setMsg(msg, t);

  const load = async (p) => {
    try {
      const status = $('#tf-status').value;
      const data = await API.get(`/scrape-tasks?status=${encodeURIComponent(status)}&page=${p}&page_size=${pageSize}`);
      page = p;
      const body = $('#gtask-body');
      if (!data.items.length) {
        body.innerHTML = '<tr><td colspan="6" class="empty">暂无刮削任务</td></tr>';
      } else {
        body.innerHTML = data.items.map((t) => `
          <tr>
            <td>${statusBadge(SCRAPE_BADGE, t.status)}</td>
            <td class="wide">${esc(t.data_path)}</td>
            <td class="wide">${esc(t.script_path)}</td>
            <td>${esc(t.trigger_by === 'auto' ? '自动' : t.trigger_by)}</td>
            <td>${fmtTime(t.created_at)}</td>
            <td class="wide">${esc(t.error || (t.result_tags ? '产出 ' + Object.keys(t.result_tags).length + ' 个标签' : '-'))}</td>
          </tr>`).join('');
      }
      const totalPage = Math.max(1, Math.ceil(data.total / pageSize));
      $('#gtask-pager').innerHTML = `
        <button class="btn btn-sm" id="gp-prev" ${page <= 1 ? 'disabled' : ''}>上一页</button>
        <span>第 ${page} / ${totalPage} 页 · 共 ${data.total} 条</span>
        <button class="btn btn-sm" id="gp-next" ${page >= totalPage ? 'disabled' : ''}>下一页</button>`;
      $('#gp-prev').onclick = () => { if (page > 1) load(page - 1); };
      $('#gp-next').onclick = () => { if (page < totalPage) load(page + 1); };
    } catch (e) { setMsgBox(e.message); }
  };

  $('#tf-search').onclick = () => load(1);
  await load(1);
}

// ==================== 个人设置 ====================
async function profile(root) {
  const me = App.user;
  root.innerHTML = `
    <div class="page-head"><h1>个人设置</h1></div>
    <div id="pf-msg"></div>
    <div class="card glass">
      <div class="card-title">个人信息</div>
      <div class="detail-line"><span class="k">用户名</span><span class="v">${esc(me.username)}</span></div>
      <div class="detail-line"><span class="k">角色</span><span class="v">${me.role === 'admin' ? '系统管理员' : '普通用户'}</span></div>
      <div class="detail-line"><span class="k">用户 ID</span><span class="v">${esc(me.id)}</span></div>
    </div>
    <div class="card glass">
      <div class="card-title">修改密码</div>
      <label class="field"><span>原密码</span><input type="password" class="input" id="pf-old" autocomplete="current-password"></label>
      <label class="field"><span>新密码（至少 6 位）</span><input type="password" class="input" id="pf-new" autocomplete="new-password"></label>
      <label class="field"><span>确认新密码</span><input type="password" class="input" id="pf-new2" autocomplete="new-password"></label>
      <button class="btn btn-primary" id="pf-save">保存</button>
    </div>`;

  $('#pf-save').onclick = async () => {
    try {
      const oldPass = $('#pf-old').value;
      const newPass = $('#pf-new').value;
      const new2 = $('#pf-new2').value;
      if (!oldPass || !newPass) throw new Error('请填写原密码与新密码');
      if (newPass !== new2) throw new Error('两次输入的新密码不一致');
      if (newPass.length < 6) throw new Error('新密码至少 6 位');
      await API.post('/auth/password', { old_password: oldPass, new_password: newPass });
      $('#pf-old').value = '';
      $('#pf-new').value = '';
      $('#pf-new2').value = '';
      setMsg($('#pf-msg'), '密码修改成功', 'success');
    } catch (e) {
      setMsg($('#pf-msg'), e.message);
    }
  };
}

// ==================== 数据查询（DQL） ====================
async function query(root) {
  const pageSize = 10;
  let page = 1;
  let colMap = {}; // collection_id -> 集合名
  let allCols = []; // 可访问集合（新增数据项时选择）

  root.innerHTML = `
    <div class="page-head">
      <h1>数据查询</h1>
      <span class="sub">DQL 跨集合查询：AND/OR（AND 优先）、括号、=、!=、&gt;、&gt;=、&lt;、&lt;=、IN、EXISTS、LIKE；collection = "集合名" 限定范围</span>
    </div>
    <div class="card glass">
      <div class="card-title">DQL 语句</div>
      <textarea class="input" id="dql-input" rows="3" placeholder='示例：model_name = "demo" AND age >= 3'></textarea>
      <div style="margin-top:12px;display:flex;gap:10px;flex-wrap:wrap">
        <button class="btn btn-primary" id="btn-dql-run">查询</button>
        <button class="btn btn-sm" id="btn-dql-example">示例</button>
        <button class="btn btn-sm" id="btn-dql-clear">清空</button>
        <button class="btn btn-sm" id="btn-q-batch" style="margin-left:auto">批量添加</button>
        <button class="btn btn-sm" id="btn-q-add">新增数据项</button>
      </div>
    </div>
    <div class="card glass">
      <div class="card-title">💡 查询示例 <span class="sub">点击自动填入并查询 · 完整语法见 <a href="#/help">DQL 语法帮助</a></span></div>
      <div id="dql-examples"></div>
    </div>
    <div class="card glass">
      <div class="card-title">查询结果 <span class="sub" id="q-count"></span>
        <span style="float:right;display:flex;gap:8px">
          <button class="btn btn-sm" id="btn-q-agg">分组统计</button>
          <button class="btn btn-sm" id="btn-q-export">导出 CSV</button>
        </span>
      </div>
      <div id="q-msg"></div>
      <table class="table">
        <thead><tr><th>所属集合</th><th>路径</th><th>标签</th><th>来源</th><th>刮削状态</th><th>创建时间</th><th></th></tr></thead>
        <tbody id="q-body"></tbody>
      </table>
      <div class="pager" id="q-pager"></div>
    </div>`;

  const msg = $('#q-msg');
  const setMsgBox = (t, k) => setMsg(msg, t, k);

  try {
    allCols = (await API.get('/collections?page=1&page_size=200')).items;
    for (const c of allCols) colMap[c.id] = c.name;
  } catch (e) { setMsgBox(e.message); }

  // 策略化删除确认弹窗（delete-impact 预检 → 动态选项 → 带参数删除）
  const openDeleteConfirm = async (itemId) => {
    try {
      const impact = await API.get(`/items/${itemId}/delete-impact`);
      const children = impact.children || [];
      const incoming = impact.incoming || [];
      const policy = impact.policy || { children: 'deny', incoming: 'deny' };
      const body = document.createElement('div');
      body.innerHTML = `
        <div id="dc-msg"></div>
        <div class="msg msg-error">该数据项存在关联关系，删除将按集合策略处理</div>
        <div class="detail-line"><span class="k">子节点</span><span class="v">${children.length} 个（父子）· 策略：${policy.children}</span></div>
        ${children.length ? `<div class="detail-line"><span class="k">子节点清单</span><span class="v">${children.slice(0, 5).map((c) => esc(c.path)).join('<br>')}${children.length > 5 ? '<br>…' : ''}</span></div>` : ''}
        <div class="detail-line"><span class="k">被引用/调用</span><span class="v">${incoming.length} 处 · 策略：${policy.incoming}</span></div>
        ${incoming.length ? `<div class="detail-line"><span class="k">引用方</span><span class="v">${incoming.slice(0, 5).map((c) => esc(c.path)).join('<br>')}${incoming.length > 5 ? '<br>…' : ''}</span></div>` : ''}
        <div style="margin-top:12px">
          ${impact.children_deny || impact.will_cascade ? `
            <label class="check-line"><input type="checkbox" id="dc-cascade" ${impact.will_cascade && !impact.children_deny ? 'checked' : ''}> 级联删除全部子节点（${children.length} 个）</label>` : ''}
          ${impact.incoming_deny ? `
            <label class="check-line"><input type="checkbox" id="dc-force"> 强制解除外部引用（${incoming.length} 处）</label>` : ''}
        </div>`;
      const mask = openModal('删除确认', body);
      const delBtn = document.createElement('button');
      delBtn.className = 'btn btn-danger';
      delBtn.textContent = '确认删除';
      mask.querySelector('.modal-foot').prepend(delBtn);
      delBtn.onclick = async () => {
        try {
          const cascade = $('#dc-cascade') ? $('#dc-cascade').checked : impact.will_cascade;
          const force = $('#dc-force') ? $('#dc-force').checked : false;
          const params = new URLSearchParams();
          if (cascade) params.set('cascade', 'true');
          if (force) params.set('force', 'true');
          const qs = params.toString();
          const result = await API.del(`/items/${itemId}${qs ? '?' + qs : ''}`);
          mask.remove();
          setMsgBox(`已删除（影响 ${result.affected_item_count || 1} 个数据项${result.detach_incoming_count ? '，解除引用 ' + result.detach_incoming_count + ' 处' : ''}）`, 'success');
          load(page);
        } catch (e) {
          setMsg($('#dc-msg'), e.message);
        }
      };
    } catch (e) { setMsgBox(e.message); }
  };

  // 关联徽标（批量）
  const loadBadges = async (items) => {
    if (!items.length) return;
    try {
      const ids = items.map((it) => it.id).join(',');
      const badges = await API.get(`/items/relation-badges?ids=${ids}`);
      $$('#q-body tr').forEach((tr) => {
        const id = tr.dataset.itemId;
        if (!id || !badges[id]) return;
        const b = badges[id];
        const parts = [];
        if (b.out) parts.push(`<span class="badge badge-blue">子 ${b.out}</span>`);
        if (b.in) parts.push(`<span class="badge badge-orange">被引用 ${b.in}</span>`);
        if (parts.length) {
          const cell = tr.querySelector('[data-rel-badge]');
          if (cell) cell.innerHTML = parts.join(' ');
        }
      });
    } catch (e) { /* 徽标失败不阻塞列表 */ }
  };

  const load = async (p) => {
    const dqlStr = $('#dql-input').value.trim();
    if (!dqlStr) { setMsgBox('请输入 DQL 语句'); return; }
    try {
      const data = await API.post('/dql/query', { dql: dqlStr, page: p, page_size: pageSize });
      page = p;
      dataItems = data.items; // 供 CSV 导出（系统优化 1.2）
      const body = $('#q-body');
      if (!data.items.length) {
        body.innerHTML = '<tr><td colspan="8" class="empty">无匹配数据项</td></tr>';
      } else {
        body.innerHTML = data.items.map((it) => `
          <tr data-item-id="${it.id}">
            <td>${esc(colMap[it.collection_id] || it.collection_id)}</td>
            <td class="wide">${esc(it.path)}</td>
            <td>${tagsSummary(it.tags, 2)}</td>
            <td><span data-rel-badge></span></td>
            <td>${statusBadge(SOURCE_BADGE, it.tag_source)}</td>
            <td>${statusBadge(SCRAPE_BADGE, it.scrape_status)}</td>
            <td>${fmtTime(it.created_at)}</td>
            <td class="actions">
              <button class="btn btn-sm" data-q-detail="${it.id}">详情</button>
              <button class="btn btn-sm" data-q-scrape="${it.id}">重刮</button>
              <button class="btn btn-sm btn-danger" data-q-del="${it.id}">删除</button>
            </td>
          </tr>`).join('');
        $$('[data-q-detail]', body).forEach((b) => { b.onclick = () => { location.hash = '#/items/' + b.dataset.qDetail; }; });
        $$('[data-q-scrape]', body).forEach((b) => {
          b.onclick = async () => {
            try { await API.post(`/items/${b.dataset.qScrape}/scrape`); setMsgBox('已触发刮削，稍后刷新查看', 'success'); }
            catch (e) { setMsgBox(e.message); }
          };
        });
        $$('[data-q-del]', body).forEach((b) => {
          b.onclick = () => openDeleteConfirm(b.dataset.qDel);
        });
        loadBadges(data.items);
      }
      $('#q-count').textContent = `共 ${data.total} 条`;
      const totalPage = Math.max(1, Math.ceil(data.total / pageSize));
      $('#q-pager').innerHTML = `
        <button class="btn btn-sm" id="qp-prev" ${page <= 1 ? 'disabled' : ''}>上一页</button>
        <span>第 ${page} / ${totalPage} 页</span>
        <button class="btn btn-sm" id="qp-next" ${page >= totalPage ? 'disabled' : ''}>下一页</button>`;
      $('#qp-prev').onclick = () => { if (page > 1) load(page - 1); };
      $('#qp-next').onclick = () => { if (page < totalPage) load(page + 1); };
    } catch (e) { setMsgBox(e.message); }
  };

  // 新增数据项：选择集合 → 动态标签表单
  const openAdd = () => {
    if (!allCols.length) { setMsgBox('暂无可用集合'); return; }
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="qa-msg"></div>
      <label class="field"><span>所属集合</span><select class="input" id="qa-col">
        ${allCols.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('')}
      </select></label>
      <div id="qa-fields"></div>`;
    const mask = openModal('新增数据项', body);
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '添加';
    mask.querySelector('.modal-foot').prepend(saveBtn);

    let curCol = null;
    let renderSeq = 0; // 竞态防护：仅接受最新一次渲染请求（快速切换集合时旧响应不得覆盖新表单）
    const renderFields = async () => {
      const seq = ++renderSeq;
      try {
        curCol = await API.get('/collections/' + $('#qa-col').value);
        if (seq !== renderSeq) return;
        curCol.tag_schema = curCol.tag_schema || [];
        $('#qa-fields').innerHTML = `
          <label class="field"><span>NFS 路径（文件或文件夹，需存在）</span><input class="input" id="qa-path" placeholder="/nfs/data/..."></label>
          <label class="check-line"><input type="checkbox" id="qa-autoscrape" checked> 刮削添加（自动执行集合刮削脚本）</label>
          <div class="card-title" style="margin-bottom:10px">初始标签（可选）</div>
          ${curCol.tag_schema.map((t) => `
            <label class="field"><span>${esc(t.name)}${t.required ? ' *' : ''}（${esc(t.type)}）</span>${tagInputHtml(t)}</label>`).join('')}`;
      } catch (e) { if (seq === renderSeq) setMsg($('#qa-msg'), e.message); }
    };
    $('#qa-col').onchange = renderFields;
    renderFields();

    saveBtn.onclick = async () => {
      try {
        if (!curCol) throw new Error('请选择集合');
        const path = $('#qa-path').value.trim();
        if (!path) throw new Error('路径不能为空');
        const payload = { path, auto_scrape: $('#qa-autoscrape').checked };
        const tags = collectTagValues($('#qa-fields'), curCol.tag_schema);
        if (Object.keys(tags).length) payload.tags = tags;
        await API.post(`/collections/${curCol.id}/items`, payload);
        mask.remove();
        setMsgBox('数据项已添加', 'success');
        // 仅在已有查询语句时刷新结果，避免覆盖成功提示
        if ($('#dql-input').value.trim()) load(page);
      } catch (e) { setMsg($('#qa-msg'), e.message); }
    };
  };

  // 批量添加数据项（系统优化 1.1）：每行一个 NFS 路径，≤500 条，返回成功/失败明细
  const openBatch = () => {
    if (!allCols.length) { setMsgBox('暂无可用集合'); return; }
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="qb-msg"></div>
      <label class="field"><span>所属集合</span><select class="input" id="qb-col">
        ${allCols.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('')}
      </select></label>
      <label class="field"><span>NFS 路径清单（每行一个，需存在；单次最多 500 条）</span>
        <textarea class="input" id="qb-paths" rows="8" placeholder="/nfs/data/layout/LAY_0001.gds&#10;/nfs/data/layout/LAY_0002.gds"></textarea>
      </label>
      <label class="check-line"><input type="checkbox" id="qb-autoscrape"> 刮削添加（批量导入默认直接注册，勾选后执行集合刮削脚本）</label>
      <div id="qb-failed"></div>`;
    const mask = openModal('批量添加数据项', body);
    const saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '批量添加';
    mask.querySelector('.modal-foot').prepend(saveBtn);

    saveBtn.onclick = async () => {
      const paths = $('#qb-paths').value.split('\n').map((s) => s.trim()).filter(Boolean);
      const msg = $('#qb-msg');
      if (!paths.length) { setMsg(msg, '请输入至少一个路径'); return; }
      if (paths.length > 500) { setMsg(msg, '单次最多批量添加 500 个数据项'); return; }
      saveBtn.disabled = true;
      try {
        const result = await API.post(`/collections/${$('#qb-col').value}/items/batch`, {
          items: paths.map((p) => ({ path: p })),
          auto_scrape: $('#qb-autoscrape').checked,
        });
        const ok = result.success.length;
        const bad = result.failed || [];
        if (bad.length) {
          $('#qb-failed').innerHTML = `<div class="msg msg-error">失败 ${bad.length} 条：</div>
            <div style="max-height:140px;overflow:auto;font-size:12.5px;line-height:1.7">${bad.map((f) => esc(f.path) + ' — ' + esc(f.error)).join('<br>')}</div>`;
        }
        if (ok > 0) {
          // 全部成功才关闭；部分失败保留弹窗展示明细
          setMsgBox(`批量添加成功 ${ok} 条${bad.length ? `，失败 ${bad.length} 条` : ''}`, bad.length ? 'error' : 'success');
          if (!bad.length) mask.remove();
          if ($('#dql-input').value.trim()) load(page);
        } else {
          setMsg(msg, '全部失败，请查看下方明细', 'error');
          saveBtn.disabled = false;
        }
      } catch (e) { setMsg(msg, e.message); saveBtn.disabled = false; }
    };
  };

  $('#btn-dql-run').onclick = () => load(1);
  $('#btn-dql-clear').onclick = () => { $('#dql-input').value = ''; };
  $('#btn-dql-example').onclick = () => {
    const ex = DQL_EXAMPLES[Math.floor(Math.random() * DQL_EXAMPLES.length)];
    $('#dql-input').value = ex;
  };
  $('#btn-q-add').onclick = openAdd;
  $('#btn-q-batch').onclick = openBatch;

  // 导出 CSV（系统优化 1.2）：导出当前页结果（UTF-8 BOM，Excel 中文兼容）
  const exportCSV = () => {
    const rows = $('#q-body').rows ? Array.from($('#q-body').rows) : [];
    if (!rows.length) { setMsgBox('当前页无数据可导出'); return; }
    const csvEsc = (s) => '"' + String(s ?? '').replace(/"/g, '""') + '"';
    const head = ['所属集合', '路径', '标签', '来源', '刮削状态', '创建时间'].map(csvEsc).join(',');
    const lines = rows.map((tr) => {
      const cells = Array.from(tr.cells);
      const id = tr.dataset.itemId;
      const it = dataItems.find((x) => x.id === id) || {};
      return [colMap[it.collection_id] || it.collection_id, it.path || cells[1]?.textContent,
        JSON.stringify(it.tags || {}), it.tag_source || '', it.scrape_status || '',
        fmtTime(it.created_at)].map(csvEsc).join(',');
    });
    const blob = new Blob(['\ufeff' + head + '\n' + lines.join('\n')], { type: 'text/csv;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'dql-export.csv';
    a.click();
    URL.revokeObjectURL(a.href);
    setMsgBox(`已导出 ${lines.length} 行 CSV`, 'success');
  };
  let dataItems = []; // 当前页数据项（导出用）
  $('#btn-q-export').onclick = exportCSV;

  // 分组统计（系统优化 1.2）：DQL 范围 + GROUP BY 字段 → 分布条
  const openAgg = () => {
    const dqlStr = $('#dql-input').value.trim();
    if (!dqlStr) { setMsgBox('请先输入 DQL 语句'); return; }
    const body = document.createElement('div');
    body.innerHTML = `
      <div id="agg-msg"></div>
      <label class="field"><span>分组字段（string/enum 标签）</span><input class="input" id="agg-field" placeholder="node"></label>
      <div id="agg-res" style="margin-top:10px"></div>`;
    const mask = openModal('分组统计', body);
    const runBtn = document.createElement('button');
    runBtn.className = 'btn btn-primary';
    runBtn.textContent = '统计';
    mask.querySelector('.modal-foot').prepend(runBtn);
    runBtn.onclick = async () => {
      const field = $('#agg-field').value.trim();
      if (!field) { setMsg($('#agg-msg'), '请输入分组字段'); return; }
      runBtn.disabled = true;
      try {
        const results = await API.post('/dql/aggregate', { dql: dqlStr, group_by: field, limit: 50 });
        const box = $('#agg-res');
        if (!results.length) { box.innerHTML = '<div class="empty">无匹配数据</div>'; }
        else {
          const max = Math.max(...results.map((r) => r.count));
          box.innerHTML = `<div class="card-title" style="margin-bottom:8px">按 ${esc(field)} 分组（共 ${results.reduce((a, r) => a + r.count, 0)} 项）</div>` +
            results.map((r) => `
              <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px">
                <span style="width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:12.5px">${esc(r.value || '(空)')}</span>
                <div style="flex:1;background:rgba(77,107,254,0.10);border-radius:6px;height:16px;overflow:hidden">
                  <div style="width:${Math.max(2, Math.round(r.count / max * 100))}%;height:100%;background:linear-gradient(90deg,#4d6bfe,#7b95ff);border-radius:6px"></div>
                </div>
                <span style="width:52px;text-align:right;font-size:12px;color:var(--text-2)">${r.count}</span>
              </div>`).join('');
        }
        runBtn.disabled = false;
      } catch (e) { setMsg($('#agg-msg'), e.message); runBtn.disabled = false; }
    };
  };
  $('#btn-q-agg').onclick = openAgg;

  // 示例提示：点击自动填入并查询
  $('#dql-examples').innerHTML = DQL_EXAMPLE_HINTS.map((e, i) => `
    <button type="button" class="example-chip" data-example="${i}" title="${esc(e.title)}">${esc(e.dql)}</button>`).join('');
  $$('#dql-examples .example-chip').forEach((b) => {
    b.onclick = () => {
      const ex = DQL_EXAMPLE_HINTS[parseInt(b.dataset.example, 10)];
      $('#dql-input').value = ex.dql;
      load(1);
    };
  });
}

// ==================== DQL 语法帮助 ====================
async function dqlHelp(root) {
  root.innerHTML = `
    <div class="page-head">
      <h1>DQL 语法帮助</h1>
      <a class="btn btn-primary btn-sm" href="#/query">去查询 →</a>
    </div>
    <div class="card glass">
      <div class="card-title">什么是 DQL？</div>
      <p>DQL（DataQueryLanguage）是类似 JQL/SQL 的数据查询语言，用于在「数据查询」页按<b>标签字段</b>检索数据项（NFS 路径 + 标签）。
      查询范围默认覆盖你有权访问的全部集合，可用 <code class="json">collection = "集合名"</code> 限定在某个集合内。</p>
    </div>
    <div class="card glass">
      <div class="card-title">语法规则</div>
      <ul style="padding-left:18px;line-height:2">
        <li><b>条件</b>：<code class="json">字段 运算符 值</code>，如 <code class="json">model_name = "demo-model"</code></li>
        <li><b>逻辑</b>：AND 优先级高于 OR，可用括号改变优先级，如 <code class="json">(a = 1 OR b = 2) AND c = 3</code></li>
        <li><b>字段</b>：标签名直接书写（支持中文；含特殊字符时用双引号包裹）；<code class="json">collection</code> 为集合名限定字段（仅支持 =、!=、IN）；<code class="json">parent</code> / <code class="json">ancestor</code> 为关联关系字段（仅支持 =、IN，值为数据项 ID：parent = 直接子节点，ancestor = 子树内节点）</li>
        <li><b>值</b>：字符串用单引号或双引号（支持 \\ 转义）；数字支持整数/浮点/负数；布尔 true/false；裸词按字符串处理</li>
        <li><b>类型限制</b>：范围查询仅限 int/float/date；LIKE 仅限 string/enum；引用不存在的标签或集合会提示错误</li>
      </ul>
      <table class="table" style="margin-top:10px">
        <thead><tr><th>运算符</th><th>说明</th><th>示例</th></tr></thead>
        <tbody>
          <tr><td>=</td><td>等于</td><td>model_name = "demo-model"</td></tr>
          <tr><td>!=</td><td>不等于</td><td>stage != "prod"</td></tr>
          <tr><td>&gt; &gt;= &lt; &lt;=</td><td>范围比较（int/float/date）</td><td>age &gt;= 3</td></tr>
          <tr><td>IN</td><td>枚举匹配</td><td>stage IN ("dev", "test")</td></tr>
          <tr><td>EXISTS</td><td>字段是否存在</td><td>config EXISTS true</td></tr>
          <tr><td>LIKE</td><td>包含匹配（string/enum，大小写不敏感）</td><td>model_name LIKE "opc"</td></tr>
        </tbody>
      </table>
    </div>
    <div class="card glass">
      <div class="card-title">查询示例 <span class="sub">点击「复制」后到数据查询页粘贴执行</span></div>
      <div id="dh-examples"></div>
    </div>`;

  $('#dh-examples').innerHTML = DQL_EXAMPLE_HINTS.map((e, i) => `
    <div class="list-row">
      <div class="row-main">
        <div class="row-sub">${esc(e.title)}</div>
        <div class="row-title"><code class="json">${esc(e.dql)}</code></div>
      </div>
      <button class="btn btn-sm" data-dh-copy="${i}">复制</button>
    </div>`).join('');
  $$('[data-dh-copy]').forEach((b) => {
    b.onclick = (ev) => {
      ev.stopPropagation();
      copyText(DQL_EXAMPLE_HINTS[parseInt(b.dataset.dhCopy, 10)].dql);
      b.textContent = '已复制';
      setTimeout(() => { b.textContent = '复制'; }, 1500);
    };
  });
}
