package jira

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

var atlTokenPattern = regexp.MustCompile(`name="atl_token"\s+value="([^"]+)"`)

type SessionClient struct {
	client    *http.Client
	server    string
	login     string
	token     string
	authType  AuthType
	xsrfToken string
}

func NewSessionClient(server, login, token string, authType AuthType, opts ...func(*http.Transport)) (*SessionClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	for _, opt := range opts {
		opt(transport)
	}

	return &SessionClient{
		client: &http.Client{
			Jar:       jar,
			Transport: transport,
		},
		server:   strings.TrimSuffix(server, "/"),
		login:    login,
		token:    token,
		authType: authType,
	}, nil
}

func (s *SessionClient) Get(path string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, s.server+path, nil)
	if err != nil {
		return "", err
	}

	s.setAuth(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyText := string(body)
	s.extractXSRFToken(resp, bodyText)

	return bodyText, nil
}

func (s *SessionClient) PostForm(path string, data url.Values) (string, string, error) {
	form := url.Values{}
	for key, values := range data {
		copied := make([]string, len(values))
		copy(copied, values)
		form[key] = copied
	}

	if s.xsrfToken != "" && form.Get("atl_token") == "" {
		form.Set("atl_token", s.xsrfToken)
	}

	req, err := http.NewRequest(http.MethodPost, s.server+path, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.setAuth(req)

	finalURL := req.URL.String()
	client := *s.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		finalURL = req.URL.String()
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", finalURL, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", finalURL, err
	}

	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	bodyText := string(body)
	s.extractXSRFToken(resp, bodyText)

	return bodyText, finalURL, nil
}

func (s *SessionClient) GetXSRFToken() string {
	return s.xsrfToken
}

func (s *SessionClient) extractXSRFToken(resp *http.Response, body string) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "atlassian.xsrf.token" && cookie.Value != "" {
			s.xsrfToken = cookie.Value
			return
		}
	}

	match := atlTokenPattern.FindStringSubmatch(body)
	if len(match) == 2 {
		s.xsrfToken = match[1]
	}
}

func (s *SessionClient) setAuth(req *http.Request) {
	switch s.authType.String() {
	case string(AuthTypeMTLS):
		if s.token != "" {
			req.Header.Add("Authorization", "Bearer "+s.token)
		}
	case string(AuthTypeBearer):
		req.Header.Add("Authorization", "Bearer "+s.token)
	case string(AuthTypeBasic):
		req.SetBasicAuth(s.login, s.token)
	}
}
