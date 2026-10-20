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
