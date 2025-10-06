package log

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() (*zap.Logger, error) {

	config := zap.NewProductionEncoderConfig()

	config.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	writer := io.MultiWriter(os.Stdout)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)

	lgr := zap.New(core)

	return lgr, nil
}

type LoggingRoundTripper struct {
	Proxied http.RoundTripper
	Logger  *zap.Logger
}

func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	// Check request body
	var reqBody []byte
	var err error
	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			lrt.Logger.Error("Failed to read request body", zap.Error(err))
			return nil, err
		}
	}

	req.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	// Log the outgoing request
	lrt.Logger.Info("Started HTTP call",
		zap.String("Method", req.Method),
		zap.String("URL", req.URL.String()),
		zap.Any("Headers", req.Header),
		zap.ByteString("Request body", reqBody),
	)
	start := time.Now()

	// Perform the request
	resp, err := lrt.Proxied.RoundTrip(req)
	if err != nil {
		lrt.Logger.Error("Request failed", zap.Error(err))
		return nil, err
	}

	// Log the incoming response
	duration := time.Since(start)
	var respBody []byte
	if resp != nil && resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			lrt.Logger.Error("Failed to read response body", zap.Error(err))
			return nil, err
		}

		var respBodyMap interface{}
		err := json.Unmarshal(respBody, &respBodyMap)
		if err != nil {
			lrt.Logger.Error("Failed to unmarshal response body", zap.Error(err))
			return nil, err
		}
		lrt.Logger.Info("Incoming response",
			zap.String("Method", req.Method),
			zap.String("URL", req.URL.String()),
			zap.Int("Status", resp.StatusCode),
			zap.Duration("Duration", duration),
			zap.Any("Response body", respBodyMap),
		)
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	return resp, nil
}
