> #301 / #1030 [用户确认的浏览器安装完整退休边界](architecture/legacy-browser-install-closure-retirement-boundary.md)仅替代下文属于该闭包的旧KEEP/安装指引；其它有效规则不变。用户已确认冻结设计 f5e560cd7 的 §1–§7（含 §4.2/§6.1），IMPLEMENTATION_READY；同一 #1030 按36路径整批实施，不机械拆批。退出旧预装基础镜像、两个仓库安装CLI及browserinstall/httpclient/timeout专属闭包；官方SDK标准安装、显式指纹浏览器/匹配Profile、当前能力与Store观察保持。无新安装器、alias或试用前置；workflow/所有inventory/既有实例数据凭据缓存进程零改，#301 OPEN。源码自检、原CI及最终复核另记同一PR，不代表产品或运行验收，合并另授。以下原Git正文、既有顶注、历史计数和勾选逐字保留。

> #301 / #1010 — USER_CONFIRMED / IMPLEMENTING. [完整退休边界](architecture/legacy-restored-debug-closure-retirement-boundary.md) §1–§7（含 §4.2/§6.1）替代下文仅属于本批的旧 KEEP：旧 Go Amazon/1688 手工浏览器执行树、五个 debug 入口、手机号探针、db-query/json-simplifier、公开 YAML 与隐式配置入口退出。Chromium 安装闭包归 `integration/browserinstall`，当前 Collector 配置合同原样保留。当前采集、图像、身份权限、Store 观察同步保持；实际消费配置的四个命令须显式 `-config`，worker canary 无需配置。tools module 仅保留原 CI lint 固定用途；test-all 仅测 root。下方原文保留为历史，不作为已退休入口的现行操作指引。源级/CI 不等于运行验收；workflow/完整 inventory/计数/ceilings/工具版本及既有资源零改，#301 OPEN，无新增试用前置。

# Amazon Listing Workbench API

> **Classification: SUPERSEDED / RETIRED** by user-confirmed #822,
> [complete Amazon closure](architecture/legacy-amazon-listing-closure-retirement-boundary.md).
> The old Task/Workbench routes and contract below are historical evidence only;
> they are not registered in current-application and must not be restored through
> a compatibility endpoint. Current Amazon platform API and source acquisition
> retain their owners. This source retirement does not operate a deployment or
> alter any existing database/table/row. The original protocol below is preserved.


## Task Queue

`GET /api/v1/amazon/listings/tasks`

Use this endpoint to build an operator queue instead of loading one task at a time.

Supported query params:

- `status=needs_review,failed`
- `action=fill_brand`
- `field=brand`
- `severity=warning`
- `source=llm`
- `needs_human=true`
- `limit=20`

Example response:

```json
{
  "count": 1,
  "query": {
    "status": ["needs_review"],
    "action": "fill_brand",
    "needs_human": true,
    "limit": 20
  },
  "items": [
    {
      "task_id": "listing-queue-1",
      "status": "needs_review",
      "ready": false,
      "needs_review": true,
      "review_items": [
        {
          "field": "brand",
          "action": "fill_brand",
          "severity": "warning",
          "reason": "missing brand",
          "source": "llm,user_text",
          "confidence": 0.58,
          "is_inferred": true,
          "needs_human": true,
          "recommended_fix": "confirm or fill the selling brand"
        }
      ],
      "review_summary": {
        "total_count": 1,
        "blocking_count": 0,
        "needs_human_count": 1,
        "by_action": {
          "fill_brand": 1
        },
        "by_field": {
          "brand": 1
        },
        "by_severity": {
          "warning": 1
        }
      },
      "total_items": 1,
      "top_action": "fill_brand",
      "action_buckets": [
        {
          "action": "fill_brand",
          "label": "待补品牌",
          "count": 1,
          "blocking_count": 0,
          "priority": 7,
          "rank": 1,
          "items": [
            {
              "message": "missing brand",
              "severity": "warning",
              "target": "brand",
              "operator_action": "fill_brand",
              "operator_advice": "confirm or fill the selling brand"
            }
          ]
        }
      ]
    }
  ]
}
```

## Workbench Response

`GET /api/v1/amazon/listings/tasks/:task_id/workbench`

