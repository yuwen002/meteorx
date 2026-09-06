/* eslint-disable */
// 一次性脚本：将 Wiki 知识库模块在代码中已注册、但 apifox 文档缺失的接口补齐到
// docs/apifox/MeteorX-backend.apifox.json 的「知识库」文件夹（id=94455852），
// 并刷新「回收站」三个接口的语义描述（真实恢复 / 级联物理删除 / 软删进回收站）。
const fs = require('fs');
const path = require('path');

const file = path.join(__dirname, '..', 'docs', 'apifox', 'MeteorX-backend.apifox.json');
const data = JSON.parse(fs.readFileSync(file, 'utf-8'));

const root = data.apiCollection.find((c) => c.id === 84837743);
const kb = root.items.find((f) => f.id === 94455852);
if (!kb) {
  console.error('未找到「知识库」文件夹 (id=94455852)');
  process.exit(1);
}

// 已存在路径集合，保证脚本可重复执行（幂等）
const existing = new Set();
const scan = (items) => {
  for (const it of items || []) {
    if (it.api) existing.add(`${it.api.method.toUpperCase()} ${it.api.path.replace(/\/+$/, '')}`);
    else if (it.items && Array.isArray(it.items)) scan(it.items);
  }
};
scan(root.items);

let nextApiId = 507500000; // 避开既有 id（最大值 507490433）
const seenApi = new Set();
function uniqueId() {
  while (seenApi.has(nextApiId) || existing.has(`id:${nextApiId}`)) nextApiId += 1;
  const id = String(nextApiId);
  seenApi.add(id);
  nextApiId += 1;
  return id;
}

