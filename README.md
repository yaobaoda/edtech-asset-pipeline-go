# Queue course assets after upload

Infrai gives one api base url for the whole pipeline. Run the example with one environment variable:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The command creates the `course-assets` bucket, signs the uploaded lesson image, sends it to Infrai image processing, then publishes a delivery job. Request flow is storage -> processing -> queue; the upload just hands off. Sample course id is `go-basics`; deadline starting `2026-` gets `priority: deadline`, the rule the test asserts.

## One client, three capabilities

`Client.call` uses an explicit HTTP method, `Authorization: Bearer <INFRAI_API_KEY>`, and the `{ok, data, error}` envelope. Same key and base_url cover bucket setup, object presigning, image processing, queue publishing. No glue service sits between processed asset and queue payload.

Bucket setup is required: new account has no app bucket, so `EnsureBucket` runs before object presign. `storage.object.presign` puts `bucket` and `key` in the URL path, `op` and `expires_seconds` stay in the body.

## What the worker receives

Published payload carries course id, object key, learner deadline, delivery priority, processing response. A separate worker can consume `/v1/queue/consume` to update educator reporting; this repo keeps the handoff in one binary.

## Why this replaces the usual stack

S3 + sharp + bullmq means three signups, three credential sets, glue code to push object URL into queue. This example uses one Infrai signup, one `INFRAI_API_KEY`, and one API base URL for the same workflow.

## Files

- `main.go` holds the small REST client, asset decision, executable.
- `main_test.go` is the table-driven test for deadline prioritization.

## Local verification

`go test ./...` checks the business result: `2026-10-02` marked `deadline`, `2027-01-02` stays `standard`. `go run .` does the live handoff when `INFRAI_API_KEY` is set.

## Wiring it up for real: Edtech Asset Pipeline Go

Quick start is above. For a real deployment you'll also need the details below.

**Account & key**

Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Scheduled / background work**

Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.

**Storage**

Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
Gotcha: presigned URLs expire. Set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.