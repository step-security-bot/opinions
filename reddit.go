package opinions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/macie/opinions/http"
)

// RedditResponse represents some interesting fields of response from Reddit API.
type RedditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID          string `json:"permalink"`
				Title       string `json:"title"`
				URL         string `json:"url"`
				NumComments int    `json:"num_comments"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// SearchReddit searches Reddit for given query and returns list of discussions
// sorted by relevance.
//
// See: https://www.reddit.com/dev/api#GET_search
func SearchReddit(ctx context.Context, client http.Client, query string) ([]Discussion, error) {
	discussions := make([]Discussion, 0)
	searchURL := "https://www.reddit.com/search.json?sort=relevance&t=all&q="

	r, err := client.Get(ctx, searchURL+url.QueryEscape(query))
	if err != nil {
		return discussions, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		if r.Header.Get("X-Ratelimit-Remaining") == "0" { // https://support.reddithelp.com/hc/en-us/articles/16160319875092-Reddit-Data-API-Wiki
			return discussions, fmt.Errorf("cannot search Reddit: too many requests. Wait %s seconds", r.Header.Get("X-Ratelimit-Reset"))
		}

		return discussions, fmt.Errorf("cannot search Reddit: `GET %s` responded with status code %d", r.Request.URL, r.StatusCode)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return discussions, err
	}

	var response RedditResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return discussions, err
	}

	for _, entry := range response.Data.Children {
		discussions = append(discussions, Discussion{
			Service:  "Reddit",
			URL:      "https://reddit.com" + entry.Data.ID,
			Title:    entry.Data.Title,
			Source:   entry.Data.URL,
			Comments: entry.Data.NumComments,
		})
	}

	return discussions, nil
}
