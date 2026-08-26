/**
 * useFaults(filter)：故障事件列表 hook。
 * 被「故障事件」与「配网总览」页共用。
 * 用法：const { list, loading, error, refresh } = GridHooks.useFaults({ status: '', feederId: '', longOutage: false });
 */
window.GridHooks = window.GridHooks || {};

window.GridHooks.useFaults = function (filter) {
  filter = filter || {};
  let list = [];
  let loading = true;
  let error = null;

  function qs() {
    const params = new URLSearchParams();
    if (filter.status) params.set('status', filter.status);
    if (filter.feederId) params.set('feederId', filter.feederId);
    if (filter.longOutage) params.set('longOutage', 'true');
    const s = params.toString();
    return s ? '?' + s : '';
  }

  async function refresh() {
    loading = true;
    error = null;
    try {
      list = await GridAPI.get('/api/faults' + qs());
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
