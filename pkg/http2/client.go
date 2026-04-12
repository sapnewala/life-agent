package http2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ============================================================
// 타입 정의
// ============================================================

// Method는 HTTP 메서드를 나타냅니다.
type Method string

const (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	PATCH  Method = "PATCH"
	DELETE Method = "DELETE"
)

// ContentType은 요청 본문의 Content-Type을 나타냅니다.
type ContentType string

const (
	ContentTypeJSON ContentType = "application/json"
	ContentTypeForm ContentType = "application/x-www-form-urlencoded"
)

// RetryConfig는 재시도 정책을 설정합니다.
type RetryConfig struct {
	MaxAttempts  int           // 최대 시도 횟수 (초기 요청 포함). 0이면 재시도 없음
	WaitBase     time.Duration // 첫 번째 재시도 대기 시간
	WaitMax      time.Duration // 최대 대기 시간 (지수 백오프 상한)
	Backoff      BackoffStrategy
	RetryOnCodes []int // 재시도할 HTTP 상태 코드 목록 (기본: 429, 500, 502, 503, 504)
}

// BackoffStrategy는 재시도 대기 시간 계산 전략입니다.
type BackoffStrategy int

const (
	BackoffFixed       BackoffStrategy = iota // 고정 대기
	BackoffExponential                        // 지수 백오프 (기본)
	BackoffLinear                             // 선형 증가
)

// Request는 HTTP 요청에 필요한 모든 옵션을 담습니다.
type Request struct {
	Method      Method
	URL         string
	Headers     map[string]string
	QueryParams map[string]string
	Body        interface{} // JSON: struct / map, Form: map[string]string 또는 url.Values
	ContentType ContentType
	Timeout     time.Duration
	Retry       RetryConfig
	Context     context.Context // nil이면 context.Background() 사용
}

// Response는 HTTP 응답을 래핑합니다.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Attempts   int // 실제 시도 횟수
}

// DecodeJSON은 응답 본문을 JSON으로 디코딩합니다.
func (r *Response) DecodeJSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// ============================================================
// 기본값
// ============================================================

var defaultRetryOnCodes = []int{429, 500, 502, 503, 504}

// DefaultRetryConfig는 일반적인 환경에 적합한 재시도 기본 설정입니다.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	WaitBase:     300 * time.Millisecond,
	WaitMax:      5 * time.Second,
	Backoff:      BackoffExponential,
	RetryOnCodes: defaultRetryOnCodes,
}

// ============================================================
// 핵심 함수
// ============================================================

// Do는 범용 HTTP 요청을 실행합니다.
// GET/POST(JSON/Form), 재시도, 타임아웃을 모두 지원합니다.
func Do(req Request) (*Response, error) {
	// --- 컨텍스트 ---
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// 타임아웃이 설정된 경우 컨텍스트에 데드라인 추가
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// --- URL 구성 ---
	rawURL, err := buildURL(req.URL, req.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("httpclient: URL 구성 실패: %w", err)
	}

	// --- 재시도 설정 정규화 ---
	retry := req.Retry
	if retry.MaxAttempts <= 0 {
		retry.MaxAttempts = 1 // 재시도 없이 1회 시도
	}
	if len(retry.RetryOnCodes) == 0 {
		retry.RetryOnCodes = defaultRetryOnCodes
	}
	if retry.WaitBase == 0 {
		retry.WaitBase = 300 * time.Millisecond
	}
	if retry.WaitMax == 0 {
		retry.WaitMax = 30 * time.Second
	}

	// --- HTTP 클라이언트 (타임아웃은 컨텍스트로 제어) ---
	client := &http.Client{}

	var (
		resp    *Response
		lastErr error
	)

	for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
		// 요청 본문 빌드 (매 시도마다 새로 생성 — Body 스트림 재사용 방지)
		httpReq, err := buildHTTPRequest(ctx, req.Method, rawURL, req.Body, req.ContentType, req.Headers)
		if err != nil {
			return nil, fmt.Errorf("httpclient: 요청 생성 실패: %w", err)
		}

		httpResp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("httpclient: 요청 실패 (시도 %d/%d): %w", attempt, retry.MaxAttempts, err)
			// 네트워크 오류 → 재시도 대기 후 계속
			if attempt < retry.MaxAttempts {
				wait := calcWait(retry, attempt)
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("httpclient: 컨텍스트 취소됨: %w", ctx.Err())
				case <-time.After(wait):
				}
			}
			continue
		}

		body, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("httpclient: 응답 본문 읽기 실패: %w", readErr)
		}

		resp = &Response{
			StatusCode: httpResp.StatusCode,
			Headers:    httpResp.Header,
			Body:       body,
			Attempts:   attempt,
		}

		// 재시도 대상 상태 코드 확인
		if attempt < retry.MaxAttempts && containsCode(retry.RetryOnCodes, httpResp.StatusCode) {
			lastErr = fmt.Errorf("httpclient: 재시도 대상 상태 코드 %d (시도 %d/%d)", httpResp.StatusCode, attempt, retry.MaxAttempts)
			wait := calcWait(retry, attempt)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("httpclient: 컨텍스트 취소됨: %w", ctx.Err())
			case <-time.After(wait):
			}
			continue
		}

		// 성공 또는 재시도 불필요
		return resp, nil
	}

	// 모든 시도 소진
	if resp != nil {
		return resp, lastErr
	}
	return nil, lastErr
}

