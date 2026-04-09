package service

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
)

type DepartmentService struct {
	departmentRepo repo.DepartmentRepository
}

func NewDepartmentService(dr repo.DepartmentRepository) *DepartmentService {
	return &DepartmentService{dr}
}

func (s *DepartmentService) GetAllDepartments() ([]*model.Departments, error) {
	departments, err := s.departmentRepo.FindAllDepartments()
	if err != nil {
		logger.Log.Error(err.Error())
		return nil, &constant.QueryDepartmentsFailedErr
	}
	return departments, nil
}
