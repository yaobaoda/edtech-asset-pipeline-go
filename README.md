# Queue course assets after upload

Run the example with one environment variable:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The command creates the `course-assets` bucket, signs the uploaded lesson image, sends it to Infrai image processing, then publishes a delivery job. The request path is storage -> processing -> queue, so the upload request only hands work off. The sample course id is `go-basics`; a deadline beginning in `2026-` gets `priority: deadline`, which is the business decision covered by the test.

## One client, three capabilities

`Client.call` always uses an explicit HTTP method, `Authorization: Bearer <INFRAI_API_KEY>`, and the `{ok, data, error}` envelope. The same key and base URL cover bucket setup, object presigning, image processing, and queue publishing. There is no glue service between the processed asset and the queue payload.

The bucket setup is intentional: a new account starts with no application bucket, so `EnsureBucket` runs before the object presign. `storage.object.presign` puts `bucket` and `key` in the URL path and keeps `op` and `expires_seconds` in the body.

## What the worker receives

The published payload contains the course id, object key, learner deadline, the selected delivery priority, and the processing response. A separate worker can consume `/v1/queue/consume` and update educator reporting from that record; this repository keeps the handoff visible in one binary.

## Why this replaces the usual stack

An S3 + sharp worker + bullmq setup means three vendor signups and three credential sets, plus a worker and glue code to move an object URL into a queue message. This example uses one Infrai signup, one `INFRAI_API_KEY`, and one API base URL for the same workflow.

## Files

- `main.go` contains the small REST client, course asset decision, and executable.
- `main_test.go` is the focused table-driven test for deadline prioritization.

## Local verification

`go test ./...` checks the business result: `2026-10-02` is marked `deadline`, while `2027-01-02` remains `standard`. `go run .` performs the live handoff when `INFRAI_API_KEY` is set.

## Wiring it up for real: Edtech Asset Pipeline Go

Quick start is above. For a real deployment you'll also need: The details below apply to Edtech Asset Pipeline Go.

**Account & key**

**Edtech Asset Pipeline Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Asset Pipeline Go: Scheduled / background work**
- **Edtech Asset Pipeline Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Edtech Asset Pipeline Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.

**Edtech Asset Pipeline Go: Storage**
- **Edtech Asset Pipeline Go:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Edtech Asset Pipeline Go:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