This endpoint returns the current AmazonListing review projection for operator follow-up. It does not expose retired ProductEnrich/ProductImage child-task progress.

Example response:

```json
{
  "task_id": "listing-123",
  "status": "needs_review",
  "ready": false,
  "needs_review": true,
  "review_items": [
    {
      "field": "brand",
      "action": "fill_brand",
      "severity": "warning",
      "reason": "missing brand",
      "source": "llm,user_text",
      "confidence": 0.58,
      "is_inferred": true,
      "needs_human": true,
      "recommended_fix": "confirm or fill the selling brand",
      "evidence": [
        {
          "type": "user_text",
          "detail": "user input: \"portable blender bottle for smoothies\""
        },
        {
          "type": "llm",
          "detail": "LLM-generated product normalization"
        },
        {
          "type": "field_value",
          "detail": "brand = \"Generic\""
        }
      ]
    }
  ],
  "review_summary": {
    "total_count": 1,
    "blocking_count": 0,
    "needs_human_count": 1,
    "by_action": {
      "fill_brand": 1
    },
    "by_field": {
      "brand": 1
    },
    "by_severity": {
      "warning": 1
    }
  },
  "total_items": 1,
  "top_action": "fill_brand",
  "action_buckets": [
    {
      "action": "fill_brand",
      "label": "待补品牌",
      "count": 1,
      "blocking_count": 0,
      "priority": 7,
      "rank": 1,
      "items": [
        {
          "message": "missing brand",
          "severity": "warning",
          "target": "brand",
          "operator_action": "fill_brand",
          "operator_advice": "confirm or fill the selling brand"
        }
      ]
    }
  ]
}
```

## Apply Field Edits

`POST /api/v1/amazon/listings/tasks/:task_id/review`

Use `action=apply_edits` to write operator fixes back into the Amazon listing draft, rebuild export payloads, and re-run validation.

Example request:

```json
{
  "action": "apply_edits",
  "edits": [
    {
      "field": "brand",
      "string_value": "Acme"
    },
    {
      "field": "title",
      "string_value": "High Quality Ceramic Coffee Mug for Home Kitchen Use"
    },
    {
      "field": "category_path",
      "string_list": ["Home & Kitchen", "Drinkware"]
    },
    {
      "field": "bullet_points",
      "string_list": [
        "Durable ceramic material",
        "Suitable for coffee and tea",
        "Comfortable daily-use mug"
      ]
    },
    {
      "field": "pricing.suggested_price",
      "number_value": 19.99
    }
  ]
}
```

Current accepted edit paths:

- Core text/list fields: `title`, `brand`, `description`, `category_path`, `bullet_points`, `search_terms`
- Images: `images.main_image`, `images.white_bg_image`, `images.gallery`
- Pricing: `pricing.currency`, `pricing.suggested_price`, `pricing.min_price`, `pricing.source_cost`
- Product attributes: `attributes.<key>`, `specifications.technical.<key>`
- Dimensions/weight: `dimensions.length`, `dimensions.width`, `dimensions.height`, `dimensions.unit`, `weight.value`, `weight.unit`
- Package: `package.quantity`, `package.dimensions.length`, `package.dimensions.width`, `package.dimensions.height`, `package.dimensions.unit`, `package.weight.value`, `package.weight.unit`
- Variants: `variants[n].sku`, `variants[n].barcode`, `variants[n].inventory`, `variants[n].is_default`, `variants[n].main_image`, `variants[n].price.amount`, `variants[n].price.currency`, `variants[n].cost_price.amount`, `variants[n].cost_price.currency`, `variants[n].attributes.<key>`

Review item evidence fields:

- `confidence`: numeric confidence score from canonical trace
- `is_inferred`: whether the field is primarily inferred/generated
- `evidence[]`: source details plus field snippets for operator review
- `evidence[].type`: source type such as `user_text`, `user_image`, `product_url`, `scraped_data`, `llm`, `field_value`
- `evidence[].detail`: human-readable evidence text, for example scraped title/spec fragments or the current field snapshot

Behavior after `apply_edits`:

- Matching `review_items` are removed.
- Listing export payloads are rebuilt.
- Validator runs again immediately.
- Task status becomes `completed` when no blocking issues or review items remain.
- Otherwise task remains `needs_review` with refreshed `review_items`.
