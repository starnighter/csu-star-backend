package repo

import (
	"csu-star-backend/internal/model"

	"gorm.io/gorm"
)

type DepartmentRepository interface {
	FindAllDepartments() ([]*model.Departments, error)
}

type departmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) FindAllDepartments() ([]*model.Departments, error) {
	var departments []*model.Departments
	result := r.db.Order("id ASC").Find(&departments)

	if result.Error != nil {
		return nil, result.Error
	}

	return departments, nil
}
