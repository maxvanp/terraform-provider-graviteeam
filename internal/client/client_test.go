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
