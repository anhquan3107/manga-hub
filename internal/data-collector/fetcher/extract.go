package fetcher

// pickLocalizedText retrieves a localized text value from attributes, trying multiple languages.
func pickLocalizedText(attributes map[string]any, key string) string {
	raw, ok := attributes[key]
	if !ok {
		return ""
	}

	localized, ok := raw.(map[string]any)
	if !ok {
		return ""
	}

	// Try preferred languages first
	for _, lang := range []string{"en", "ja-ro", "ja"} {
		if value, ok := localized[lang].(string); ok && value != "" {
			return value
		}
	}

	// Fallback to any non-empty value
	for _, value := range localized {
		if text, ok := value.(string); ok && text != "" {
			return text
		}
	}

	return ""
}

// extractAuthor retrieves the author name from MangaDex relationships.
func extractAuthor(rels []struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}) string {
	for _, rel := range rels {
		if rel.Type != "author" {
			continue
		}
		if rel.Attributes == nil {
			continue
		}
		if name, ok := rel.Attributes["name"].(string); ok && name != "" {
			return name
		}
	}
	return ""
}

// extractTagNames retrieves tag names from MangaDex attributes.
func extractTagNames(attributes map[string]any) []string {
	rawTags, ok := attributes["tags"].([]any)
	if !ok {
		return nil
	}

	tags := make([]string, 0, len(rawTags))
	for _, rawTag := range rawTags {
		tagMap, ok := rawTag.(map[string]any)
		if !ok {
			continue
		}
		attr, ok := tagMap["attributes"].(map[string]any)
		if !ok {
			continue
		}
		name := pickLocalizedText(attr, "name")
		if name != "" {
			tags = append(tags, name)
		}
	}

	return tags
}
