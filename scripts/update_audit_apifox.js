/* eslint-disable */
// 向 MeteorX-backend.apifox.json 追加审计日志增强功能（告警管理 + 会话分析）
const fs = require('fs');
const path = require('path');

const file = path.join(__dirname, '..', 'docs', 'apifox', 'MeteorX-backend.apifox.json');
const data = JSON.parse(fs.readFileSync(file, 'utf-8'));

const root = data.apiCollection.find((c) => c.id === 84837743);
if (!root) {
  console.error('未找到根目录 apiCollection (id=84837743)');
  process.exit(1);
}

// 查找现有的审计日志文件夹
const auditFolder = root.items.find((item) => item.name === '审计日志');
if (!auditFolder) {
  console.error('未找到审计日志文件夹');
  process.exit(1);
}

// ---------- 通用构造器 ----------
function makeApiRequest(fields) {
  return {
    type: 'http',
    path: fields.path,
    method: fields.method,
    parameters: fields.parameters || {},
    auth: { type: 'bearer', bearer: {} },
    securityScheme: {},
    commonParameters: { query: [], body: [], cookie: [], header: [] },
    responses: [
      {
        id: String(Math.floor(100000000 + Math.random() * 900000000)),
        code: 200,
        name: '成功',
        headers: [],
        jsonSchema: fields.responseSchema || { type: 'object', properties: {} },
        itemSchema: {},
        description: '',
        contentType: 'json',
        mediaType: '',
      },
    ],
    responseExamples: [],
    requestBody: fields.requestBody || {},
    description: fields.description || '',
    tags: [],
    status: 'developing',
    serverId: '',
    operationId: '',
    sourceUrl: '',
    ordering: fields.ordering || 0,
    cases: [],
    mocks: [],
    customApiFields: '{}',
    advancedSettings: { disabledSystemHeaders: {} },
    mockScript: {},
    codeSamples: [],
    commonResponseStatus: {},
    responseChildren: [],
    visibility: 'INHERITED',
    moduleId: 7675120,
    oasExtensions: '',
    preProcessors: [],
    postProcessors: [],
    inheritPostProcessors: {},
    inheritPreProcessors: {},
  };
}

function makeBody(properties, required = []) {
  const orders = Object.keys(properties);
  return {
    type: 'application/json',
    parameters: [],
    jsonSchema: {
      type: 'object',
      properties,
      'x-apifox-orders': orders,
      required,
    },
    required: true,
    mediaType: '',
    examples: [],
    oasExtensions: '',
  };
}

function makeItem(name, api) {
  return { name, api };
}

// ---------- 告警规则接口 ----------
const alertRuleSchemaProps = {
  id: { type: 'string', title: '规则ID' },
  name: { type: 'string', title: '规则名称' },
  description: { type: 'string', title: '规则描述' },
  enabled: { type: 'boolean', title: '是否启用' },
  trigger_type: { type: 'string', title: '触发类型', enum: ['risk_level', 'action', 'user'] },
  trigger_value: { type: 'string', title: '触发值' },
  notify_channels: { type: 'array', title: '通知渠道', items: { type: 'string' } },
  notify_targets: { type: 'array', title: '通知目标', items: { type: 'string' } },
  notify_template: { type: 'string', title: '通知模板' },
  cooldown_minutes: { type: 'integer', title: '冷却时间（分钟）' },
  created_at: { type: 'string', title: '创建时间' },
  updated_at: { type: 'string', title: '更新时间' },
};

const alertRuleCreateProps = {
  name: { type: 'string', title: '规则名称' },
  description: { type: 'string', title: '规则描述' },
  enabled: { type: 'boolean', title: '是否启用' },
  trigger_type: { type: 'string', title: '触发类型', enum: ['risk_level', 'action', 'user'] },
  trigger_value: { type: 'string', title: '触发值' },
  notify_channels: { type: 'array', title: '通知渠道', items: { type: 'string' } },
  notify_targets: { type: 'array', title: '通知目标', items: { type: 'string' } },
  notify_template: { type: 'string', title: '通知模板' },
  cooldown_minutes: { type: 'integer', title: '冷却时间（分钟）' },
};

const createAlertRule = makeItem('创建告警规则', makeApiRequest({
  path: '/api/v1/audit/alert-rules',
  method: 'POST',
  ordering: 10,
  requestBody: makeBody(alertRuleCreateProps, ['name', 'trigger_type', 'trigger_value', 'notify_channels', 'notify_targets']),
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: alertRuleSchemaProps },
    },
  },
  description: '创建新的告警规则',
}));

const listAlertRules = makeItem('获取所有告警规则', makeApiRequest({
  path: '/api/v1/audit/alert-rules',
  method: 'GET',
  ordering: 20,
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: {
        type: 'array',
        items: { type: 'object', properties: alertRuleSchemaProps },
      },
    },
  },
  description: '获取所有告警规则列表',
}));

const getAlertRule = makeItem('告警规则详情', makeApiRequest({
  path: '/api/v1/audit/alert-rules/{id}',
  method: 'GET',
  ordering: 30,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '规则ID', required: true, enable: true }],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: alertRuleSchemaProps },
    },
  },
  description: '获取单个告警规则详情',
}));

const updateAlertRule = makeItem('更新告警规则', makeApiRequest({
  path: '/api/v1/audit/alert-rules/{id}',
  method: 'PUT',
  ordering: 40,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '规则ID', required: true, enable: true }],
  },
  requestBody: makeBody(alertRuleCreateProps, []),
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: alertRuleSchemaProps },
    },
  },
  description: '更新告警规则',
}));

