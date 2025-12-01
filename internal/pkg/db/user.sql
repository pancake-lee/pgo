DROP TABLE IF EXISTS "user";
CREATE TABLE "user" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,

  "user_name" varchar(32)  NOT NULL,
  "password" varchar(100)  NOT NULL,

  CONSTRAINT  "idx_user_name" UNIQUE ("user_name")
);
ALTER SEQUENCE user_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_job";
CREATE TABLE "user_job" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,

  "job_name" varchar(32)  NOT NULL,
  CONSTRAINT  "idx_job_name" UNIQUE ("job_name")
);
ALTER SEQUENCE user_job_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_dept";
CREATE TABLE "user_dept" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,

  "dept_path" varchar(100)  NOT NULL,
  "dept_name" varchar(32)  NOT NULL,
  CONSTRAINT  "idx_dept_path" UNIQUE ("dept_path")
);
ALTER SEQUENCE user_dept_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_dept_assoc";
CREATE TABLE "user_dept_assoc" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,

  "user_id" int NOT NULL,
  "dept_id" int NOT NULL,
  "job_id" int NOT NULL,
  CONSTRAINT  "idx_user_dept" UNIQUE ("user_id", "dept_id")
);
ALTER SEQUENCE user_dept_assoc_id_seq RESTART WITH 10;

-- --------------------------------------------------
DROP TABLE IF EXISTS "project";
CREATE TABLE "project" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,
  
  "proj_name" varchar(100) NOT NULL,
  CONSTRAINT "idx_proj_name" UNIQUE ("proj_name")
);
ALTER SEQUENCE project_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_project_assoc";
CREATE TABLE "user_project_assoc" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  
  "user_id" int NOT NULL,
  "proj_id" int NOT NULL,
  CONSTRAINT "idx_user_proj" UNIQUE ("user_id", "proj_id")
);
ALTER SEQUENCE user_project_assoc_id_seq RESTART WITH 10;

-- --------------------------------------------------
DROP TABLE IF EXISTS "user_role";
CREATE TABLE "user_role" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  "update_time" timestamp NOT NULL DEFAULT NOW(),
  "update_user" int NOT NULL,
  
  "proj_id" int NOT NULL,
  "role_name" varchar(100) NOT NULL,
  "is_default" int NOT NULL
);
ALTER SEQUENCE user_role_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_role_assoc";
CREATE TABLE "user_role_assoc" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  
  "user_id" int NOT NULL,
  "role_id" int NOT NULL,
  CONSTRAINT "idx_user_role" UNIQUE ("user_id", "role_id")
);
ALTER SEQUENCE user_role_assoc_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_role_permission_assoc";
CREATE TABLE "user_role_permission_assoc" (
  "id" serial PRIMARY KEY,
  "create_time" timestamp NOT NULL DEFAULT NOW(),
  "create_user" int NOT NULL,
  
  "role_id" int NOT NULL,
  "action" varchar(32) NOT NULL,
  "path_pattern" varchar(767) NOT NULL -- 这是mysql建索引时的限制，pg应该不是767，但数据不超就不纠结
);
ALTER SEQUENCE user_role_permission_assoc_id_seq RESTART WITH 10;