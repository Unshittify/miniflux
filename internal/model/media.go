// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"encoding/base64"
	"fmt"
)

// Medium represents an instance of external media like images and videos
type Medium struct {
	ID       int64  `json:"id"`
	Hash     string `json:"hash"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"-"`
	Text     string `json:"text"`

	Entry *EntryMedium `json:"entry"`
}

// DataURL returns the data URL of the media.
func (i *Medium) DataURL() string {
	return fmt.Sprintf("%s;base64,%s", i.MimeType, base64.StdEncoding.EncodeToString(i.Content))
}

// Media represents a list of icons.
type Media []*Medium

// EntryMedium is a junction table between entries and media.
type EntryMedium struct {
	EntryID  int64  `json:"entry_id"`
	MediumID int64  `json:"medium_id"`
	Source   string `json:"src"`
}

// MediaText represents an instance of a text transcription of external media
type MediaText struct {
	EntryID  int64  `json:"entry_id"`
	MediumID int64  `json:"medium_id"`
	Source   string `json:"src"`
}
