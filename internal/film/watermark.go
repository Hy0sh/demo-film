package film

import (
	_ "embed"
	"html/template"
	"strings"

	"github.com/mxschmitt/playwright-go"
)

//go:embed watermark.html
var watermarkSource string

var watermarkPage = template.Must(template.New("watermark").Parse(watermarkSource))

// renderWatermark draws a text watermark into png, on a transparent
// background.
func renderWatermark(page playwright.Page, text, png string) error {
	var b strings.Builder
	if err := watermarkPage.Execute(&b, text); err != nil {
		return err
	}
	if err := page.SetContent(b.String()); err != nil {
		return err
	}
	_, err := page.Locator("#w").Screenshot(playwright.LocatorScreenshotOptions{Path: &png, OmitBackground: playwright.Bool(true)})
	return err
}
