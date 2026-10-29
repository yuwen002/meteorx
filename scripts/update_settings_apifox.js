/* eslint-disable */
// 一次性脚本：向 MeteorX-backend.apifox.json 的根目录 items 末尾追加
// 「系统配置」文件夹（含管理列表/批量更新/公开配置三个接口）。
const fs = require('fs');
const path = require('path');

const file = path.join(__dirname, '..', 'docs', 'apifox', 'MeteorX-backend.apifox.json');
const data = JSON.parse(fs.readFileSync(file, 'utf-8'));

const root = data.apiCollection.find((c) => c.id === 84837743);
if (!root) {
  console.error('未找到根目录 apiCollection (id=84837743)');
  process.exit(1);
}

// 避免重复追加
if (root.items.some((i) => i.name === '系统配置')) {
  console.log('「系统配置」文件夹已存在，跳过');
  process.exit(0);
}

// ---------- 通用构造器 ----------
function makeApiRequest(fields) {
  return {
    type: 'http',
    path: fields.path,
    method: fields.method,
    parameters: fields.parameters || {},
    auth: fields.noAuth ? { type: 'noauth' } : { type: 'bearer', bearer: {} },
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
    jsonSchema: { type: 'object', properties, 'x-apifox-orders': orders, required },
    required: true,
    mediaType: '',
    examples: [],
    oasExtensions: '',
  };
}

function makeItem(name, api) {
  return { name, api };
}

// ---------- 系统配置接口 ----------
const settingListView = {
  type: 'object',
  properties: {
    code: { type: 'integer' },
    message: { type: 'string' },
    data: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          key: { type: 'string' },
          value: { type: 'string' },
          value_type: { type: 'string' },
          default_value: { type: 'string' },
          is_public: { type: 'boolean' },
          source: { type: 'string' },
          description: { type: 'string' },
          updated_at: { type: 'string' },
        },
      },
    },
  },
};

const listSettings = makeItem('配置列表', makeApiRequest({
  path: '/api/v1/admin/settings',
  method: 'GET',
  ordering: 10,
  responseSchema: settingListView,
  description: '列出全部已登记配置键，含当前值、默认值、类型、来源和描述',
}));

const updateSettings = makeItem('批量更新配置', makeApiRequest({
  path: '/api/v1/admin/settings',
  method: 'PUT',
  ordering: 20,
  requestBody: makeBody({
    settings: {
      type: 'array',
      title: '待更新项',
      items: {
        type: 'object',
        properties: {
          key: { type: 'string', title: '配置键' },
          value: { type: 'string', title: '新值（字符串形态）' },
        },
        required: ['key', 'value'],
      },
    },
  }, ['settings']),
  responseSchema: settingListView,
  description: '整批校验（白名单键+类型归一化），任一非法则整批拒绝；成功后返回全量列表',
}));

const publicSettings = makeItem('公开配置', makeApiRequest({
  path: '/api/v1/settings/public',
  method: 'GET',
  ordering: 30,
  noAuth: true,
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: {} },
    },
  },
  description: '免鉴权接口：仅返回 is_public=true 的配置键，值还原原生 JSON 类型；供登录/注册页消费',
}));

const settingsFolder = {
  name: '系统配置',
  id: 85990010,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '平台运行时配置中心（注册开关/维护模式等），管理接口仅超管，公开接口免鉴权',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [listSettings, updateSettings, publicSettings],
};

// ---------- 写入 ----------
root.items.push(settingsFolder);

fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8');
console.log('Apifox JSON 更新完成：已追加「系统配置」文件夹（3 接口）');
