package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func (s *Server) createClient(timeoutSeconds int) *http.Client {
	if timeoutSeconds <= 0 {
		timeoutSeconds = s.config.DefaultTimeoutSeconds
	}

	return &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !s.config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= s.config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func (s *Server) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http/https allowed)", u.Scheme)
	}

	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if isInternalIP(ip) {
			return fmt.Errorf("internal IP addresses are not allowed")
		}
	}

	for _, denied := range s.config.DeniedDomains {
		if strings.Contains(host, denied) {
			return fmt.Errorf("domain %s is blocked", host)
		}
	}

	if len(s.config.AllowedDomains) > 0 {
		allowed := false
		for _, domain := range s.config.AllowedDomains {
			if strings.Contains(host, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain %s is not in allowed list", host)
		}
	}

	return nil
}

func isInternalIP(ip net.IP) bool {
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	}

	for _, cidr := range privateCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *Server) fetchURLTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_url",
		Description: "Fetch the raw content of a URL",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":             mcp.StringProperty("URL to fetch"),
				"method":          mcp.StringProperty("HTTP method (default: GET)"),
				"headers":         mcp.MapProperty("Custom headers"),
				"body":            mcp.StringProperty("Request body"),
				"timeout_seconds": mcp.IntProperty("Request timeout"),
			},
			[]string{"url"},
		),
		Handler: s.handleFetchURL,
	}
}

func (s *Server) handleFetchURL(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	method, _ := mcp.GetStringParam(params, "method", false)
	if method == "" {
		method = "GET"
	}

	headers, _ := mcp.GetMapParam(params, "headers", false)
	body, _ := mcp.GetStringParam(params, "body", false)
	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, s.config.DefaultTimeoutSeconds)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := s.createClient(timeout)
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, int64(s.config.MaxResponseSizeBytes))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	fetchTime := time.Since(startTime)

	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	return mcp.JSONResult(map[string]interface{}{
		"url":            rawURL,
		"status_code":    resp.StatusCode,
		"headers":        respHeaders,
		"content":        string(content),
		"content_length": len(content),
		"fetch_time_ms":  fetchTime.Milliseconds(),
	})
}

func (s *Server) fetchHTMLTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_html",
		Description: "Fetch and return cleaned HTML",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":             mcp.StringProperty("URL to fetch"),
				"timeout_seconds": mcp.IntProperty("Request timeout"),
			},
			[]string{"url"},
		),
		Handler: s.handleFetchHTML,
	}
}

func (s *Server) handleFetchHTML(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, s.config.DefaultTimeoutSeconds)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	client := s.createClient(timeout)
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, int64(s.config.MaxResponseSizeBytes))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	cleanedHTML := cleanHTML(string(content))
	fetchTime := time.Since(startTime)

	return mcp.JSONResult(map[string]interface{}{
		"url":           rawURL,
		"status_code":   resp.StatusCode,
		"content":       cleanedHTML,
		"fetch_time_ms": fetchTime.Milliseconds(),
	})
}

func (s *Server) fetchTextTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_text",
		Description: "Fetch and extract text content (no HTML)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":             mcp.StringProperty("URL to fetch"),
				"timeout_seconds": mcp.IntProperty("Request timeout"),
			},
			[]string{"url"},
		),
		Handler: s.handleFetchText,
	}
}

func (s *Server) handleFetchText(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, s.config.DefaultTimeoutSeconds)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	client := s.createClient(timeout)
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, int64(s.config.MaxResponseSizeBytes))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	text, title := extractText(string(content))
	fetchTime := time.Since(startTime)

	return mcp.JSONResult(map[string]interface{}{
		"url":           rawURL,
		"status_code":   resp.StatusCode,
		"content_type":  resp.Header.Get("Content-Type"),
		"content":       text,
		"title":         title,
		"fetch_time_ms": fetchTime.Milliseconds(),
	})
}

func (s *Server) fetchMarkdownTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_markdown",
		Description: "Fetch and convert to Markdown",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":             mcp.StringProperty("URL to fetch"),
				"timeout_seconds": mcp.IntProperty("Request timeout"),
			},
			[]string{"url"},
		),
		Handler: s.handleFetchMarkdown,
	}
}

func (s *Server) handleFetchMarkdown(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, s.config.DefaultTimeoutSeconds)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	client := s.createClient(timeout)
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, int64(s.config.MaxResponseSizeBytes))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	markdown := htmlToMarkdown(string(content))
	fetchTime := time.Since(startTime)

	return mcp.JSONResult(map[string]interface{}{
		"url":           rawURL,
		"status_code":   resp.StatusCode,
		"content":       markdown,
		"fetch_time_ms": fetchTime.Milliseconds(),
	})
}

func (s *Server) fetchJSONTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_json",
		Description: "Fetch and parse JSON response",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":     mcp.StringProperty("URL to fetch"),
				"method":  mcp.StringProperty("HTTP method"),
				"headers": mcp.MapProperty("Custom headers"),
				"body":    mcp.StringProperty("Request body"),
			},
			[]string{"url"},
		),
		Handler: s.handleFetchJSON,
	}
}