const deleteAlertRule = makeItem('删除告警规则', makeApiRequest({
  path: '/api/v1/audit/alert-rules/{id}',
  method: 'DELETE',
  ordering: 50,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '规则ID', required: true, enable: true }],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
    },
  },
  description: '删除告警规则',
}));

// ---------- 告警记录接口 ----------
const alertSchemaProps = {
  id: { type: 'string', title: '告警ID' },
  rule_id: { type: 'string', title: '规则ID' },
  rule_name: { type: 'string', title: '规则名称' },
  audit_log_id: { type: 'string', title: '审计日志ID' },
  user_id: { type: 'string', title: '用户ID' },
  username: { type: 'string', title: '用户名' },
  risk_level: { type: 'string', title: '风险等级', enum: ['low', 'medium', 'high', 'critical'] },
  action: { type: 'string', title: '操作类型' },
  message: { type: 'string', title: '告警消息' },
  notified: { type: 'boolean', title: '是否已通知' },
  notify_time: { type: 'string', title: '通知时间' },
  created_at: { type: 'string', title: '创建时间' },
};

const listAlerts = makeItem('告警记录列表', makeApiRequest({
  path: '/api/v1/audit/alerts',
  method: 'GET',
  ordering: 60,
  parameters: {
    query: [
      { name: 'page', type: 'integer', description: '页码', required: false, enable: true },
      { name: 'page_size', type: 'integer', description: '每页数量', required: false, enable: true },
      { name: 'rule_id', type: 'string', description: '规则ID', required: false, enable: true },
      { name: 'user_id', type: 'string', description: '用户ID', required: false, enable: true },
      { name: 'risk_level', type: 'string', description: '风险等级', required: false, enable: true },
    ],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: {
        type: 'object',
        properties: {
          items: {
            type: 'array',
            items: { type: 'object', properties: alertSchemaProps },
          },
          total: { type: 'integer' },
        },
      },
    },
  },
  description: '分页查询告警记录',
}));

const getAlert = makeItem('告警记录详情', makeApiRequest({
  path: '/api/v1/audit/alerts/{id}',
  method: 'GET',
  ordering: 70,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '告警ID', required: true, enable: true }],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: alertSchemaProps },
    },
  },
  description: '获取单个告警记录详情',
}));

// ---------- 会话分析接口 ----------
const sessionSchemaProps = {
  session_id: { type: 'string', title: '会话ID' },
  user_id: { type: 'string', title: '用户ID' },
  username: { type: 'string', title: '用户名' },
  total_requests: { type: 'integer', title: '总请求数' },
  success_count: { type: 'integer', title: '成功次数' },
  failure_count: { type: 'integer', title: '失败次数' },
  avg_duration: { type: 'integer', title: '平均耗时（毫秒）' },
  first_request: { type: 'string', title: '首次请求时间' },
  last_request: { type: 'string', title: '最后请求时间' },
  duration_minutes: { type: 'integer', title: '会话时长（分钟）' },
};

const auditLogSchemaProps = {
  id: { type: 'string', title: '日志ID' },
  user_id: { type: 'string', title: '用户ID' },
  username: { type: 'string', title: '用户名' },
  module: { type: 'string', title: '模块' },
  action: { type: 'string', title: '操作类型' },
  method: { type: 'string', title: 'HTTP方法' },
  path: { type: 'string', title: '请求路径' },
  status_code: { type: 'integer', title: 'HTTP状态码' },
  result: { type: 'string', title: '结果' },
  client_ip: { type: 'string', title: '客户端IP' },
  ip_location: { type: 'string', title: 'IP地理位置' },
  duration: { type: 'integer', title: '请求耗时（毫秒）' },
  error_message: { type: 'string', title: '错误信息' },
  created_at: { type: 'string', title: '创建时间' },
};

const listSessions = makeItem('会话摘要列表', makeApiRequest({
  path: '/api/v1/audit/sessions',
  method: 'GET',
  ordering: 80,
  parameters: {
    query: [
      { name: 'page', type: 'integer', description: '页码', required: false, enable: true },
      { name: 'page_size', type: 'integer', description: '每页数量', required: false, enable: true },
      { name: 'user_id', type: 'string', description: '用户ID', required: false, enable: true },
    ],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: {
        type: 'object',
        properties: {
          items: {
            type: 'array',
            items: { type: 'object', properties: sessionSchemaProps },
          },
          total: { type: 'integer' },
        },
      },
    },
  },
  description: '分页查询会话摘要列表',
}));

const getSessionLogs = makeItem('获取会话的所有日志', makeApiRequest({
  path: '/api/v1/audit/sessions/{id}/logs',
  method: 'GET',
  ordering: 90,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '会话ID', required: true, enable: true }],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: {
        type: 'object',
        properties: {
          ...sessionSchemaProps,
          logs: {
            type: 'array',
            items: { type: 'object', properties: auditLogSchemaProps },
          },
        },
      },
    },
  },
  description: '获取指定会话的所有审计日志',
}));

// ---------- 写入 ----------
const alertRuleFolder = {
  name: '告警管理',
  id: 85990004,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '审计告警规则管理和告警记录查询',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [
    createAlertRule,
    listAlertRules,
    getAlertRule,
    updateAlertRule,
    deleteAlertRule,
    listAlerts,
    getAlert,
  ],
};

const sessionFolder = {
  name: '会话分析',
  id: 85990005,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '用户会话追踪和操作时间线分析',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [
    listSessions,
    getSessionLogs,
  ],
};

root.items.push(alertRuleFolder, sessionFolder);

fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8');
console.log('Apifox JSON 更新完成：已追加 告警管理 / 会话分析 两个文件夹');