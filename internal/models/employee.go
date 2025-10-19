package models

import (
	"time"
)

type Employee struct {
	EmployeeID int       `db:"EMPLOYEE_ID" json:"employee_id"`
	FirstName  string    `db:"FIRST_NAME" json:"first_name"`
	LastName   string    `db:"LAST_NAME" json:"last_name"`
	Email      string    `db:"EMAIL" json:"email"`
	HireDate   time.Time `db:"HIRE_DATE" json:"hire_date"`
	Salary     float64   `db:"SALARY" json:"salary"`
}
