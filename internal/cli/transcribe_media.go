// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cli // import "miniflux.app/v2/internal/cli"

import (
	"github.com/otiai10/gosseract/v2"
	"log/slog"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/contentmedia"
	"sync"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/storage"
)

func transcribeMedia(store *storage.Storage, count int) {
	var wg sync.WaitGroup

	startTime := time.Now()

	entries, err := store.MediaWithNoTexts(count)
	if err != nil {
		slog.Error("Unable to fetch jobs from database", slog.Any("error", err))
		return
	}

	nbJobs := len(entries)

	slog.Info("Created a batch of feeds",
		slog.Int("nb_jobs", nbJobs),
		slog.Int("batch_size", config.Opts.BatchSize()),
	)

	var jobQueue = make(chan *model.Medium, nbJobs)

	slog.Info("Starting a pool of workers",
		slog.Int("nb_workers", config.Opts.WorkerPoolSize()),
	)

	for i := range config.Opts.WorkerPoolSize() {
		wg.Add(1)
		go func(workerID int) {
			client := gosseract.NewClient()
			defer wg.Done()
			defer client.Close()
			for media := range jobQueue {
				slog.Info("Transcribing media",
					slog.Int64("media_id", media.ID),
					slog.Int("worker_id", workerID),
				)

				err = contentmedia.TranscribeMedia(client, store, media)

				if err != nil {
					slog.Error("Unable to transcribe media",
						slog.Int64("media_id", media.ID),
						slog.String("text", media.Text),
						slog.Any("error", err),
					)
					continue
				}

				slog.Info("Transcribed media",
					slog.Int64("media_id", media.ID),
					slog.String("text", media.Text),
				)
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