function makeApiRequest({ path: p, method, name, order, params = {}, bodyProps = null, bodyRequired = [], description = '' }) {
  const key = `${method.toUpperCase()} ${p.replace(/\/+$/, '')}`;
  if (existing.has(key)) return null; // 已存在则跳过

  const api = {
    id: uniqueId(),
    type: 'http',
    path: p,
    method: method.toLowerCase(),
    parameters: {
      path: params.path || [],
      query: params.query || [],
      cookie: [],
      header: [],
    },
    auth: { type: 'bearer', bearer: {} },
    securityScheme: {},
    commonParameters: { query: [], body: [], cookie: [], header: [] },
    responses: [
      {
        id: uniqueId(),
        code: 200,
        name: '成功',
        headers: [],
        jsonSchema: { type: 'object', properties: {} },
        itemSchema: {},
        description: '',
        contentType: 'json',
        mediaType: '',
      },
    ],
    responseExamples: [],
    requestBody: bodyProps
      ? makeBody(bodyProps, bodyRequired)
      : { type: 'none', parameters: [], jsonSchema: {}, required: false, mediaType: '', examples: [], oasExtensions: '' },
    description,
    tags: [],
    status: 'developing',
    serverId: '',
    operationId: '',
    sourceUrl: '',
    ordering: order,
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
  existing.add(key);
  return { name, api };
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

// ---------- 通用参数 ----------
const docIdPath = { name: 'id', type: 'string', description: '文档ID', required: true, enable: true };
const idPath = { name: 'id', type: 'string', description: 'ID', required: true, enable: true };
const pageQuery = { name: 'page', type: 'integer', description: '页码', required: false, enable: true };
const pageSizeQuery = { name: 'page_size', type: 'integer', description: '每页数量', required: false, enable: true };
const tokenQuery = { name: 'token', type: 'string', description: '分享 Token', required: true, enable: true };

// 接口路径与展示顺序（对齐 wiki_handler.go / wiki_handler_extended.go 实际注册）
const items = [
  // Markdown 实时预览
  { name: 'Markdown 实时预览', p: '/api/v1/wiki/documents/preview', method: 'POST', order: 200,
    bodyProps: { content: { type: 'string', title: 'Markdown 原文' } }, bodyRequired: ['content'],
    description: '将 Markdown 内容渲染为 HTML（不做持久化）' },

  // 空间级标签（作用于当前租户，路径不含 spaceId）
  { name: '创建空间级标签', p: '/api/v1/wiki/spaces/tags', method: 'POST', order: 210,
    bodyProps: { name: { type: 'string', title: '标签名' }, color: { type: 'string', title: '颜色' } }, bodyRequired: ['name'] },
  { name: '空间级标签列表', p: '/api/v1/wiki/spaces/tags', method: 'GET', order: 220 },
  { name: '删除空间级标签', p: '/api/v1/wiki/spaces/tags/{id}', method: 'DELETE', order: 230, params: { path: [idPath] } },

  // 节点批量操作
  { name: '节点批量操作', p: '/api/v1/wiki/spaces/nodes/batch', method: 'POST', order: 240,
    bodyProps: {
      action: { type: 'string', title: '操作：delete / move' },
      node_ids: { type: 'array', items: { type: 'string' }, title: '节点ID列表' },
      target: { type: 'string', title: '目标父节点ID（action=move 时使用；移动至空间根传空）' },
    }, bodyRequired: ['action', 'node_ids'],
    description: '事务内批量删除或移动节点，逐节点校验归属与权限' },

  // 文档模板
  { name: '创建文档模板', p: '/api/v1/wiki/spaces/templates', method: 'POST', order: 250,
    bodyProps: {
      name: { type: 'string', title: '模板名' },
      description: { type: 'string', title: '模板描述' },
      format: { type: 'string', title: '格式：markdown/rich_text 等' },
      category: { type: 'string', title: '分类' },
      content: { type: 'string', title: '模板正文' },
      is_public: { type: 'boolean', title: '是否公开模板' },
    }, bodyRequired: ['name', 'content'] },
  { name: '文档模板列表', p: '/api/v1/wiki/spaces/templates', method: 'GET', order: 260,
    params: { query: [{ name: 'category', type: 'string', description: '分类过滤', required: false, enable: true }] } },
  { name: '文档模板详情', p: '/api/v1/wiki/spaces/templates/{id}', method: 'GET', order: 270, params: { path: [idPath] } },
  { name: '更新文档模板', p: '/api/v1/wiki/spaces/templates/{id}', method: 'PUT', order: 280, params: { path: [idPath] },
    bodyProps: {
      name: { type: 'string', title: '模板名' },
      description: { type: 'string', title: '模板描述' },
      format: { type: 'string', title: '格式' },
      category: { type: 'string', title: '分类' },
      content: { type: 'string', title: '模板正文' },
      is_public: { type: 'boolean', title: '是否公开模板' },
    } },
  { name: '删除文档模板', p: '/api/v1/wiki/spaces/templates/{id}', method: 'DELETE', order: 290, params: { path: [idPath] } },

  // 通知
  { name: '通知列表', p: '/api/v1/wiki/spaces/notifications', method: 'GET', order: 300, params: { query: [pageQuery, pageSizeQuery] } },
  { name: '全部通知标记已读', p: '/api/v1/wiki/spaces/notifications/read-all', method: 'PUT', order: 310 },
  { name: '通知未读数', p: '/api/v1/wiki/spaces/notifications/unread-count', method: 'GET', order: 320 },
  { name: '标记通知已读', p: '/api/v1/wiki/spaces/notifications/{id}/read', method: 'PUT', order: 330, params: { path: [idPath] } },

  // 订阅
  { name: '我的文档订阅列表', p: '/api/v1/wiki/spaces/subscriptions', method: 'GET', order: 340 },
  { name: '订阅文档', p: '/api/v1/wiki/documents/{id}/subscribe', method: 'POST', order: 350,
    params: { path: [docIdPath], query: [{ name: 'notify_type', type: 'string', description: '通知方式（默认 all）', required: false, enable: true }] } },
  { name: '取消订阅文档', p: '/api/v1/wiki/documents/{id}/subscribe', method: 'DELETE', order: 360, params: { path: [docIdPath] } },

  // 文档标签绑定
  { name: '为文档绑定标签', p: '/api/v1/wiki/documents/{id}/tags/{tagId}', method: 'POST', order: 370,
    params: { path: [docIdPath, { name: 'tagId', type: 'string', description: '标签ID', required: true, enable: true }] } },
  { name: '移除文档标签', p: '/api/v1/wiki/documents/{id}/tags/{tagId}', method: 'DELETE', order: 380,
    params: { path: [docIdPath, { name: 'tagId', type: 'string', description: '标签ID', required: true, enable: true }] } },
  { name: '文档标签列表', p: '/api/v1/wiki/documents/{id}/tags', method: 'GET', order: 390, params: { path: [docIdPath] } },

  // 评论
  { name: '发表评论', p: '/api/v1/wiki/documents/{id}/comments', method: 'POST', order: 400,
    params: { path: [docIdPath] },
    bodyProps: {
      parent_id: { type: 'string', title: '父评论ID（回复时传）' },
      content: { type: 'string', title: '评论内容' },
      mention_ids: { type: 'string', title: '@提及用户ID（逗号分隔）' },
    }, bodyRequired: ['content'] },
  { name: '评论树列表', p: '/api/v1/wiki/documents/{id}/comments', method: 'GET', order: 410, params: { path: [docIdPath] } },
  { name: '更新评论', p: '/api/v1/wiki/documents/comments/{id}', method: 'PUT', order: 420,
    params: { path: [idPath] }, bodyProps: { content: { type: 'string', title: '评论内容' } }, bodyRequired: ['content'],
    description: '仅评论作者可更新' },
  { name: '删除评论', p: '/api/v1/wiki/documents/comments/{id}', method: 'DELETE', order: 430,
    params: { path: [idPath] }, description: '作者本人或具备节点 delete 权限者' },

  // 分享管理
  { name: '创建文档分享', p: '/api/v1/wiki/documents/{id}/share', method: 'POST', order: 440,
    params: { path: [docIdPath] },
    bodyProps: {
      password: { type: 'string', title: '访问密码（留空为无密码）' },
      expire_at: { type: 'string', title: '过期时间 ISO8601' },
      max_views: { type: 'integer', title: '最大访问次数（0=不限）' },
      allow_download: { type: 'boolean', title: '是否允许下载原文' },
    } },
  { name: '文档分享列表', p: '/api/v1/wiki/documents/{id}/shares', method: 'GET', order: 450, params: { path: [docIdPath] } },
  { name: '删除分享', p: '/api/v1/wiki/documents/shares/{id}', method: 'DELETE', order: 460,
    params: { path: [idPath] }, description: 'id 为分享记录主键，需文档 update 权限' },

  // 公开分享落地（免登录）
  { name: '公开访问分享文档', p: '/api/v1/wiki/share/{token}', method: 'GET', order: 470,
    params: { path: [{ name: 'token', type: 'string', description: '分享 Token', required: true, enable: true }],
      query: [{ name: 'password', type: 'string', description: '访问密码（有密码时必填）', required: false, enable: true }] },
    description: '免登录公开接口（挂载于公开路由分组）。校验 token/有效期/最大访问次数/密码，通过后返回标题、content、content_html、format、allow_download、view_count 等只读数据；访问计数在密码通过后原子自增。文档已被删除（回收站）时返回 403' },

  // 统计与访问日志
  { name: '文档统计', p: '/api/v1/wiki/documents/{id}/stats', method: 'GET', order: 480, params: { path: [docIdPath] },
    description: '返回 total_views / total_edits / total_downloads / total_shares / unique_viewers / last_viewed_at' },
  { name: '文档访问日志', p: '/api/v1/wiki/documents/{id}/access-logs', method: 'GET', order: 490,
    params: { path: [docIdPath], query: [pageQuery, pageSizeQuery] } },

  // 编辑锁
  { name: '获取编辑锁', p: '/api/v1/wiki/documents/{id}/edit-lock', method: 'POST', order: 500,
    params: { path: [docIdPath] }, description: '抢编辑锁；被他人持有时返回冲突信息（默认 30 分钟过期）' },
  { name: '刷新编辑锁', p: '/api/v1/wiki/documents/{id}/edit-lock', method: 'PUT', order: 510,
    params: { path: [docIdPath] }, description: '延长当前持有锁的过期时间' },
  { name: '释放编辑锁', p: '/api/v1/wiki/documents/{id}/edit-lock', method: 'DELETE', order: 520,
    params: { path: [docIdPath] } },
  { name: '查询编辑锁状态', p: '/api/v1/wiki/documents/{id}/edit-lock', method: 'GET', order: 530,
    params: { path: [docIdPath] }, description: '返回 document_id / user_id / user_name / locked_at / expires_at / can_edit' },

  // 导入导出
  { name: '导出文档', p: '/api/v1/wiki/documents/{id}/export', method: 'POST', order: 540,
    params: { path: [docIdPath] },
    bodyProps: { format: { type: 'string', title: '导出格式：markdown / pdf / html' } },
    bodyRequired: ['format'],
    description: '响应直接回写文件流（Content-Disposition: attachment）' },
  { name: '导入文档', p: '/api/v1/wiki/documents/{id}/import', method: 'POST', order: 550,
    params: { path: [docIdPath] },
    bodyProps: null,
    description: 'multipart/form-data，字段 file（必填）与 format（默认 markdown）；文件内容作为新修订写入目标文档' },

  // 版本对比
  { name: '版本对比', p: '/api/v1/wiki/documents/{id}/revisions/compare', method: 'GET', order: 560,
    params: { path: [docIdPath], query: [
      { name: 'version1', type: 'string', description: '旧版本号', required: true, enable: true },
      { name: 'version2', type: 'string', description: '新版本号', required: true, enable: true },
    ] },
    description: '行级对比，返回 { old_version, new_version, diffs[] }，diffs 含 type/line_num/content/old_line/new_line' },
].map((d) => ({ ...d }));

let added = 0;
for (const spec of items) {
  const { name, p, method, order, params, bodyProps, bodyRequired, description } = spec;
  const item = makeApiRequest({ path: p, method, name, order, params, bodyProps, bodyRequired, description });
  if (item) {
    kb.items.push(item);
    added += 1;
  } else {
    console.log(`跳过（已存在）：${method.toUpperCase()} ${p}`);
  }
}

// ---------- 回收站接口语义刷新（真实恢复 / 级联物理删除 / 软删进回收站） ----------
const descMap = {
  '507439995': '列出当前租户回收站项目（含已软删的空间/目录/文档），支持 space_id 与 item_type 过滤、分页；每项含 deleted_by / deleted_at / expires_at（默认 30 天）',
  '507439996': '从回收站恢复：将软删实体真实还原（目录恢复会递归还原整棵软删子树及文档内容），成功后再移除回收站记录；实体已不存在时返回 404',
  '507439997': '永久删除回收站项目：级联物理清除实体及其关联数据（版本/附件/标签/评论/分享/编辑锁等），不可恢复',
};
for (const item of kb.items) {
  if (!item.api) continue;
  if (descMap[item.api.id]) item.api.description = descMap[item.api.id];
}

// 软删落回收站提示（按 apifox item id 精确定位，避免误伤子资源删除接口）
const deleteDescMap = {
  '507439978': '删除空间（软删）：空间数据进入回收站，可在 /api/v1/wiki/trash 恢复或永久删除',
  '507439983': '删除节点及其子树（软删）：节点与子树内文档进入回收站，可在 /api/v1/wiki/trash 恢复（恢复时整棵子树一并还原）',
  '507439990': '删除文档（软删）：文档进入回收站，可在 /api/v1/wiki/trash 恢复或永久删除；删除后原分享链接立即失效',
};
for (const item of kb.items) {
  if (!item.api) continue;
  if (deleteDescMap[item.api.id]) item.api.description = deleteDescMap[item.api.id];
}

fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8');
console.log(`完成：知识库文件夹新增 ${added} 个接口，并已刷新回收站/删除接口的语义描述`);
