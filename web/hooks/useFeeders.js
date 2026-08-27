/**
 * useFeeders：线路列表 hook。
 * 被「配网总览」与「拓扑管理」页共用。
 * 用法：const { list, loading, error, refresh } = GridHooks.useFeeders();
 */
window.GridHooks = window.GridHooks || {};

window.GridHooks.useFeeders = function () {
  let list = [];
  let loading = true;
  let error = null;

  async function refresh() {
    loading = true;
    error = null;
    try {
      list = await GridAPI.get('/api/feeders');
    } catch (e) {
      error = e.message;
      list = [];
    } finally {
      loading = false;
      emit();
    }
  }

  const listeners = [];
  function subscribe(fn) { listeners.push(fn); }
  function emit() { listeners.forEach((fn) => { try { fn(); } catch (e) { console.error(e); } }); }

  refresh();
  return { get list() { return list; }, get loading() { return loading; }, get error() { return error; }, refresh, subscribe };
};
