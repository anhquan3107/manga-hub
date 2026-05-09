package sources

// Options specifies parameters for the Collect operation.
type Options struct {
	SeedFile string
	Source   string
	Limit    int
}

// Result contains the outcome of a Collect operation.
type Result struct {
	Source         string
	RequestedLimit int
	FetchedCount    int
	ExistingCount   int
	FinalCount      int
}

// mangadexResponse represents the JSON response from MangaDex API.
type mangadexResponse struct {
	Data []struct {
		ID            string         `json:"id"`
		Attributes    map[string]any `json:"attributes"`
		Relationships []struct {
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

// jikanResponse represents the JSON response from Jikan API.
type jikanResponse struct {
	Data []struct {
		MalID        int    `json:"mal_id"`
		Status       string `json:"status"`
		Chapters     int    `json:"chapters"`
		Synopsis     string `json:"synopsis"`
		Title        string `json:"title"`
		TitleEnglish string `json:"title_english"`
		Authors      []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Genres []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Demographics []struct {
			Name string `json:"name"`
		} `json:"demographics"`
	} `json:"data"`
}
