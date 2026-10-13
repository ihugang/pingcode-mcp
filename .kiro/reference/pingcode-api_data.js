define({
  "publication": {
    "status": "public-contract-only",
    "description": "公开发布用的 PingCode REST API 契约摘要；不包含租户、项目、工作项、运行快照、操作时间、内部链接或本机信息。"
  },
  "api": [
    {
      "version": "public",
      "type": "GET",
      "url": "/v1/auth/token",
      "title": "获取企业令牌",
      "permission": ["企业令牌"],
      "scopes": [],
      "group": "客户端凭据",
      "name": "获取企业令牌",
      "parameter": {
        "fields": {
          "Query": [
            {"field": "grant_type", "type": "String", "optional": false, "allowedValues": ["client_credentials"]},
            {"field": "client_id", "type": "String", "optional": false},
            {"field": "client_secret", "type": "String", "optional": false}
          ]
        }
      },
      "success": {
        "fields": {
          "资源属性": [
            {"field": "access_token", "type": "String", "optional": false},
            {"field": "token_type", "type": "String", "optional": false},
            {"field": "expires_in", "type": "String|Number", "optional": false}
          ]
        }
      }
    },
    {
      "version": "public",
      "type": "GET",
      "url": "/v1/pjm/projects",
      "title": "获取项目列表",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:read:pjm:project"],
      "group": "项目",
      "name": "获取项目列表",
      "parameter": {
        "fields": {
          "Query": [
            {"field": "scope_type", "type": "String", "optional": true},
            {"field": "scope_id", "type": "String", "optional": true},
            {"field": "keywords", "type": "String", "optional": true},
            {"field": "member_type", "type": "String", "optional": true},
            {"field": "member_id", "type": "String", "optional": true},
            {"field": "type", "type": "String", "optional": true},
            {"field": "created_between", "type": "String", "optional": true},
            {"field": "updated_between", "type": "String", "optional": true},
            {"field": "include_deleted", "type": "Boolean", "optional": true},
            {"field": "include_archived", "type": "Boolean", "optional": true},
            {"field": "page_size", "type": "Number", "optional": true},
            {"field": "page_index", "type": "Number", "optional": true}
          ]
        }
      },
      "success": {"fields": {"资源属性": [
        {"field": "page_size", "type": "Number", "optional": false},
        {"field": "page_index", "type": "Number", "optional": false},
        {"field": "total", "type": "Number", "optional": false},
        {"field": "values", "type": "Object[]", "optional": false}
      ]}}
    },
    {
      "version": "public",
      "type": "GET",
      "url": "/v1/pjm/projects/{project_id}",
      "title": "获取一个项目",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:read:pjm:project"],
      "group": "项目",
      "name": "获取一个项目",
      "parameter": {"fields": {"Path": [
        {"field": "project_id", "type": "String", "optional": false}
      ], "Query": [
        {"field": "include_deleted", "type": "Boolean", "optional": true},
        {"field": "include_archived", "type": "Boolean", "optional": true}
      ]}},
      "success": {"fields": {"资源属性": [
        {"field": "id", "type": "String", "optional": false},
        {"field": "name", "type": "String", "optional": false},
        {"field": "identifier", "type": "String", "optional": false},
        {"field": "type", "type": "String", "optional": false},
        {"field": "state", "type": "Object", "optional": false},
        {"field": "assignee", "type": "Object", "optional": false},
        {"field": "created_at", "type": "Number", "optional": false},
        {"field": "updated_at", "type": "Number", "optional": false}
      ]}}
    },
    {
      "version": "public",
      "type": "GET",
      "url": "/v1/pjm/workitems",
      "title": "获取工作项列表",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:read:pjm:workitem"],
      "group": "工作项",
      "name": "获取工作项列表",
      "parameter": {"fields": {"Query": [
        {"field": "identifier", "type": "String", "optional": true},
        {"field": "project_id", "type": "String", "optional": true},
        {"field": "type_id", "type": "String", "optional": true},
        {"field": "parent_id", "type": "String", "optional": true},
        {"field": "assignee_id", "type": "String", "optional": true},
        {"field": "state_id", "type": "String", "optional": true},
        {"field": "priority_id", "type": "String", "optional": true},
        {"field": "keywords", "type": "String", "optional": true},
        {"field": "include_deleted", "type": "Boolean", "optional": true},
        {"field": "include_archived", "type": "Boolean", "optional": true},
        {"field": "page_size", "type": "Number", "optional": true},
        {"field": "page_index", "type": "Number", "optional": true}
      ]}},
      "success": {"fields": {"资源属性": [
        {"field": "page_size", "type": "Number", "optional": false},
        {"field": "page_index", "type": "Number", "optional": false},
        {"field": "total", "type": "Number", "optional": false},
        {"field": "values", "type": "Object[]", "optional": false}
      ]}}
    },
    {
      "version": "public",
      "type": "GET",
      "url": "/v1/pjm/workitems/{workitem_id}",
      "title": "获取一个工作项",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:read:pjm:workitem"],
      "group": "工作项",
      "name": "获取一个工作项",
      "parameter": {"fields": {"Path": [
        {"field": "workitem_id", "type": "String", "optional": false}
      ], "Query": [
        {"field": "include_deleted", "type": "Boolean", "optional": true},
        {"field": "include_archived", "type": "Boolean", "optional": true}
      ]}},
      "success": {"fields": {"资源属性": [
        {"field": "id", "type": "String", "optional": false},
        {"field": "identifier", "type": "String", "optional": false},
        {"field": "title", "type": "String", "optional": false},
        {"field": "type", "type": "String", "optional": false},
        {"field": "state", "type": "Object", "optional": false},
        {"field": "priority", "type": "Object", "optional": false},
        {"field": "assignee", "type": "Object", "optional": false},
        {"field": "created_at", "type": "Number", "optional": false},
        {"field": "updated_at", "type": "Number", "optional": false}
      ]}}
    },
    {
      "version": "public",
      "type": "POST",
      "url": "/v1/pjm/workitems",
      "title": "创建一个工作项",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:write:pjm:workitem"],
      "group": "工作项",
      "name": "创建一个工作项",
      "parameter": {"fields": {"Body": [
        {"field": "project_id", "type": "String", "optional": false},
        {"field": "type_id", "type": "String", "optional": false},
        {"field": "title", "type": "String", "optional": false},
        {"field": "description", "type": "String", "optional": true},
        {"field": "start_at", "type": "Number", "optional": true},
        {"field": "end_at", "type": "Number", "optional": true},
        {"field": "priority_id", "type": "String", "optional": true},
        {"field": "state_id", "type": "String", "optional": true},
        {"field": "assignee_id", "type": "String", "optional": true},
        {"field": "parent_id", "type": "String", "optional": true},
        {"field": "properties", "type": "Object", "optional": true}
      ]}},
      "success": {"fields": {"资源属性": [
        {"field": "id", "type": "String", "optional": false},
        {"field": "identifier", "type": "String", "optional": false},
        {"field": "title", "type": "String", "optional": false}
      ]}}
    },
    {
      "version": "public",
      "type": "PATCH",
      "url": "/v1/pjm/workitems/{workitem_id}",
      "title": "部分更新一个工作项",
      "permission": ["企业令牌/用户令牌"],
      "scopes": ["pcp:write:pjm:workitem"],
      "group": "工作项",
      "name": "部分更新一个工作项",
      "parameter": {"fields": {"Path": [
        {"field": "workitem_id", "type": "String", "optional": false}
      ], "Body": [
        {"field": "title", "type": "String", "optional": true},
        {"field": "description", "type": "String", "optional": true},
        {"field": "start_at", "type": "Number", "optional": true},
        {"field": "end_at", "type": "Number", "optional": true},
        {"field": "priority_id", "type": "String", "optional": true},
        {"field": "state_id", "type": "String", "optional": true},
        {"field": "assignee_id", "type": "String", "optional": true},
        {"field": "parent_id", "type": "String", "optional": true},
        {"field": "properties", "type": "Object", "optional": true}
      ]}},
      "success": {"fields": {"资源属性": [
        {"field": "id", "type": "String", "optional": false},
        {"field": "identifier", "type": "String", "optional": false},
        {"field": "title", "type": "String", "optional": false},
        {"field": "updated_at", "type": "Number", "optional": false}
      ]}}
    }
  ]
});
