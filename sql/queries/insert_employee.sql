INSERT INTO employees (employee_id, first_name, last_name, email, hire_date, salary)
VALUES (employee_seq.NEXTVAL, :first_name, :last_name, :email, SYSDATE, :salary)