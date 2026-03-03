package service

import (
	"testing"

	"ozon-manager/internal/model"
)

func TestShouldAgentAcquire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		mode               string
		hasOnlineExtension bool
		want               bool
	}{
		{
			name: "agent mode always uses agent",
			mode: model.ShopExecutionEngineAgent,
			want: true,
		},
		{
			name: "extension mode blocks agent",
			mode: model.ShopExecutionEngineExtension,
			want: false,
		},
		{
			name:               "auto mode prefers extension when online",
			mode:               model.ShopExecutionEngineAuto,
			hasOnlineExtension: true,
			want:               false,
		},
		{
			name:               "auto mode falls back to agent when extension offline",
			mode:               model.ShopExecutionEngineAuto,
			hasOnlineExtension: false,
			want:               true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldAgentAcquire(tc.mode, tc.hasOnlineExtension)
			if got != tc.want {
				t.Fatalf("shouldAgentAcquire(%q, %v) = %v, want %v", tc.mode, tc.hasOnlineExtension, got, tc.want)
			}
		})
	}
}

func TestShouldExtensionAcquire(t *testing.T) {
	t.Parallel()

	if !shouldExtensionAcquire(model.ShopExecutionEngineAuto) {
		t.Fatalf("auto mode should allow extension")
	}
	if !shouldExtensionAcquire(model.ShopExecutionEngineExtension) {
		t.Fatalf("extension mode should allow extension")
	}
	if shouldExtensionAcquire(model.ShopExecutionEngineAgent) {
		t.Fatalf("agent mode should block extension")
	}
}

func TestValidateJobAssignedAgent(t *testing.T) {
	t.Parallel()

	agentID := uint(9)
	wrongID := uint(10)

	tests := []struct {
		name    string
		job     *model.AutomationJob
		agentID uint
		wantErr bool
	}{
		{
			name:    "nil job",
			job:     nil,
			agentID: agentID,
			wantErr: true,
		},
		{
			name: "job without assigned agent",
			job: &model.AutomationJob{
				ID: 1,
			},
			agentID: agentID,
			wantErr: true,
		},
		{
			name: "job assigned to different agent",
			job: &model.AutomationJob{
				ID:              2,
				AssignedAgentID: &wrongID,
			},
			agentID: agentID,
			wantErr: true,
		},
		{
			name: "job assigned to current agent",
			job: &model.AutomationJob{
				ID:              3,
				AssignedAgentID: &agentID,
			},
			agentID: agentID,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateJobAssignedAgent(tc.job, tc.agentID)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
