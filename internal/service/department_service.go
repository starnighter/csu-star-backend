package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
)

type DepartmentService interface {
	GetAllDepartments() ([]*model.Departments, error)
}

type departmentService struct {
	DepartmentRepo repo.DepartmentRepository
}

func NewDepartmentService(dr repo.DepartmentRepository) DepartmentService {
	return &departmentService{dr}
}

func (s *departmentService) GetAllDepartments() ([]*model.Departments, error) {
	departments, err := s.DepartmentRepo.FindAllDepartments()
	if err != nil {
		return nil, err
	}
	return departments, nil
}
