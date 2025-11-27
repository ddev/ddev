-- Test file with invalid SQL syntax to verify error detection
-- This should cause the import to fail and report an error

CREATE TABLE test_table (
  id INT PRIMARY KEY,
  name VARCHAR(50)
);

-- This line has invalid SQL syntax that will cause an error
THIS IS INVALID SQL SYNTAX AND SHOULD FAIL;

-- This line should never be executed
CREATE TABLE should_not_exist (
  id INT
);
