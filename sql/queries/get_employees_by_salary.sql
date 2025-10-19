SELECT employee_id, first_name, last_name, email, hire_date, salary
FROM employees
WHERE salary >= :min_salary
ORDER BY salary DESC