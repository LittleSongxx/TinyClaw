package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/conf"
)

const (
	DeviceStatusActive  = "active"
	DeviceStatusRevoked = "revoked"

	PairingStatusPending  = "pending"
	PairingStatusApproved = "approved"
	PairingStatusRejected = "rejected"
	PairingStatusExpired  = "expired"
)

type DeviceRecord struct {
	DeviceID        string            `json:"device_id"`
	WorkspaceID     string            `json:"workspace_id"`
	DeviceTokenHash string            `json:"-"`
	PublicKey       string            `json:"public_key"`
	Name            string            `json:"name"`
	Platform        string            `json:"platform"`
	DeviceFamily    string            `json:"device_family,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Status          string            `json:"status"`
	CreateTime      int64             `json:"create_time"`
	UpdateTime      int64             `json:"update_time"`
	LastSeenAt      int64             `json:"last_seen_at"`
	RevokedAt       int64             `json:"revoked_at,omitempty"`
}

type DevicePairingRequest struct {
	RequestID          string                 `json:"request_id"`
	WorkspaceID        string                 `json:"workspace_id"`
	BootstrapTokenHash string                 `json:"-"`
	BootstrapCodeHash  string                 `json:"-"`
	DeviceID           string                 `json:"device_id"`
	PublicKey          string                 `json:"public_key"`
	Descriptor         map[string]interface{} `json:"descriptor,omitempty"`
	Status             string                 `json:"status"`
	ExpiresAt          int64                  `json:"expires_at"`
	IssuedTokenHash    string                 `json:"-"`
	CreateTime         int64                  `json:"create_time"`
	UpdateTime         int64                  `json:"update_time"`
}

func InsertDevicePairingRequest(ctx context.Context, req DevicePairingRequest) error {
	if DB == nil {
		return nil
	}
	req.WorkspaceID = authz.NormalizeWorkspaceID(req.WorkspaceID)
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.Status = normalizePairingStatus(req.Status)
	if req.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	now := time.Now().Unix()
	if req.CreateTime == 0 {
		req.CreateTime = now
	}
	req.UpdateTime = now
	descriptor, _ := json.Marshal(req.Descriptor)
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
		INSERT INTO device_pairing_requests (
			request_id, workspace_id, bootstrap_token_hash, bootstrap_code_hash, device_id, public_key, descriptor, status, expires_at, issued_token_hash, create_time, update_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			workspace_id = VALUES(workspace_id),
			bootstrap_token_hash = VALUES(bootstrap_token_hash),
			bootstrap_code_hash = VALUES(bootstrap_code_hash),
			device_id = VALUES(device_id),
			public_key = VALUES(public_key),
			descriptor = VALUES(descriptor),
			status = VALUES(status),
			expires_at = VALUES(expires_at),
			issued_token_hash = VALUES(issued_token_hash),
			update_time = VALUES(update_time)
	`, req.RequestID, req.WorkspaceID, req.BootstrapTokenHash, req.BootstrapCodeHash, req.DeviceID, req.PublicKey, string(descriptor), req.Status, req.ExpiresAt, req.IssuedTokenHash, req.CreateTime, req.UpdateTime)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO device_pairing_requests (
			request_id, workspace_id, bootstrap_token_hash, bootstrap_code_hash, device_id, public_key, descriptor, status, expires_at, issued_token_hash, create_time, update_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(request_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			bootstrap_token_hash = excluded.bootstrap_token_hash,
			bootstrap_code_hash = excluded.bootstrap_code_hash,
			device_id = excluded.device_id,
			public_key = excluded.public_key,
			descriptor = excluded.descriptor,
			status = excluded.status,
			expires_at = excluded.expires_at,
			issued_token_hash = excluded.issued_token_hash,
			update_time = excluded.update_time
	`, req.RequestID, req.WorkspaceID, req.BootstrapTokenHash, req.BootstrapCodeHash, req.DeviceID, req.PublicKey, string(descriptor), req.Status, req.ExpiresAt, req.IssuedTokenHash, req.CreateTime, req.UpdateTime)
	return err
}

func GetDevicePairingRequest(ctx context.Context, workspaceID, requestID string) (*DevicePairingRequest, error) {
	return scanPairing(DB.QueryRow(`
		SELECT request_id, workspace_id, bootstrap_token_hash, bootstrap_code_hash, device_id, public_key, descriptor, status, expires_at, issued_token_hash, create_time, update_time
		FROM device_pairing_requests WHERE workspace_id = ? AND request_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(requestID)))
}

