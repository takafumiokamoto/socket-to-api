-- Create a test table
CREATE TABLE employees (
    employee_id NUMBER PRIMARY KEY,
    first_name VARCHAR2(50),
    last_name VARCHAR2(50),
    email VARCHAR2(100),
    hire_date DATE,
    salary NUMBER(10,2)
);

-- Create a sequence for employee_id
CREATE SEQUENCE employee_seq START WITH 1 INCREMENT BY 1;

-- Insert some test data
INSERT INTO employees (employee_id, first_name, last_name, email, hire_date, salary)
VALUES (employee_seq.NEXTVAL, 'John', 'Doe', 'john.doe@example.com', SYSDATE, 50000);

INSERT INTO employees (employee_id, first_name, last_name, email, hire_date, salary)
VALUES (employee_seq.NEXTVAL, 'Jane', 'Smith', 'jane.smith@example.com', SYSDATE, 60000);

INSERT INTO employees (employee_id, first_name, last_name, email, hire_date, salary)
VALUES (employee_seq.NEXTVAL, 'Bob', 'Johnson', 'bob.johnson@example.com', SYSDATE, 55000);

COMMIT;
