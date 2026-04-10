package gateway

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/authz"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/google/uuid"
)

type DeviceBootstrap struct {
	RequestID   string `json:"request_id"`
	WorkspaceID string `json:"workspace_id"`
	Code        string `json:"code"`
	ExpiresAt   int64  `json:"expires_at"`
}

type PairingSubmitRequest struct {
	BootstrapCode string               `json:"bootstrap_code"`
	DeviceID      string               `json:"device_id"`
	PublicKey     string               `json:"public_key"`
	Descriptor    *node.NodeDescriptor `json:"descriptor,omitempty"`
}

type DeviceApprovalResponse struct {
	DeviceID    string `json:"device_id"`
	WorkspaceID string `json:"workspace_id"`
	DeviceToken string `json:"device_token,omitempty"`
}

func (s *Service) CreateDeviceBootstrap(ctx context.Context) (*DeviceBootstrap, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !principal.CanManageWorkspace() {
		return nil, authz.ErrForbidden
	}
	now := time.Now()
	code, err := randomCode(20)
	if err != nil {
		return nil, err
	}
	requestID := uuid.NewString()
	record := db.DevicePairingRequest{
		RequestID:          requestID,
		WorkspaceID:        principal.WorkspaceID,
		BootstrapTokenHash: db.HashSecret(code),
		BootstrapCodeHash:  db.HashSecret(code),
		Status:             db.PairingStatusPending,
		ExpiresAt:          now.Add(10 * time.Minute).Unix(),
		CreateTime:         now.Unix(),
		UpdateTime:         now.Unix(),
	}
	if err := db.InsertDevicePairingRequest(ctx, record); err != nil {
		return nil, err
	}
	_ = db.InsertAuditEvent(ctx, db.AuditEvent{
		WorkspaceID:  principal.WorkspaceID,
		ActorID:      principal.ActorID,
		Action:       "devices.bootstrap",
		ResourceType: "device_pairing_request",
		ResourceID:   requestID,
		Success:      true,
	})
	return &DeviceBootstrap{
		RequestID:   requestID,
		WorkspaceID: principal.WorkspaceID,
		Code:        code,
		ExpiresAt:   record.ExpiresAt,
	}, nil
}

func (s *Service) SubmitDevicePairing(ctx context.Context, req PairingSubmitRequest) (*db.DevicePairingRequest, error) {
	code := strings.TrimSpace(req.BootstrapCode)
	if code == "" {
		return nil, errors.New("bootstrap_code is required")
	}
	record, err := db.GetDevicePairingRequestByBootstrap(ctx, db.HashSecret(code))
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("bootstrap code not found")
	}
	if record.Status != db.PairingStatusPending {
		return nil, errors.New("pairing request is not pending")
	}
	if record.ExpiresAt <= time.Now().Unix() {
		_ = db.UpdateDevicePairingStatus(ctx, record.WorkspaceID, record.RequestID, db.PairingStatusExpired, "")
		return nil, errors.New("bootstrap code expired")
	}
	if req.Descriptor == nil {
		req.Descriptor = &node.NodeDescriptor{}
	}
	if req.DeviceID == "" {
		req.DeviceID = firstNonEmpty(req.Descriptor.DeviceID, req.Descriptor.ID)
	}
	if req.DeviceID == "" {
		req.DeviceID = uuid.NewString()
	}
	if _, err := decodePublicKey(req.PublicKey); err != nil {
		return nil, fmt.Errorf("invalid public_key: %w", err)
	}
	req.Descriptor.WorkspaceID = record.WorkspaceID
	req.Descriptor.DeviceID = req.DeviceID
	if req.Descriptor.ID == "" {
		req.Descriptor.ID = req.DeviceID
	}
	record.DeviceID = req.DeviceID
	record.PublicKey = strings.TrimSpace(req.PublicKey)
	record.Descriptor = descriptorMap(req.Descriptor)
	if err := db.InsertDevicePairingRequest(ctx, *record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) ListPendingDevices(ctx context.Context) ([]db.DevicePairingRequest, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !principal.CanManageWorkspace() {
		return nil, authz.ErrForbidden
	}
	return db.ListDevicePairingRequests(ctx, principal.WorkspaceID, db.PairingStatusPending)
}

