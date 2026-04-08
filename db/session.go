package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
)

type SessionMeta struct {
	ID             int64  `json:"id"`
	SessionID      string `json:"session_id"`
	SessionKey     string `json:"session_key"`
	Channel        string `json:"channel"`
	AccountID      string `json:"account_id"`
	PeerID         string `json:"peer_id"`
	GroupID        string `json:"group_id"`
	ThreadID       string `json:"thread_id"`
	Kind           string `json:"kind"`
	TranscriptPath string `json:"transcript_path"`
	Summary        string `json:"summary"`
	MessageCount   int    `json:"message_count"`
	LastMessageAt  int64  `json:"last_message_at"`
	CreateTime     int64  `json:"create_time"`
	UpdateTime     int64  `json:"update_time"`
}

func UpsertSessionMeta(meta *SessionMeta) error {
	if DB == nil || meta == nil || meta.SessionID == "" {
		return nil
	}

	now := time.Now().Unix()
	if meta.CreateTime == 0 {
		meta.CreateTime = now
	}
	meta.UpdateTime = now

	if conf.BaseConfInfo.DBType == "mysql" {
		query := `
		INSERT INTO sessions (
			session_id, session_key, channel, account_id, peer_id, group_id, thread_id, kind,
			transcript_path, summary, message_count, last_message_at, create_time, update_time, from_bot
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			session_key = VALUES(session_key),
			channel = VALUES(channel),
			account_id = VALUES(account_id),
			peer_id = VALUES(peer_id),
			group_id = VALUES(group_id),
			thread_id = VALUES(thread_id),
			kind = VALUES(kind),
			transcript_path = VALUES(transcript_path),
			summary = VALUES(summary),
			message_count = VALUES(message_count),
			last_message_at = VALUES(last_message_at),
			update_time = VALUES(update_time),
			from_bot = VALUES(from_bot)
		`
		_, err := DB.Exec(query,
			meta.SessionID, meta.SessionKey, meta.Channel, meta.AccountID, meta.PeerID, meta.GroupID,
			meta.ThreadID, meta.Kind, meta.TranscriptPath, meta.Summary, meta.MessageCount,
			meta.LastMessageAt, meta.CreateTime, meta.UpdateTime, conf.BaseConfInfo.BotName,
		)
		return err
	}

	query := `
	INSERT INTO sessions (
		session_id, session_key, channel, account_id, peer_id, group_id, thread_id, kind,
		transcript_path, summary, message_count, last_message_at, create_time, update_time, from_bot
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_id) DO UPDATE SET
		session_key = excluded.session_key,
		channel = excluded.channel,
		account_id = excluded.account_id,
		peer_id = excluded.peer_id,
		group_id = excluded.group_id,
		thread_id = excluded.thread_id,
		kind = excluded.kind,
		transcript_path = excluded.transcript_path,
		summary = excluded.summary,
		message_count = excluded.message_count,
		last_message_at = excluded.last_message_at,
		update_time = excluded.update_time,
		from_bot = excluded.from_bot
	`
	_, err := DB.Exec(query,
		meta.SessionID, meta.SessionKey, meta.Channel, meta.AccountID, meta.PeerID, meta.GroupID,
		meta.ThreadID, meta.Kind, meta.TranscriptPath, meta.Summary, meta.MessageCount,
		meta.LastMessageAt, meta.CreateTime, meta.UpdateTime, conf.BaseConfInfo.BotName,
	)
	return err
}

func GetSessionMeta(sessionID string) (*SessionMeta, error) {
	if DB == nil || sessionID == "" {
		return nil, nil
	}

	query := `
	SELECT id, session_id, session_key, channel, account_id, peer_id, group_id, thread_id, kind,
		transcript_path, summary, message_count, last_message_at, create_time, update_time
	FROM sessions WHERE session_id = ?
	`
	row := DB.QueryRow(query, sessionID)

	meta := new(SessionMeta)
	err := row.Scan(
		&meta.ID, &meta.SessionID, &meta.SessionKey, &meta.Channel, &meta.AccountID, &meta.PeerID,
		&meta.GroupID, &meta.ThreadID, &meta.Kind, &meta.TranscriptPath, &meta.Summary,
		&meta.MessageCount, &meta.LastMessageAt, &meta.CreateTime, &meta.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return meta, nil
}

func ListSessionMeta(limit int) ([]SessionMeta, error) {
	if DB == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
	SELECT id, session_id, session_key, channel, account_id, peer_id, group_id, thread_id, kind,
		transcript_path, summary, message_count, last_message_at, create_time, update_time
	FROM sessions
	ORDER BY update_time DESC
	LIMIT %d
	`, limit)

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metas := make([]SessionMeta, 0, limit)
	for rows.Next() {
		var meta SessionMeta
		if err := rows.Scan(
			&meta.ID, &meta.SessionID, &meta.SessionKey, &meta.Channel, &meta.AccountID, &meta.PeerID,
			&meta.GroupID, &meta.ThreadID, &meta.Kind, &meta.TranscriptPath, &meta.Summary,
			&meta.MessageCount, &meta.LastMessageAt, &meta.CreateTime, &meta.UpdateTime,
		); err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}

	return metas, rows.Err()
}
