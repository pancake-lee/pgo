DROP TABLE IF EXISTS "user";
CREATE TABLE "user" (
  "id" serial NOT NULL,
  "user_name" varchar(32)  NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT  "idx_user_name" UNIQUE ("user_name")
);

COMMENT ON TABLE "user" IS '用户';
COMMENT ON COLUMN "user"."id" IS 'The primary key of the table';
COMMENT ON COLUMN "user"."user_name" IS 'The name of
 the user';

-- user_id_seq是pgsql自动为user.id字段生成的序列名
ALTER SEQUENCE user_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_job";
CREATE TABLE "user_job" (
  "id" serial NOT NULL,
  "job_name" varchar(32)  NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT  "idx_job_name" UNIQUE ("job_name")
);
ALTER SEQUENCE user_job_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_dept";
CREATE TABLE "user_dept" (
  "id" serial NOT NULL,
  "dept_path" varchar(100)  NOT NULL,
  "dept_name" varchar(32)  NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT  "idx_dept_path" UNIQUE ("dept_path")
);
ALTER SEQUENCE user_dept_id_seq RESTART WITH 10;

DROP TABLE IF EXISTS "user_dept_assoc";
CREATE TABLE "user_dept_assoc" (
  "user_id" int NOT NULL,
  "dept_id" int NOT NULL,
  "job_id" int NOT NULL,
  PRIMARY KEY (user_id, dept_id)
);
