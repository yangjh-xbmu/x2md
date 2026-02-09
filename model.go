package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// APIResponse is the top-level response from FxTwitter API.
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Tweet   *Tweet `json:"tweet"`
}

// Tweet represents a single tweet or article from FxTwitter.
type Tweet struct {
	ID               string   `json:"id"`
	URL              string   `json:"url"`
	Text             string   `json:"text"`
	CreatedAt        string   `json:"created_at"`
	CreatedTimestamp  int64    `json:"created_timestamp"`
	Likes            int      `json:"likes"`
	Retweets         int      `json:"retweets"`
	Replies          int      `json:"replies"`
	Views            int      `json:"views"`
	Bookmarks        int      `json:"bookmarks"`
	Lang             string   `json:"lang"`
	Source           string   `json:"source"`
	Author           *Author  `json:"author"`
	Media            *Media   `json:"media"`
	Quote            *Tweet   `json:"quote"`
	Poll             *Poll    `json:"poll"`
	ReplyingTo       string   `json:"replying_to"`
	ReplyingToStatus string   `json:"replying_to_status"`
	Article          *Article `json:"article"`
	ConversationID   string   `json:"conversation_id"`
}

// Author holds the tweet author's information.
type Author struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	AvatarURL  string `json:"avatar_url"`
	Followers  int    `json:"followers"`
	Following  int    `json:"following"`
}

// Media is a container for photos and videos attached to a tweet.
type Media struct {
	All    []MediaItem `json:"all"`
	Photos []Photo     `json:"photos"`
	Videos []Video     `json:"videos"`
}

// MediaItem represents a generic media item in the "all" array.
type MediaItem struct {
	Type         string `json:"type"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// Photo represents an image attached to a tweet.
type Photo struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	AltText string `json:"altText"`
}

// Video represents a video attached to a tweet.
type Video struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     float64 `json:"duration"`
}

// Poll represents a poll in a tweet.
type Poll struct {
	Choices    []PollChoice `json:"choices"`
	TotalVotes int          `json:"total_votes"`
	EndsAt     string       `json:"ends_at"`
	Ended      bool         `json:"ended"`
}

// PollChoice is a single option in a poll.
type PollChoice struct {
	Label      string  `json:"label"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// Article represents an X Article (long-form content).
// FxTwitter returns article content in Draft.js block format.
type Article struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	PreviewText   string          `json:"preview_text"`
	CoverMedia    *ArticleMedia   `json:"cover_media"`
	Content       *ArticleContent `json:"content"`
	MediaEntities []ArticleMedia  `json:"media_entities"`
	CreatedAt     string          `json:"created_at"`
	ModifiedAt    string          `json:"modified_at"`
}

// ArticleContent holds the Draft.js block structure.
type ArticleContent struct {
	Blocks    []Block         `json:"blocks"`
	EntityMap []EntityMapItem `json:"entityMap"`
}

// Block is a single Draft.js content block.
type Block struct {
	Key               string             `json:"key"`
	Text              string             `json:"text"`
	Type              string             `json:"type"`
	InlineStyleRanges []InlineStyleRange `json:"inlineStyleRanges"`
	EntityRanges      []EntityRange      `json:"entityRanges"`
}

// InlineStyleRange marks a range of text with a style (Bold, Italic, Code, etc.).
type InlineStyleRange struct {
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Style  string `json:"style"`
}

// EntityRange references an entity in the EntityMap by key.
type EntityRange struct {
	Key    int `json:"key"`
	Offset int `json:"offset"`
	Length int `json:"length"`
}

// EntityMapItem maps a key to an entity value.
// Note: FxTwitter returns "key" as a string (e.g. "0", "1").
type EntityMapItem struct {
	Key   FlexInt     `json:"key"`
	Value EntityValue `json:"value"`
}

// FlexInt handles JSON values that may be either a number or a string.
type FlexInt int

// EntityValue describes an entity (MEDIA, DIVIDER, LINK, etc.).
type EntityValue struct {
	Type       string          `json:"type"`
	Mutability string          `json:"mutability"`
	Data       EntityData      `json:"data"`
}

// EntityData holds entity-specific data.
type EntityData struct {
	EntityKey  string           `json:"entityKey"`
	MediaItems []EntityMediaRef `json:"mediaItems"`
	URL        string           `json:"url"`
}

// EntityMediaRef references a media item by mediaId.
type EntityMediaRef struct {
	LocalMediaID string `json:"localMediaId"`
	MediaID      string `json:"mediaId"`
}

// ArticleMedia represents a media entity in an article.
type ArticleMedia struct {
	ID        string         `json:"id"`
	MediaKey  string         `json:"media_key"`
	MediaID   string         `json:"media_id"`
	MediaInfo *MediaInfo     `json:"media_info"`
}

// MediaInfo holds the actual image/video info.
type MediaInfo struct {
	TypeName         string `json:"__typename"`
	OriginalImgURL   string `json:"original_img_url"`
	OriginalImgWidth int    `json:"original_img_width"`
	OriginalImgHeight int   `json:"original_img_height"`
}

// UnmarshalJSON handles both string ("0") and number (0) JSON values.
func (f *FlexInt) UnmarshalJSON(data []byte) error {
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		*f = FlexInt(intVal)
		return nil
	}
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		n, err := strconv.Atoi(strVal)
		if err != nil {
			return fmt.Errorf("FlexInt: cannot parse %q as int", strVal)
		}
		*f = FlexInt(n)
		return nil
	}
	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(data))
}

// URLType indicates whether a URL points to a tweet or an article.
type URLType int

const (
	URLTypeTweet   URLType = iota
	URLTypeArticle
)

// URLInfo holds parsed URL information.
type URLInfo struct {
	Type       URLType
	ScreenName string
	ID         string
	OriginalURL string
}
