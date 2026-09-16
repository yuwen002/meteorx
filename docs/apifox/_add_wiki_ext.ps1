$path = "C:\works\go_works\meteorx\docs\apifox\MeteorX-backend.openapi.json"
$json = Get-Content $path -Raw -Encoding UTF8 | ConvertFrom-Json

$wikiExt = @{}

function New-Path {
    param([string]$Summary, [string]$Tag, [string]$Desc)
    $obj = @{
        summary = $Summary
        tags = @($Tag)
    }
    if ($Desc) { $obj.description = $Desc }
    return $obj
}

function Add-Param {
    param($Params, [string]$Name, [string]$In, [bool]$Required, [string]$Type, [string]$Desc, $Example)
    $p = @{ name = $Name; in = $In; required = $Required; schema = @{ type = $Type } }
    if ($Desc) { $p.description = $Desc }
    if ($null -ne $Example) { $p.schema.example = $Example }
    $Params.Add($p) | Out-Null
}

function New-Resp($SchemaRef) {
    return @{
        "200" = @{
            description = "成功"
            content = @{
                "application/json" = @{
                    schema = if ($SchemaRef) { @{ '$ref' = $SchemaRef } } else { @{ type = "object" } }
                }
            }
        }
    }
}

# ==================== 12.1 空间级标签 ====================
$wikiExt['/api/v1/wiki/spaces/tags'] = @{
    post = (New-Path "创建标签" "知识库")
    get  = (New-Path "标签列表" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/tags'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/CreateTagReq" } } }
}
$wikiExt['/api/v1/wiki/spaces/tags'].post.responses = New-Resp "#/components/schemas/ApiTagResp"
$wikiExt['/api/v1/wiki/spaces/tags'].get.responses = New-Resp "#/components/schemas/ApiListTagResp"

$wikiExt['/api/v1/wiki/spaces/tags/{id}'] = @{
    delete = (New-Path "删除标签" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/tags/{id}'].delete.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/spaces/tags/{id}'].delete.parameters "id" "path" $true "string" "标签ID"
$wikiExt['/api/v1/wiki/spaces/tags/{id}'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"

# ==================== 12.2 文档标签绑定 ====================
$wikiExt['/api/v1/wiki/documents/{id}/tags'] = @{
    get = (New-Path "文档标签列表" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/tags'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/tags'].get.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/tags'].get.responses = New-Resp "#/components/schemas/ApiListDocumentTagResp"

$wikiExt['/api/v1/wiki/documents/{id}/tags/{tagId}'] = @{
    post   = (New-Path "为文档绑定标签" "知识库")
    delete = (New-Path "移除文档标签" "知识库")
}
foreach ($m in @('post','delete')) {
    $wikiExt['/api/v1/wiki/documents/{id}/tags/{tagId}'].$m.parameters = @()
    Add-Param $wikiExt['/api/v1/wiki/documents/{id}/tags/{tagId}'].$m.parameters "id" "path" $true "string" "文档ID"
    Add-Param $wikiExt['/api/v1/wiki/documents/{id}/tags/{tagId}'].$m.parameters "tagId" "path" $true "string" "标签ID"
    $wikiExt['/api/v1/wiki/documents/{id}/tags/{tagId}'].$m.responses = New-Resp "#/components/schemas/ApiSuccess"
}

# ==================== 12.3 评论系统 ====================
$wikiExt['/api/v1/wiki/documents/{id}/comments'] = @{
    post = (New-Path "创建评论" "知识库")
    get  = (New-Path "评论列表（树状）" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/comments'].post.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/comments'].post.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/comments'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/CreateCommentReq" } } }
}
$wikiExt['/api/v1/wiki/documents/{id}/comments'].post.responses = New-Resp "#/components/schemas/ApiCommentResp"
$wikiExt['/api/v1/wiki/documents/{id}/comments'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/comments'].get.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/comments'].get.responses = New-Resp "#/components/schemas/ApiListCommentResp"

$wikiExt['/api/v1/wiki/documents/comments/{id}'] = @{
    put    = (New-Path "更新评论" "知识库")
    delete = (New-Path "删除评论" "知识库")
}
foreach ($m in @('put','delete')) {
    $wikiExt['/api/v1/wiki/documents/comments/{id}'].$m.parameters = @()
    Add-Param $wikiExt['/api/v1/wiki/documents/comments/{id}'].$m.parameters "id" "path" $true "string" "评论ID"
}
$wikiExt['/api/v1/wiki/documents/comments/{id}'].put.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/UpdateCommentReq" } } }
}
$wikiExt['/api/v1/wiki/documents/comments/{id}'].put.responses = New-Resp "#/components/schemas/ApiCommentResp"
$wikiExt['/api/v1/wiki/documents/comments/{id}'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"

# ==================== 12.4 分享链接 ====================
$wikiExt['/api/v1/wiki/documents/{id}/share'] = @{
    post = (New-Path "创建分享链接" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/share'].post.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/share'].post.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/share'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/ShareLinkReq" } } }
}
$wikiExt['/api/v1/wiki/documents/{id}/share'].post.responses = New-Resp "#/components/schemas/ApiShareLinkResp"

$wikiExt['/api/v1/wiki/documents/{id}/shares'] = @{
    get = (New-Path "文档分享列表" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/shares'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/shares'].get.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/shares'].get.responses = New-Resp "#/components/schemas/ApiListShareLinkResp"

$wikiExt['/api/v1/wiki/documents/shares/{id}'] = @{
    delete = (New-Path "删除分享链接" "知识库")
}
$wikiExt['/api/v1/wiki/documents/shares/{id}'].delete.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/shares/{id}'].delete.parameters "id" "path" $true "string" "分享ID"
$wikiExt['/api/v1/wiki/documents/shares/{id}'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"

$wikiExt['/api/v1/wiki/share/{token}'] = @{
    get = (New-Path "公开访问分享文档（免登录）" "公共" "校验 token、过期时间、最大次数、密码；成功后自增浏览计数")
}
$wikiExt['/api/v1/wiki/share/{token}'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/share/{token}'].get.parameters "token" "path" $true "string" "分享 token"
Add-Param $wikiExt['/api/v1/wiki/share/{token}'].get.parameters "password" "query" $false "string" "分享密码"
$wikiExt['/api/v1/wiki/share/{token}'].get.responses = New-Resp "#/components/schemas/ApiSharedDocumentResp"

# ==================== 12.5 文档模板 ====================
$wikiExt['/api/v1/wiki/spaces/templates'] = @{
    post = (New-Path "创建模板" "知识库")
    get  = (New-Path "模板列表" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/templates'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/CreateTemplateReq" } } }
}
$wikiExt['/api/v1/wiki/spaces/templates'].post.responses = New-Resp "#/components/schemas/ApiDocumentTemplateResp"
$wikiExt['/api/v1/wiki/spaces/templates'].get.responses = New-Resp "#/components/schemas/ApiListDocumentTemplateResp"

$wikiExt['/api/v1/wiki/spaces/templates/{id}'] = @{
    get    = (New-Path "模板详情" "知识库")
    put    = (New-Path "更新模板" "知识库")
    delete = (New-Path "删除模板" "知识库")
}
foreach ($m in @('get','put','delete')) {
    $wikiExt['/api/v1/wiki/spaces/templates/{id}'].$m.parameters = @()
    Add-Param $wikiExt['/api/v1/wiki/spaces/templates/{id}'].$m.parameters "id" "path" $true "string" "模板ID"
}
$wikiExt['/api/v1/wiki/spaces/templates/{id}'].put.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/UpdateTemplateReq" } } }
}
$wikiExt['/api/v1/wiki/spaces/templates/{id}'].get.responses = New-Resp "#/components/schemas/ApiDocumentTemplateResp"
$wikiExt['/api/v1/wiki/spaces/templates/{id}'].put.responses = New-Resp "#/components/schemas/ApiDocumentTemplateResp"
$wikiExt['/api/v1/wiki/spaces/templates/{id}'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"

# ==================== 12.6 访问统计与日志 ====================
$wikiExt['/api/v1/wiki/documents/{id}/stats'] = @{
    get = (New-Path "文档统计数据" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/stats'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/stats'].get.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/stats'].get.responses = New-Resp "#/components/schemas/ApiDocumentStatsResp"

$wikiExt['/api/v1/wiki/documents/{id}/access-logs'] = @{
    get = (New-Path "访问日志（分页）" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/access-logs'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/access-logs'].get.parameters "id" "path" $true "string" "文档ID"
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/access-logs'].get.parameters "page" "query" $false "integer" "" 1
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/access-logs'].get.parameters "page_size" "query" $false "integer" "" 10
$wikiExt['/api/v1/wiki/documents/{id}/access-logs'].get.responses = New-Resp "#/components/schemas/ApiListDocumentAccessLogResp"

# ==================== 12.7 订阅与通知 ====================
$wikiExt['/api/v1/wiki/documents/{id}/subscribe'] = @{
    post   = (New-Path "订阅文档" "知识库")
    delete = (New-Path "取消订阅" "知识库")
}
foreach ($m in @('post','delete')) {
    $wikiExt['/api/v1/wiki/documents/{id}/subscribe'].$m.parameters = @()
    Add-Param $wikiExt['/api/v1/wiki/documents/{id}/subscribe'].$m.parameters "id" "path" $true "string" "文档ID"
}
$wikiExt['/api/v1/wiki/documents/{id}/subscribe'].post.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/subscribe'].post.parameters "id" "path" $true "string" "文档ID"
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/subscribe'].post.parameters "notify_type" "query" $false "string" "通知类型，默认 all"
$wikiExt['/api/v1/wiki/documents/{id}/subscribe'].post.responses = New-Resp "#/components/schemas/ApiSubscriptionResp"
$wikiExt['/api/v1/wiki/documents/{id}/subscribe'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"

$wikiExt['/api/v1/wiki/spaces/subscriptions'] = @{
    get = (New-Path "当前用户订阅列表" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/subscriptions'].get.responses = New-Resp "#/components/schemas/ApiListSubscriptionResp"

$wikiExt['/api/v1/wiki/spaces/notifications'] = @{
    get = (New-Path "通知列表（分页）" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/notifications'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/spaces/notifications'].get.parameters "page" "query" $false "integer" "" 1
Add-Param $wikiExt['/api/v1/wiki/spaces/notifications'].get.parameters "page_size" "query" $false "integer" "" 10
$wikiExt['/api/v1/wiki/spaces/notifications'].get.responses = New-Resp "#/components/schemas/ApiListNotificationResp"

$wikiExt['/api/v1/wiki/spaces/notifications/{id}/read'] = @{
    put = (New-Path "标记通知已读" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/notifications/{id}/read'].put.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/spaces/notifications/{id}/read'].put.parameters "id" "path" $true "string" "通知ID"
$wikiExt['/api/v1/wiki/spaces/notifications/{id}/read'].put.responses = New-Resp "#/components/schemas/ApiSuccess"

$wikiExt['/api/v1/wiki/spaces/notifications/read-all'] = @{
    put = (New-Path "全部标记已读" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/notifications/read-all'].put.responses = New-Resp "#/components/schemas/ApiSuccess"

$wikiExt['/api/v1/wiki/spaces/notifications/unread-count'] = @{
    get = (New-Path "未读通知数" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/notifications/unread-count'].get.responses = New-Resp "#/components/schemas/ApiUnreadCountResp"

# ==================== 12.8 编辑锁 ====================
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'] = @{
    post   = (New-Path "获取编辑锁" "知识库")
    put    = (New-Path "刷新编辑锁" "知识库")
    delete = (New-Path "释放编辑锁" "知识库")
    get    = (New-Path "查询锁状态" "知识库")
}
foreach ($m in @('post','put','delete','get')) {
    $wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].$m.parameters = @()
    Add-Param $wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].$m.parameters "id" "path" $true "string" "文档ID"
}
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/AcquireEditLockReq" } } }
}
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].post.responses = New-Resp "#/components/schemas/ApiEditLockResp"
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].put.responses = New-Resp "#/components/schemas/ApiEditLockResp"
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].delete.responses = New-Resp "#/components/schemas/ApiSuccess"
$wikiExt['/api/v1/wiki/documents/{id}/edit-lock'].get.responses = New-Resp "#/components/schemas/ApiEditLockResp"

# ==================== 12.9 批量操作 ====================
$wikiExt['/api/v1/wiki/spaces/nodes/batch'] = @{
    post = (New-Path "批量节点操作（delete|move）" "知识库")
}
$wikiExt['/api/v1/wiki/spaces/nodes/batch'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/BatchOperationReq" } } }
}
$wikiExt['/api/v1/wiki/spaces/nodes/batch'].post.responses = New-Resp "#/components/schemas/ApiSuccess"

# ==================== 12.10 版本对比 ====================
$wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'] = @{
    get = (New-Path "版本行级对比" "知识库")
}
$wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'].get.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'].get.parameters "id" "path" $true "string" "文档ID"
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'].get.parameters "version1" "query" $true "integer" "旧版本号"
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'].get.parameters "version2" "query" $true "integer" "新版本号"
$wikiExt['/api/v1/wiki/documents/{id}/revisions/compare'].get.responses = New-Resp "#/components/schemas/ApiDiffResultResp"

# ==================== 12.11 导入导出 ====================
$wikiExt['/api/v1/wiki/documents/{id}/export'] = @{
    post = (New-Path "导出文档（markdown|pdf|html）" "知识库" "响应直接回写文件流，Content-Disposition: attachment")
}
$wikiExt['/api/v1/wiki/documents/{id}/export'].post.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/export'].post.parameters "id" "path" $true "string" "文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/export'].post.requestBody = @{
    required = $true
    content = @{ "application/json" = @{ schema = @{ '$ref' = "#/components/schemas/ExportDocumentReq" } } }
}
$wikiExt['/api/v1/wiki/documents/{id}/export'].post.responses = New-Resp "$null"
$wikiExt['/api/v1/wiki/documents/{id}/export'].post.responses."200".content."application/json" = @{}
$wikiExt['/api/v1/wiki/documents/{id}/export'].post.responses."200".content."application/octet-stream" = @{ schema = @{ type = "string"; format = "binary" } }

$wikiExt['/api/v1/wiki/documents/{id}/import'] = @{
    post = (New-Path "导入文档（multipart/form-data）" "知识库" "将文件内容作为新修订写入目标文档")
}
$wikiExt['/api/v1/wiki/documents/{id}/import'].post.parameters = @()
Add-Param $wikiExt['/api/v1/wiki/documents/{id}/import'].post.parameters "id" "path" $true "string" "目标文档ID"
$wikiExt['/api/v1/wiki/documents/{id}/import'].post.requestBody = @{
    required = $true
    content = @{ "multipart/form-data" = @{ schema = @{ '$ref' = "#/components/schemas/ImportDocumentReq" } } }
}
$wikiExt['/api/v1/wiki/documents/{id}/import'].post.responses = New-Resp "#/components/schemas/ApiImportDocumentResp"

foreach ($k in $wikiExt.Keys) {
    $json.paths | Add-Member -NotePropertyName $k -NotePropertyValue $wikiExt[$k]
}

$json | ConvertTo-Json -Depth 100 | Set-Content $path -Encoding UTF8
Write-Host "OK: Added $($wikiExt.Count) wiki extension paths"