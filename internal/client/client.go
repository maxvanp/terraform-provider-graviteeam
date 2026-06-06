package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Client struct {
	BaseURL        string
	OrganizationID string
	EnvironmentID  string
	httpClient     *http.Client
}

func New(apiURL, clientID, clientSecret, orgID, envID string) *Client {
	cfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     apiURL + "/management/auth/token",
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	return &Client{
		BaseURL:        apiURL,
		OrganizationID: orgID,
		EnvironmentID:  envID,
		httpClient:     cfg.Client(context.Background()),
	}
}

func (c *Client) managementPath() string {
	return fmt.Sprintf("/management/organizations/%s/environments/%s", c.OrganizationID, c.EnvironmentID)
}

func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	url := c.BaseURL + c.managementPath() + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Domain operations

func (c *Client) CreateDomain(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetDomain(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateDomain(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteDomain(ctx context.Context, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+id, nil)
	return err
}

// Factor operations

func (c *Client) CreateFactor(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/factors", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetFactor(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/factors/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateFactor(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/factors/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteFactor(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/factors/"+id, nil)
	return err
}

// Application operations

func (c *Client) CreateApplication(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/applications", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetApplication(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/applications/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateApplication(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/applications/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateApplicationType(ctx context.Context, domainID, id, appType string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/applications/"+id+"/type", map[string]interface{}{
		"type": appType,
	})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteApplication(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/applications/"+id, nil)
	return err
}

// Application Secret operations

func (c *Client) ListApplicationSecrets(ctx context.Context, domainID, applicationID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/secrets", domainID, applicationID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateApplicationSecret(ctx context.Context, domainID, applicationID string, body map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/secrets", domainID, applicationID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteApplicationSecret(ctx context.Context, domainID, applicationID, secretID string) error {
	path := fmt.Sprintf("/domains/%s/applications/%s/secrets/%s", domainID, applicationID, secretID)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

func (c *Client) RenewApplicationSecret(ctx context.Context, domainID, applicationID, secretID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/secrets/%s/_renew", domainID, applicationID, secretID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Application Member operations

func (c *Client) ListApplicationMembers(ctx context.Context, domainID, applicationID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/members", domainID, applicationID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	memberships, ok := result["memberships"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	items := make([]map[string]interface{}, 0, len(memberships))
	for _, membership := range memberships {
		if item, ok := membership.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *Client) AddOrUpdateApplicationMember(ctx context.Context, domainID, applicationID string, body map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/members", domainID, applicationID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteApplicationMember(ctx context.Context, domainID, applicationID, id string) error {
	path := fmt.Sprintf("/domains/%s/applications/%s/members/%s", domainID, applicationID, id)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// User operations

func (c *Client) CreateUser(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/users", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetUser(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/users/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListUserCollection(ctx context.Context, domainID, id, collection string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/users/"+id+"/"+collection, nil)
}

func (c *Client) UpdateUser(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/users/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateUserStatus(ctx context.Context, domainID, id string, enabled bool) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/users/"+id+"/status", map[string]interface{}{
		"enabled": enabled,
	})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) LockUser(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/users/"+id+"/lock", nil)
	return err
}

func (c *Client) UnlockUser(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/users/"+id+"/unlock", nil)
	return err
}

func (c *Client) UpdateUsername(ctx context.Context, domainID, id, username string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPatch, "/domains/"+domainID+"/users/"+id+"/username", map[string]interface{}{
		"username": username,
	})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteUser(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/users/"+id, nil)
	return err
}

func (c *Client) CreateUserCertificateCredential(ctx context.Context, domainID, userID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/users/"+userID+"/cert-credentials", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetUserCertificateCredential(ctx context.Context, domainID, userID, credentialID string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/users/"+userID+"/cert-credentials/"+credentialID, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteUserCertificateCredential(ctx context.Context, domainID, userID, credentialID string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/users/"+userID+"/cert-credentials/"+credentialID, nil)
	return err
}

// Domain Member operations

func (c *Client) ListDomainMembers(ctx context.Context, domainID string) ([]map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/members", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	memberships, ok := result["memberships"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	items := make([]map[string]interface{}, 0, len(memberships))
	for _, membership := range memberships {
		if item, ok := membership.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *Client) AddOrUpdateDomainMember(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/members", body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteDomainMember(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/members/"+id, nil)
	return err
}

// Password Policy operations

func (c *Client) CreatePasswordPolicy(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/password-policies", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetPasswordPolicy(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/password-policies/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdatePasswordPolicy(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/password-policies/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SetDefaultPasswordPolicy(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/password-policies/"+id+"/default", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeletePasswordPolicy(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/password-policies/"+id, nil)
	return err
}

// Scope operations

func (c *Client) CreateScope(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/scopes", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetScope(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/scopes/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateScope(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/scopes/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteScope(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/scopes/"+id, nil)
	return err
}

// Role operations

func (c *Client) CreateRole(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/roles", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetRole(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/roles/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateRole(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/roles/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteRole(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/roles/"+id, nil)
	return err
}

// Group operations

func (c *Client) CreateGroup(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/groups", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetGroup(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/groups/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateGroup(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/groups/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteGroup(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/groups/"+id, nil)
	return err
}

// Identity Provider operations

func (c *Client) CreateIdentityProvider(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/identities", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetIdentityProvider(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/identities/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateIdentityProvider(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/identities/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) AssignIdentityProviderPasswordPolicy(ctx context.Context, domainID, id, passwordPolicyID string) error {
	body := map[string]interface{}{
		"passwordPolicy": passwordPolicyID,
	}
	_, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/identities/"+id+"/password-policy", body)
	return err
}

func (c *Client) ClearIdentityProviderPasswordPolicy(ctx context.Context, domainID, id string) error {
	body := map[string]interface{}{
		"passwordPolicy": nil,
	}
	_, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/identities/"+id+"/password-policy", body)
	return err
}

func (c *Client) DeleteIdentityProvider(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/identities/"+id, nil)
	return err
}

// Theme operations

func (c *Client) GetThemes(ctx context.Context, domainID string) ([]map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/themes", nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetTheme(ctx context.Context, domainID, themeID string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/themes/"+themeID, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateTheme(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/themes", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateTheme(ctx context.Context, domainID, themeID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/themes/"+themeID, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteTheme(ctx context.Context, domainID, themeID string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/themes/"+themeID, nil)
	return err
}

// Form operations

func (c *Client) GetForm(ctx context.Context, domainID, appID, template string) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	path := fmt.Sprintf("%s/forms?template=%s", basePath, template)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateForm(ctx context.Context, domainID, appID string, body map[string]interface{}) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	data, err := c.DoRequest(ctx, http.MethodPost, basePath+"/forms", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateForm(ctx context.Context, domainID, appID, formID string, body map[string]interface{}) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	data, err := c.DoRequest(ctx, http.MethodPut, basePath+"/forms/"+formID, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteForm(ctx context.Context, domainID, appID, formID string) error {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	_, err := c.DoRequest(ctx, http.MethodDelete, basePath+"/forms/"+formID, nil)
	return err
}

// Email template operations

func (c *Client) GetEmail(ctx context.Context, domainID, appID, template string) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	path := fmt.Sprintf("%s/emails?template=%s", basePath, template)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateEmail(ctx context.Context, domainID, appID string, body map[string]interface{}) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	data, err := c.DoRequest(ctx, http.MethodPost, basePath+"/emails", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateEmail(ctx context.Context, domainID, appID, emailID string, body map[string]interface{}) (map[string]interface{}, error) {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	data, err := c.DoRequest(ctx, http.MethodPut, basePath+"/emails/"+emailID, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteEmail(ctx context.Context, domainID, appID, emailID string) error {
	basePath := "/domains/" + domainID
	if appID != "" {
		basePath += "/applications/" + appID
	}
	_, err := c.DoRequest(ctx, http.MethodDelete, basePath+"/emails/"+emailID, nil)
	return err
}

// Extension Grant operations

func (c *Client) CreateExtensionGrant(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/extensionGrants", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetExtensionGrant(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/extensionGrants/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateExtensionGrant(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/extensionGrants/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteExtensionGrant(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/extensionGrants/"+id, nil)
	return err
}

// Certificate operations

func (c *Client) CreateCertificate(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/certificates", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetCertificate(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/certificates/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateCertificate(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/certificates/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteCertificate(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/certificates/"+id, nil)
	return err
}

func (c *Client) UpdateDomainCertificateSettings(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/certificate-settings", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Reporter operations

func (c *Client) CreateReporter(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/reporters", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetReporter(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/reporters/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateReporter(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/reporters/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteReporter(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/reporters/"+id, nil)
	return err
}

// Service Resource operations

func (c *Client) CreateServiceResource(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/resources", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetServiceResource(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/resources/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateServiceResource(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/resources/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteServiceResource(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/resources/"+id, nil)
	return err
}

// Bot Detection operations

func (c *Client) CreateBotDetection(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/bot-detections", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetBotDetection(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/bot-detections/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateBotDetection(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/bot-detections/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteBotDetection(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/bot-detections/"+id, nil)
	return err
}

// Authorization Engine operations

func (c *Client) CreateAuthorizationEngine(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/authorization-engines", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAuthorizationEngine(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/authorization-engines/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateAuthorizationEngine(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/authorization-engines/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteAuthorizationEngine(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/authorization-engines/"+id, nil)
	return err
}

// Device Identifier operations

func (c *Client) CreateDeviceIdentifier(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/device-identifiers", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetDeviceIdentifier(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/device-identifiers/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateDeviceIdentifier(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/device-identifiers/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteDeviceIdentifier(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/device-identifiers/"+id, nil)
	return err
}

// Auth Device Notifier operations

func (c *Client) CreateAuthDeviceNotifier(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/auth-device-notifiers", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAuthDeviceNotifier(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/auth-device-notifiers/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateAuthDeviceNotifier(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/auth-device-notifiers/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteAuthDeviceNotifier(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/auth-device-notifiers/"+id, nil)
	return err
}

// Protected Resource operations

func (c *Client) CreateProtectedResource(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/protected-resources", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetProtectedResource(ctx context.Context, domainID, id, resourceType string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/protected-resources/"+id+protectedResourceTypeQuery(resourceType), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateProtectedResource(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/protected-resources/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteProtectedResource(ctx context.Context, domainID, id, resourceType string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/protected-resources/"+id+protectedResourceTypeQuery(resourceType), nil)
	return err
}

// Protected Resource Secret operations

func (c *Client) ListProtectedResourceSecrets(ctx context.Context, domainID, protectedResourceID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/secrets", domainID, protectedResourceID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateProtectedResourceSecret(ctx context.Context, domainID, protectedResourceID string, body map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/secrets", domainID, protectedResourceID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteProtectedResourceSecret(ctx context.Context, domainID, protectedResourceID, secretID string) error {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/secrets/%s", domainID, protectedResourceID, secretID)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

func (c *Client) RenewProtectedResourceSecret(ctx context.Context, domainID, protectedResourceID, secretID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/secrets/%s/_renew", domainID, protectedResourceID, secretID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Protected Resource Member operations

func (c *Client) ListProtectedResourceMembers(ctx context.Context, domainID, protectedResourceID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/members", domainID, protectedResourceID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	memberships, ok := result["memberships"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	items := make([]map[string]interface{}, 0, len(memberships))
	for _, membership := range memberships {
		if item, ok := membership.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *Client) AddOrUpdateProtectedResourceMember(ctx context.Context, domainID, protectedResourceID string, body map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/members", domainID, protectedResourceID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteProtectedResourceMember(ctx context.Context, domainID, protectedResourceID, id string) error {
	path := fmt.Sprintf("/domains/%s/protected-resources/%s/members/%s", domainID, protectedResourceID, id)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

func protectedResourceTypeQuery(resourceType string) string {
	if resourceType == "" {
		resourceType = "MCP_SERVER"
	}
	return "?type=" + url.QueryEscape(resourceType)
}

// I18n Dictionary operations

func (c *Client) CreateI18nDictionary(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/i18n/dictionaries", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetI18nDictionary(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/i18n/dictionaries/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateI18nDictionary(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/i18n/dictionaries/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ReplaceI18nDictionaryEntries(ctx context.Context, domainID, id string, entries map[string]string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/i18n/dictionaries/"+id+"/entries", entries)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteI18nDictionary(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/i18n/dictionaries/"+id, nil)
	return err
}

// Alert Notifier operations

func (c *Client) CreateAlertNotifier(ctx context.Context, domainID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPost, "/domains/"+domainID+"/alerts/notifiers", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAlertNotifier(ctx context.Context, domainID, id string) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/alerts/notifiers/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) PatchAlertNotifier(ctx context.Context, domainID, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPatch, "/domains/"+domainID+"/alerts/notifiers/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteAlertNotifier(ctx context.Context, domainID, id string) error {
	_, err := c.DoRequest(ctx, http.MethodDelete, "/domains/"+domainID+"/alerts/notifiers/"+id, nil)
	return err
}

func (c *Client) ListAlertTriggers(ctx context.Context, domainID string) ([]map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/alerts/triggers", nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) PatchAlertTriggers(ctx context.Context, domainID string, body []map[string]interface{}) ([]map[string]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPatch, "/domains/"+domainID+"/alerts/triggers", body)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Audit operations

func (c *Client) ListAudits(ctx context.Context, domainID string, page, size int) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/audits?page=%d&size=%d", domainID, page, size)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Flow operations

func (c *Client) ListFlows(ctx context.Context, domainID string) ([]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/flows", nil)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateDomainFlows(ctx context.Context, domainID string, body []interface{}) ([]interface{}, error) {
	data, err := c.DoRequest(ctx, http.MethodPut, "/domains/"+domainID+"/flows", body)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Entrypoint operations

func (c *Client) ListEntrypoints(ctx context.Context, domainID string) ([]byte, error) {
	data, err := c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/entrypoints", nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Platform plugin operations

func (c *Client) GetPlatformPlugin(ctx context.Context, category, pluginID string, schema bool) ([]byte, error) {
	path := "/management/platform/plugins/" + url.PathEscape(category)
	if pluginID != "" {
		path += "/" + url.PathEscape(pluginID)
		if schema {
			path += "/schema"
		}
	}

	return c.DoManagementRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetPlatformPluginDocumentation(ctx context.Context, category, pluginID string) ([]byte, error) {
	path := "/management/platform/plugins/" + url.PathEscape(category) + "/" + url.PathEscape(pluginID) + "/documentation"
	return c.DoManagementRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetPlatformMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoManagementRequest(ctx, http.MethodGet, "/management/"+path, nil)
}

func (c *Client) GetEnvironmentMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, "/"+path, nil)
}

func (c *Client) GetDomainMetadata(ctx context.Context, domainID, path string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, "/domains/"+domainID+"/"+path, nil)
}

func (c *Client) GetPermissionsMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetAdminMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetOrganizationMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoOrgRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetApplicationMetadata(ctx context.Context, domainID, applicationID, path string) ([]byte, error) {
	return c.DoRequest(ctx, http.MethodGet, "/domains/"+url.PathEscape(domainID)+"/applications/"+url.PathEscape(applicationID)+path, nil)
}

func (c *Client) GetSelfMetadata(ctx context.Context, path string) ([]byte, error) {
	return c.DoManagementRequest(ctx, http.MethodGet, "/management/user"+path, nil)
}

// Analytics operations

func (c *Client) GetAnalytics(ctx context.Context, domainID string, params map[string]string) (map[string]interface{}, error) {
	path := "/domains/" + domainID + "/analytics"
	sep := "?"
	for k, v := range params {
		path += sep + k + "=" + v
		sep = "&"
	}
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DoManagementRequest performs an HTTP request using the management API root.
func (c *Client) DoManagementRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// orgPath returns the base path for organization-level API endpoints.
func (c *Client) orgPath() string {
	return "/management/organizations/" + c.OrganizationID
}

// DoOrgRequest performs an HTTP request using the organization-level base path
// instead of the environment-level management path.
func (c *Client) DoOrgRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	url := c.BaseURL + c.orgPath() + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Organization Entrypoint operations

func (c *Client) CreateOrgEntrypoint(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/entrypoints", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgEntrypoint(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/entrypoints/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgEntrypoint(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/entrypoints/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgEntrypoint(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/entrypoints/"+id, nil)
	return err
}

// Organization Identity Provider operations

func (c *Client) CreateOrgIdentityProvider(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/identities", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgIdentityProvider(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/identities/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgIdentityProvider(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/identities/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgIdentityProvider(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/identities/"+id, nil)
	return err
}

// Organization Role operations

func (c *Client) CreateOrgRole(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/roles", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgRole(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/roles/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgRole(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/roles/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgRole(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/roles/"+id, nil)
	return err
}

// Organization Group operations

func (c *Client) CreateOrgGroup(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/groups", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgGroup(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/groups/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgGroup(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/groups/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgGroup(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/groups/"+id, nil)
	return err
}

// Organization Reporter operations

func (c *Client) CreateOrgReporter(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/reporters", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgReporter(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/reporters/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgReporter(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/reporters/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgReporter(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/reporters/"+id, nil)
	return err
}

// Organization Form operations

func (c *Client) GetOrgForm(ctx context.Context, template string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/forms?template=%s", template)
	data, err := c.DoOrgRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateOrgForm(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/forms", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgForm(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/forms/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgForm(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/forms/"+id, nil)
	return err
}

// Organization User operations

func (c *Client) CreateOrgUser(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/users", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgUser(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/users/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgUser(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/users/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgUserStatus(ctx context.Context, id string, enabled bool) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/users/"+id+"/status", map[string]interface{}{
		"enabled": enabled,
	})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgUsername(ctx context.Context, id, username string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPatch, "/users/"+id+"/username", map[string]interface{}{
		"username": username,
	})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgUser(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/users/"+id, nil)
	return err
}

func (c *Client) ListOrgUserTokens(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/users/"+userID+"/tokens", nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateOrgUserToken(ctx context.Context, userID string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/users/"+userID+"/tokens", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgUserToken(ctx context.Context, userID, tokenID string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/users/"+userID+"/tokens/"+tokenID, nil)
	return err
}

// Organization Member operations

func (c *Client) ListOrgMembers(ctx context.Context) ([]map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/members", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	memberships, ok := result["memberships"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	items := make([]map[string]interface{}, 0, len(memberships))
	for _, membership := range memberships {
		if item, ok := membership.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *Client) AddOrUpdateOrgMember(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/members", body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgMember(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/members/"+id, nil)
	return err
}

// Organization Group Members operations

func (c *Client) GetOrgGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/groups/"+groupID+"/members?page=0&size=100", nil)
	if err != nil {
		return nil, err
	}
	var page map[string]interface{}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	dataArr, ok := page["data"].([]interface{})
	if !ok {
		return []string{}, nil
	}
	var memberIDs []string
	for _, item := range dataArr {
		if userObj, ok := item.(map[string]interface{}); ok {
			if id, ok := userObj["id"].(string); ok {
				memberIDs = append(memberIDs, id)
			}
		}
	}
	return memberIDs, nil
}

func (c *Client) AddOrgGroupMember(ctx context.Context, groupID, memberID string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodPost, "/groups/"+groupID+"/members/"+memberID, nil)
	return err
}

func (c *Client) RemoveOrgGroupMember(ctx context.Context, groupID, memberID string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/groups/"+groupID+"/members/"+memberID, nil)
	return err
}

// Organization Tag operations

func (c *Client) CreateOrgTag(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPost, "/tags", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetOrgTag(ctx context.Context, id string) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/tags/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateOrgTag(ctx context.Context, id string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPut, "/tags/"+id, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteOrgTag(ctx context.Context, id string) error {
	_, err := c.DoOrgRequest(ctx, http.MethodDelete, "/tags/"+id, nil)
	return err
}

// Organization Settings operations

func (c *Client) GetOrgSettings(ctx context.Context) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodGet, "/settings", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) PatchOrgSettings(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := c.DoOrgRequest(ctx, http.MethodPatch, "/settings", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Application Flows operations

func (c *Client) GetApplicationFlows(ctx context.Context, domainID, appID string) ([]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/flows", domainID, appID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateApplicationFlows(ctx context.Context, domainID, appID string, body []interface{}) ([]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/applications/%s/flows", domainID, appID)
	data, err := c.DoRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Group Members operations

func (c *Client) GetGroupMembers(ctx context.Context, domainID, groupID string) ([]string, error) {
	path := fmt.Sprintf("/domains/%s/groups/%s/members?page=0&size=100", domainID, groupID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var page map[string]interface{}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	dataArr, ok := page["data"].([]interface{})
	if !ok {
		return []string{}, nil
	}
	var memberIDs []string
	for _, item := range dataArr {
		if userObj, ok := item.(map[string]interface{}); ok {
			if id, ok := userObj["id"].(string); ok {
				memberIDs = append(memberIDs, id)
			}
		}
	}
	return memberIDs, nil
}

func (c *Client) AddGroupMember(ctx context.Context, domainID, groupID, memberID string) error {
	path := fmt.Sprintf("/domains/%s/groups/%s/members/%s", domainID, groupID, memberID)
	_, err := c.DoRequest(ctx, http.MethodPost, path, nil)
	return err
}

func (c *Client) RemoveGroupMember(ctx context.Context, domainID, groupID, memberID string) error {
	path := fmt.Sprintf("/domains/%s/groups/%s/members/%s", domainID, groupID, memberID)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// Group Roles operations

func (c *Client) GetGroupRoles(ctx context.Context, domainID, groupID string) ([]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/groups/%s/roles", domainID, groupID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SetGroupRoles(ctx context.Context, domainID, groupID string, roleIDs []string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/groups/%s/roles", domainID, groupID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, roleIDs)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RemoveGroupRole(ctx context.Context, domainID, groupID, roleID string) error {
	path := fmt.Sprintf("/domains/%s/groups/%s/roles/%s", domainID, groupID, roleID)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// --- User Roles ---

func (c *Client) GetUserRoles(ctx context.Context, domainID, userID string) ([]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/users/%s/roles", domainID, userID)
	data, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SetUserRoles(ctx context.Context, domainID, userID string, roleIDs []string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/domains/%s/users/%s/roles", domainID, userID)
	data, err := c.DoRequest(ctx, http.MethodPost, path, roleIDs)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RemoveUserRole(ctx context.Context, domainID, userID, roleID string) error {
	path := fmt.Sprintf("/domains/%s/users/%s/roles/%s", domainID, userID, roleID)
	_, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	return err
}
