package utils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/LittleSongxx/TinyClaw/param"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	LogId   string      `json:"log_id"`
}

func Failure(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, message string, data interface{}) {
	_, logId := logger.GetBotNameAndLogId(ctx)
	start := logger.StartTimeFromContext(ctx)

	resp := Response{
		Code:    code,
		Message: message,
		Data:    data,
		LogId:   logId,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)

	strCode := strconv.Itoa(code)
	metrics.HTTPResponseCount.WithLabelValues(r.URL.Path, strCode).Inc()
	metrics.HTTPResponseDuration.WithLabelValues(r.URL.Path, strCode).Observe(time.Since(start).Seconds())
}

func Success(ctx context.Context, w http.ResponseWriter, r *http.Request, data interface{}) {
	_, logId := logger.GetBotNameAndLogId(ctx)
	start := logger.StartTimeFromContext(ctx)

	resp := Response{
		Code:    param.CodeSuccess,
		Message: param.MsgSuccess,
		Data:    data,
		LogId:   logId,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)

	strCode := strconv.Itoa(param.CodeSuccess)
	metrics.HTTPResponseCount.WithLabelValues(r.URL.Path, strCode).Inc()
	metrics.HTTPResponseDuration.WithLabelValues(r.URL.Path, strCode).Observe(time.Since(start).Seconds())
}

func HandleJsonBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if err = json.Unmarshal(body, v); err != nil {
		return err
	}

	return nil
}
