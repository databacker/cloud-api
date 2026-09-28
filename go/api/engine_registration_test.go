package api

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type registrationFixture struct {
	AuthenticationKeyID           string `json:"authenticationKeyId"`
	AuthenticationPublicKeyBase64 string `json:"authenticationPublicKeyBase64"`
	Body                          string `json:"body"`
	ContentDigest                 string `json:"contentDigest"`
	ConfigurationKeyID            string `json:"configurationKeyId"`
	ConfigurationPublicKeyBase64  string `json:"configurationPublicKeyBase64"`
	IdempotencyKey                string `json:"idempotencyKey"`
	SignatureBase                 string `json:"signatureBase"`
	SignatureBase64               string `json:"signatureBase64"`
	SignatureInput                string `json:"signatureInput"`
}

func loadRegistrationFixture(t *testing.T) registrationFixture {
	t.Helper()
	data, err := os.ReadFile("../../testdata/registration-proof-v1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture registrationFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return fixture
}

func TestRegistrationProofFixture(t *testing.T) {
	fixture := loadRegistrationFixture(t)

	publicKey, err := base64.StdEncoding.DecodeString(fixture.AuthenticationPublicKeyBase64)
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	signature, err := base64.StdEncoding.DecodeString(fixture.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if !ed25519.Verify(publicKey, []byte(fixture.SignatureBase), signature) {
		t.Fatal("fixture signature did not verify")
	}

	digest := sha256.Sum256([]byte(fixture.Body))
	wantDigest := "sha-256=:" + base64.StdEncoding.EncodeToString(digest[:]) + ":"
	if fixture.ContentDigest != wantDigest {
		t.Fatalf("content digest = %q, want %q", fixture.ContentDigest, wantDigest)
	}

	var body EngineBody
	if err := json.Unmarshal([]byte(fixture.Body), &body); err != nil {
		t.Fatalf("decode registration body: %v", err)
	}
	if body.PublicKeys.Authentication.Algorithm != Ed25519 {
		t.Fatalf("authentication algorithm = %q", body.PublicKeys.Authentication.Algorithm)
	}
	if body.PublicKeys.ConfigurationEncryption.Algorithm != X25519 {
		t.Fatalf("configuration algorithm = %q", body.PublicKeys.ConfigurationEncryption.Algorithm)
	}
	if got := deterministicKeyID("auth", "databacker/engine-key-id/authentication/ed25519/v1\x00", body.PublicKeys.Authentication.Generation, fixture.AuthenticationPublicKeyBase64); got != fixture.AuthenticationKeyID {
		t.Fatalf("authentication key ID = %q, want %q", got, fixture.AuthenticationKeyID)
	}
	if got := deterministicKeyID("config", "databacker/engine-key-id/configuration-encryption/x25519/v1\x00", body.PublicKeys.ConfigurationEncryption.Generation, fixture.ConfigurationPublicKeyBase64); got != fixture.ConfigurationKeyID {
		t.Fatalf("configuration key ID = %q, want %q", got, fixture.ConfigurationKeyID)
	}
	if !strings.Contains(fixture.SignatureInput, `keyid="`+fixture.AuthenticationKeyID+`"`) {
		t.Fatal("signature input does not contain the fixture key ID")
	}
}

func deterministicKeyID(prefix, domain string, generation uint64, encodedPublicKey string) string {
	publicKey, err := base64.StdEncoding.DecodeString(encodedPublicKey)
	if err != nil {
		panic(err)
	}
	input := append([]byte(nil), domain...)
	var encodedGeneration [8]byte
	binary.BigEndian.PutUint64(encodedGeneration[:], generation)
	input = append(input, encodedGeneration[:]...)
	input = append(input, publicKey...)
	digest := sha256.Sum256(input)
	return prefix + ":" + fmt.Sprintf("%x", digest)
}

func TestRegisterEngineRequiresProofHeaders(t *testing.T) {
	handler := Handler(Unimplemented{})
	req := httptest.NewRequest(http.MethodPost, "/admin/accounts/account-123/engines", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("registration without proof returned %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterEngineBindsProofHeaders(t *testing.T) {
	fixture := loadRegistrationFixture(t)
	handler := Handler(Unimplemented{})
	req := httptest.NewRequest(http.MethodPost, "/admin/accounts/account-123/engines", strings.NewReader(fixture.Body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Signature-Input", fixture.SignatureInput)
	req.Header.Set("Signature", "databacker-engine=:"+fixture.SignatureBase64+":")
	req.Header.Set("Content-Digest", fixture.ContentDigest)
	req.Header.Set("Idempotency-Key", fixture.IdempotencyKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("registration with bound proof headers returned %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}
