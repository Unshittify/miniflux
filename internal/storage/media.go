// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"database/sql"
	"fmt"
	"miniflux.app/v2/internal/model"
)

// MediumByID returns a medium by the ID.
func (s *Storage) MediumByID(mediaID int64) (*model.Medium, error) {
	var media model.Medium
	query := `SELECT id, hash, mime_type, content, text FROM media WHERE id=$1`
	err := s.db.QueryRow(query, mediaID).Scan(&media.ID, &media.Hash, &media.MimeType, &media.Content, &media.Text)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("store: unable to fetch media #%d: %w", mediaID, err)
	}

	return &media, nil
}

// MediumBySrc returns medium by source url.
func (s *Storage) MediumBySrc(userID int64, src string) (*model.Medium, error) {
	var media = model.Medium{
		Entry: &model.EntryMedium{},
	}
	query := `
		SELECT
			m.id,
			m.hash,
			m.mime_type,
			m.content,
			m.text,
			em.src
		FROM entry_media as em
		LEFT JOIN 
		    media as m ON em.medium_id=m.id
		LEFT JOIN 
		    entries as e ON em.entry_id=e.id
		WHERE
		    e.user_id=$1
			and em.src=$2
		LIMIT 1
	`

	err := s.db.QueryRow(query, userID, src).Scan(&media.ID, &media.Hash, &media.MimeType, &media.Content, &media.Text, &media.Entry.Source)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("store: unable to fetch media %s: %w", src, err)
	}

	return &media, nil
}

// MediaByEntryId returns entry media.
func (s *Storage) MediaByEntryId(userID int64, entryID int64) (model.Media, error) {
	query := `
		SELECT
			media.id,
			media.hash,
			media.mime_type,
			media.content,
			media.text,
			entry_media.src
		FROM entry_media
		LEFT JOIN 
		    media ON entry_media.medium_id=media.id
		LEFT JOIN 
		    entry ON entry_media.entry_id=entry.id
		WHERE
		    entry.user_id=$1
			entry.id=$2
	`
	rows, err := s.db.Query(query, userID, entryID)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch media: %v`, err)
	}
	defer rows.Close()

	var media model.Media
	for rows.Next() {
		var medium model.Medium
		err := rows.Scan(&medium.ID, &medium.Hash, &medium.MimeType, &medium.Content, &medium.Text)
		if err != nil {
			return nil, fmt.Errorf(`store: unable to fetch media row: %v`, err)
		}
		media = append(media, &medium)
	}

	return media, nil
}

// MediumByHash returns a medium by the hash (checksum).
func (s *Storage) MediumByHash(medium *model.Medium) error {
	err := s.db.QueryRow(`SELECT id FROM media WHERE hash=$1`, medium.Hash).Scan(&medium.ID)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return fmt.Errorf(`store: unable to fetch medium by hash %q: %v`, medium.Hash, err)
	}

	return nil
}

// CreateMedium creates a new media.
func (s *Storage) CreateMedium(media *model.Medium) error {
	query := `
		INSERT INTO media
			(hash, mime_type, content, text, text_vectors)
		VALUES
			($1, $2, $3, $4, to_tsvector($4))
		RETURNING
			id
	`
	err := s.db.QueryRow(
		query,
		media.Hash,
		normalizeMimeType(media.MimeType),
		media.Content,
		media.Text,
	).Scan(&media.ID)

	if err != nil {
		return fmt.Errorf(`store: unable to create media: %v`, err)
	}

	return nil
}

// CreateEntryMedium creates a medium and associate the media to the given feed.
func (s *Storage) CreateEntryMedium(entryID int64, entrySourceUrl string, media *model.Medium) error {
	err := s.MediumByHash(media)
	if err != nil {
		return err
	}

	if media.ID == 0 {
		err := s.CreateMedium(media)
		if err != nil {
			return err
		}
	}

	_, err = s.db.Exec(`INSERT INTO entry_media (entry_id, medium_id, src) VALUES ($1, $2, $3)`, entryID, media.ID, entrySourceUrl)
	if err != nil {
		return fmt.Errorf(`store: unable to create feed media: %v`, err)
	}

	return nil
}

// DeleteEntryMedia deletes all media that belong to an entry
func (s *Storage) DeleteEntryMedia(entryID int64) error {
	_, err := s.db.Exec(`DELETE FROM entry_media WHERE entry_id = $1`, entryID)
	if err != nil {
		return fmt.Errorf(`store: unable to create feed media: %v`, err)
	}

	return nil
}

// MediaWithNoTexts returns medium by source url.
func (s *Storage) MediaWithNoTexts(limit int) (model.Media, error) {
	query := `
		SELECT
			m.id,
			m.hash,
			m.mime_type,
			m.content,
			m.text
		FROM media as m
		WHERE m.text = ''
		LIMIT $1
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch media: %v`, err)
	}
	defer rows.Close()

	var media model.Media
	for rows.Next() {
		var medium model.Medium
		var text sql.NullString
		err := rows.Scan(&medium.ID, &medium.Hash, &medium.MimeType, &medium.Content, &text)
		if err != nil {
			return nil, fmt.Errorf(`store: unable to fetch media row: %v`, err)
		}
		medium.Text = text.String
		media = append(media, &medium)
	}

	return media, nil
}

func (s *Storage) UpdateMediaText(media *model.Medium) (err error) {
	query := `
		UPDATE
			media
		SET
			text=$1,
			text_vectors=to_tsvector($1)
		WHERE
			id=$2
	`
	_, err = s.db.Exec(query,
		media.Text,
		media.ID,
	)

	if err != nil {
		return fmt.Errorf(`store: unable to media text #%d (%s): %v`, media.ID, media.Text, err)
	}

	return nil
}
