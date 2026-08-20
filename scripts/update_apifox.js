/* eslint-disable */
// 一次性脚本：向 MeteorX-backend.apifox.json 的根目录 items 末尾追加
// 注销审批 / 运营看板 / 通知公告 三个文件夹（含全部新接口）。
const fs = require('fs');
const path = require('path');

const file = path.join(__dirname, '..', 'docs', 'apifox', 'MeteorX-backend.apifox.json');
const data = JSON.parse(fs.readFileSync(file, 'utf-8'));

const root = data.apiCollection.find((c) => c.id === 84837743);
if (!root) {
  console.error('未找到根目录 apiCollection (id=84837743)');
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

// ---------- 注销审批接口 ----------
const cancelRequestList = makeItem('注销申请列表', makeApiRequest({
  path: '/api/v1/admin/cancel-requests',
  method: 'GET',
  ordering: 10,
  parameters: {
    query: [
      { name: 'page', type: 'integer', description: '页码', required: false, enable: true },
      { name: 'page_size', type: 'integer', description: '每页数量', required: false, enable: true },
      { name: 'status', type: 'integer', description: '状态：1待审批 2已通过 3已驳回 4已完成', required: false, enable: true },
      { name: 'tenant_id', type: 'string', description: '租户ID', required: false, enable: true },
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
            items: {
              type: 'object',
              properties: {
                id: { type: 'string' },
                tenant_id: { type: 'string' },
                tenant_name: { type: 'string' },
                reason: { type: 'string' },
                status: { type: 'integer' },
                status_text: { type: 'string' },
                approver_id: { type: 'string' },
                review_remark: { type: 'string' },
                effective_at: { type: 'string' },
                applied_at: { type: 'string' },
                approved_at: { type: 'string' },
                completed_at: { type: 'string' },
                created_at: { type: 'string' },
                updated_at: { type: 'string' },
              },
            },
          },
          total: { type: 'integer' },
        },
      },
    },
  },
}));

const cancelRequestApprove = makeItem('审批通过注销申请', makeApiRequest({
  path: '/api/v1/admin/cancel-requests/{id}/approve',
  method: 'PUT',
  ordering: 20,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '申请ID', required: true, enable: true }],
  },
  requestBody: makeBody(
    {
      review_remark: { type: 'string', title: '审批备注' },
      effective_days: { type: 'integer', title: '生效天数（0=立即执行）' },
    },
    []
  ),
}));

const cancelRequestReject = makeItem('驳回注销申请', makeApiRequest({
  path: '/api/v1/admin/cancel-requests/{id}/reject',
  method: 'PUT',
  ordering: 30,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '申请ID', required: true, enable: true }],
  },
  requestBody: makeBody(
    {
      review_remark: { type: 'string', title: '审批备注' },
    },
    []
  ),
}));

const cancelFolder = {
  name: '注销审批',
  id: 85990001,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '租户注销申请审批（通过/驳回）',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [cancelRequestList, cancelRequestApprove, cancelRequestReject],
};

// ---------- 运营看板接口 ----------
const dashboardOverview = makeItem('平台运营数据总览', makeApiRequest({
  path: '/api/v1/admin/dashboard/overview',
  method: 'GET',
  ordering: 10,
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: {
        type: 'object',
        properties: {
          tenant_stats: {
            type: 'object',
            properties: {
              total: { type: 'integer' },
              enabled: { type: 'integer' },
              disabled: { type: 'integer' },
              today_new: { type: 'integer' },
              week_new: { type: 'integer' },
              month_new: { type: 'integer' },
            },
          },
          user_stats: {
            type: 'object',
            properties: {
              total: { type: 'integer' },
              today_new: { type: 'integer' },
              week_new: { type: 'integer' },
              month_new: { type: 'integer' },
            },
          },
          subscription_stats: {
            type: 'object',
            properties: {
              total: { type: 'integer' },
              active: { type: 'integer' },
              expired: { type: 'integer' },
              cancelled: { type: 'integer' },
              active_tenant: { type: 'integer' },
            },
          },
          audit_stats: {
            type: 'object',
            properties: {
              total: { type: 'integer' },
              today: { type: 'integer' },
              success: { type: 'integer' },
              failure: { type: 'integer' },
              action_stats: { type: 'object', additionalProperties: { type: 'integer' } },
              module_stats: { type: 'object', additionalProperties: { type: 'integer' } },
            },
          },
        },
      },
    },
  },
}));

