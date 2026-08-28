// Command bucket-cors shows or applies the CORS rules on the configured
// object-storage bucket (STORAGE_* env, same names the server reads).
//
// The browser uploads videos and photos DIRECTLY to the bucket via presigned
// PUT, so the bucket must answer CORS preflights for every origin the app is
// served from. The old DO Spaces bucket carried these rules as console-only
// config; the B2 cutover (B-172) moved the objects but not usable rules — the
// bucket's one native rule allowed only http://localhost:8080, so every real
// origin's PUT died in preflight: presign 200, then /videos/process never
// called, "Upload failed" in the Builder (observed 2026-08-26). This tool
// makes the rules a repo artifact so the next storage move carries them.
//
// 🛑 B2 NATIVE API ON PURPOSE. The bucket holds B2-native CORS rules, and B2
// refuses S3-API PutBucketCors while any native rule exists ("The bucket
// contains B2 Native CORS rules. Please use B2 Native API instead."). Do not
// "simplify" this back to the aws-sdk S3 calls — that path 400s forever.
//
// Usage (from backend/, with STORAGE_* in the environment):
//
//	go run ./cmd/bucket-cors -show
//	go run ./cmd/bucket-cors -apply                  # canonical origins below
//	go run ./cmd/bucket-cors -apply -origins a,b,c   # override the origin list
//
// Prod: run from the prod clone with .env.prod sourced — the rules are
// per-bucket, and prod uses its own bucket.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// canonicalOrigins is every origin the app is served from. Prod first; the
// dev origins are the Tailscale, LAN and localhost faces of `task dev:*`
// (backend/Taskfile.yml network addresses). localhost:8080 predates the 8484
// port and is kept for older bookmarks.
var canonicalOrigins = []string{
	"https://hq.yumyums.kitchen",
	"http://100.90.128.69:8484",
	"http://192.168.8.164:8484",
	"http://localhost:8484",
	"http://localhost:8080",
}

type b2Auth struct {
	AccountID          string `json:"accountId"`
	APIURL             string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

type b2Bucket struct {
	BucketID  string           `json:"bucketId"`
	CORSRules []map[string]any `json:"corsRules"`
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bucket-cors: "+format+"\n", args...)
	os.Exit(1)
}

func b2Call(client *http.Client, url, token string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d: %s", url, resp.StatusCode, raw)
	}
	return json.Unmarshal(raw, out)
}

func main() {
	show := flag.Bool("show", false, "print the bucket's current CORS rules and exit")
	apply := flag.Bool("apply", false, "apply the canonical CORS rules to the bucket")
	origins := flag.String("origins", strings.Join(canonicalOrigins, ","), "comma-separated allowed origins (with -apply)")
	flag.Parse()

	key, secret, bucket := os.Getenv("STORAGE_KEY"), os.Getenv("STORAGE_SECRET"), os.Getenv("STORAGE_BUCKET")
	if key == "" || secret == "" || bucket == "" {
		die("STORAGE_KEY, STORAGE_SECRET and STORAGE_BUCKET must all be set")
	}
	if *show == *apply {
		die("pass exactly one of -show or -apply")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// b2_authorize_account uses basic auth on the application key.
	authReq, err := http.NewRequest(http.MethodGet, "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		die("%v", err)
	}
	authReq.SetBasicAuth(key, secret)
	authResp, err := client.Do(authReq)
	if err != nil {
		die("authorize: %v", err)
	}
	defer authResp.Body.Close()
	authRaw, _ := io.ReadAll(authResp.Body)
	if authResp.StatusCode != http.StatusOK {
		die("authorize: HTTP %d", authResp.StatusCode)
	}
	var auth b2Auth
	if err := json.Unmarshal(authRaw, &auth); err != nil {
		die("authorize decode: %v", err)
	}

	var buckets struct {
		Buckets []b2Bucket `json:"buckets"`
	}
	err = b2Call(client, auth.APIURL+"/b2api/v2/b2_list_buckets", auth.AuthorizationToken,
		map[string]any{"accountId": auth.AccountID, "bucketName": bucket}, &buckets)
	if err != nil {
		die("list buckets: %v", err)
	}
	if len(buckets.Buckets) == 0 {
		die("bucket %q not found on this account", bucket)
	}
	b := buckets.Buckets[0]

	if *apply {
		list := strings.Split(*origins, ",")
		for i := range list {
			list[i] = strings.TrimSpace(list[i])
		}
		rules := []map[string]any{{
			"corsRuleName":      "hq-browser-uploads",
			"allowedOrigins":    list,
			"allowedOperations": []string{"s3_head", "s3_put", "s3_get", "b2_download_file_by_name"},
			"allowedHeaders":    []string{"*"},
			"exposeHeaders":     []string{"etag"},
			"maxAgeSeconds":     3600,
		}}
		var updated b2Bucket
		err = b2Call(client, auth.APIURL+"/b2api/v2/b2_update_bucket", auth.AuthorizationToken,
			map[string]any{"accountId": auth.AccountID, "bucketId": b.BucketID, "corsRules": rules}, &updated)
		if err != nil {
			die("update bucket: %v", err)
		}
		b = updated
		fmt.Printf("applied CORS rules to %q\n", bucket)
	}

	if len(b.CORSRules) == 0 {
		fmt.Printf("bucket %q has NO CORS rules — browser uploads will fail preflight\n", bucket)
		return
	}
	pretty, _ := json.MarshalIndent(b.CORSRules, "", "  ")
	fmt.Printf("bucket %q CORS rules:\n%s\n", bucket, pretty)
}
