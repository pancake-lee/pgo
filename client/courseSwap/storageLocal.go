package courseSwap

import (
	"context"
	"time"

	"github.com/pancake-lee/pgo/client/swagger"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type LocalRepo struct {
	db *gorm.DB
}

type CourseSwapRequestModel struct {
	ID           int32 `gorm:"primaryKey;autoIncrement"`
	SrcTeacher   string
	SrcDate      string
	SrcCourseNum int32
	SrcCourse    string
	SrcClass     string
	DstTeacher   string
	DstDate      string
	DstCourseNum int32
	DstCourse    string
	DstClass     string
	CreateTime   string
	Status       int32
}

func (CourseSwapRequestModel) TableName() string {
	return "course_swap_request"
}

func NewLocalRepo(dbPath string) *LocalRepo {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	db.AutoMigrate(&CourseSwapRequestModel{})
	return &LocalRepo{db: db}
}

func (r *LocalRepo) GetCourseSwapRequestList(ctx context.Context) ([]swagger.ApiCourseSwapRequestInfo, error) {
	var models []CourseSwapRequestModel
	result := r.db.Find(&models)
	if result.Error != nil {
		plogger.Debug("Local GetList failed: ", result.Error)
		return nil, result.Error
	}

	var infos []swagger.ApiCourseSwapRequestInfo
	for _, m := range models {
		infos = append(infos, swagger.ApiCourseSwapRequestInfo{
			ID:           m.ID,
			SrcTeacher:   m.SrcTeacher,
			SrcDate:      m.SrcDate,
			SrcCourseNum: m.SrcCourseNum,
			SrcCourse:    m.SrcCourse,
			SrcClass:     m.SrcClass,
			DstTeacher:   m.DstTeacher,
			DstDate:      m.DstDate,
			DstCourseNum: m.DstCourseNum,
			DstCourse:    m.DstCourse,
			DstClass:     m.DstClass,
			CreateTime:   m.CreateTime,
			Status:       m.Status,
		})
	}
	return infos, nil
}

func (r *LocalRepo) AddCourseSwapRequest(ctx context.Context, req *swagger.ApiCourseSwapRequestInfo) error {
	model := CourseSwapRequestModel{
		SrcTeacher:   req.SrcTeacher,
		SrcDate:      req.SrcDate,
		SrcCourseNum: req.SrcCourseNum,
		SrcCourse:    req.SrcCourse,
		SrcClass:     req.SrcClass,
		DstTeacher:   req.DstTeacher,
		DstDate:      req.DstDate,
		DstCourseNum: req.DstCourseNum,
		DstCourse:    req.DstCourse,
		DstClass:     req.DstClass,
		CreateTime:   time.Now().Format("2006-01-02 15:04:05"), // Assuming format
		Status:       req.Status,
	}
	if req.CreateTime != "" {
		model.CreateTime = req.CreateTime
	}

	result := r.db.Create(&model)
	if result.Error != nil {
		plogger.Debug("Local Add failed: ", result.Error)
		return result.Error
	}
	return nil
}
