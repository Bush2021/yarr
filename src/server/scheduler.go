package server

import "github.com/nkanaev/yarr/src/storage/model"

type FeedScheduler interface {
	FeedsPending() int32
	RefreshFeeds()
	RefreshFeed(feed *model.Feed)
	SetRefreshRate(minutes int64)
}