const dashboardFolder = {
  name: '运营看板',
  id: 85990002,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '平台运营数据总览',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [dashboardOverview],
};

// ---------- 通知公告接口 ----------
const announcementSchemaProps = {
  id: { type: 'string' },
  title: { type: 'string' },
  content: { type: 'string' },
  scope: { type: 'string' },
  target_tenant_id: { type: 'string' },
  status: { type: 'integer' },
  status_text: { type: 'string' },
  publisher_id: { type: 'string' },
  publish_at: { type: 'string' },
  expire_at: { type: 'string' },
  created_at: { type: 'string' },
  updated_at: { type: 'string' },
};

const announcementCreateReqProps = {
  title: { type: 'string', title: '公告标题' },
  content: { type: 'string', title: '公告内容' },
  scope: { type: 'string', title: '范围：all/tenant' },
  target_tenant_id: { type: 'string', title: '目标租户ID' },
  status: { type: 'integer', title: '初始状态：0草稿 1已发布 2已下架' },
  publish_at: { type: 'string', title: '发布时间' },
  expire_at: { type: 'string', title: '过期时间' },
};

const announcementList = makeItem('公告列表', makeApiRequest({
  path: '/api/v1/admin/announcements',
  method: 'GET',
  ordering: 10,
  parameters: {
    query: [
      { name: 'page', type: 'integer', description: '页码', required: false, enable: true },
      { name: 'page_size', type: 'integer', description: '每页数量', required: false, enable: true },
      { name: 'keyword', type: 'string', description: '标题关键字', required: false, enable: true },
      { name: 'status', type: 'integer', description: '状态：0草稿 1已发布 2已下架', required: false, enable: true },
      { name: 'scope', type: 'string', description: '范围：all/tenant', required: false, enable: true },
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
          items: { type: 'array', items: { type: 'object', properties: announcementSchemaProps } },
          total: { type: 'integer' },
        },
      },
    },
  },
}));

const announcementCreate = makeItem('创建公告', makeApiRequest({
  path: '/api/v1/admin/announcements',
  method: 'POST',
  ordering: 20,
  requestBody: makeBody(announcementCreateReqProps, ['title', 'content', 'scope']),
}));

const announcementDetail = makeItem('公告详情', makeApiRequest({
  path: '/api/v1/admin/announcements/{id}',
  method: 'GET',
  ordering: 30,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '公告ID', required: true, enable: true }],
  },
  responseSchema: {
    type: 'object',
    properties: {
      code: { type: 'integer' },
      message: { type: 'string' },
      data: { type: 'object', properties: announcementSchemaProps },
    },
  },
}));

const announcementUpdate = makeItem('更新公告', makeApiRequest({
  path: '/api/v1/admin/announcements/{id}',
  method: 'PUT',
  ordering: 40,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '公告ID', required: true, enable: true }],
  },
  requestBody: makeBody(announcementCreateReqProps, ['title', 'content', 'scope']),
}));

const announcementDelete = makeItem('删除公告', makeApiRequest({
  path: '/api/v1/admin/announcements/{id}',
  method: 'DELETE',
  ordering: 50,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '公告ID', required: true, enable: true }],
  },
}));

const announcementStatus = makeItem('发布/下架公告', makeApiRequest({
  path: '/api/v1/admin/announcements/{id}/status',
  method: 'PUT',
  ordering: 60,
  parameters: {
    path: [{ name: 'id', type: 'string', description: '公告ID', required: true, enable: true }],
  },
  requestBody: makeBody(
    { status: { type: 'integer', title: '状态：1发布 2下架' } },
    ['status']
  ),
}));

const notificationFolder = {
  name: '通知公告',
  id: 85990003,
  auth: {},
  securityScheme: {},
  parentId: 0,
  serverId: '',
  description: '平台通知公告（CRUD/发布/下架）',
  identityPattern: { httpApi: { type: 'inherit', bodyType: '', fields: [] } },
  shareSettings: {},
  visibility: 'INHERITED',
  moduleId: 7675120,
  preProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  postProcessors: [{ id: 'inheritProcessors', type: 'inheritProcessors', data: {} }],
  inheritPostProcessors: {},
  inheritPreProcessors: {},
  items: [announcementList, announcementCreate, announcementDetail, announcementUpdate, announcementDelete, announcementStatus],
};

// ---------- 写入 ----------
root.items.push(cancelFolder, dashboardFolder, notificationFolder);

fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8');
console.log('Apifox JSON 更新完成：已追加 注销审批 / 运营看板 / 通知公告 三个文件夹');
