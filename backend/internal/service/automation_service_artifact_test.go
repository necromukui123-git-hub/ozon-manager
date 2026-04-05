package service

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

func openAutomationArtifactTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Shop{},
		&model.AutomationJob{},
		&model.AutomationJobItem{},
		&model.AutomationArtifact{},
	); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func createAutomationArtifactTestJob(t *testing.T, db *gorm.DB) uint {
	t.Helper()

	user := &model.User{
		Username:     "artifact-owner-" + strings.ReplaceAll(t.Name(), "/", "-"),
		PasswordHash: "hash",
		DisplayName:  "owner",
		Role:         model.RoleShopAdmin,
		Status:       "active",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	shop := &model.Shop{
		Name:                "artifact-shop",
		ClientID:            "artifact-client-" + strings.ReplaceAll(t.Name(), "/", "-"),
		ApiKey:              "artifact-key",
		IsActive:            true,
		ExecutionEngineMode: model.ShopExecutionEngineAuto,
		OwnerID:             user.ID,
	}
	if err := db.Create(shop).Error; err != nil {
		t.Fatalf("db.Create(shop): %v", err)
	}

	job := &model.AutomationJob{
		ShopID:     shop.ID,
		CreatedBy:  user.ID,
		JobType:    model.AutomationJobTypeSyncActionCandidates,
		Status:     model.AutomationJobStatusSuccess,
		RateLimit:  1,
		TotalItems: 1,
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("db.Create(job): %v", err)
	}

	item := &model.AutomationJobItem{
		JobID:             job.ID,
		SourceSKU:         "__sync_action_candidates__:1",
		TargetPrice:       0.01,
		OverallStatus:     model.AutomationStepStatusSuccess,
		StepExitStatus:    model.AutomationStepStatusSuccess,
		StepRepriceStatus: model.AutomationStepStatusSuccess,
		StepReaddStatus:   model.AutomationStepStatusSuccess,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("db.Create(item): %v", err)
	}

	return job.ID
}

func TestGetLatestArtifactWaitsForRecentlyCommittedSnapshot(t *testing.T) {
	t.Parallel()

	db := openAutomationArtifactTestDB(t)
	repo := repository.NewAutomationRepository(db)
	service := NewAutomationService(repo, nil, nil)
	jobID := createAutomationArtifactTestJob(t, db)

	go func() {
		time.Sleep(80 * time.Millisecond)
		_ = repo.CreateArtifact(jobID, "action_candidates_snapshot", map[string]interface{}{
			"items": []map[string]interface{}{
				{"source_sku": "SKU-1"},
			},
		})
	}()

	artifact, err := service.GetLatestArtifact(jobID, "action_candidates_snapshot")
	if err != nil {
		t.Fatalf("GetLatestArtifact() error = %v", err)
	}
	if artifact == nil {
		t.Fatal("GetLatestArtifact() returned nil artifact")
	}
	if artifact.ArtifactType != "action_candidates_snapshot" {
		t.Fatalf("artifact_type = %q, want %q", artifact.ArtifactType, "action_candidates_snapshot")
	}
}
