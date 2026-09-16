package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	path := `C:\works\go_works\meteorx\docs\apifox\MeteorX-backend.openapi.json`
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal: %v\n", err)
		os.Exit(1)
	}

	paths := doc["paths"].(map[string]any)

	type pathOp struct {
		method  string
		summary string
		tag     string
		desc    string
		path    string
		params  []map[string]any
		reqBody string // schema ref
		resp    string // schema ref, "" for none
		octet   bool
		multipart bool
	}

	ops := []pathOp{
		// 12.1 标签
		{"post", "创建标签", "知识库", "", "/api/v1/wiki/spaces/tags", nil, "#/components/schemas/CreateTagReq", "#/components/schemas/ApiTagResp", false, false},
		{"get", "标签列表", "知识库", "", "/api/v1/wiki/spaces/tags", nil, "", "#/components/schemas/ApiListTagResp", false, false},
		{"delete", "删除标签", "知识库", "", "/api/v1/wiki/spaces/tags/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "标签ID"}}, "", "#/components/schemas/ApiSuccess", false, false},

		// 12.2 文档标签绑定
		{"get", "文档标签列表", "知识库", "", "/api/v1/wiki/documents/{id}/tags", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiListDocumentTagResp", false, false},
		{"post", "为文档绑定标签", "知识库", "", "/api/v1/wiki/documents/{id}/tags/{tagId}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}, {"name": "tagId", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "标签ID"}}, "", "#/components/schemas/ApiSuccess", false, false},
		{"delete", "移除文档标签", "知识库", "", "/api/v1/wiki/documents/{id}/tags/{tagId}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}, {"name": "tagId", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "标签ID"}}, "", "#/components/schemas/ApiSuccess", false, false},

		// 12.3 评论
		{"post", "创建评论", "知识库", "", "/api/v1/wiki/documents/{id}/comments", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "#/components/schemas/CreateCommentReq", "#/components/schemas/ApiCommentResp", false, false},
		{"get", "评论列表（树状）", "知识库", "", "/api/v1/wiki/documents/{id}/comments", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiListCommentResp", false, false},
		{"put", "更新评论", "知识库", "", "/api/v1/wiki/documents/comments/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "评论ID"}}, "#/components/schemas/UpdateCommentReq", "#/components/schemas/ApiCommentResp", false, false},
		{"delete", "删除评论", "知识库", "", "/api/v1/wiki/documents/comments/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "评论ID"}}, "", "#/components/schemas/ApiSuccess", false, false},

		// 12.4 分享
		{"post", "创建分享链接", "知识库", "", "/api/v1/wiki/documents/{id}/share", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "#/components/schemas/ShareLinkReq", "#/components/schemas/ApiShareLinkResp", false, false},
		{"get", "文档分享列表", "知识库", "", "/api/v1/wiki/documents/{id}/shares", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiListShareLinkResp", false, false},
		{"delete", "删除分享链接", "知识库", "", "/api/v1/wiki/documents/shares/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "分享ID"}}, "", "#/components/schemas/ApiSuccess", false, false},
		{"get", "公开访问分享文档（免登录）", "公共", "校验 token、过期时间、最大次数、密码；成功后自增浏览计数", "/api/v1/wiki/share/{token}", []map[string]any{{"name": "token", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "分享 token"}, {"name": "password", "in": "query", "required": false, "schema": map[string]any{"type": "string"}, "description": "分享密码"}}, "", "#/components/schemas/ApiSharedDocumentResp", false, false},

		// 12.5 模板
		{"post", "创建模板", "知识库", "", "/api/v1/wiki/spaces/templates", nil, "#/components/schemas/CreateTemplateReq", "#/components/schemas/ApiDocumentTemplateResp", false, false},
		{"get", "模板列表", "知识库", "", "/api/v1/wiki/spaces/templates", nil, "", "#/components/schemas/ApiListDocumentTemplateResp", false, false},
		{"get", "模板详情", "知识库", "", "/api/v1/wiki/spaces/templates/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "模板ID"}}, "", "#/components/schemas/ApiDocumentTemplateResp", false, false},
		{"put", "更新模板", "知识库", "", "/api/v1/wiki/spaces/templates/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "模板ID"}}, "#/components/schemas/UpdateTemplateReq", "#/components/schemas/ApiDocumentTemplateResp", false, false},
		{"delete", "删除模板", "知识库", "", "/api/v1/wiki/spaces/templates/{id}", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "模板ID"}}, "", "#/components/schemas/ApiSuccess", false, false},

		// 12.6 统计与日志
		{"get", "文档统计数据", "知识库", "", "/api/v1/wiki/documents/{id}/stats", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiDocumentStatsResp", false, false},
		{"get", "访问日志（分页）", "知识库", "", "/api/v1/wiki/documents/{id}/access-logs", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}, {"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "example": 1}}, {"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "example": 10}}}, "", "#/components/schemas/ApiListDocumentAccessLogResp", false, false},

		// 12.7 订阅与通知
		{"post", "订阅文档", "知识库", "", "/api/v1/wiki/documents/{id}/subscribe", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}, {"name": "notify_type", "in": "query", "required": false, "schema": map[string]any{"type": "string"}, "description": "通知类型，默认 all"}}, "", "#/components/schemas/ApiSubscriptionResp", false, false},
		{"delete", "取消订阅", "知识库", "", "/api/v1/wiki/documents/{id}/subscribe", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiSuccess", false, false},
		{"get", "当前用户订阅列表", "知识库", "", "/api/v1/wiki/spaces/subscriptions", nil, "", "#/components/schemas/ApiListSubscriptionResp", false, false},
		{"get", "通知列表（分页）", "知识库", "", "/api/v1/wiki/spaces/notifications", []map[string]any{{"name": "page", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "example": 1}}, {"name": "page_size", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "example": 10}}}, "", "#/components/schemas/ApiListNotificationResp", false, false},
		{"put", "标记通知已读", "知识库", "", "/api/v1/wiki/spaces/notifications/{id}/read", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "通知ID"}}, "", "#/components/schemas/ApiSuccess", false, false},
		{"put", "全部标记已读", "知识库", "", "/api/v1/wiki/spaces/notifications/read-all", nil, "", "#/components/schemas/ApiSuccess", false, false},
		{"get", "未读通知数", "知识库", "", "/api/v1/wiki/spaces/notifications/unread-count", nil, "", "#/components/schemas/ApiUnreadCountResp", false, false},

		// 12.8 编辑锁
		{"post", "获取编辑锁", "知识库", "", "/api/v1/wiki/documents/{id}/edit-lock", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "#/components/schemas/AcquireEditLockReq", "#/components/schemas/ApiEditLockResp", false, false},
		{"put", "刷新编辑锁", "知识库", "", "/api/v1/wiki/documents/{id}/edit-lock", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiEditLockResp", false, false},
		{"delete", "释放编辑锁", "知识库", "", "/api/v1/wiki/documents/{id}/edit-lock", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiSuccess", false, false},
		{"get", "查询锁状态", "知识库", "", "/api/v1/wiki/documents/{id}/edit-lock", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "", "#/components/schemas/ApiEditLockResp", false, false},

		// 12.9 批量
		{"post", "批量节点操作（delete|move）", "知识库", "", "/api/v1/wiki/spaces/nodes/batch", nil, "#/components/schemas/BatchOperationReq", "#/components/schemas/ApiSuccess", false, false},

		// 12.10 版本对比
		{"get", "版本行级对比", "知识库", "", "/api/v1/wiki/documents/{id}/revisions/compare", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}, {"name": "version1", "in": "query", "required": true, "schema": map[string]any{"type": "integer"}, "description": "旧版本号"}, {"name": "version2", "in": "query", "required": true, "schema": map[string]any{"type": "integer"}, "description": "新版本号"}}, "", "#/components/schemas/ApiDiffResultResp", false, false},

		// 12.11 导入导出
		{"post", "导出文档（markdown|pdf|html）", "知识库", "响应直接回写文件流，Content-Disposition: attachment", "/api/v1/wiki/documents/{id}/export", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "文档ID"}}, "#/components/schemas/ExportDocumentReq", "", true, false},
		{"post", "导入文档（multipart/form-data）", "知识库", "将文件内容作为新修订写入目标文档", "/api/v1/wiki/documents/{id}/import", []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "目标文档ID"}}, "", "#/components/schemas/ApiImportDocumentResp", false, true},
	}

	for _, op := range ops {
		method := strings.ToLower(op.method)
		existing, ok := paths[op.path]
		if !ok {
			existing = map[string]any{}
			paths[op.path] = existing
		}
		ep := existing.(map[string]any)

		opObj := map[string]any{
			"summary": op.summary,
			"tags":    []string{op.tag},
		}
		if op.desc != "" {
			opObj["description"] = op.desc
		}
		if len(op.params) > 0 {
			opObj["parameters"] = op.params
		}
		if op.reqBody != "" {
			contentKey := "application/json"
			if op.multipart {
				contentKey = "multipart/form-data"
			}
			opObj["requestBody"] = map[string]any{
				"required": true,
				"content": map[string]any{
					contentKey: map[string]any{
						"schema": map[string]any{
							"$ref": op.reqBody,
						},
					},
				},
			}
		}

		resp := map[string]any{
			"description": "成功",
		}
		if op.octet {
			resp["content"] = map[string]any{
				"application/octet-stream": map[string]any{
					"schema": map[string]any{
						"type":   "string",
						"format": "binary",
					},
				},
			}
		} else if op.resp != "" {
			resp["content"] = map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"$ref": op.resp,
					},
				},
			}
		}
		opObj["responses"] = map[string]any{
			"200": resp,
		}

		ep[method] = opObj
	}

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	extraSchemas := buildExtraSchemas()
	for k, v := range extraSchemas {
		if _, exists := schemas[k]; !exists {
			schemas[k] = v
		}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("OK: added %d operations across %d paths, %d schemas\n",
		len(ops), countPathsWithOps(paths), len(extraSchemas))
}

