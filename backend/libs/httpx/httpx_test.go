package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		payload    interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success with map",
			status:     http.StatusOK,
			payload:    map[string]string{"message": "hello"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"hello"}`,
		},
		{
			name:       "success with struct",
			status:     http.StatusCreated,
			payload:    struct{ ID int }{ID: 42},
			wantStatus: http.StatusCreated,
			wantBody:   `{"ID":42}`,
		},
		{
			name:       "empty payload",
			status:     http.StatusNoContent,
			payload:    nil,
			wantStatus: http.StatusNoContent,
			wantBody:   `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

			WriteJSON(w, tt.status, tt.payload, log)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteJSON() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("WriteJSON() Content-Type = %v, want application/json", ct)
			}

			got := bytes.TrimSpace(w.Body.Bytes())
			want := []byte(tt.wantBody)
			if !bytes.Equal(got, want) {
				t.Errorf("WriteJSON() body = %s, want %s", got, want)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		message string
		details map[string]interface{}
	}{
		{
			name:    "simple error",
			status:  http.StatusBadRequest,
			code:    "invalid_input",
			message: "Invalid input provided",
			details: nil,
		},
		{
			name:    "error with details",
			status:  http.StatusBadRequest,
			code:    "validation_failed",
			message: "Validation failed",
			details: map[string]interface{}{"field": "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

			WriteError(w, tt.status, tt.code, tt.message, tt.details, log)

			if w.Code != tt.status {
				t.Errorf("WriteError() status = %v, want %v", w.Code, tt.status)
			}

			var got ErrorPayload
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("WriteError() failed to decode response: %v", err)
			}

			if got.Error != tt.code {
				t.Errorf("WriteError() error = %v, want %v", got.Error, tt.code)
			}
			if got.Message != tt.message {
				t.Errorf("WriteError() message = %v, want %v", got.Message, tt.message)
			}
		})
	}
}

func TestWriteBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	WriteBadRequest(w, "Invalid request", nil, log)

	if w.Code != http.StatusBadRequest {
		t.Errorf("WriteBadRequest() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var got ErrorPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("WriteBadRequest() failed to decode response: %v", err)
	}

	if got.Error != "bad_request" {
		t.Errorf("WriteBadRequest() error = %v, want bad_request", got.Error)
	}
}

func TestWriteUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	WriteUnauthorized(w, "Authentication required", log)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("WriteUnauthorized() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}

	var got ErrorPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("WriteUnauthorized() failed to decode response: %v", err)
	}

	if got.Error != "unauthorized" {
		t.Errorf("WriteUnauthorized() error = %v, want unauthorized", got.Error)
	}
}

func TestWriteForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	WriteForbidden(w, "Access denied", log)

	if w.Code != http.StatusForbidden {
		t.Errorf("WriteForbidden() status = %v, want %v", w.Code, http.StatusForbidden)
	}

	var got ErrorPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("WriteForbidden() failed to decode response: %v", err)
	}

	if got.Error != "forbidden" {
		t.Errorf("WriteForbidden() error = %v, want forbidden", got.Error)
	}
}

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	WriteNotFound(w, "Resource not found", log)

	if w.Code != http.StatusNotFound {
		t.Errorf("WriteNotFound() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var got ErrorPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("WriteNotFound() failed to decode response: %v", err)
	}

	if got.Error != "not_found" {
		t.Errorf("WriteNotFound() error = %v, want not_found", got.Error)
	}
}

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	WriteInternalError(w, "Something went wrong", nil, log)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("WriteInternalError() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}

	var got ErrorPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("WriteInternalError() failed to decode response: %v", err)
	}

	if got.Error != "internal_error" {
		t.Errorf("WriteInternalError() error = %v, want internal_error", got.Error)
	}
}

func TestWriteNoContent(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("WriteNoContent() status = %v, want %v", w.Code, http.StatusNoContent)
	}

	if w.Body.Len() != 0 {
		t.Errorf("WriteNoContent() body length = %v, want 0", w.Body.Len())
	}
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "valid json",
			body:    `{"name":"test","value":42}`,
			target:  &map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "invalid json",
			body:    `{invalid}`,
			target:  &map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    "",
			target:  &map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *http.Request
			if tt.body != "" {
				r = httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			} else {
				r = httptest.NewRequest(http.MethodPost, "/test", nil)
			}

			err := DecodeJSON(r, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