func (s *Service) ApproveDevice(ctx context.Context, requestID string) (*DeviceApprovalResponse, error) {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !principal.CanManageWorkspace() {
		return nil, authz.ErrForbidden
	}
	record, err := db.GetDevicePairingRequest(ctx, principal.WorkspaceID, requestID)
	if err != nil {
		return nil, err
	}
	if !sameWorkspace(principal.WorkspaceID, record.WorkspaceID) {
		return nil, authz.ErrForbidden
	}
	if record.Status != db.PairingStatusPending {
		return nil, errors.New("pairing request is not pending")
	}
	if record.ExpiresAt <= time.Now().Unix() {
		_ = db.UpdateDevicePairingStatus(ctx, record.WorkspaceID, record.RequestID, db.PairingStatusExpired, "")
		return nil, errors.New("pairing request expired")
	}
	deviceToken, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	desc := descriptorFromPairing(record)
	if desc.ID == "" {
		desc.ID = record.DeviceID
	}
	if desc.DeviceID == "" {
		desc.DeviceID = record.DeviceID
	}
	if desc.WorkspaceID == "" {
		desc.WorkspaceID = record.WorkspaceID
	}
	metadata := desc.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	device := db.DeviceRecord{
		DeviceID:        record.DeviceID,
		WorkspaceID:     record.WorkspaceID,
		DeviceTokenHash: db.HashSecret(deviceToken),
		PublicKey:       record.PublicKey,
		Name:            firstNonEmpty(desc.Name, desc.ID),
		Platform:        desc.Platform,
		DeviceFamily:    firstNonEmpty(metadata["kind"], desc.Platform),
		Metadata:        metadata,
		Status:          db.DeviceStatusActive,
	}
	if err := db.UpsertDevice(ctx, device); err != nil {
		return nil, err
	}
	if err := db.UpdateDevicePairingStatus(ctx, record.WorkspaceID, record.RequestID, db.PairingStatusApproved, db.HashSecret(deviceToken)); err != nil {
		return nil, err
	}
	_ = db.InsertAuditEvent(ctx, db.AuditEvent{
		WorkspaceID:  principal.WorkspaceID,
		ActorID:      principal.ActorID,
		Action:       "devices.approve",
		ResourceType: "device",
		ResourceID:   record.DeviceID,
		Success:      true,
	})
	return &DeviceApprovalResponse{DeviceID: record.DeviceID, WorkspaceID: record.WorkspaceID, DeviceToken: deviceToken}, nil
}

func (s *Service) RejectDevice(ctx context.Context, requestID, reason string) error {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	if !principal.CanManageWorkspace() {
		return authz.ErrForbidden
	}
	record, err := db.GetDevicePairingRequest(ctx, principal.WorkspaceID, requestID)
	if err != nil {
		return err
	}
	if !sameWorkspace(principal.WorkspaceID, record.WorkspaceID) {
		return authz.ErrForbidden
	}
	if err := db.UpdateDevicePairingStatus(ctx, principal.WorkspaceID, requestID, db.PairingStatusRejected, ""); err != nil {
		return err
	}
	_ = db.InsertAuditEvent(ctx, db.AuditEvent{
		WorkspaceID:  principal.WorkspaceID,
		ActorID:      principal.ActorID,
		Action:       "devices.reject",
		ResourceType: "device_pairing_request",
		ResourceID:   requestID,
		Success:      true,
		Detail:       reason,
	})
	return nil
}

