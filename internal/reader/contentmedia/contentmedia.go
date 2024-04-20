// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package contentmedia // import "miniflux.app/v2/internal/reader/contentmedia"

import (
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"log/slog"
	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/storage"
	"regexp"
)

var imgRegex = regexp.MustCompile("<img.*src=\"(?P<imgsrc>.*?)\"")
var requestBuilder = fetcher.NewRequestBuilder()

func DownloadFile(mediaURL string) (*model.Medium, error) {
	slog.Debug("Downloading media",
		slog.String("media_url", mediaURL),
	)

	responseHandler := fetcher.NewResponseHandler(requestBuilder.ExecuteRequest(mediaURL))
	defer responseHandler.Close()

	if localizedError := responseHandler.LocalizedError(); localizedError != nil {
		return nil, fmt.Errorf("contentmedia: unable to download (%s): %w", mediaURL, localizedError.Error())
	}

	responseBody, localizedError := responseHandler.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if localizedError != nil {
		return nil, fmt.Errorf("contentmedia: unable to read response body (%s): %w", mediaURL, localizedError.Error())
	}

	media := &model.Medium{
		Hash:     crypto.HashFromBytes(responseBody),
		MimeType: responseHandler.ContentType(),
		Content:  responseBody,
	}

	return media, nil
}

func FetchEntryMedia(store *storage.Storage, entry *model.Entry, client *gosseract.Client) error {
	err := store.DeleteEntryMedia(entry.ID)
	if err != nil {
		return fmt.Errorf("contentmedia: unable to get entries: %v", err)
	}

	matches := imgRegex.FindAllStringSubmatch(entry.Content, -1)

	slog.Info("Image tags found:",
		slog.Int("matches", len(matches)),
		slog.Int64("entry_id", entry.ID),
		slog.Int64("feed_id", entry.FeedID),
	)

	for _, match := range matches {
		url := match[1]
		medium, fileErr := DownloadFile(url)

		if fileErr != nil {
			slog.Warn("Unable to download media",
				slog.Int64("feedID", entry.FeedID),
				slog.Int64("entryID", entry.ID),
				slog.Any("error", fileErr),
			)
			continue
		}

		fileErr = store.CreateEntryMedium(entry.ID, url, medium)

		if fileErr != nil {
			slog.Warn("Unable to save media for entry",
				slog.Int64("feedID", entry.FeedID),
				slog.Int64("entryID", entry.ID),
				slog.Any("error", fileErr),
			)
			continue
		}

		if client != nil {
			fileErr = TranscribeMedia(client, store, medium)

			if fileErr != nil {
				slog.Warn("Unable transcribe media for entry",
					slog.Int64("feedID", entry.FeedID),
					slog.Int64("entryID", entry.ID),
					slog.Int64("mediaID", medium.ID),
					slog.Any("error", fileErr),
				)
			}
			continue
		}
	}

	err = store.UpdateArchiveDate(entry.UserID, entry.ID)
	if err != nil {
		return fmt.Errorf("contentmedia: to delete entry media: %v", err)
	}

	return nil
}

func TranscribeMedia(client *gosseract.Client, store *storage.Storage, media *model.Medium) error {
	err := client.SetImageFromBytes(media.Content)

	if err != nil {
		return fmt.Errorf("contentmedia: unabble to set image: %v", err)
	}

	media.Text, err = client.Text()

	if err != nil {
		return fmt.Errorf("contentmedia: unabble to parse image: %v", err)
	}

	if media.Text == "" {
		media.Text = "-" // set something to recognize entries that were processed
	}

	return store.UpdateMediaText(media)
}
