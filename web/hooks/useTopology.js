/**
 * useTopology(feederId)：线路拓扑数据 hook（拓扑管理页使用）。
 */
window.GridHooks = window.GridHooks || {};

window.GridHooks.useTopology = function (feederId) {
  let data = null;
  let loading = false;
  let error = null;
  const listeners = [];

  function emit() { listeners.forEach((fn) => { try { fn(); } catch (e) { console.error(e); } }); }
  function subscribe(fn) { listeners.push(fn); }

  async function refresh() {
    if (!feederId) { data = null; loading = false; emit(); return; }
    loading = true;
    error = null;
    try {
      data = await GridAPI.get('/api/feeders/' + encodeURIComponent(feederId) + '/topology');
    } catch (e) {
      error = e.message;
      data = null;
    } finally {
      loading = false;
      emit();
    }
  }

  refresh();
  return {
    get data() { return data; },
    get loading() { return loading; },
    get error() { return error; },
    refresh,
    subscribe,
  };
};
