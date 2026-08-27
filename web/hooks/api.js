/**
 * API 工具：统一请求后端，解析 {code,message,data} 响应信封。
 */
window.GridAPI = (function () {
  async function request(method, path, body) {
    const opts = { method, headers: {} };
    if (body !== undefined && body !== null) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const resp = await fetch(path, opts);
    let payload = null;
    try {
      payload = await resp.json();
    } catch (e) {
      payload = null;
    }
    if (!resp.ok || (payload && payload.code !== 0)) {
      const msg = (payload && payload.message) || ('HTTP ' + resp.status + ' ' + resp.statusText);
      const err = new Error(msg);
      err.status = resp.status;
      err.code = payload ? payload.code : -1;
      throw err;
    }
    return payload ? payload.data : null;
  }

  return {
    get: (path) => request('GET', path),
    post: (path, body) => request('POST', path, body === undefined ? {} : body),
    put: (path, body) => request('PUT', path, body === undefined ? {} : body),
    del: (path) => request('DELETE', path),
  };
})();
