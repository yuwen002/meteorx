/* eslint-disable */
// 向 MeteorX-backend.openapi.json 追加告警管理和会话分析接口
const fs = require('fs');
const path = require('path');

const file = path.join(__dirname, '..', 'docs', 'apifox', 'MeteorX-backend.openapi.json');
const data = JSON.parse(fs.readFileSync(file, 'utf-8'));

// ---------- 添加 tags ----------
const newTags = [
  {
    name: '告警管理',
    description: '审计告警规则管理和告警记录查询'
  },
  {
    name: '会话分析',
    description: '用户会话追踪和操作时间线分析'
  }
];

newTags.forEach(tag => {
  if (!data.tags.find(t => t.name === tag.name)) {
    data.tags.push(tag);
  }
});

// ---------- 添加 paths ----------
const newPaths = {
  // 告警规则
  '/api/v1/audit/alert-rules': {
    post: {
      summary: '创建告警规则',
      tags: ['告警管理'],
      requestBody: {
        required: true,
        content: {
          'application/json': {
            schema: {
              $ref: '#/components/schemas/CreateAlertRuleReq'
            }
          }
        }
      },
      responses: {
        '201': {
          description: '创建成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer', example: 201 },
                  message: { type: 'string', example: '创建成功' },
                  data: { $ref: '#/components/schemas/AlertRuleResp' }
                }
              }
            }
          }
        }
      }
    },
    get: {
      summary: '获取所有告警规则',
      tags: ['告警管理'],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer', example: 200 },
                  data: {
                    type: 'array',
                    items: { $ref: '#/components/schemas/AlertRuleResp' }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  '/api/v1/audit/alert-rules/{id}': {
    get: {
      summary: '告警规则详情',
      tags: ['告警管理'],
      parameters: [
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: { $ref: '#/components/schemas/AlertRuleResp' }
                }
              }
            }
          }
        }
      }
    },
    put: {
      summary: '更新告警规则',
      tags: ['告警管理'],
      parameters: [
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } }
      ],
      requestBody: {
        required: true,
        content: {
          'application/json': {
            schema: {
              $ref: '#/components/schemas/UpdateAlertRuleReq'
            }
          }
        }
      },
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: { $ref: '#/components/schemas/AlertRuleResp' }
                }
              }
            }
          }
        }
      }
    },
    delete: {
      summary: '删除告警规则',
      tags: ['告警管理'],
      parameters: [
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  message: { type: 'string' }
                }
              }
            }
          }
        }
      }
    }
  },
  '/api/v1/audit/alerts': {
    get: {
      summary: '告警记录列表',
      tags: ['告警管理'],
      parameters: [
        { name: 'page', in: 'query', schema: { type: 'integer', default: 1 } },
        { name: 'page_size', in: 'query', schema: { type: 'integer', default: 20 } },
        { name: 'rule_id', in: 'query', schema: { type: 'string' } },
        { name: 'user_id', in: 'query', schema: { type: 'string' } },
        { name: 'risk_level', in: 'query', schema: { type: 'string', enum: ['low', 'medium', 'high', 'critical'] } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: {
                    type: 'object',
                    properties: {
                      items: {
                        type: 'array',
                        items: { $ref: '#/components/schemas/AlertResp' }
                      },
                      total: { type: 'integer' }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  '/api/v1/audit/alerts/{id}': {
    get: {
      summary: '告警记录详情',
      tags: ['告警管理'],
      parameters: [
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: { $ref: '#/components/schemas/AlertResp' }
                }
              }
            }
          }
        }
      }
    }
  },
  // 会话分析
  '/api/v1/audit/sessions': {
    get: {
      summary: '会话摘要列表',
      tags: ['会话分析'],
      parameters: [
        { name: 'page', in: 'query', schema: { type: 'integer', default: 1 } },
        { name: 'page_size', in: 'query', schema: { type: 'integer', default: 20 } },
        { name: 'user_id', in: 'query', schema: { type: 'string' } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: {
                    type: 'object',
                    properties: {
                      items: {
                        type: 'array',
                        items: { $ref: '#/components/schemas/SessionSummaryResp' }
                      },
                      total: { type: 'integer' }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  '/api/v1/audit/sessions/{id}/logs': {
    get: {
      summary: '获取会话的所有日志',
      tags: ['会话分析'],
      parameters: [
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } }
      ],
      responses: {
        '200': {
          description: '成功',
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  code: { type: 'integer' },
                  data: {
                    allOf: [
                      { $ref: '#/components/schemas/SessionSummaryResp' },
                      {
                        type: 'object',
                        properties: {
                          logs: {
                            type: 'array',
                            items: { $ref: '#/components/schemas/AuditLogResp' }
                          }
                        }
                      }
                    ]
                  }
                }
              }
            }
          }
        }
      }
    }
  }
};

