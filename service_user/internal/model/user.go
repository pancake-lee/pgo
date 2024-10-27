package model

type User struct {
	ID       uint64 `gorm:"column:id;type:int4;primary_key" json:"id"`
	UserName string `gorm:"column:user_name;type:varchar(32);NOT NULL" json:"userName"` // The name of the user
}

// TableName table name
func (m *User) TableName() string {
	return "user"
}