func GetDevicePairingRequestByBootstrap(ctx context.Context, tokenHash string) (*DevicePairingRequest, error) {
	return scanPairing(DB.QueryRow(`
		SELECT request_id, workspace_id, bootstrap_token_hash, bootstrap_code_hash, device_id, public_key, descriptor, status, expires_at, issued_token_hash, create_time, update_time
		FROM device_pairing_requests WHERE bootstrap_token_hash = ?
	`, strings.TrimSpace(tokenHash)))
}

func ListDevicePairingRequests(ctx context.Context, workspaceID, status string) ([]DevicePairingRequest, error) {
	if DB == nil {
		return nil, nil
	}
	workspaceID = authz.NormalizeWorkspaceID(workspaceID)
	status = strings.TrimSpace(status)
	query := `
		SELECT request_id, workspace_id, bootstrap_token_hash, bootstrap_code_hash, device_id, public_key, descriptor, status, expires_at, issued_token_hash, create_time, update_time
		FROM device_pairing_requests WHERE workspace_id = ?`
	args := []interface{}{workspaceID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY create_time DESC"
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DevicePairingRequest, 0)
	for rows.Next() {
		item, err := scanPairingRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func UpdateDevicePairingStatus(ctx context.Context, workspaceID, requestID, status, issuedTokenHash string) error {
	if DB == nil {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE device_pairing_requests SET status = ?, issued_token_hash = ?, update_time = ?
		WHERE workspace_id = ? AND request_id = ?
	`, normalizePairingStatus(status), strings.TrimSpace(issuedTokenHash), time.Now().Unix(), authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(requestID))
	return err
}

func UpsertDevice(ctx context.Context, device DeviceRecord) error {
	if DB == nil {
		return nil
	}
	device.WorkspaceID = authz.NormalizeWorkspaceID(device.WorkspaceID)
	device.DeviceID = strings.TrimSpace(device.DeviceID)
	device.Status = normalizeDeviceStatus(device.Status)
	if device.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	now := time.Now().Unix()
	if device.CreateTime == 0 {
		device.CreateTime = now
	}
	device.UpdateTime = now
	if device.LastSeenAt == 0 {
		device.LastSeenAt = now
	}
	metadata, _ := json.Marshal(device.Metadata)
	if conf.BaseConfInfo.DBType == "mysql" {
		_, err := DB.Exec(`
			INSERT INTO devices (device_id, workspace_id, device_token_hash, public_key, name, platform, device_family, metadata, status, create_time, update_time, last_seen_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				workspace_id = VALUES(workspace_id),
				device_token_hash = VALUES(device_token_hash),
				public_key = VALUES(public_key),
				name = VALUES(name),
				platform = VALUES(platform),
				device_family = VALUES(device_family),
				metadata = VALUES(metadata),
				status = VALUES(status),
				update_time = VALUES(update_time),
				last_seen_at = VALUES(last_seen_at),
				revoked_at = VALUES(revoked_at)
		`, device.DeviceID, device.WorkspaceID, device.DeviceTokenHash, device.PublicKey, device.Name, device.Platform, device.DeviceFamily, string(metadata), device.Status, device.CreateTime, device.UpdateTime, device.LastSeenAt, device.RevokedAt)
		return err
	}
	_, err := DB.Exec(`
		INSERT INTO devices (device_id, workspace_id, device_token_hash, public_key, name, platform, device_family, metadata, status, create_time, update_time, last_seen_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			device_token_hash = excluded.device_token_hash,
			public_key = excluded.public_key,
			name = excluded.name,
			platform = excluded.platform,
			device_family = excluded.device_family,
			metadata = excluded.metadata,
			status = excluded.status,
			update_time = excluded.update_time,
			last_seen_at = excluded.last_seen_at,
			revoked_at = excluded.revoked_at
	`, device.DeviceID, device.WorkspaceID, device.DeviceTokenHash, device.PublicKey, device.Name, device.Platform, device.DeviceFamily, string(metadata), device.Status, device.CreateTime, device.UpdateTime, device.LastSeenAt, device.RevokedAt)
	return err
}

func GetDevice(ctx context.Context, workspaceID, deviceID string) (*DeviceRecord, error) {
	if DB == nil {
		return nil, nil
	}
	return scanDevice(DB.QueryRow(`
		SELECT device_id, workspace_id, device_token_hash, public_key, name, platform, device_family, metadata, status, create_time, update_time, last_seen_at, revoked_at
		FROM devices WHERE workspace_id = ? AND device_id = ?
	`, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(deviceID)))
}

func ListDevices(ctx context.Context, workspaceID string) ([]DeviceRecord, error) {
	if DB == nil {
		return nil, nil
	}
	rows, err := DB.Query(`
		SELECT device_id, workspace_id, device_token_hash, public_key, name, platform, device_family, metadata, status, create_time, update_time, last_seen_at, revoked_at
		FROM devices WHERE workspace_id = ? ORDER BY update_time DESC
	`, authz.NormalizeWorkspaceID(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DeviceRecord, 0)
	for rows.Next() {
		item, err := scanDeviceRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func RevokeDevice(ctx context.Context, workspaceID, deviceID string) error {
	if DB == nil {
		return nil
	}
	now := time.Now().Unix()
	_, err := DB.Exec(`
		UPDATE devices SET status = ?, revoked_at = ?, update_time = ? WHERE workspace_id = ? AND device_id = ?
	`, DeviceStatusRevoked, now, now, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(deviceID))
	return err
}

func MarkDeviceSeen(ctx context.Context, workspaceID, deviceID string) error {
	if DB == nil {
		return nil
	}
	now := time.Now().Unix()
	_, err := DB.Exec(`UPDATE devices SET last_seen_at = ?, update_time = ? WHERE workspace_id = ? AND device_id = ?`, now, now, authz.NormalizeWorkspaceID(workspaceID), strings.TrimSpace(deviceID))
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPairing(row rowScanner) (*DevicePairingRequest, error) {
	item, err := scanPairingRows(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func scanPairingRows(row rowScanner) (*DevicePairingRequest, error) {
	var (
		item          DevicePairingRequest
		descriptorRaw string
	)
	if err := row.Scan(&item.RequestID, &item.WorkspaceID, &item.BootstrapTokenHash, &item.BootstrapCodeHash, &item.DeviceID, &item.PublicKey, &descriptorRaw, &item.Status, &item.ExpiresAt, &item.IssuedTokenHash, &item.CreateTime, &item.UpdateTime); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(descriptorRaw), &item.Descriptor)
	return &item, nil
}

func scanDevice(row rowScanner) (*DeviceRecord, error) {
	item, err := scanDeviceRows(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func scanDeviceRows(row rowScanner) (*DeviceRecord, error) {
	var (
		item        DeviceRecord
		metadataRaw string
	)
	if err := row.Scan(&item.DeviceID, &item.WorkspaceID, &item.DeviceTokenHash, &item.PublicKey, &item.Name, &item.Platform, &item.DeviceFamily, &metadataRaw, &item.Status, &item.CreateTime, &item.UpdateTime, &item.LastSeenAt, &item.RevokedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(metadataRaw), &item.Metadata)
	return &item, nil
}

func normalizePairingStatus(status string) string {
	switch strings.TrimSpace(status) {
	case PairingStatusApproved, PairingStatusRejected, PairingStatusExpired:
		return strings.TrimSpace(status)
	default:
		return PairingStatusPending
	}
}

func normalizeDeviceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case DeviceStatusRevoked:
		return DeviceStatusRevoked
	default:
		return DeviceStatusActive
	}
}
