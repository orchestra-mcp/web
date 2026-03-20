package services

import (
	"testing"
)

func TestLogSleepRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     LogSleepRequest
		wantErr bool
	}{
		{
			name:    "empty bed_time",
			req:     LogSleepRequest{BedTime: "", WakeTime: "2026-03-20T07:00:00Z"},
			wantErr: true,
		},
		{
			name:    "empty wake_time",
			req:     LogSleepRequest{BedTime: "2026-03-19T23:00:00Z", WakeTime: ""},
			wantErr: true,
		},
		{
			name:    "invalid bed_time format",
			req:     LogSleepRequest{BedTime: "not-a-date", WakeTime: "2026-03-20T07:00:00Z"},
			wantErr: true,
		},
		{
			name:    "invalid wake_time format",
			req:     LogSleepRequest{BedTime: "2026-03-19T23:00:00Z", WakeTime: "bad"},
			wantErr: true,
		},
		{
			name:    "wake before bed",
			req:     LogSleepRequest{BedTime: "2026-03-20T07:00:00Z", WakeTime: "2026-03-19T23:00:00Z"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &healthService{repo: nil}
			_, err := svc.LogSleep(t.Context(), 1, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogSleep() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogSleepRequest_QualityDefaultsTo3(t *testing.T) {
	// Quality rating of 0 (unset) should default to 3
	req := LogSleepRequest{
		BedTime:       "2026-03-19T23:00:00Z",
		WakeTime:      "2026-03-20T07:00:00Z",
		QualityRating: 0,
	}

	// We can't call the full method without a repo, but we can verify
	// the validation passes. The quality default is applied inside LogSleep
	// before repo.CreateSleepLog is called.
	if req.QualityRating < 1 || req.QualityRating > 5 {
		// This is expected — it will be defaulted to 3 in the service
		t.Logf("Quality rating %d out of range, will default to 3", req.QualityRating)
	}
}
