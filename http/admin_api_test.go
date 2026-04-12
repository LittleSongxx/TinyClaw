package http

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

func TestDeviceApproveRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/admin/device/approve", strings.NewReader("{"))
	req = req.WithContext(logger.WithStartTime(req.Context(), time.Now()))
	rr := httptest.NewRecorder()

	DeviceApprove(rr, req)

	var resp utils.Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != param.CodeParamError {
		t.Fatalf("expected param error code %d, got %d", param.CodeParamError, resp.Code)
	}
}

func TestPluginsValidateRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/admin/plugins/validate", strings.NewReader("{"))
	req = req.WithContext(logger.WithStartTime(req.Context(), time.Now()))
	rr := httptest.NewRecorder()

	PluginsValidate(rr, req)

	var resp utils.Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != param.CodeParamError {
		t.Fatalf("expected param error code %d, got %d", param.CodeParamError, resp.Code)
	}
}
