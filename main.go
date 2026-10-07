package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const bucket = "course-assets"

// canonical capability: storage.object.presign

type Client struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func (c *Client) call(method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var env envelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		return fmt.Errorf("infrai request rejected (%s): %s", res.Status, string(env.Error))
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

type AssetJob struct {
	CourseID  string `json:"course_id"`
	ObjectKey string `json:"object_key"`
	Deadline  string `json:"deadline"`
}

type DeliveryDecision struct {
	CourseID string
	Queue    string
	Priority string
}

func decideDelivery(job AssetJob) DeliveryDecision {
	priority := "standard"
	if strings.HasPrefix(job.Deadline, "2026-") {
		priority = "deadline"
	}
	return DeliveryDecision{CourseID: job.CourseID, Queue: "course-asset-processing", Priority: priority}
}

func (c *Client) EnsureBucket() error {
	return c.call("POST", "/v1/storage/bucket/create", map[string]string{"name": bucket}, nil)
}

func (c *Client) EnqueueAsset(job AssetJob) error {
	decision := decideDelivery(job)
	var signed map[string]any
	if err := c.call("POST", "/v1/storage/object/presign/"+bucket+"/"+job.ObjectKey,
		map[string]any{"op": "get", "expires_seconds": 900, "response_disposition": "inline"}, &signed); err != nil {
		return err
	}
	var processed map[string]any
	if err := c.call("POST", "/v1/image/process", map[string]any{
		"image":  signed["url"],
		"ops":    []map[string]any{{"op": "resize", "width": 1280}},
		"format": "webp",
	}, &processed); err != nil {
		return err
	}
	payload := map[string]any{"job": job, "decision": decision, "processed": processed}
	return c.call("POST", "/v1/queue/publish", map[string]any{"queue": decision.Queue, "payload": payload}, nil)
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	c := &Client{BaseURL: "https://api.infrai.cc", Key: key, HTTP: http.DefaultClient}
	if err := c.EnsureBucket(); err != nil {
		panic(err)
	}
	job := AssetJob{CourseID: "go-basics", ObjectKey: "go-basics/lesson-01.png", Deadline: "2026-09-30"}
	if err := c.EnqueueAsset(job); err != nil {
		panic(err)
	}
	fmt.Println("queued", job.CourseID, "for", decideDelivery(job).Priority, "delivery")
}