// 合并 paths
Object.assign(data.paths, newPaths);

// ---------- 添加 schemas ----------
const newSchemas = {
  CreateAlertRuleReq: {
    type: 'object',
    required: ['name', 'trigger_type', 'trigger_value', 'notify_channels', 'notify_targets'],
    properties: {
      name: { type: 'string', title: '规则名称' },
      description: { type: 'string', title: '规则描述' },
      enabled: { type: 'boolean', title: '是否启用', default: true },
      trigger_type: { type: 'string', title: '触发类型', enum: ['risk_level', 'action', 'user'] },
      trigger_value: { type: 'string', title: '触发值' },
      notify_channels: {
        type: 'array',
        title: '通知渠道',
        items: { type: 'string', enum: ['email', 'dingtalk', 'wechat', 'webhook'] }
      },
      notify_targets: {
        type: 'array',
        title: '通知目标',
        items: { type: 'string' }
      },
      notify_template: { type: 'string', title: '通知模板' },
      cooldown_minutes: { type: 'integer', title: '冷却时间（分钟）', default: 30 }
    }
  },
  UpdateAlertRuleReq: {
    type: 'object',
    properties: {
      name: { type: 'string', title: '规则名称' },
      description: { type: 'string', title: '规则描述' },
      enabled: { type: 'boolean', title: '是否启用' },
      trigger_type: { type: 'string', title: '触发类型', enum: ['risk_level', 'action', 'user'] },
      trigger_value: { type: 'string', title: '触发值' },
      notify_channels: {
        type: 'array',
        title: '通知渠道',
        items: { type: 'string', enum: ['email', 'dingtalk', 'wechat', 'webhook'] }
      },
      notify_targets: {
        type: 'array',
        title: '通知目标',
        items: { type: 'string' }
      },
      notify_template: { type: 'string', title: '通知模板' },
      cooldown_minutes: { type: 'integer', title: '冷却时间（分钟）' }
    }
  },
  AlertRuleResp: {
    type: 'object',
    properties: {
      id: { type: 'string', title: '规则ID' },
      name: { type: 'string', title: '规则名称' },
      description: { type: 'string', title: '规则描述' },
      enabled: { type: 'boolean', title: '是否启用' },
      trigger_type: { type: 'string', title: '触发类型' },
      trigger_value: { type: 'string', title: '触发值' },
      notify_channels: { type: 'array', title: '通知渠道', items: { type: 'string' } },
      notify_targets: { type: 'array', title: '通知目标', items: { type: 'string' } },
      notify_template: { type: 'string', title: '通知模板' },
      cooldown_minutes: { type: 'integer', title: '冷却时间（分钟）' },
      created_at: { type: 'string', format: 'date-time', title: '创建时间' },
      updated_at: { type: 'string', format: 'date-time', title: '更新时间' }
    }
  },
  AlertResp: {
    type: 'object',
    properties: {
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
      notify_time: { type: 'string', format: 'date-time', title: '通知时间' },
      created_at: { type: 'string', format: 'date-time', title: '创建时间' }
    }
  },
  SessionSummaryResp: {
    type: 'object',
    properties: {
      session_id: { type: 'string', title: '会话ID' },
      user_id: { type: 'string', title: '用户ID' },
      username: { type: 'string', title: '用户名' },
      total_requests: { type: 'integer', title: '总请求数' },
      success_count: { type: 'integer', title: '成功次数' },
      failure_count: { type: 'integer', title: '失败次数' },
      avg_duration: { type: 'integer', title: '平均耗时（毫秒）' },
      first_request: { type: 'string', format: 'date-time', title: '首次请求时间' },
      last_request: { type: 'string', format: 'date-time', title: '最后请求时间' },
      duration_minutes: { type: 'integer', title: '会话时长（分钟）' }
    }
  }
};

// 合并 schemas
Object.assign(data.components.schemas, newSchemas);

// 写入文件
fs.writeFileSync(file, JSON.stringify(data, null, 2), 'utf-8');
console.log('OpenAPI JSON 更新完成：已追加 告警管理 / 会话分析 接口和 Schema');