func countPathsWithOps(paths map[string]any) int {
	n := 0
	for _, v := range paths {
		if m, ok := v.(map[string]any); ok {
			for k := range m {
				switch k {
				case "get", "post", "put", "delete", "patch":
					n++
				}
			}
		}
	}
	return n
}

func buildExtraSchemas() map[string]any {
	strType := map[string]any{"type": "string"}
	intType := map[string]any{"type": "integer"}
	boolType := map[string]any{"type": "boolean"}

	api := func(schema map[string]any) map[string]any {
		return map[string]any{
			"allOf": []any{
				map[string]any{"$ref": "#/components/schemas/ApiSuccess"},
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"data": schema,
					},
				},
			},
		}
	}

	apiList := func(itemSchema map[string]any) map[string]any {
		return api(map[string]any{
			"type": "array",
			"items": itemSchema,
		})
	}

	tagResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":        strType,
			"name":      strType,
			"color":     strType,
			"created_by": strType,
			"created_at": strType,
		},
	}

	commentResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":           strType,
			"document_id":  strType,
			"node_id":      strType,
			"parent_id":    strType,
			"content":      strType,
			"mention_ids":  map[string]any{"type": "array", "items": strType},
			"status":       strType,
			"created_by":   strType,
			"created_at":   strType,
			"updated_at":   strType,
			"replies":      map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/CommentResp"}},
		},
	}

	shareResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":             strType,
			"document_id":    strType,
			"token":          strType,
			"password":       strType,
			"expire_at":      strType,
			"max_views":      intType,
			"view_count":     intType,
			"allow_download": boolType,
			"created_at":     strType,
		},
	}

	templateResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":          strType,
			"name":        strType,
			"description": strType,
			"content":     strType,
			"format":      strType,
			"category":    strType,
			"is_public":   boolType,
			"created_by":  strType,
			"created_at":  strType,
			"updated_at":  strType,
		},
	}

	docStatsResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"total_views":      intType,
			"total_edits":      intType,
			"total_downloads":  intType,
			"total_shares":     intType,
			"unique_viewers":   intType,
			"last_viewed_at":   strType,
		},
	}

	accessLogResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":          strType,
			"document_id": strType,
			"user_id":     strType,
			"action":      strType,
			"ip_address":  strType,
			"user_agent":  strType,
			"created_at":  strType,
		},
	}

	subResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":          strType,
			"document_id": strType,
			"user_id":     strType,
			"notify_type": strType,
			"created_at":  strType,
		},
	}

	notifResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":           strType,
			"type":         strType,
			"title":        strType,
			"content":      strType,
			"related_id":   strType,
			"related_type": strType,
			"is_read":      boolType,
			"created_at":   strType,
		},
	}

	editLockResp := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"document_id": strType,
			"user_id":     strType,
			"user_name":   strType,
			"locked_at":   strType,
			"expires_at":  strType,
			"can_edit":    boolType,
		},
	}

	diffLine := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     strType,
			"line_num": intType,
			"content":  strType,
			"old_line": strType,
			"new_line": strType,
		},
	}

	diffResult := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"old_version": intType,
			"new_version": intType,
			"diffs":       map[string]any{"type": "array", "items": diffLine},
		},
	}

	sharedDoc := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"document_id":   strType,
			"title":         strType,
			"need_password": boolType,
			"content_html":  strType,
			"format":        strType,
			"current_ver":   intType,
			"updated_at":    strType,
		},
	}

	return map[string]any{
		// 请求体
		"CreateTagReq": map[string]any{
			"type":     "object",
			"required": []string{"name"},
			"properties": map[string]any{
				"name":  strType,
				"color": strType,
			},
		},
		"CreateCommentReq": map[string]any{
			"type":     "object",
			"required": []string{"content"},
			"properties": map[string]any{
				"document_id":  strType,
				"node_id":      strType,
				"parent_id":    strType,
				"content":      strType,
				"mention_ids":  map[string]any{"type": "array", "items": strType},
			},
		},
		"UpdateCommentReq": map[string]any{
			"type":     "object",
			"required": []string{"content"},
			"properties": map[string]any{
				"content": strType,
			},
		},
		"ShareLinkReq": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"document_id":   strType,
				"password":      strType,
				"expire_at":     strType,
				"max_views":     intType,
				"allow_download": boolType,
			},
		},
		"CreateTemplateReq": map[string]any{
			"type":     "object",
			"required": []string{"name", "content"},
			"properties": map[string]any{
				"name":        strType,
				"description": strType,
				"format":      strType,
				"category":    strType,
				"content":     strType,
				"is_public":   boolType,
			},
		},
		"UpdateTemplateReq": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        strType,
				"description": strType,
				"format":      strType,
				"category":    strType,
				"content":     strType,
				"is_public":   boolType,
			},
		},
		"AcquireEditLockReq": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"duration": intType,
			},
		},
		"BatchOperationReq": map[string]any{
			"type":     "object",
			"required": []string{"action", "node_ids"},
			"properties": map[string]any{
				"action":    map[string]any{"type": "string", "enum": []string{"delete", "move"}},
				"node_ids":  map[string]any{"type": "array", "items": strType},
				"target":    strType,
			},
		},
		"ExportDocumentReq": map[string]any{
			"type":     "object",
			"required": []string{"format"},
			"properties": map[string]any{
				"format": map[string]any{"type": "string", "enum": []string{"markdown", "pdf", "html"}},
			},
		},
		"ImportDocumentReq": map[string]any{
			"type": "object",
			"required": []string{"file"},
			"properties": map[string]any{
				"file":   map[string]any{"type": "string", "format": "binary"},
				"format": map[string]any{"type": "string", "enum": []string{"markdown"}},
			},
		},

		// 内部模型
		"TagResp":               tagResp,
		"DocumentTagResp":       map[string]any{"type": "object", "properties": map[string]any{"id": strType, "document_id": strType, "tag_id": strType, "tag": tagResp, "created_at": strType}},
		"CommentResp":           commentResp,
		"ShareLinkResp":         shareResp,
		"SharedDocumentResp":    sharedDoc,
		"DocumentTemplateResp": templateResp,
		"DocumentStatsResp":     docStatsResp,
		"DocumentAccessLogResp": accessLogResp,
		"SubscriptionResp":      subResp,
		"NotificationResp":      notifResp,
		"EditLockResp":          editLockResp,
		"DiffLine":              diffLine,
		"DiffResult":            diffResult,

		// Api* 包装类型
		"ApiTagResp":                      api(tagResp),
		"ApiListTagResp":                   apiList(tagResp),
		"ApiListDocumentTagResp":           apiList(map[string]any{"$ref": "#/components/schemas/DocumentTagResp"}),
		"ApiCommentResp":                   api(commentResp),
		"ApiListCommentResp":               apiList(commentResp),
		"ApiShareLinkResp":                 api(shareResp),
		"ApiListShareLinkResp":             apiList(shareResp),
		"ApiSharedDocumentResp":            api(sharedDoc),
		"ApiDocumentTemplateResp":          api(templateResp),
		"ApiListDocumentTemplateResp":      apiList(templateResp),
		"ApiDocumentStatsResp":             api(docStatsResp),
		"ApiListDocumentAccessLogResp":     apiList(accessLogResp),
		"ApiSubscriptionResp":              api(subResp),
		"ApiListSubscriptionResp":          apiList(subResp),
		"ApiListNotificationResp":          apiList(notifResp),
		"ApiEditLockResp":                  api(editLockResp),
		"ApiDiffResultResp":                api(diffResult),
		"ApiImportDocumentResp":            api(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"document_id": strType,
				"node_id":     strType,
				"filename":    strType,
				"size":        intType,
				"format":      strType,
			},
		}),
		"ApiUnreadCountResp": api(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"count": intType,
			},
		}),
	}
}