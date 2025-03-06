package db

// Grade represents a student's grade record.
type Grade struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	StudentName string `json:"student_name"`
	Email       string `json:"email"`
	Class       string `json:"class"`
	Grade       int    `json:"grade"`
}
