package repository

import (
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/okamoto/socket-to-api/internal/models"
)

//go:embed sql/queries/get_all_employees.sql
var getAllEmployeesQuery string

//go:embed sql/queries/get_employee_by_id.sql
var getEmployeeByIDQuery string

//go:embed sql/queries/get_employees_by_salary.sql
var getEmployeesBySalaryQuery string

//go:embed sql/queries/insert_employee.sql
var insertEmployeeQuery string

//go:embed sql/queries/update_employee_salary.sql
var updateEmployeeSalaryQuery string

//go:embed sql/queries/delete_employee.sql
var deleteEmployeeQuery string

//go:embed sql/queries/test_dual.sql
var testDualQuery string

type EmployeeRepository struct {
	db *sqlx.DB
}

func NewEmployeeRepository(db *sqlx.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

// GetAll retrieves all employees from the database
func (r *EmployeeRepository) GetAll() ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Select(&employees, getAllEmployeesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get all employees: %w", err)
	}
	return employees, nil
}

// GetByID retrieves an employee by ID
func (r *EmployeeRepository) GetByID(id int) (*models.Employee, error) {
	var employee models.Employee
	err := r.db.Get(&employee, getEmployeeByIDQuery, sql.Named("id", id))
	if err != nil {
		return nil, fmt.Errorf("failed to get employee by id: %w", err)
	}
	return &employee, nil
}

// GetBySalary retrieves employees with salary >= minSalary
func (r *EmployeeRepository) GetBySalary(minSalary float64) ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Select(&employees, getEmployeesBySalaryQuery, sql.Named("min_salary", minSalary))
	if err != nil {
		return nil, fmt.Errorf("failed to get employees by salary: %w", err)
	}
	return employees, nil
}

// Create inserts a new employee
func (r *EmployeeRepository) Create(firstName, lastName, email string, salary float64) error {
	_, err := r.db.Exec(insertEmployeeQuery,
		sql.Named("first_name", firstName),
		sql.Named("last_name", lastName),
		sql.Named("email", email),
		sql.Named("salary", salary),
	)
	if err != nil {
		return fmt.Errorf("failed to insert employee: %w", err)
	}
	return nil
}

// UpdateSalary updates an employee's salary
func (r *EmployeeRepository) UpdateSalary(id int, newSalary float64) error {
	result, err := r.db.Exec(updateEmployeeSalaryQuery,
		sql.Named("id", id),
		sql.Named("salary", newSalary),
	)
	if err != nil {
		return fmt.Errorf("failed to update employee salary: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("employee with id %d not found", id)
	}

	return nil
}

// Delete removes an employee by ID
func (r *EmployeeRepository) Delete(id int) error {
	result, err := r.db.Exec(deleteEmployeeQuery, sql.Named("id", id))
	if err != nil {
		return fmt.Errorf("failed to delete employee: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("employee with id %d not found", id)
	}

	return nil
}

// TestDual tests Oracle DUAL table and functions
func (r *EmployeeRepository) TestDual() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	rows, err := r.db.Query(testDualQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query DUAL: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var currentDate string
		var currentUser string
		var message string

		err = rows.Scan(&currentDate, &currentUser, &message)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DUAL result: %w", err)
		}

		result["current_date"] = currentDate
		result["current_user"] = currentUser
		result["message"] = message
	}

	return result, nil
}
