/**
 * File              : structs.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 14.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

type SlackVerificationStruct struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
}

type GiphyResponse struct {
	Meta       giphyResponseMeta       `json:"meta"`
	Pagination giphyResponsePagination `json:"pagination,omitempty"`
	Data       []giphyResponseData     `json:"data,omitempty"`
}

type giphyResponseMeta struct {
	Msg        string `json:"msg"`
	Status     int    `json:"status"`
	ResponseID string `json:"response_id"`
}

type giphyResponsePagination struct {
	Total  int `json:"total_count"`
	Actual int `json:"count"`
	Offset int `json:"offset"`
}

type giphyResponseData struct {
	Type       string                  `json:"type,omitempty"`
	Id         string                  `json:"id,omitempty"`
	Url        string                  `json:"url,omitempty"`
	Slug       string                  `json:"slug,omitempty"`
	Username   string                  `json:"username,omitempty"`
	Source     string                  `json:"source,omitempty"`
	Title      string                  `json:"title,omitempty"`
	Rating     string                  `json:"rating,omitempty"`
	ContentUrl string                  `json:"content_url,omitempty"`
	Images     giphyResponseDataImages `json:"images,omitempty"`
}

type giphyResponseDataImages struct {
	Original  giphyResponseDataImage `json:"original"`
	DownSized giphyResponseDataImage `json:"downsized"`
}

type giphyResponseDataImage struct {
	Height string `json:"height,omitempty"`
	Width  string `json:"width,omitempty"`
	Size   string `json:"size,omitempty"`
	Url    string `json:"url,omitempty"`
	Frames string `json:"frames,omitempty"`
	Hash   string `json:"hash,omitempty"`
}
