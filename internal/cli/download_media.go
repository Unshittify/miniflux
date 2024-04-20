// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"log/slog"
	"miniflux.app/v2/internal/model"
	"sync"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/reader/contentmedia"
	"miniflux.app/v2/internal/storage"
)

func downloadMedia(store *storage.Storage, count int) {
	var wg sync.WaitGroup

	startTime := time.Now()

	builder := storage.NewAnonymousQueryBuilder(store)
	builder.With("e.archived_at is null")
	builder.WithSorting("created_at", "asc")
	builder.WithLimit(count)

	entries, err := builder.GetEntries()
	if err != nil {
		slog.Error("Unable to fetch jobs from database", slog.Any("error", err))
		return
	}

	nbJobs := len(entries)

	slog.Info("Created a batch of feeds",
		slog.Int("nb_jobs", nbJobs),
		slog.Int("batch_size", config.Opts.BatchSize()),
	)

	var jobQueue = make(chan *model.Entry, nbJobs)

	slog.Info("Starting a pool of workers",
		slog.Int("nb_workers", config.Opts.WorkerPoolSize()),
	)

	for i := range config.Opts.WorkerPoolSize() {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for entry := range jobQueue {
				slog.Info("Downloading media for entry",
					slog.Int64("entry_id", entry.ID),
					slog.Int64("feed_id", entry.FeedID),
					slog.Int("worker_id", workerID),
				)

				err = contentmedia.FetchEntryMedia(store, entry, nil)

				if err != nil {
					slog.Error("Unable to delete entry media", slog.Any("error", err))
					continue
				}
			}
		}(i)
	}

	for _, job := range entries {
		jobQueue <- job
	}
	close(jobQueue)

	wg.Wait()

	slog.Info("Refreshed a batch of feeds",
		slog.Int("nb_feeds", nbJobs),
		slog.String("duration", time.Since(startTime).String()),
	)
}
