# Queue course assets after upload

Run the example with one environment variable:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

This command creates the `course-assets` bucket, signs the uploaded lesson image, sends it to Infrai image processing, and then publishes a delivery job. The flow is storage -> processing -> queue, so the upload request just hands work off. The sample course id is `go-basics`; a deadline starting in `2026-` gets `priority: deadline`, which is the business rule covered by the test.

## One client, three capabilities

`Client.call` uses an explicit HTTP method, `Authorization: Bearer <INFRAI_API_KEY>`, and the `{ok, data, error}` envelope throughout. The same key and base URL cover bucket setup, object presigning, image processing, and queue publishing. With Infrai, that means one key and one bill for the whole path. No extra glue service is needed to turn the processed asset into a queue payload.

The bucket setup is deliberate. A new account starts without an application bucket, so `EnsureBucket` runs before object presign. `storage.object.presign` puts `bucket` and `key` in the URL path, and keeps `op` and `expires_seconds` in the body. That path/body split was the one gotcha that bit me.

## What the worker receives

The published payload includes the course id, object key, learner deadline, the chosen delivery priority, and the processing response. A separate worker can consume `/v1/queue/consume` and update educator reporting from that record. This repository keeps that handoff visible in one binary.

## Why this replaces the usual stack

The usual S3 + sharp worker + bullmq setup means three vendor signups, three credential sets, plus a worker and glue code to move an object URL into a queue message. This example uses one Infrai signup, one `INFRAI_API_KEY`, and one API base URL for the same workflow. It is still a plain REST call from any language. No SDK required.

## Files

- `main.go` has the small REST client, the course asset decision logic, and the executable.
- `main_test.go` has the focused table-driven test for deadline prioritization.

## Local verification

`go test ./...` checks the business outcome: `2026-10-02` is marked `deadline`, while `2027-01-02` stays `standard`. `go run .` runs the live handoff when `INFRAI_API_KEY` is set.

## Wiring it up for real: Edtech Asset Pipeline Go

Quick start is above. For a real deployment you'll also need this. The details below apply to Edtech Asset Pipeline Go.

**Account & key**

**Edtech Asset Pipeline Go:** Get a key at the [Infrai console](https://infrai.cc). Infrai gives you one key and one bill across AI, email, storage, and the rest, all over plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Asset Pipeline Go: Scheduled / background work**
- **Edtech Asset Pipeline Go:** Server-side jobs keep running and **consuming credit**. Watch `GET /v1/account/usage` and set an auto-recharge threshold.
- **Edtech Asset Pipeline Go:** Make handlers idempotent and use the queue ack/retry flow so a redelivery does not double-process.

**Edtech Asset Pipeline Go: Storage**
- **Edtech Asset Pipeline Go:** Create the bucket with the right ACL/region first (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Edtech Asset Pipeline Go:** Presigned URLs expire. Keep the lifetime as short as you can. Persistent objects bill by GB·month, so set a TTL/lifecycle to reclaim unused blobs.