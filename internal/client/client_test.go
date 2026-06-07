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

func TestUpdateDomain_Success(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["name"] != "updated-domain" {
			t.Errorf("expected updated-domain body, got %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "domain-123",
			"name": "updated-domain",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	result, err := c.UpdateDomain(context.Background(), "domain-123", map[string]interface{}{"name": "updated-domain"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "updated-domain" {
		t.Errorf("expected updated-domain, got %v", result["name"])
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

func TestFactorCRUDOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create factor body: %v", err)
		}
		if body["name"] != "factor-name" {
			t.Fatalf("unexpected create factor body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "factor-123", "name": "factor-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors/factor-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "factor-123", "name": "factor-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update factor body: %v", err)
			}
			if body["name"] != "factor-updated" {
				t.Fatalf("unexpected update factor body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "factor-123", "name": "factor-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateFactor(context.Background(), "domain-123", map[string]interface{}{"name": "factor-name"})
	if err != nil {
		t.Fatalf("create factor: %v", err)
	}
	if created["id"] != "factor-123" {
		t.Fatalf("unexpected created factor: %#v", created)
	}
	got, err := c.GetFactor(context.Background(), "domain-123", "factor-123")
	if err != nil {
		t.Fatalf("get factor: %v", err)
	}
	if got["name"] != "factor-name" {
		t.Fatalf("unexpected factor: %#v", got)
	}
	updated, err := c.UpdateFactor(context.Background(), "domain-123", "factor-123", map[string]interface{}{"name": "factor-updated"})
	if err != nil {
		t.Fatalf("update factor: %v", err)
	}
	if updated["name"] != "factor-updated" {
		t.Fatalf("unexpected updated factor: %#v", updated)
	}
	if err := c.DeleteFactor(context.Background(), "domain-123", "factor-123"); err != nil {
		t.Fatalf("delete factor: %v", err)
	}
}

func TestApplicationCRUDAndTypeOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create application body: %v", err)
		}
		if body["name"] != "app-name" {
			t.Fatalf("unexpected create application body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-123", "name": "app-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-123", "name": "app-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update application body: %v", err)
			}
			if body["name"] != "app-updated" {
				t.Fatalf("unexpected update application body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-123", "name": "app-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/type", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update application type body: %v", err)
		}
		if body["type"] != "SERVICE" {
			t.Fatalf("unexpected update application type body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-123", "type": "SERVICE"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateApplication(context.Background(), "domain-123", map[string]interface{}{"name": "app-name"})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if created["id"] != "app-123" {
		t.Fatalf("unexpected created application: %#v", created)
	}
	got, err := c.GetApplication(context.Background(), "domain-123", "app-123")
	if err != nil {
		t.Fatalf("get application: %v", err)
	}
	if got["name"] != "app-name" {
		t.Fatalf("unexpected application: %#v", got)
	}
	updated, err := c.UpdateApplication(context.Background(), "domain-123", "app-123", map[string]interface{}{"name": "app-updated"})
	if err != nil {
		t.Fatalf("update application: %v", err)
	}
	if updated["name"] != "app-updated" {
		t.Fatalf("unexpected updated application: %#v", updated)
	}
	typed, err := c.UpdateApplicationType(context.Background(), "domain-123", "app-123", "SERVICE")
	if err != nil {
		t.Fatalf("update application type: %v", err)
	}
	if typed["type"] != "SERVICE" {
		t.Fatalf("unexpected updated application type: %#v", typed)
	}
	if err := c.DeleteApplication(context.Background(), "domain-123", "app-123"); err != nil {
		t.Fatalf("delete application: %v", err)
	}
}

func TestApplicationSecretOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "secret-1"}, {"id": "secret-2"}})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create application secret body: %v", err)
			}
			if body["name"] != "created-secret" {
				t.Fatalf("unexpected create application secret body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "secret-3", "name": "created-secret"})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-2/_renew", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "secret-2", "secret": "new-value"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	secrets, err := c.ListApplicationSecrets(context.Background(), "domain-123", "app-123")
	if err != nil {
		t.Fatalf("list application secrets: %v", err)
	}
	if len(secrets) != 2 {
		t.Fatalf("unexpected application secrets: %#v", secrets)
	}
	created, err := c.CreateApplicationSecret(context.Background(), "domain-123", "app-123", map[string]interface{}{"name": "created-secret"})
	if err != nil {
		t.Fatalf("create application secret: %v", err)
	}
	if created["id"] != "secret-3" {
		t.Fatalf("unexpected created application secret: %#v", created)
	}
	renewed, err := c.RenewApplicationSecret(context.Background(), "domain-123", "app-123", "secret-2")
	if err != nil {
		t.Fatalf("renew application secret: %v", err)
	}
	if renewed["secret"] != "new-value" {
		t.Fatalf("unexpected renewed application secret: %#v", renewed)
	}
	if err := c.DeleteApplicationSecret(context.Background(), "domain-123", "app-123", "secret-3"); err != nil {
		t.Fatalf("delete application secret: %v", err)
	}
}

func TestApplicationAndDomainMemberOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"memberships": []map[string]interface{}{{"id": "membership-1"}, {"id": "membership-2"}},
			})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode add application member body: %v", err)
			}
			if body["memberId"] != "user-3" {
				t.Fatalf("unexpected add application member body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "membership-3", "memberId": "user-3"})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members/membership-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"memberships": []map[string]interface{}{{"id": "domain-membership-1"}},
			})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode add domain member body: %v", err)
			}
			if body["memberId"] != "user-2" {
				t.Fatalf("unexpected add domain member body: %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/members/domain-membership-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	appMembers, err := c.ListApplicationMembers(context.Background(), "domain-123", "app-123")
	if err != nil {
		t.Fatalf("list application members: %v", err)
	}
	if len(appMembers) != 2 {
		t.Fatalf("unexpected application members: %#v", appMembers)
	}
	appMember, err := c.AddOrUpdateApplicationMember(context.Background(), "domain-123", "app-123", map[string]interface{}{"memberId": "user-3"})
	if err != nil {
		t.Fatalf("add application member: %v", err)
	}
	if appMember["id"] != "membership-3" {
		t.Fatalf("unexpected application member: %#v", appMember)
	}
	if err := c.DeleteApplicationMember(context.Background(), "domain-123", "app-123", "membership-3"); err != nil {
		t.Fatalf("delete application member: %v", err)
	}

	domainMembers, err := c.ListDomainMembers(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("list domain members: %v", err)
	}
	if len(domainMembers) != 1 {
		t.Fatalf("unexpected domain members: %#v", domainMembers)
	}
	domainMember, err := c.AddOrUpdateDomainMember(context.Background(), "domain-123", map[string]interface{}{"memberId": "user-2"})
	if err != nil {
		t.Fatalf("add domain member: %v", err)
	}
	if len(domainMember) != 0 {
		t.Fatalf("expected empty domain member result for 204 response, got %#v", domainMember)
	}
	if err := c.DeleteDomainMember(context.Background(), "domain-123", "domain-membership-1"); err != nil {
		t.Fatalf("delete domain member: %v", err)
	}
}

func TestUserCertificateCredentialOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode certificate credential body: %v", err)
		}
		if body["certificatePem"] != "pem" {
			t.Fatalf("unexpected certificate credential body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "credential-1", "certificatePem": "pem"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials/credential-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "credential-1", "certificatePem": "pem"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateUserCertificateCredential(context.Background(), "domain-123", "user-123", map[string]interface{}{"certificatePem": "pem"})
	if err != nil {
		t.Fatalf("create user certificate credential: %v", err)
	}
	if created["id"] != "credential-1" {
		t.Fatalf("unexpected certificate credential: %#v", created)
	}
	got, err := c.GetUserCertificateCredential(context.Background(), "domain-123", "user-123", "credential-1")
	if err != nil {
		t.Fatalf("get user certificate credential: %v", err)
	}
	if got["certificatePem"] != "pem" {
		t.Fatalf("unexpected certificate credential: %#v", got)
	}
	if err := c.DeleteUserCertificateCredential(context.Background(), "domain-123", "user-123", "credential-1"); err != nil {
		t.Fatalf("delete user certificate credential: %v", err)
	}
}

func TestThemeFormEmailAndPluginCRUDOperations(t *testing.T) {
	mux := testMux()
	handleJSON := func(path string, methods map[string]map[string]interface{}) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			body, ok := methods[r.Method]
			if !ok {
				t.Errorf("unexpected method %s for %s", r.Method, path)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				var requestBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("decode %s %s body: %v", r.Method, path, err)
				}
				if requestBody["name"] != nil && requestBody["name"] != body["name"] {
					t.Fatalf("unexpected %s %s body: %#v", r.Method, path, requestBody)
				}
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_ = json.NewEncoder(w).Encode(body)
		})
	}
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-1", map[string]map[string]interface{}{
		http.MethodGet:    {"id": "theme-1", "name": "theme"},
		http.MethodPut:    {"id": "theme-1", "name": "updated-theme"},
		http.MethodDelete: nil,
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "theme-1"}})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create theme body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "theme-1", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-1", map[string]map[string]interface{}{
		http.MethodPut:    {"id": "form-1", "name": "updated-form"},
		http.MethodDelete: nil,
	})
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/app-form-1", map[string]map[string]interface{}{
		http.MethodPut:    {"id": "app-form-1", "name": "updated-app-form"},
		http.MethodDelete: nil,
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("template") != "LOGIN" {
				t.Errorf("unexpected form lookup: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "form-1", "template": "LOGIN"})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create form body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "form-1", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("template") != "LOGIN" {
				t.Errorf("unexpected application form lookup: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-form-1", "template": "LOGIN"})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create application form body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-form-1", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_, _ = w.Write([]byte("<html>preview</html>"))
	})
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails/email-1", map[string]map[string]interface{}{
		http.MethodPut:    {"id": "email-1", "name": "updated-email"},
		http.MethodDelete: nil,
	})
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/app-email-1", map[string]map[string]interface{}{
		http.MethodPut:    {"id": "app-email-1", "name": "updated-app-email"},
		http.MethodDelete: nil,
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("template") != "LOGIN" {
				t.Errorf("unexpected email lookup: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "email-1", "template": "LOGIN"})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create email body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "email-1", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("template") != "LOGIN" {
				t.Errorf("unexpected application email lookup: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-email-1", "template": "LOGIN"})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create application email body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "app-email-1", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	for _, tc := range []struct {
		path        string
		createdName string
		updatedName string
	}{
		{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants", "created-extension-grant", "updated-extension-grant"},
		{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters", "created-reporter", "updated-reporter"},
		{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections", "created-bot-detection", "updated-bot-detection"},
		{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines", "created-authorization-engine", "updated-authorization-engine"},
	} {
		path := tc.path
		createdName := tc.createdName
		updatedName := tc.updatedName
		handleJSON(path, map[string]map[string]interface{}{
			http.MethodPost: {"id": "plugin-1", "name": createdName},
		})
		handleJSON(path+"/plugin-1", map[string]map[string]interface{}{
			http.MethodGet:    {"id": "plugin-1", "name": createdName},
			http.MethodPut:    {"id": "plugin-1", "name": updatedName},
			http.MethodDelete: nil,
		})
	}
	handleJSON("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificate-settings", map[string]map[string]interface{}{
		http.MethodPut: {"enabled": true},
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	themes, err := c.GetThemes(context.Background(), "domain-123")
	if err != nil || len(themes) != 1 {
		t.Fatalf("get themes: themes=%#v err=%v", themes, err)
	}
	if theme, err := c.CreateTheme(context.Background(), "domain-123", map[string]interface{}{"name": "created-theme"}); err != nil || theme["id"] != "theme-1" {
		t.Fatalf("create theme: theme=%#v err=%v", theme, err)
	}
	if theme, err := c.GetTheme(context.Background(), "domain-123", "theme-1"); err != nil || theme["id"] != "theme-1" {
		t.Fatalf("get theme: theme=%#v err=%v", theme, err)
	}
	if theme, err := c.UpdateTheme(context.Background(), "domain-123", "theme-1", map[string]interface{}{"name": "updated-theme"}); err != nil || theme["name"] != "updated-theme" {
		t.Fatalf("update theme: theme=%#v err=%v", theme, err)
	}
	if err := c.DeleteTheme(context.Background(), "domain-123", "theme-1"); err != nil {
		t.Fatalf("delete theme: %v", err)
	}
	if form, err := c.GetForm(context.Background(), "domain-123", "", "LOGIN"); err != nil || form["id"] != "form-1" {
		t.Fatalf("get form: form=%#v err=%v", form, err)
	}
	if form, err := c.GetForm(context.Background(), "domain-123", "app-123", "LOGIN"); err != nil || form["id"] != "app-form-1" {
		t.Fatalf("get application form: form=%#v err=%v", form, err)
	}
	if form, err := c.CreateForm(context.Background(), "domain-123", "", map[string]interface{}{"name": "created-form"}); err != nil || form["id"] != "form-1" {
		t.Fatalf("create form: form=%#v err=%v", form, err)
	}
	if form, err := c.CreateForm(context.Background(), "domain-123", "app-123", map[string]interface{}{"name": "created-app-form"}); err != nil || form["id"] != "app-form-1" {
		t.Fatalf("create application form: form=%#v err=%v", form, err)
	}
	if form, err := c.UpdateForm(context.Background(), "domain-123", "", "form-1", map[string]interface{}{"name": "updated-form"}); err != nil || form["name"] != "updated-form" {
		t.Fatalf("update form: form=%#v err=%v", form, err)
	}
	if form, err := c.UpdateForm(context.Background(), "domain-123", "app-123", "app-form-1", map[string]interface{}{"name": "updated-app-form"}); err != nil || form["name"] != "updated-app-form" {
		t.Fatalf("update application form: form=%#v err=%v", form, err)
	}
	if preview, err := c.PreviewForm(context.Background(), "domain-123", map[string]interface{}{"name": "preview"}); err != nil || string(preview) != "<html>preview</html>" {
		t.Fatalf("preview form: preview=%q err=%v", string(preview), err)
	}
	if err := c.DeleteForm(context.Background(), "domain-123", "", "form-1"); err != nil {
		t.Fatalf("delete form: %v", err)
	}
	if err := c.DeleteForm(context.Background(), "domain-123", "app-123", "app-form-1"); err != nil {
		t.Fatalf("delete application form: %v", err)
	}
	if email, err := c.GetEmail(context.Background(), "domain-123", "", "LOGIN"); err != nil || email["id"] != "email-1" {
		t.Fatalf("get email: email=%#v err=%v", email, err)
	}
	if email, err := c.GetEmail(context.Background(), "domain-123", "app-123", "LOGIN"); err != nil || email["id"] != "app-email-1" {
		t.Fatalf("get application email: email=%#v err=%v", email, err)
	}
	if email, err := c.CreateEmail(context.Background(), "domain-123", "", map[string]interface{}{"name": "created-email"}); err != nil || email["id"] != "email-1" {
		t.Fatalf("create email: email=%#v err=%v", email, err)
	}
	if email, err := c.CreateEmail(context.Background(), "domain-123", "app-123", map[string]interface{}{"name": "created-app-email"}); err != nil || email["id"] != "app-email-1" {
		t.Fatalf("create application email: email=%#v err=%v", email, err)
	}
	if email, err := c.UpdateEmail(context.Background(), "domain-123", "", "email-1", map[string]interface{}{"name": "updated-email"}); err != nil || email["name"] != "updated-email" {
		t.Fatalf("update email: email=%#v err=%v", email, err)
	}
	if email, err := c.UpdateEmail(context.Background(), "domain-123", "app-123", "app-email-1", map[string]interface{}{"name": "updated-app-email"}); err != nil || email["name"] != "updated-app-email" {
		t.Fatalf("update application email: email=%#v err=%v", email, err)
	}
	if err := c.DeleteEmail(context.Background(), "domain-123", "", "email-1"); err != nil {
		t.Fatalf("delete email: %v", err)
	}
	if err := c.DeleteEmail(context.Background(), "domain-123", "app-123", "app-email-1"); err != nil {
		t.Fatalf("delete application email: %v", err)
	}
	if grant, err := c.CreateExtensionGrant(context.Background(), "domain-123", map[string]interface{}{"name": "created-extension-grant"}); err != nil || grant["id"] != "plugin-1" {
		t.Fatalf("create extension grant: grant=%#v err=%v", grant, err)
	}
	if grant, err := c.GetExtensionGrant(context.Background(), "domain-123", "plugin-1"); err != nil || grant["id"] != "plugin-1" {
		t.Fatalf("get extension grant: grant=%#v err=%v", grant, err)
	}
	if grant, err := c.UpdateExtensionGrant(context.Background(), "domain-123", "plugin-1", map[string]interface{}{"name": "updated-extension-grant"}); err != nil || grant["name"] != "updated-extension-grant" {
		t.Fatalf("update extension grant: grant=%#v err=%v", grant, err)
	}
	if err := c.DeleteExtensionGrant(context.Background(), "domain-123", "plugin-1"); err != nil {
		t.Fatalf("delete extension grant: %v", err)
	}
	if reporter, err := c.CreateReporter(context.Background(), "domain-123", map[string]interface{}{"name": "created-reporter"}); err != nil || reporter["id"] != "plugin-1" {
		t.Fatalf("create reporter: reporter=%#v err=%v", reporter, err)
	}
	if reporter, err := c.GetReporter(context.Background(), "domain-123", "plugin-1"); err != nil || reporter["id"] != "plugin-1" {
		t.Fatalf("get reporter: reporter=%#v err=%v", reporter, err)
	}
	if reporter, err := c.UpdateReporter(context.Background(), "domain-123", "plugin-1", map[string]interface{}{"name": "updated-reporter"}); err != nil || reporter["name"] != "updated-reporter" {
		t.Fatalf("update reporter: reporter=%#v err=%v", reporter, err)
	}
	if err := c.DeleteReporter(context.Background(), "domain-123", "plugin-1"); err != nil {
		t.Fatalf("delete reporter: %v", err)
	}
	if bot, err := c.CreateBotDetection(context.Background(), "domain-123", map[string]interface{}{"name": "created-bot-detection"}); err != nil || bot["id"] != "plugin-1" {
		t.Fatalf("create bot detection: bot=%#v err=%v", bot, err)
	}
	if bot, err := c.GetBotDetection(context.Background(), "domain-123", "plugin-1"); err != nil || bot["id"] != "plugin-1" {
		t.Fatalf("get bot detection: bot=%#v err=%v", bot, err)
	}
	if bot, err := c.UpdateBotDetection(context.Background(), "domain-123", "plugin-1", map[string]interface{}{"name": "updated-bot-detection"}); err != nil || bot["name"] != "updated-bot-detection" {
		t.Fatalf("update bot detection: bot=%#v err=%v", bot, err)
	}
	if err := c.DeleteBotDetection(context.Background(), "domain-123", "plugin-1"); err != nil {
		t.Fatalf("delete bot detection: %v", err)
	}
	if engine, err := c.CreateAuthorizationEngine(context.Background(), "domain-123", map[string]interface{}{"name": "created-authorization-engine"}); err != nil || engine["id"] != "plugin-1" {
		t.Fatalf("create authorization engine: engine=%#v err=%v", engine, err)
	}
	if engine, err := c.GetAuthorizationEngine(context.Background(), "domain-123", "plugin-1"); err != nil || engine["id"] != "plugin-1" {
		t.Fatalf("get authorization engine: engine=%#v err=%v", engine, err)
	}
	if engine, err := c.UpdateAuthorizationEngine(context.Background(), "domain-123", "plugin-1", map[string]interface{}{"name": "updated-authorization-engine"}); err != nil || engine["name"] != "updated-authorization-engine" {
		t.Fatalf("update authorization engine: engine=%#v err=%v", engine, err)
	}
	if err := c.DeleteAuthorizationEngine(context.Background(), "domain-123", "plugin-1"); err != nil {
		t.Fatalf("delete authorization engine: %v", err)
	}
	if settings, err := c.UpdateDomainCertificateSettings(context.Background(), "domain-123", map[string]interface{}{"enabled": true}); err != nil || settings["enabled"] != true {
		t.Fatalf("update domain certificate settings: settings=%#v err=%v", settings, err)
	}
}

func TestProtectedResourceI18nAndAlertOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create protected resource body: %v", err)
		}
		if body["name"] != "resource" {
			t.Fatalf("unexpected create protected resource body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-1", "name": "resource"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-2", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "UMA Resource" {
			t.Fatalf("unexpected protected resource type query: %s", r.URL.RawQuery)
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-2", "name": "resource"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			if r.URL.Query().Get("type") != "MCP_SERVER" {
				t.Fatalf("unexpected protected resource type query: %s", r.URL.RawQuery)
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-1", "name": "resource"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update protected resource body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-1", "name": body["name"]})
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "secret-1"}, {"id": "secret-2"}})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create protected resource secret body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "secret-3", "name": body["name"]})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1/secrets/secret-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1/secrets/secret-2/_renew", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "secret-2", "secret": "renewed"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"memberships": []map[string]interface{}{{"id": "membership-1"}, {"id": "membership-2"}},
			})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode protected resource member body: %v", err)
			}
			if body["memberId"] != "user-1" {
				t.Fatalf("unexpected protected resource member body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "membership-3", "memberId": "user-1"})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-1/members/membership-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create i18n dictionary body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dictionary-1", "name": body["name"]})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dictionary-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dictionary-1", "locale": "fr"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update i18n dictionary body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dictionary-1", "name": body["name"]})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dictionary-1/entries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode replace i18n entries body: %v", err)
		}
		if body["login.title"] != "Bonjour" {
			t.Fatalf("unexpected i18n entries body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dictionary-1", "entries": body})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/notifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create alert notifier body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-1", "name": body["name"]})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/notifiers/notifier-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-1", "name": "notifier"})
		case http.MethodPatch:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode patch alert notifier body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-1", "name": body["name"]})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PATCH, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/triggers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "trigger-1", "enabled": false}})
		case http.MethodPatch:
			var body []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode patch alert triggers body: %v", err)
			}
			if len(body) != 1 || body[0]["id"] != "trigger-1" {
				t.Fatalf("unexpected alert triggers body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("expected GET or PATCH, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	resource, err := c.CreateProtectedResource(context.Background(), "domain-123", map[string]interface{}{"name": "resource"})
	if err != nil || resource["id"] != "resource-1" {
		t.Fatalf("create protected resource: resource=%#v err=%v", resource, err)
	}
	resource, err = c.GetProtectedResource(context.Background(), "domain-123", "resource-1", "")
	if err != nil || resource["id"] != "resource-1" {
		t.Fatalf("get protected resource with default type: resource=%#v err=%v", resource, err)
	}
	resource, err = c.GetProtectedResource(context.Background(), "domain-123", "resource-2", "UMA Resource")
	if err != nil || resource["id"] != "resource-2" {
		t.Fatalf("get protected resource with explicit type: resource=%#v err=%v", resource, err)
	}
	resource, err = c.UpdateProtectedResource(context.Background(), "domain-123", "resource-1", map[string]interface{}{"name": "updated-resource"})
	if err != nil || resource["name"] != "updated-resource" {
		t.Fatalf("update protected resource: resource=%#v err=%v", resource, err)
	}
	secrets, err := c.ListProtectedResourceSecrets(context.Background(), "domain-123", "resource-1")
	if err != nil || len(secrets) != 2 {
		t.Fatalf("list protected resource secrets: secrets=%#v err=%v", secrets, err)
	}
	secret, err := c.CreateProtectedResourceSecret(context.Background(), "domain-123", "resource-1", map[string]interface{}{"name": "secret"})
	if err != nil || secret["id"] != "secret-3" {
		t.Fatalf("create protected resource secret: secret=%#v err=%v", secret, err)
	}
	secret, err = c.RenewProtectedResourceSecret(context.Background(), "domain-123", "resource-1", "secret-2")
	if err != nil || secret["secret"] != "renewed" {
		t.Fatalf("renew protected resource secret: secret=%#v err=%v", secret, err)
	}
	if err := c.DeleteProtectedResourceSecret(context.Background(), "domain-123", "resource-1", "secret-3"); err != nil {
		t.Fatalf("delete protected resource secret: %v", err)
	}
	members, err := c.ListProtectedResourceMembers(context.Background(), "domain-123", "resource-1")
	if err != nil || len(members) != 2 {
		t.Fatalf("list protected resource members: members=%#v err=%v", members, err)
	}
	member, err := c.AddOrUpdateProtectedResourceMember(context.Background(), "domain-123", "resource-1", map[string]interface{}{"memberId": "user-1"})
	if err != nil || member["id"] != "membership-3" {
		t.Fatalf("add protected resource member: member=%#v err=%v", member, err)
	}
	if err := c.DeleteProtectedResourceMember(context.Background(), "domain-123", "resource-1", "membership-3"); err != nil {
		t.Fatalf("delete protected resource member: %v", err)
	}
	if err := c.DeleteProtectedResource(context.Background(), "domain-123", "resource-1", ""); err != nil {
		t.Fatalf("delete protected resource: %v", err)
	}
	if err := c.DeleteProtectedResource(context.Background(), "domain-123", "resource-2", "UMA Resource"); err != nil {
		t.Fatalf("delete protected resource with explicit type: %v", err)
	}
	dictionary, err := c.CreateI18nDictionary(context.Background(), "domain-123", map[string]interface{}{"name": "dictionary"})
	if err != nil || dictionary["id"] != "dictionary-1" {
		t.Fatalf("create i18n dictionary: dictionary=%#v err=%v", dictionary, err)
	}
	dictionary, err = c.GetI18nDictionary(context.Background(), "domain-123", "dictionary-1")
	if err != nil || dictionary["locale"] != "fr" {
		t.Fatalf("get i18n dictionary: dictionary=%#v err=%v", dictionary, err)
	}
	dictionary, err = c.UpdateI18nDictionary(context.Background(), "domain-123", "dictionary-1", map[string]interface{}{"name": "updated-dictionary"})
	if err != nil || dictionary["name"] != "updated-dictionary" {
		t.Fatalf("update i18n dictionary: dictionary=%#v err=%v", dictionary, err)
	}
	dictionary, err = c.ReplaceI18nDictionaryEntries(context.Background(), "domain-123", "dictionary-1", map[string]string{"login.title": "Bonjour"})
	if err != nil || dictionary["id"] != "dictionary-1" {
		t.Fatalf("replace i18n dictionary entries: dictionary=%#v err=%v", dictionary, err)
	}
	if err := c.DeleteI18nDictionary(context.Background(), "domain-123", "dictionary-1"); err != nil {
		t.Fatalf("delete i18n dictionary: %v", err)
	}
	notifier, err := c.CreateAlertNotifier(context.Background(), "domain-123", map[string]interface{}{"name": "notifier"})
	if err != nil || notifier["id"] != "notifier-1" {
		t.Fatalf("create alert notifier: notifier=%#v err=%v", notifier, err)
	}
	notifier, err = c.GetAlertNotifier(context.Background(), "domain-123", "notifier-1")
	if err != nil || notifier["name"] != "notifier" {
		t.Fatalf("get alert notifier: notifier=%#v err=%v", notifier, err)
	}
	notifier, err = c.PatchAlertNotifier(context.Background(), "domain-123", "notifier-1", map[string]interface{}{"name": "updated-notifier"})
	if err != nil || notifier["name"] != "updated-notifier" {
		t.Fatalf("patch alert notifier: notifier=%#v err=%v", notifier, err)
	}
	triggers, err := c.ListAlertTriggers(context.Background(), "domain-123")
	if err != nil || len(triggers) != 1 {
		t.Fatalf("list alert triggers: triggers=%#v err=%v", triggers, err)
	}
	triggers, err = c.PatchAlertTriggers(context.Background(), "domain-123", []map[string]interface{}{{"id": "trigger-1", "enabled": true}})
	if err != nil || len(triggers) != 1 {
		t.Fatalf("patch alert triggers: triggers=%#v err=%v", triggers, err)
	}
	if err := c.DeleteAlertNotifier(context.Background(), "domain-123", "notifier-1"); err != nil {
		t.Fatalf("delete alert notifier: %v", err)
	}
}

func TestUserLifecycleOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create user body: %v", err)
		}
		if body["username"] != "user-name" {
			t.Fatalf("unexpected create user body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "user-123", "username": "user-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "user-123", "username": "user-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update user body: %v", err)
			}
			if body["firstName"] != "Updated" {
				t.Fatalf("unexpected update user body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "user-123", "firstName": "Updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update user status body: %v", err)
		}
		if body["enabled"] != false {
			t.Fatalf("unexpected update user status body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "user-123", "enabled": false})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/unlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/username", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update username body: %v", err)
		}
		if body["username"] != "user-renamed" {
			t.Fatalf("unexpected update username body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "user-123", "username": "user-renamed"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/collections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`[{"id":"collection-item"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateUser(context.Background(), "domain-123", map[string]interface{}{"username": "user-name"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created["id"] != "user-123" {
		t.Fatalf("unexpected created user: %#v", created)
	}
	got, err := c.GetUser(context.Background(), "domain-123", "user-123")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got["username"] != "user-name" {
		t.Fatalf("unexpected user: %#v", got)
	}
	updated, err := c.UpdateUser(context.Background(), "domain-123", "user-123", map[string]interface{}{"firstName": "Updated"})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updated["firstName"] != "Updated" {
		t.Fatalf("unexpected updated user: %#v", updated)
	}
	status, err := c.UpdateUserStatus(context.Background(), "domain-123", "user-123", false)
	if err != nil {
		t.Fatalf("update user status: %v", err)
	}
	if status["enabled"] != false {
		t.Fatalf("unexpected updated user status: %#v", status)
	}
	if err := c.LockUser(context.Background(), "domain-123", "user-123"); err != nil {
		t.Fatalf("lock user: %v", err)
	}
	if err := c.UnlockUser(context.Background(), "domain-123", "user-123"); err != nil {
		t.Fatalf("unlock user: %v", err)
	}
	renamed, err := c.UpdateUsername(context.Background(), "domain-123", "user-123", "user-renamed")
	if err != nil {
		t.Fatalf("update username: %v", err)
	}
	if renamed["username"] != "user-renamed" {
		t.Fatalf("unexpected updated username: %#v", renamed)
	}
	collection, err := c.ListUserCollection(context.Background(), "domain-123", "user-123", "collections")
	if err != nil {
		t.Fatalf("list user collection: %v", err)
	}
	if string(collection) != `[{"id":"collection-item"}]` {
		t.Fatalf("unexpected user collection: %s", collection)
	}
	if err := c.DeleteUser(context.Background(), "domain-123", "user-123"); err != nil {
		t.Fatalf("delete user: %v", err)
	}
}

func TestPasswordPolicyOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create password policy body: %v", err)
		}
		if body["name"] != "policy-name" {
			t.Fatalf("unexpected create password policy body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "policy-123", "name": "policy-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "policy-123", "name": "policy-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update password policy body: %v", err)
			}
			if body["name"] != "policy-updated" {
				t.Fatalf("unexpected update password policy body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "policy-123", "name": "policy-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/default", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "policy-123", "defaultPolicy": true})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode evaluate password policy body: %v", err)
		}
		if body["password"] != "SecurePass123!" {
			t.Fatalf("unexpected evaluate password policy body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"valid":true}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreatePasswordPolicy(context.Background(), "domain-123", map[string]interface{}{"name": "policy-name"})
	if err != nil {
		t.Fatalf("create password policy: %v", err)
	}
	if created["id"] != "policy-123" {
		t.Fatalf("unexpected created password policy: %#v", created)
	}
	got, err := c.GetPasswordPolicy(context.Background(), "domain-123", "policy-123")
	if err != nil {
		t.Fatalf("get password policy: %v", err)
	}
	if got["name"] != "policy-name" {
		t.Fatalf("unexpected password policy: %#v", got)
	}
	updated, err := c.UpdatePasswordPolicy(context.Background(), "domain-123", "policy-123", map[string]interface{}{"name": "policy-updated"})
	if err != nil {
		t.Fatalf("update password policy: %v", err)
	}
	if updated["name"] != "policy-updated" {
		t.Fatalf("unexpected updated password policy: %#v", updated)
	}
	defaulted, err := c.SetDefaultPasswordPolicy(context.Background(), "domain-123", "policy-123")
	if err != nil {
		t.Fatalf("set default password policy: %v", err)
	}
	if defaulted["defaultPolicy"] != true {
		t.Fatalf("unexpected default password policy response: %#v", defaulted)
	}
	evaluation, err := c.EvaluatePasswordPolicy(context.Background(), "domain-123", "policy-123", map[string]interface{}{"password": "SecurePass123!"})
	if err != nil {
		t.Fatalf("evaluate password policy: %v", err)
	}
	if string(evaluation) != `{"valid":true}` {
		t.Fatalf("unexpected password policy evaluation: %s", evaluation)
	}
	if err := c.DeletePasswordPolicy(context.Background(), "domain-123", "policy-123"); err != nil {
		t.Fatalf("delete password policy: %v", err)
	}
}

func TestScopeRoleGroupCRUDOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create scope body: %v", err)
		}
		if body["key"] != "scope-key" {
			t.Fatalf("unexpected create scope body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "scope-123", "key": "scope-key"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes/scope-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "scope-123", "key": "scope-key"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update scope body: %v", err)
			}
			if body["name"] != "scope-updated" {
				t.Fatalf("unexpected update scope body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "scope-123", "name": "scope-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create role body: %v", err)
		}
		if body["name"] != "role-name" {
			t.Fatalf("unexpected create role body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "role-123", "name": "role-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "role-123", "name": "role-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update role body: %v", err)
			}
			if body["name"] != "role-updated" {
				t.Fatalf("unexpected update role body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "role-123", "name": "role-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create group body: %v", err)
		}
		if body["name"] != "group-name" {
			t.Fatalf("unexpected create group body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "group-123", "name": "group-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "group-123", "name": "group-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update group body: %v", err)
			}
			if body["name"] != "group-updated" {
				t.Fatalf("unexpected update group body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "group-123", "name": "group-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	scope, err := c.CreateScope(context.Background(), "domain-123", map[string]interface{}{"key": "scope-key"})
	if err != nil {
		t.Fatalf("create scope: %v", err)
	}
	if scope["id"] != "scope-123" {
		t.Fatalf("unexpected created scope: %#v", scope)
	}
	scope, err = c.GetScope(context.Background(), "domain-123", "scope-123")
	if err != nil {
		t.Fatalf("get scope: %v", err)
	}
	if scope["key"] != "scope-key" {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	scope, err = c.UpdateScope(context.Background(), "domain-123", "scope-123", map[string]interface{}{"name": "scope-updated"})
	if err != nil {
		t.Fatalf("update scope: %v", err)
	}
	if scope["name"] != "scope-updated" {
		t.Fatalf("unexpected updated scope: %#v", scope)
	}
	if err := c.DeleteScope(context.Background(), "domain-123", "scope-123"); err != nil {
		t.Fatalf("delete scope: %v", err)
	}

	role, err := c.CreateRole(context.Background(), "domain-123", map[string]interface{}{"name": "role-name"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if role["id"] != "role-123" {
		t.Fatalf("unexpected created role: %#v", role)
	}
	role, err = c.GetRole(context.Background(), "domain-123", "role-123")
	if err != nil {
		t.Fatalf("get role: %v", err)
	}
	if role["name"] != "role-name" {
		t.Fatalf("unexpected role: %#v", role)
	}
	role, err = c.UpdateRole(context.Background(), "domain-123", "role-123", map[string]interface{}{"name": "role-updated"})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if role["name"] != "role-updated" {
		t.Fatalf("unexpected updated role: %#v", role)
	}
	if err := c.DeleteRole(context.Background(), "domain-123", "role-123"); err != nil {
		t.Fatalf("delete role: %v", err)
	}

	group, err := c.CreateGroup(context.Background(), "domain-123", map[string]interface{}{"name": "group-name"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if group["id"] != "group-123" {
		t.Fatalf("unexpected created group: %#v", group)
	}
	group, err = c.GetGroup(context.Background(), "domain-123", "group-123")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if group["name"] != "group-name" {
		t.Fatalf("unexpected group: %#v", group)
	}
	group, err = c.UpdateGroup(context.Background(), "domain-123", "group-123", map[string]interface{}{"name": "group-updated"})
	if err != nil {
		t.Fatalf("update group: %v", err)
	}
	if group["name"] != "group-updated" {
		t.Fatalf("unexpected updated group: %#v", group)
	}
	if err := c.DeleteGroup(context.Background(), "domain-123", "group-123"); err != nil {
		t.Fatalf("delete group: %v", err)
	}
}

func TestIdentityProviderOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create identity provider body: %v", err)
		}
		if body["name"] != "idp-name" {
			t.Fatalf("unexpected create identity provider body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "idp-123", "name": "idp-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "idp-123", "name": "idp-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update identity provider body: %v", err)
			}
			if body["name"] != "idp-updated" {
				t.Fatalf("unexpected update identity provider body: %#v", body)
			}
			if _, ok := body["groupMapper"]; !ok {
				t.Fatalf("expected groupMapper in update identity provider body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "idp-123", "name": "idp-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123/password-policy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode identity provider password policy body: %v", err)
		}
		if _, ok := body["passwordPolicy"]; !ok {
			t.Fatalf("expected passwordPolicy field, got %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateIdentityProvider(context.Background(), "domain-123", map[string]interface{}{"name": "idp-name"})
	if err != nil {
		t.Fatalf("create identity provider: %v", err)
	}
	if created["id"] != "idp-123" {
		t.Fatalf("unexpected created identity provider: %#v", created)
	}
	got, err := c.GetIdentityProvider(context.Background(), "domain-123", "idp-123")
	if err != nil {
		t.Fatalf("get identity provider: %v", err)
	}
	if got["name"] != "idp-name" {
		t.Fatalf("unexpected identity provider: %#v", got)
	}
	updated, err := c.UpdateIdentityProvider(context.Background(), "domain-123", "idp-123", map[string]interface{}{
		"name":        "idp-updated",
		"groupMapper": map[string]interface{}{"groups": []string{"group-1"}},
		"roleMapper":  map[string]interface{}{"roles": []string{"role-1"}},
	})
	if err != nil {
		t.Fatalf("update identity provider: %v", err)
	}
	if updated["name"] != "idp-updated" {
		t.Fatalf("unexpected updated identity provider: %#v", updated)
	}
	if err := c.AssignIdentityProviderPasswordPolicy(context.Background(), "domain-123", "idp-123", "policy-123"); err != nil {
		t.Fatalf("assign identity provider password policy: %v", err)
	}
	if err := c.ClearIdentityProviderPasswordPolicy(context.Background(), "domain-123", "idp-123"); err != nil {
		t.Fatalf("clear identity provider password policy: %v", err)
	}
	if err := c.DeleteIdentityProvider(context.Background(), "domain-123", "idp-123"); err != nil {
		t.Fatalf("delete identity provider: %v", err)
	}
}

func TestOrgIdentityProviderOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/identities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create organization identity provider body: %v", err)
		}
		if body["name"] != "org-idp-name" {
			t.Fatalf("unexpected create organization identity provider body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-idp-123", "name": "org-idp-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/identities/org-idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-idp-123", "name": "org-idp-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update organization identity provider body: %v", err)
			}
			if body["name"] != "org-idp-updated" {
				t.Fatalf("unexpected update organization identity provider body: %#v", body)
			}
			if _, ok := body["roleMapper"]; !ok {
				t.Fatalf("expected roleMapper in update organization identity provider body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-idp-123", "name": "org-idp-updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateOrgIdentityProvider(context.Background(), map[string]interface{}{"name": "org-idp-name"})
	if err != nil {
		t.Fatalf("create organization identity provider: %v", err)
	}
	if created["id"] != "org-idp-123" {
		t.Fatalf("unexpected created organization identity provider: %#v", created)
	}
	got, err := c.GetOrgIdentityProvider(context.Background(), "org-idp-123")
	if err != nil {
		t.Fatalf("get organization identity provider: %v", err)
	}
	if got["name"] != "org-idp-name" {
		t.Fatalf("unexpected organization identity provider: %#v", got)
	}
	updated, err := c.UpdateOrgIdentityProvider(context.Background(), "org-idp-123", map[string]interface{}{
		"name":        "org-idp-updated",
		"groupMapper": map[string]interface{}{"groups": []string{"group-1"}},
		"roleMapper":  map[string]interface{}{"roles": []string{"role-1"}},
	})
	if err != nil {
		t.Fatalf("update organization identity provider: %v", err)
	}
	if updated["name"] != "org-idp-updated" {
		t.Fatalf("unexpected updated organization identity provider: %#v", updated)
	}
	if err := c.DeleteOrgIdentityProvider(context.Background(), "org-idp-123"); err != nil {
		t.Fatalf("delete organization identity provider: %v", err)
	}
}

func TestOrgUserLifecycleOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create organization user body: %v", err)
		}
		if body["username"] != "org-user-name" {
			t.Fatalf("unexpected create organization user body: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-user-123", "username": "org-user-name"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-user-123", "username": "org-user-name"})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update organization user body: %v", err)
			}
			if body["firstName"] != "Updated" {
				t.Fatalf("unexpected update organization user body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-user-123", "firstName": "Updated"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update organization user status body: %v", err)
		}
		if body["enabled"] != false {
			t.Fatalf("unexpected update organization user status body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-user-123", "enabled": false})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/username", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update organization username body: %v", err)
		}
		if body["username"] != "org-user-renamed" {
			t.Fatalf("unexpected update organization username body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "org-user-123", "username": "org-user-renamed"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	created, err := c.CreateOrgUser(context.Background(), map[string]interface{}{"username": "org-user-name"})
	if err != nil {
		t.Fatalf("create organization user: %v", err)
	}
	if created["id"] != "org-user-123" {
		t.Fatalf("unexpected created organization user: %#v", created)
	}
	got, err := c.GetOrgUser(context.Background(), "org-user-123")
	if err != nil {
		t.Fatalf("get organization user: %v", err)
	}
	if got["username"] != "org-user-name" {
		t.Fatalf("unexpected organization user: %#v", got)
	}
	updated, err := c.UpdateOrgUser(context.Background(), "org-user-123", map[string]interface{}{"firstName": "Updated"})
	if err != nil {
		t.Fatalf("update organization user: %v", err)
	}
	if updated["firstName"] != "Updated" {
		t.Fatalf("unexpected updated organization user: %#v", updated)
	}
	status, err := c.UpdateOrgUserStatus(context.Background(), "org-user-123", false)
	if err != nil {
		t.Fatalf("update organization user status: %v", err)
	}
	if status["enabled"] != false {
		t.Fatalf("unexpected organization user status: %#v", status)
	}
	renamed, err := c.UpdateOrgUsername(context.Background(), "org-user-123", "org-user-renamed")
	if err != nil {
		t.Fatalf("update organization username: %v", err)
	}
	if renamed["username"] != "org-user-renamed" {
		t.Fatalf("unexpected organization username: %#v", renamed)
	}
	if err := c.DeleteOrgUser(context.Background(), "org-user-123"); err != nil {
		t.Fatalf("delete organization user: %v", err)
	}
}

func TestOrgUserTokenOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "token-1"},
				{"id": "token-2"},
			})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create organization user token body: %v", err)
			}
			if body["name"] != "token-name" {
				t.Fatalf("unexpected create organization user token body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "token-3", "name": "token-name"})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/tokens/token-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	tokens, err := c.ListOrgUserTokens(context.Background(), "org-user-123")
	if err != nil {
		t.Fatalf("list organization user tokens: %v", err)
	}
	if len(tokens) != 2 || tokens[0]["id"] != "token-1" || tokens[1]["id"] != "token-2" {
		t.Fatalf("unexpected organization user tokens: %#v", tokens)
	}
	created, err := c.CreateOrgUserToken(context.Background(), "org-user-123", map[string]interface{}{"name": "token-name"})
	if err != nil {
		t.Fatalf("create organization user token: %v", err)
	}
	if created["id"] != "token-3" {
		t.Fatalf("unexpected created organization user token: %#v", created)
	}
	if err := c.DeleteOrgUserToken(context.Background(), "org-user-123", "token-3"); err != nil {
		t.Fatalf("delete organization user token: %v", err)
	}
}

func TestOrgMemberOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"id": "member-1"},
					{"id": "member-2"},
				},
			})
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode add organization member body: %v", err)
			}
			if body["member"] != "user@example.com" {
				t.Fatalf("unexpected add organization member body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "member-3"})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/members/member-3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	members, err := c.ListOrgMembers(context.Background())
	if err != nil {
		t.Fatalf("list organization members: %v", err)
	}
	if len(members) != 2 || members[0]["id"] != "member-1" || members[1]["id"] != "member-2" {
		t.Fatalf("unexpected organization members: %#v", members)
	}
	created, err := c.AddOrUpdateOrgMember(context.Background(), map[string]interface{}{"member": "user@example.com"})
	if err != nil {
		t.Fatalf("add organization member: %v", err)
	}
	if created["id"] != "member-3" {
		t.Fatalf("unexpected organization member: %#v", created)
	}
	if err := c.DeleteOrgMember(context.Background(), "member-3"); err != nil {
		t.Fatalf("delete organization member: %v", err)
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

func TestDomainFlowOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/flows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "login", "enabled": true},
				{"id": "mfa", "enabled": false},
			})
		case http.MethodPut:
			var body []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update domain flows body: %v", err)
			}
			if len(body) != 1 || body[0]["id"] != "login" || body[0]["enabled"] != true {
				t.Fatalf("unexpected update domain flows body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("expected GET or PUT, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	flows, err := c.ListFlows(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("list domain flows: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("unexpected domain flows: %#v", flows)
	}

	updated, err := c.UpdateDomainFlows(context.Background(), "domain-123", []interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	if err != nil {
		t.Fatalf("update domain flows: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("unexpected updated domain flows: %#v", updated)
	}
}

func TestApplicationFlowOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/flows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "login", "enabled": true},
			})
		case http.MethodPut:
			var body []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update application flows body: %v", err)
			}
			if len(body) != 1 || body[0]["id"] != "consent" || body[0]["enabled"] != false {
				t.Fatalf("unexpected update application flows body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("expected GET or PUT, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	flows, err := c.GetApplicationFlows(context.Background(), "domain-123", "app-123")
	if err != nil {
		t.Fatalf("get application flows: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("unexpected application flows: %#v", flows)
	}

	updated, err := c.UpdateApplicationFlows(context.Background(), "domain-123", "app-123", []interface{}{
		map[string]interface{}{"id": "consent", "enabled": false},
	})
	if err != nil {
		t.Fatalf("update application flows: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("unexpected updated application flows: %#v", updated)
	}
}

func TestPluginLikeResourceOperations(t *testing.T) {
	cases := map[string]struct {
		collection string
		create     func(context.Context, *Client, map[string]interface{}) (map[string]interface{}, error)
		get        func(context.Context, *Client) (map[string]interface{}, error)
		update     func(context.Context, *Client, map[string]interface{}) (map[string]interface{}, error)
		delete     func(context.Context, *Client) error
	}{
		"auth device notifier": {
			collection: "auth-device-notifiers",
			create: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.CreateAuthDeviceNotifier(ctx, "domain-123", body)
			},
			get: func(ctx context.Context, c *Client) (map[string]interface{}, error) {
				return c.GetAuthDeviceNotifier(ctx, "domain-123", "resource-123")
			},
			update: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.UpdateAuthDeviceNotifier(ctx, "domain-123", "resource-123", body)
			},
			delete: func(ctx context.Context, c *Client) error {
				return c.DeleteAuthDeviceNotifier(ctx, "domain-123", "resource-123")
			},
		},
		"certificate": {
			collection: "certificates",
			create: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.CreateCertificate(ctx, "domain-123", body)
			},
			get: func(ctx context.Context, c *Client) (map[string]interface{}, error) {
				return c.GetCertificate(ctx, "domain-123", "resource-123")
			},
			update: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.UpdateCertificate(ctx, "domain-123", "resource-123", body)
			},
			delete: func(ctx context.Context, c *Client) error {
				return c.DeleteCertificate(ctx, "domain-123", "resource-123")
			},
		},
		"device identifier": {
			collection: "device-identifiers",
			create: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.CreateDeviceIdentifier(ctx, "domain-123", body)
			},
			get: func(ctx context.Context, c *Client) (map[string]interface{}, error) {
				return c.GetDeviceIdentifier(ctx, "domain-123", "resource-123")
			},
			update: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.UpdateDeviceIdentifier(ctx, "domain-123", "resource-123", body)
			},
			delete: func(ctx context.Context, c *Client) error {
				return c.DeleteDeviceIdentifier(ctx, "domain-123", "resource-123")
			},
		},
		"service resource": {
			collection: "resources",
			create: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.CreateServiceResource(ctx, "domain-123", body)
			},
			get: func(ctx context.Context, c *Client) (map[string]interface{}, error) {
				return c.GetServiceResource(ctx, "domain-123", "resource-123")
			},
			update: func(ctx context.Context, c *Client, body map[string]interface{}) (map[string]interface{}, error) {
				return c.UpdateServiceResource(ctx, "domain-123", "resource-123", body)
			},
			delete: func(ctx context.Context, c *Client) error {
				return c.DeleteServiceResource(ctx, "domain-123", "resource-123")
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mux := testMux()
			collectionPath := "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/" + tc.collection
			itemPath := collectionPath + "/resource-123"
			mux.HandleFunc(collectionPath, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode create body: %v", err)
				}
				if body["name"] != "created" {
					t.Fatalf("unexpected create body: %#v", body)
				}
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-123", "name": "created"})
			})
			mux.HandleFunc(itemPath, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-123", "name": "created"})
				case http.MethodPut:
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatalf("decode update body: %v", err)
					}
					if body["name"] != "updated" {
						t.Fatalf("unexpected update body: %#v", body)
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "resource-123", "name": "updated"})
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("expected GET, PUT, or DELETE, got %s", r.Method)
				}
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			c := newTestClient(server)
			created, err := tc.create(context.Background(), c, map[string]interface{}{"name": "created"})
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created["id"] != "resource-123" {
				t.Fatalf("unexpected create result: %#v", created)
			}
			got, err := tc.get(context.Background(), c)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got["name"] != "created" {
				t.Fatalf("unexpected get result: %#v", got)
			}
			updated, err := tc.update(context.Background(), c, map[string]interface{}{"name": "updated"})
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			if updated["name"] != "updated" {
				t.Fatalf("unexpected update result: %#v", updated)
			}
			if err := tc.delete(context.Background(), c); err != nil {
				t.Fatalf("delete: %v", err)
			}
		})
	}
}

func TestGroupMembersOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.RawQuery != "page=0&size=100" {
			t.Errorf("expected pagination query, got %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "user-1"},
				{"id": "user-2"},
				{"displayName": "ignored"},
			},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members/user-3", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected POST or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	members, err := c.GetGroupMembers(context.Background(), "domain-123", "group-123")
	if err != nil {
		t.Fatalf("get group members: %v", err)
	}
	if len(members) != 2 || members[0] != "user-1" || members[1] != "user-2" {
		t.Fatalf("unexpected members: %#v", members)
	}
	if err := c.AddGroupMember(context.Background(), "domain-123", "group-123", "user-3"); err != nil {
		t.Fatalf("add group member: %v", err)
	}
	if err := c.RemoveGroupMember(context.Background(), "domain-123", "group-123", "user-3"); err != nil {
		t.Fatalf("remove group member: %v", err)
	}
}

func TestOrgGroupMembersOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.RawQuery != "page=0&size=100" {
			t.Errorf("expected pagination query, got %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "org-user-1"},
				{"id": "org-user-2"},
			},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members/org-user-3", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("expected POST or DELETE, got %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	members, err := c.GetOrgGroupMembers(context.Background(), "group-123")
	if err != nil {
		t.Fatalf("get organization group members: %v", err)
	}
	if len(members) != 2 || members[0] != "org-user-1" || members[1] != "org-user-2" {
		t.Fatalf("unexpected members: %#v", members)
	}
	if err := c.AddOrgGroupMember(context.Background(), "group-123", "org-user-3"); err != nil {
		t.Fatalf("add organization group member: %v", err)
	}
	if err := c.RemoveOrgGroupMember(context.Background(), "group-123", "org-user-3"); err != nil {
		t.Fatalf("remove organization group member: %v", err)
	}
}

func TestGroupRolesOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "role-1"},
				{"id": "role-2"},
			})
		case http.MethodPost:
			var body []string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode set group roles body: %v", err)
			}
			if len(body) != 2 || body[0] != "role-2" || body[1] != "role-3" {
				t.Fatalf("unexpected set group roles body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles/role-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	roles, err := c.GetGroupRoles(context.Background(), "domain-123", "group-123")
	if err != nil {
		t.Fatalf("get group roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("unexpected roles: %#v", roles)
	}
	result, err := c.SetGroupRoles(context.Background(), "domain-123", "group-123", []string{"role-2", "role-3"})
	if err != nil {
		t.Fatalf("set group roles: %v", err)
	}
	if result["updated"] != true {
		t.Fatalf("unexpected set group roles response: %#v", result)
	}
	if err := c.RemoveGroupRole(context.Background(), "domain-123", "group-123", "role-1"); err != nil {
		t.Fatalf("remove group role: %v", err)
	}
}

func TestUserRolesOperations(t *testing.T) {
	mux := testMux()
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "role-1"}})
		case http.MethodPost:
			var body []string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode set user roles body: %v", err)
			}
			if len(body) != 1 || body[0] != "role-2" {
				t.Fatalf("unexpected set user roles body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles/role-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)
	roles, err := c.GetUserRoles(context.Background(), "domain-123", "user-123")
	if err != nil {
		t.Fatalf("get user roles: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("unexpected roles: %#v", roles)
	}
	result, err := c.SetUserRoles(context.Background(), "domain-123", "user-123", []string{"role-2"})
	if err != nil {
		t.Fatalf("set user roles: %v", err)
	}
	if result["updated"] != true {
		t.Fatalf("unexpected set user roles response: %#v", result)
	}
	if err := c.RemoveUserRole(context.Background(), "domain-123", "user-123", "role-1"); err != nil {
		t.Fatalf("remove user role: %v", err)
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
