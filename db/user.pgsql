DROP TABLE IF EXISTS "user";
CREATE TABLE "user" (
  "id" serial NOT NULL,
  "user_name" varchar(32)  NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT  "idx_user_name" UNIQUE ("user_name")
);

COMMENT ON TABLE "user" IS '用户';
COMMENT ON COLUMN "user"."id" IS 'The primary key of the table';
COMMENT ON COLUMN "user"."user_name" IS 'The name of the user';

-- user_id_seq是pgsql自动为user.id字段生成的序列名
ALTER SEQUENCE user_id_seq RESTART WITH 10;