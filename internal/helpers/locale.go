package helpers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
)

// SupportedLocales lists the CMS content locales.
var SupportedLocales = map[string]bool{"en": true, "ar": true}

// DefaultLocale is the canonical content locale.
const DefaultLocale = "en"

// ContentLocale extracts the content locale from the ?locale= query param.
func ContentLocale(c fiber.Ctx) string {
	locale := c.Query("locale", DefaultLocale)
	if !SupportedLocales[locale] {
		return DefaultLocale
	}
	return locale
}

// ResolveTranslation extracts translated fields for a locale from a JSONB blob.
// Returns nil if locale is default or no translation exists.
func ResolveTranslation(translationsJSON []byte, locale string) map[string]interface{} {
	if locale == DefaultLocale || len(translationsJSON) == 0 {
		return nil
	}
	var all map[string]map[string]interface{}
	if err := json.Unmarshal(translationsJSON, &all); err != nil {
		return nil
	}
	return all[locale]
}

// MergeTranslation merges fields into the JSONB translations blob for the given locale.
func MergeTranslation(existing []byte, locale string, fields map[string]interface{}) []byte {
	var all map[string]map[string]interface{}
	if err := json.Unmarshal(existing, &all); err != nil || all == nil {
		all = make(map[string]map[string]interface{})
	}
	if all[locale] == nil {
		all[locale] = make(map[string]interface{})
	}
	for k, v := range fields {
		all[locale][k] = v
	}
	raw, _ := json.Marshal(all)
	return raw
}
