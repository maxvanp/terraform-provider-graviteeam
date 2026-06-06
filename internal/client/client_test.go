package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// newTestClient creates a Client pointing at a test server.
// The server must handle /management/auth/token for OAuth2.
func newTestClient(server *httptest.Server) *Client {
	return New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT")
}

// testMux returns an http.ServeMux with the OAuth2 token endpoint pre-configured.
func testMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})
	return mux
}

func TestCreateDomain_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["name"] != "test-domain" {
			t.Errorf("expected name test-domain, got %v", body["name"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "domain-123",
			"name": "test-domain",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	result, err := c.CreateDomain(context.Background(), map[string]interface{}{"name": "test-domain"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "domain-123" {
		t.Errorf("expected id domain-123, got %v", result["id"])
	}
	if result["name"] != "test-domain" {
		t.Errorf("expected name test-domain, got %v", result["name"])
	}
}

func TestGetDomain_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "domain-123",
			"name":        "test-domain",
			"description": "A test domain",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	result, err := c.GetDomain(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "domain-123" {
		t.Errorf("expected id domain-123, got %v", result["id"])
	}
}

func TestGetDomain_NotFound(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Domain not found"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	_, err := c.GetDomain(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestDeleteDomain_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteDomain(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRotateCertificate_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "cert-123",
			"name": "Generated certificate",
			"type": "CERTIFICATE",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	result, err := c.RotateCertificate(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "cert-123" {
		t.Errorf("expected id cert-123, got %v", result["id"])
	}
}

func TestResetUserPassword_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/resetPassword", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["password"] != "SecurePass123!" {
			t.Errorf("expected password body, got %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	if err := c.ResetUserPassword(context.Background(), "domain-123", "user-123", "SecurePass123!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendUserRegistrationConfirmation_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/sendRegistrationConfirmation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	if err := c.SendUserRegistrationConfirmation(context.Background(), "domain-123", "user-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetOrgUserPassword_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/resetPassword", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["password"] != "SecurePass123!" {
			t.Errorf("expected password body, got %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	if err := c.ResetOrgUserPassword(context.Background(), "user-123", "SecurePass123!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccRotateCertificateLifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	ctx := context.Background()
	c := New("http://localhost:8093", "admin", "adminadmin", "DEFAULT", "DEFAULT")
	domain, err := c.CreateDomain(ctx, map[string]interface{}{
		"name":        "tf-acc-rotate-" + time.Now().UTC().Format("20060102150405"),
		"dataPlaneId": "default",
	})
	if err != nil {
		t.Fatalf("create domain: %v", err)
	}
	domainID, ok := domain["id"].(string)
	if !ok || domainID == "" {
		t.Fatalf("expected created domain id, got %#v", domain["id"])
	}
	defer func() {
		if err := c.DeleteDomain(ctx, domainID); err != nil {
			t.Logf("delete domain %s: %v", domainID, err)
		}
	}()

	certificate, err := c.RotateCertificate(ctx, domainID)
	if err != nil {
		t.Fatalf("rotate certificate: %v", err)
	}
	certificateID, ok := certificate["id"].(string)
	if !ok || certificateID == "" {
		t.Fatalf("expected rotated certificate id, got %#v", certificate["id"])
	}

	if _, err := c.GetCertificate(ctx, domainID, certificateID); err != nil {
		t.Fatalf("get rotated certificate %s: %v", certificateID, err)
	}
	if err := c.DeleteCertificate(ctx, domainID, certificateID); err != nil {
		t.Fatalf("delete rotated certificate %s: %v", certificateID, err)
	}
}

func TestDoRequest_ServerError(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Internal server error"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	_, err := c.CreateDomain(context.Background(), map[string]interface{}{"name": "test"})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestCreateFactor_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/d1/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "factor-123",
			"name": "TOTP",
			"type": "otp-am-factor",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	result, err := c.CreateFactor(context.Background(), "d1", map[string]interface{}{
		"name": "TOTP",
		"type": "otp-am-factor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "factor-123" {
		t.Errorf("expected id factor-123, got %v", result["id"])
	}
}

func TestManagementPath(t *testing.T) {
	c := &Client{
		OrganizationID: "myorg",
		EnvironmentID:  "myenv",
	}
	expected := "/management/organizations/myorg/environments/myenv"
	if got := c.managementPath(); got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}
