/**
 * 共享枚举/常量 —— 与后端 Go domain 层保持一致。
 * 后端位置：
 *   domain/switch.go    SwitchStatus / SwitchType
 *   domain/indicator.go IndicatorStatus
 *   domain/fault.go     FaultStatus / FaultAction
 *   domain/feeder.go    FeederStatus
 *   config/config.go    电压等级
 */
window.GridEnums = {
  // 开关状态
  SwitchStatus: {
    CLOSED: 'closed',
    OPEN: 'open',
  },
  // 开关类型
  SwitchType: {
    SECTIONALIZER: 'sectionalizer',
    TIE: 'tie',
    FEEDER_OUTLET: 'feeder_outlet',
  },
  // 指示器状态
  IndicatorStatus: {
    TRIGGERED: 'triggered',
    RESET: 'reset',
  },
  // 故障事件状态
  FaultStatus: {
    LOCATED: 'located',
    REPAIRING: 'repairing',
    RESTORED: 'restored',
    ARCHIVED: 'archived',
  },
  // 线路状态
  FeederStatus: {
    ACTIVE: 'active',
    INACTIVE: 'inactive',
  },
  // 电压等级
  VoltageLevels: ['10kV', '20kV', '35kV'],
  // 长时停电阈值（分钟）
  LONG_OUTAGE_MINUTES: 120,
};

// 状态中文标签与样式
window.GridLabels = {
  switchStatus: { closed: '合闸', open: '分闸' },
  switchType: { sectionalizer: '分段', tie: '联络', feeder_outlet: '出线' },
  indicatorStatus: { triggered: '翻牌', reset: '复位' },
  faultStatus: { located: '已定位', repairing: '抢修中', restored: '已复电', archived: '已归档' },
  feederStatus: { active: '在运', inactive: '停运' },
};
