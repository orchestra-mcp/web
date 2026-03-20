package services

import (
	"fmt"
	"html"
	"strings"

	"github.com/orchestra-mcp/web/internal/models"
)

// ExportService handles presentation export to various formats.
type ExportService struct{}

func NewExportService() *ExportService {
	return &ExportService{}
}

// SlideHTML renders a single slide as an HTML string for PDF generation.
func (s *ExportService) SlideHTML(slide models.PresentationSlide) string {
	var b strings.Builder
	b.WriteString(`<section class="slide" data-layout="`)
	b.WriteString(html.EscapeString(slide.Layout))
	b.WriteString(`">`)

	if slide.Title != "" {
		b.WriteString(`<h2>`)
		b.WriteString(html.EscapeString(slide.Title))
		b.WriteString(`</h2>`)
	}

	if slide.Content != "" {
		b.WriteString(`<div class="content">`)
		b.WriteString(markdownToHTMLBasic(slide.Content))
		b.WriteString(`</div>`)
	}

	b.WriteString(`</section>`)
	return b.String()
}

// RenderPresentationHTML generates a full HTML document for all slides.
// This can be captured by a headless browser (wkhtmltopdf, Chromium) to produce PDF.
func (s *ExportService) RenderPresentationHTML(pres models.Presentation, slides []models.PresentationSlide) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>`)
	b.WriteString(html.EscapeString(pres.Title))
	b.WriteString(`</title>
<style>
  @page { size: landscape; margin: 0; }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: system-ui, -apple-system, sans-serif; color: #1a1a1a; }
  .slide {
    width: 100vw; height: 100vh;
    display: flex; flex-direction: column;
    justify-content: center; align-items: center;
    padding: 60px 80px;
    page-break-after: always;
    position: relative;
  }
  .slide h2 { font-size: 2.5rem; margin-bottom: 1.5rem; text-align: center; }
  .slide .content { font-size: 1.25rem; line-height: 1.8; max-width: 80%; }
  .slide .content ul, .slide .content ol { margin-left: 2rem; }
  .slide .content li { margin-bottom: 0.5rem; }
  .slide .content p { margin-bottom: 1rem; }
  .slide .content pre { background: #f5f5f5; padding: 1rem; border-radius: 8px; overflow-x: auto; font-size: 0.9rem; }
  .slide .content code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; }
  .slide[data-layout="title"] { justify-content: center; }
  .slide[data-layout="title"] h2 { font-size: 3.5rem; }
  .slide[data-layout="two-column"] .content { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; }
  .slide[data-layout="quote"] .content { font-style: italic; font-size: 1.5rem; text-align: center; }
  .slide[data-layout="image-full"] { padding: 0; }
  .slide-number { position: absolute; bottom: 20px; right: 40px; font-size: 0.8rem; color: #999; }
</style>
</head>
<body>
`)

	for i, slide := range slides {
		b.WriteString(s.SlideHTML(slide))
		b.WriteString(fmt.Sprintf(`<div class="slide-number">%d / %d</div>`, i+1, len(slides)))
	}

	b.WriteString(`
</body>
</html>`)

	return b.String()
}

// GeneratePPTXMarkdown generates a simplified markdown representation
// suitable for conversion to PPTX via go-pptx or external tools.
func (s *ExportService) GeneratePPTXMarkdown(pres models.Presentation, slides []models.PresentationSlide) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(pres.Title)
	b.WriteString("\n\n")

	if pres.Description != "" {
		b.WriteString("> ")
		b.WriteString(pres.Description)
		b.WriteString("\n\n")
	}

	for i, slide := range slides {
		b.WriteString(fmt.Sprintf("---\n\n## Slide %d", i+1))
		if slide.Title != "" {
			b.WriteString(": ")
			b.WriteString(slide.Title)
		}
		b.WriteString("\n\n")

		if slide.Content != "" {
			b.WriteString(slide.Content)
			b.WriteString("\n\n")
		}

		if slide.Notes != "" {
			b.WriteString("**Notes:** ")
			b.WriteString(slide.Notes)
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

// GoogleSlidesURL returns a Google Slides creation URL with pre-filled title.
func (s *ExportService) GoogleSlidesURL(pres models.Presentation) string {
	return "https://docs.google.com/presentation/create?title=" + strings.ReplaceAll(pres.Title, " ", "+")
}

// markdownToHTMLBasic does minimal markdown to HTML conversion for slide content.
func markdownToHTMLBasic(content string) string {
	var b strings.Builder
	lines := strings.Split(content, "\n")
	inList := false
	inPre := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inPre {
				b.WriteString("</code></pre>")
				inPre = false
			} else {
				b.WriteString("<pre><code>")
				inPre = true
			}
			continue
		}
		if inPre {
			b.WriteString(html.EscapeString(line))
			b.WriteString("\n")
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				b.WriteString("<ul>")
				inList = true
			}
			b.WriteString("<li>")
			b.WriteString(html.EscapeString(trimmed[2:]))
			b.WriteString("</li>")
			continue
		}
		if inList {
			b.WriteString("</ul>")
			inList = false
		}

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(line, "### ") {
			b.WriteString("<h4>")
			b.WriteString(html.EscapeString(strings.TrimPrefix(line, "### ")))
			b.WriteString("</h4>")
		} else if strings.HasPrefix(line, "## ") {
			b.WriteString("<h3>")
			b.WriteString(html.EscapeString(strings.TrimPrefix(line, "## ")))
			b.WriteString("</h3>")
		} else if strings.HasPrefix(line, "# ") {
			b.WriteString("<h2>")
			b.WriteString(html.EscapeString(strings.TrimPrefix(line, "# ")))
			b.WriteString("</h2>")
		} else {
			b.WriteString("<p>")
			b.WriteString(html.EscapeString(line))
			b.WriteString("</p>")
		}
	}

	if inList {
		b.WriteString("</ul>")
	}
	return b.String()
}