func (s *Service) RevokeDevice(ctx context.Context, deviceID string) error {
	principal, err := authz.RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	if !principal.CanManageWorkspace() {
		return authz.ErrForbidden
	}
	if err := db.RevokeDevice(ctx, principal.WorkspaceID, deviceID); err != nil {
		return err
	}
	_ = db.InsertAuditEvent(ctx, db.AuditEvent{
		WorkspaceID:  principal.WorkspaceID,
		ActorID:      principal.ActorID,
		Action:       "devices.revoke",
		ResourceType: "device",
		ResourceID:   deviceID,
		Success:      true,
	})
	return nil
}

func (s *Service) VerifyDeviceConnect(ctx context.Context, connect ConnectFrame) (*db.DeviceRecord, error) {
	auth := connect.Auth
	if auth.DeviceID == "" {
		auth.DeviceID = connect.DeviceID()
	}
	token := firstNonEmpty(auth.Token, connect.Token)
	if auth.DeviceID == "" || token == "" {
		return nil, errors.New("device_id and device token are required")
	}
	device, err := db.GetDevice(ctx, authz.NormalizeWorkspaceID(connect.WorkspaceID), auth.DeviceID)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, errors.New("device not found")
	}
	if device.Status != db.DeviceStatusActive {
		return nil, errors.New("device is not active")
	}
	if db.HashSecret(token) != device.DeviceTokenHash {
		return nil, errors.New("invalid device token")
	}
	if connect.Node != nil {
		if connect.Node.Platform != "" && device.Platform != "" && connect.Node.Platform != device.Platform {
			return nil, errors.New("device platform metadata mismatch")
		}
		if connect.Node.WorkspaceID != "" && !sameWorkspace(connect.Node.WorkspaceID, device.WorkspaceID) {
			return nil, errors.New("device workspace mismatch")
		}
	}
	if auth.Nonce != "" || auth.Signature != "" {
		publicKey, err := decodePublicKey(device.PublicKey)
		if err != nil {
			return nil, err
		}
		signature, err := base64.RawStdEncoding.DecodeString(auth.Signature)
		if err != nil {
			signature, err = base64.StdEncoding.DecodeString(auth.Signature)
		}
		if err != nil {
			return nil, errors.New("invalid device signature")
		}
		message := []byte(device.WorkspaceID + ":" + device.DeviceID + ":" + auth.Nonce)
		if !ed25519.Verify(publicKey, message, signature) {
			return nil, errors.New("invalid device signature")
		}
	}
	if err := db.MarkDeviceSeen(ctx, device.WorkspaceID, device.DeviceID); err != nil {
		return nil, err
	}
	return device, nil
}

func (f ConnectFrame) DeviceID() string {
	if f.Device != nil && f.Device.ID != "" {
		return f.Device.ID
	}
	if f.Node != nil {
		return firstNonEmpty(f.Node.DeviceID, f.Node.ID)
	}
	return ""
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("public_key is required")
	}
	body, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		body, err = base64.StdEncoding.DecodeString(value)
	}
	if err != nil {
		body, err = hex.DecodeString(value)
	}
	if err != nil {
		return nil, err
	}
	if len(body) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(body))
	}
	return ed25519.PublicKey(body), nil
}

func randomCode(bytesLen int) (string, error) {
	token, err := randomToken(bytesLen)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimRight(token, "=")), nil
}

func randomToken(bytesLen int) (string, error) {
	if bytesLen <= 0 {
		bytesLen = 32
	}
	body := make([]byte, bytesLen)
	if _, err := rand.Read(body); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func descriptorMap(desc *node.NodeDescriptor) map[string]interface{} {
	if desc == nil {
		return map[string]interface{}{}
	}
	body, err := json.Marshal(desc)
	if err != nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func descriptorFromPairing(record *db.DevicePairingRequest) node.NodeDescriptor {
	var desc node.NodeDescriptor
	if record == nil || len(record.Descriptor) == 0 {
		return desc
	}
	body, _ := json.Marshal(record.Descriptor)
	_ = json.Unmarshal(body, &desc)
	return desc
}

func sameWorkspace(left, right string) bool {
	return authz.NormalizeWorkspaceID(left) == authz.NormalizeWorkspaceID(right)
}
