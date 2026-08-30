// API 客户端：统一处理 {code, message, data} 响应与 JWT
const API = {
  token: localStorage.getItem('dc_token') || '',

  async req(method, path, body) {
    const headers = { 'Content-Type': 'application/json' };
    if (this.token) headers['Authorization'] = 'Bearer ' + this.token;
    let res;
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 20000); // 请求超时防挂起（20s）
    try {
      res = await fetch('/api/v1' + path, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: ctrl.signal,
      });
    } catch (e) {
      throw new Error('网络错误：无法连接服务器（请求超时或连接失败）');
    } finally {
      clearTimeout(timer);
    }
    let data = null;
    try { data = await res.json(); } catch (e) { data = { code: -1, message: '响应解析失败' }; }
    if (res.status === 401 && !path.startsWith('/auth/login')) {
      this.logout();
      throw new Error(data.message || '登录已过期');
    }
    if (data.code !== 0) throw new Error(data.message || '请求失败');
    return data.data;
  },

  get(p) { return this.req('GET', p); },
  post(p, b) { return this.req('POST', p, b); },
  put(p, b) { return this.req('PUT', p, b); },
  patch(p, b) { return this.req('PATCH', p, b); },
  del(p) { return this.req('DELETE', p); },

  logout() {
    this.token = '';
    localStorage.removeItem('dc_token');
    if (location.hash !== '#/login') location.hash = '#/login';
  },
};
