package main

import (
	"fmt"
	"strings"
)

const maxThreadDepth = 50

// FetchThread fetches an entire thread by traversing replying_to_status upward.
// It returns tweets in chronological order (oldest first).
func FetchThread(screenName, id string) ([]*Tweet, error) {
	var chain []*Tweet

	currentScreenName := screenName
	currentID := id

	for i := 0; i < maxThreadDepth; i++ {
		tweet, err := FetchTweet(currentScreenName, currentID)
		if err != nil {
			if len(chain) == 0 {
				return nil, fmt.Errorf("failed to fetch tweet %s: %w", currentID, err)
			}
			// If we fail to fetch a parent tweet, stop traversal and return what we have.
			break
		}

		chain = append(chain, tweet)

		// Check if this tweet is a reply to another tweet by the same author (thread).
		if tweet.ReplyingToStatus == "" {
			break
		}

		// Only follow the chain if replying to the same author (self-thread).
		if tweet.ReplyingTo != "" && tweet.Author != nil &&
			!strings.EqualFold(tweet.ReplyingTo, tweet.Author.ScreenName) {
			break
		}

		currentID = tweet.ReplyingToStatus
		// Use the same screen name for the parent tweet in the thread.
		if tweet.Author != nil {
			currentScreenName = tweet.Author.ScreenName
		}
	}

	// Reverse to chronological order (oldest first).
	reverse(chain)

	return chain, nil
}

// reverse reverses a slice of tweets in place.
func reverse(tweets []*Tweet) {
	for i, j := 0, len(tweets)-1; i < j; i, j = i+1, j-1 {
		tweets[i], tweets[j] = tweets[j], tweets[i]
	}
}

