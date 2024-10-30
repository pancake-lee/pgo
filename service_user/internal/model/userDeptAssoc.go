package model

type UserDeptAssoc struct {
	UserID int `gorm:"column:user_id;type:int4;NOT NULL" json:"userID"`
	DeptID int `gorm:"column:dept_id;type:int4;NOT NULL" json:"deptID"`
	JobID  int `gorm:"column:job_id;type:int4;NOT NULL" json:"jobID"`
}

// TableName table name
func (m *UserDeptAssoc) TableName() string {
	return "user_dept_assoc"
}


