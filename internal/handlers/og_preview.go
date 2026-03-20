package handlers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/net/html"
)

// OgPreviewHandler handles Open Graph metadata preview endpoints.
type OgPreviewHandler struct {
	client *http.Client
	cache  sync.Map // url string -> *ogCacheEntry
}

type ogCacheEntry struct {
	data      OgData
	expiresAt time.Time
}

// OgData holds parsed Open Graph metadata.
type OgData struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
}

// NewOgPreviewHandler creates a new OgPreviewHandler.
func NewOgPreviewHandler() *OgPreviewHandler {
	return &OgPreviewHandler{
		client: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// Preview returns Open Graph metadata for a URL.
// GET /api/og-preview?url=...
func (h *OgPreviewHandler) Preview(c fiber.Ctx) error {
	rawURL := c.Query("url")
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url query parameter required"})
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid URL"})
	}

	// Check cache
	if entry, ok := h.cache.Load(rawURL); ok {
		ce := entry.(*ogCacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return c.JSON(ce.data)
		}
		h.cache.Delete(rawURL)
	}

	data, err := h.fetchOg(rawURL, parsed)
	if err != nil {
		// Return minimal data on fetch error
		return c.JSON(OgData{
			URL:     rawURL,
			Favicon: "https://www.google.com/s2/favicons?domain=" + parsed.Hostname() + "&sz=32",
		})
	}

	// Cache for 10 minutes
	h.cache.Store(rawURL, &ogCacheEntry{data: data, expiresAt: time.Now().Add(10 * time.Minute)})

	return c.JSON(data)
}

func (h *OgPreviewHandler) fetchOg(rawURL string, parsed *url.URL) (OgData, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return OgData{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OrchestraBot/1.0)")
	req.Header.Set("Accept", "text/html")

	resp, err := h.client.Do(req)
	if err != nil {
		return OgData{}, err
	}
	defer resp.Body.Close()

	// Limit read to 512KB
	body := io.LimitReader(resp.Body, 512*1024)

	data := OgData{
		URL:     rawURL,
		Favicon: "https://www.google.com/s2/favicons?domain=" + parsed.Hostname() + "&sz=32",
	}

	tokenizer := html.NewTokenizer(body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			tn, hasAttr := tokenizer.TagName()
			tagName := string(tn)

			if tagName == "body" {
				break // Stop parsing at body
			}

			if tagName == "meta" && hasAttr {
				var property, name, content string
				for {
					key, val, more := tokenizer.TagAttr()
					k := string(key)
					v := string(val)
					switch k {
					case "property":
						property = v
					case "name":
						name = v
					case "content":
						content = v
					}
					if !more {
						break
					}
				}

				switch property {
				case "og:title":
					data.Title = content
				case "og:description":
					data.Description = content
				case "og:image":
					data.Image = content
				case "og:site_name":
					data.SiteName = content
				}

				// Fallback to meta name tags
				if data.Title == "" && name == "title" {
					data.Title = content
				}
				if data.Description == "" && name == "description" {
					data.Description = content
				}
			}

			if tagName == "title" && data.Title == "" {
				tokenizer.Next()
				data.Title = strings.TrimSpace(tokenizer.Token().Data)
			}
		}
	}

	// Truncate description
	if len(data.Description) > 200 {
		data.Description = data.Description[:197] + "..."
	}

	return data, nil
}
