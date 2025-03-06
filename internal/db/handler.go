package db

import (
	"github.com/hashicorp/go-metrics"
	"gorm.io/gorm"
	"time"
)

// Handler DBHandler wraps the gorm.DB instance.
type Handler struct {
	DB *gorm.DB
}

// NewHandler creates a new DBHandler and auto-migrates the Grade model.
func NewHandler(db *gorm.DB) (*Handler, error) {
	if err := db.AutoMigrate(&Grade{}); err != nil {
		return nil, err
	}
	return &Handler{DB: db}, nil
}

// GetGrades retrieves all Grade records.
func (h *Handler) GetGrades() ([]Grade, error) {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "get_grades"}}
	defer metrics.MeasureSinceWithLabels([]string{"get_grades_avg_time"}, time.Now(), tags)
	var grades []Grade
	err := h.DB.Find(&grades).Error
	return grades, err
}

// GetGradeByID retrieves a Grade record by its ID.
func (h *Handler) GetGradeByID(id uint) (*Grade, error) {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "get_grades_id"}}
	defer metrics.MeasureSinceWithLabels([]string{"get_grades_id_time"}, time.Now(), tags)
	var grade Grade
	err := h.DB.First(&grade, id).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

// CreateGrade creates a new Grade record.
func (h *Handler) CreateGrade(g *Grade) error {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "post_grades"}}
	defer metrics.MeasureSinceWithLabels([]string{"post_grades_time"}, time.Now(), tags)
	return h.DB.Create(g).Error
}

// UpdateGrade updates an existing Grade record.
func (h *Handler) UpdateGrade(g *Grade) error {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "put_grades_id"}}
	defer metrics.MeasureSinceWithLabels([]string{"put_grades_id_time"}, time.Now(), tags)
	return h.DB.Save(g).Error
}

// DeleteGrade deletes a Grade record.
func (h *Handler) DeleteGrade(g *Grade) error {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "delete_grades_id"}}
	defer metrics.MeasureSinceWithLabels([]string{"delete_grades_id_time"}, time.Now(), tags)
	return h.DB.Delete(g).Error
}

// GetAverageGrade computes the average of all grades.
func (h *Handler) GetAverageGrade() (float64, error) {
	tags := []metrics.Label{{Name: "layer", Value: "db"}, {Name: "operation", Value: "get_grades_avg"}}
	defer metrics.MeasureSinceWithLabels([]string{"get_grades_avg_time"}, time.Now(), tags)
	var avg float64
	if err := h.DB.Model(&Grade{}).Select("AVG(grade)").Scan(&avg).Error; err != nil {
		return 0, err
	}
	return avg, nil
}

func (h *Handler) Ping() error {
	dbase, err := h.DB.DB()
	if err != nil {
		return err
	}
	return dbase.Ping()
}