func (s *Server) handleFetchJSON(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	method, _ := mcp.GetStringParam(params, "method", false)
	if method == "" {
		method = "GET"
	}

	headers, _ := mcp.GetMapParam(params, "headers", false)
	body, _ := mcp.GetStringParam(params, "body", false)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := s.createClient(s.config.DefaultTimeoutSeconds)
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, int64(s.config.MaxResponseSizeBytes))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	var jsonData interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	fetchTime := time.Since(startTime)

	return mcp.JSONResult(map[string]interface{}{
		"url":           rawURL,
		"status_code":   resp.StatusCode,
		"data":          jsonData,
		"fetch_time_ms": fetchTime.Milliseconds(),
	})
}

func (s *Server) extractLinksTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "extract_links",
		Description: "Extract all links from a webpage",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":            mcp.StringProperty("URL to fetch"),
				"filter_pattern": mcp.StringProperty("Regex pattern to filter links"),
			},
			[]string{"url"},
		),
		Handler: s.handleExtractLinks,
	}
}

func (s *Server) handleExtractLinks(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	rawURL, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	filterPattern, _ := mcp.GetStringParam(params, "filter_pattern", false)

	if err := s.validateURL(rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	client := s.createClient(s.config.DefaultTimeoutSeconds)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	baseURL, _ := url.Parse(rawURL)

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	var links []map[string]string
	seen := make(map[string]bool)

	var filter *regexp.Regexp
	if filterPattern != "" {
		filter, _ = regexp.Compile(filterPattern)
	}

	var extractLinks func(*html.Node)
	extractLinks = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, text, rel string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "href":
					href = attr.Val
				case "rel":
					rel = attr.Val
				}
			}

			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				text = strings.TrimSpace(n.FirstChild.Data)
			}

			if href != "" && !seen[href] {
				resolvedURL := href
				if parsedHref, err := url.Parse(href); err == nil {
					resolvedURL = baseURL.ResolveReference(parsedHref).String()
				}

				if filter == nil || filter.MatchString(resolvedURL) {
					seen[href] = true
					links = append(links, map[string]string{
						"href": resolvedURL,
						"text": text,
						"rel":  rel,
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractLinks(c)
		}
	}

	extractLinks(doc)

	return mcp.JSONResult(map[string]interface{}{
		"url":         rawURL,
		"links":       links,
		"total_count": len(links),
	})
}

func cleanHTML(content string) string {
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	commentRe := regexp.MustCompile(`<!--[\s\S]*?-->`)

	content = scriptRe.ReplaceAllString(content, "")
	content = styleRe.ReplaceAllString(content, "")
	content = commentRe.ReplaceAllString(content, "")

	return content
}

func extractText(content string) (string, string) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content, ""
	}

	var textBuilder strings.Builder
	var title string

	var extractNode func(*html.Node)
	extractNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				return
			case "title":
				if n.FirstChild != nil {
					title = n.FirstChild.Data
				}
				return
			case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li":
				textBuilder.WriteString("\n")
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				textBuilder.WriteString(text)
				textBuilder.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractNode(c)
		}
	}

	extractNode(doc)

	text := textBuilder.String()
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(text, "\n\n")
	text = strings.TrimSpace(text)

	return text, title
}

func htmlToMarkdown(content string) string {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content
	}

	var mdBuilder strings.Builder

	var convertNode func(*html.Node)
	convertNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				return
			case "h1":
				mdBuilder.WriteString("\n# ")
			case "h2":
				mdBuilder.WriteString("\n## ")
			case "h3":
				mdBuilder.WriteString("\n### ")
			case "h4":
				mdBuilder.WriteString("\n#### ")
			case "p":
				mdBuilder.WriteString("\n\n")
			case "br":
				mdBuilder.WriteString("\n")
			case "li":
				mdBuilder.WriteString("\n- ")
			case "strong", "b":
				mdBuilder.WriteString("**")
			case "em", "i":
				mdBuilder.WriteString("*")
			case "a":
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						mdBuilder.WriteString("[")
						for c := n.FirstChild; c != nil; c = c.NextSibling {
							convertNode(c)
						}
						mdBuilder.WriteString("](")
						mdBuilder.WriteString(attr.Val)
						mdBuilder.WriteString(")")
						return
					}
				}
			case "code":
				mdBuilder.WriteString("`")
			case "pre":
				mdBuilder.WriteString("\n```\n")
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				mdBuilder.WriteString(text)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(c)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "strong", "b":
				mdBuilder.WriteString("**")
			case "em", "i":
				mdBuilder.WriteString("*")
			case "code":
				mdBuilder.WriteString("`")
			case "pre":
				mdBuilder.WriteString("\n```\n")
			case "h1", "h2", "h3", "h4":
				mdBuilder.WriteString("\n")
			}
		}
	}

	convertNode(doc)

	md := mdBuilder.String()
	md = regexp.MustCompile(`\n{3,}`).ReplaceAllString(md, "\n\n")
	return strings.TrimSpace(md)
}
