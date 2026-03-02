package service

import "ozon-manager/internal/model"

// === 代理方法（供 PromotionService 等外部调用） ===

func (s *AutomationService) CreateJobWithItems(job *model.AutomationJob, items []model.AutomationJobItem) error {
	return s.automationRepo.CreateJobWithItems(job, items)
}

func (s *AutomationService) CreateArtifact(jobID uint, artifactType string, payload interface{}) error {
	return s.automationRepo.CreateArtifact(jobID, artifactType, payload)
}

func (s *AutomationService) FindJobByIDAndShop(jobID, shopID uint) (*model.AutomationJob, error) {
	return s.automationRepo.FindJobByIDAndShop(jobID, shopID)
}
