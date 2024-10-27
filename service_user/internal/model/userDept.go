package model

type UserDept struct {
	ID       uint64 `gorm:"column:id;type:int4;primary_key" json:"id"`
	DeptPath string `gorm:"column:dept_path;type:varchar(100);NOT NULL" json:"deptPath"`
	DeptName string `gorm:"column:dept_name;type:varchar(32);NOT NULL" json:"deptName"`
}

// TableName table name
func (m *UserDept) TableName() string {
	return "user_dept"
}
