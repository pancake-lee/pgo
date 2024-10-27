package model

type UserJob struct {
	ID      uint64 `gorm:"column:id;type:int4;primary_key" json:"id"`
	JobName string `gorm:"column:job_name;type:varchar(32);NOT NULL" json:"jobName"`
}

// TableName table name
func (m *UserJob) TableName() string {
	return "user_job"
}
