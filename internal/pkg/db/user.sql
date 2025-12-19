DROP TABLE IF EXISTS user;
CREATE TABLE user (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,

  user_name varchar(32)  NOT NULL,
  password varchar(100)  NOT NULL,

  mtbl_record_id varchar(16)  NOT NULL,
  last_edit_from varchar(8)  NOT NULL,

  UNIQUE KEY idx_user_name (user_name)
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_job;
CREATE TABLE user_job (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,

  job_name varchar(32)  NOT NULL,
  UNIQUE KEY idx_job_name (job_name)
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_dept;
CREATE TABLE user_dept (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,

  dept_path varchar(100)  NOT NULL,
  dept_name varchar(32)  NOT NULL,
  UNIQUE KEY idx_dept_path (dept_path)
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_dept_assoc;
CREATE TABLE user_dept_assoc (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,

  user_id int NOT NULL,
  dept_id int NOT NULL,
  job_id int NOT NULL,
  UNIQUE KEY idx_user_dept (user_id, dept_id)
) AUTO_INCREMENT=10;

-- --------------------------------------------------
DROP TABLE IF EXISTS project;
CREATE TABLE project (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,
  
  proj_name varchar(100) NOT NULL,
  UNIQUE KEY idx_proj_name (proj_name)
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_project_assoc;
CREATE TABLE user_project_assoc (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  
  user_id int NOT NULL,
  proj_id int NOT NULL,
  UNIQUE KEY idx_user_proj (user_id, proj_id)
) AUTO_INCREMENT=10;

-- --------------------------------------------------
DROP TABLE IF EXISTS user_role;
CREATE TABLE user_role (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_user int NOT NULL,
  
  proj_id int NOT NULL,
  role_name varchar(100) NOT NULL,
  is_default int NOT NULL
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_role_assoc;
CREATE TABLE user_role_assoc (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  
  user_id int NOT NULL,
  role_id int NOT NULL,
  UNIQUE KEY idx_user_role (user_id, role_id)
) AUTO_INCREMENT=10;

DROP TABLE IF EXISTS user_role_permission_assoc;
CREATE TABLE user_role_permission_assoc (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  create_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  create_user int NOT NULL,
  
  role_id int NOT NULL,
  action varchar(32) NOT NULL,
  path_pattern varchar(767) NOT NULL -- 这是mysql建索引时的限制，pg应该不是767，但数据不超就不纠结
) AUTO_INCREMENT=10;