// ============================================================
// 편의 함수
// ============================================================

// Get은 GET 요청을 수행합니다.
//
//	resp, err := httpclient.Get("https://api.example.com/users",
//	    map[string]string{"page": "1"},
//	    nil,
//	    5*time.Second,
//	    httpclient.DefaultRetryConfig,
//	)
func Get(
	rawURL string,
	queryParams map[string]string,
	headers map[string]string,
	timeout time.Duration,
	retry RetryConfig,
) (*Response, error) {
	return Do(Request{
		Method:      GET,
		URL:         rawURL,
		QueryParams: queryParams,
		Headers:     headers,
		Timeout:     timeout,
		Retry:       retry,
	})
}

// PostJSON은 JSON 본문으로 POST 요청을 수행합니다.
//
//	resp, err := httpclient.PostJSON("https://api.example.com/users",
//	    map[string]interface{}{"name": "Alice"},
//	    nil,
//	    5*time.Second,
//	    httpclient.DefaultRetryConfig,
//	)
func PostJSON(
	rawURL string,
	body interface{},
	headers map[string]string,
	timeout time.Duration,
	retry RetryConfig,
) (*Response, error) {
	return Do(Request{
		Method:      POST,
		URL:         rawURL,
		Body:        body,
		ContentType: ContentTypeJSON,
		Headers:     headers,
		Timeout:     timeout,
		Retry:       retry,
	})
}

// PostForm은 Form 데이터로 POST 요청을 수행합니다.
//
//	resp, err := httpclient.PostForm("https://api.example.com/login",
//	    map[string]string{"username": "alice", "password": "secret"},
//	    nil,
//	    5*time.Second,
//	    httpclient.DefaultRetryConfig,
//	)
func PostForm(
	rawURL string,
	formData map[string]string,
	headers map[string]string,
	timeout time.Duration,
	retry RetryConfig,
) (*Response, error) {
	return Do(Request{
		Method:      POST,
		URL:         rawURL,
		Body:        formData,
		ContentType: ContentTypeForm,
		Headers:     headers,
		Timeout:     timeout,
		Retry:       retry,
	})
}

// ============================================================
// 내부 헬퍼
// ============================================================

func buildURL(rawURL string, params map[string]string) (string, error) {
	if len(params) == 0 {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildHTTPRequest(
	ctx context.Context,
	method Method,
	rawURL string,
	body interface{},
	ct ContentType,
	headers map[string]string,
) (*http.Request, error) {
	var (
		bodyReader  io.Reader
		contentType string
	)

	if body != nil {
		switch ct {
		case ContentTypeForm:
			encoded, err := encodeForm(body)
			if err != nil {
				return nil, err
			}
			bodyReader = strings.NewReader(encoded)
			contentType = string(ContentTypeForm)

		default: // JSON (기본값)
			b, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("JSON 직렬화 실패: %w", err)
			}
			bodyReader = bytes.NewReader(b)
			contentType = string(ContentTypeJSON)
		}
	}

	req, err := http.NewRequestWithContext(ctx, string(method), rawURL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Content-Type 설정
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 사용자 정의 헤더 (Content-Type 덮어쓰기 가능)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// encodeForm은 map[string]string 또는 url.Values를 form-encoded 문자열로 변환합니다.
func encodeForm(body interface{}) (string, error) {
	switch v := body.(type) {
	case url.Values:
		return v.Encode(), nil
	case map[string]string:
		vals := url.Values{}
		for key, val := range v {
			vals.Set(key, val)
		}
		return vals.Encode(), nil
	case map[string][]string:
		return url.Values(v).Encode(), nil
	default:
		return "", fmt.Errorf("form 본문은 map[string]string 또는 url.Values 여야 합니다. 받은 타입: %T", body)
	}
}

// calcWait는 재시도 전략에 따라 대기 시간을 계산합니다.
func calcWait(cfg RetryConfig, attempt int) time.Duration {
	var wait time.Duration
	switch cfg.Backoff {
	case BackoffFixed:
		wait = cfg.WaitBase
	case BackoffLinear:
		wait = cfg.WaitBase * time.Duration(attempt)
	default: // BackoffExponential
		multiplier := math.Pow(2, float64(attempt-1))
		wait = time.Duration(float64(cfg.WaitBase) * multiplier)
	}
	if wait > cfg.WaitMax {
		wait = cfg.WaitMax
	}
	return wait
}

func containsCode(codes []int, code int) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
