// Command mintjwt prints an HS256 JSON Web Token to stdout.
//
// This is the whole point of the spike: it demonstrates that HQ's Go backend
// can mint a token Supabase's substrate accepts, with NO GoTrue, NO hosted
// Supabase project, and NO third-party JWT library. The imports below are
// exhaustively stdlib — crypto/hmac, crypto/sha256, encoding/base64,
// encoding/json. If a future reader is tempted to reach for
// github.com/golang-jwt/jwt to "do this properly", the sign() function below is
// the counter-argument: an HS256 JWT is a base64url-joined header, payload and
// HMAC-SHA256, and that is all it is.
//
// The real HQ endpoint (card `sync-jwt-bridge-endpoint`) will do exactly this
// with the session's real user id and HQ's real secret. Which claims map to
// which grants is deliberately NOT decided here.
//
//	go run ./mintjwt -secret <JWT_SECRET> -sub user-alice
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

// b64 is base64url WITHOUT padding, which is what the JWS compact serialization
// requires. Using StdEncoding here produces a token every verifier rejects.
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// sign builds a complete HS256 JWT from a claims map. ~10 lines of real work.
func sign(claims map[string]any, secret string) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signing := b64(header) + "." + b64(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	return signing + "." + b64(mac.Sum(nil)), nil
}

func main() {
	secret := flag.String("secret", "", "HS256 signing secret; must equal the stack's JWT_SECRET")
	sub := flag.String("sub", "user-alice", "subject claim — the end user's id")
	role := flag.String("role", "authenticated", "Postgres role PostgREST will SET ROLE into")
	ttl := flag.Duration("ttl", time.Hour, "token lifetime")
	expired := flag.Bool("expired", false, "mint an ALREADY-EXPIRED token (negative proof: exp is enforced)")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "mintjwt: -secret is required")
		os.Exit(2)
	}

	now := time.Now()
	exp := now.Add(*ttl)
	if *expired {
		exp = now.Add(-1 * time.Minute)
	}

	// `role` is the claim PostgREST reads to decide which Postgres role to
	// SET ROLE into for the request. `sub` is what the RLS policies in
	// sql/spike-fixture.sql compare against. `exp` is what makes the token
	// expire; PostgREST rejects a stale one with 401 JWTExpired.
	tok, err := sign(map[string]any{
		"role": *role,
		"sub":  *sub,
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
	}, *secret)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mintjwt:", err)
		os.Exit(1)
	}
	fmt.Println(tok)
}
