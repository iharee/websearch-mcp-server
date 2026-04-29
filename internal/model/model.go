package model

type SearchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type FetchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
