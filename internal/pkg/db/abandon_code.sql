-- 用于生成curd代码的示例表
-- a开头的表名是为了方便排序，abandon是排序最前的单词了
-- 两个单词组成表明，才能兼顾到下划线和驼峰的转换代码
-- TODO 应该要把各种类型的字段都写上，以便生成代码时能够覆盖到所有的类型
-- TODO 唯一索引的字段，可以生成出对应的查询方法，单键和复合键都可以

DROP TABLE IF EXISTS "abandon_code";
CREATE TABLE "abandon_code" (
  "idx1" serial NOT NULL,
  "col1" varchar(32)  NOT NULL,
  PRIMARY KEY ("idx1"),
  CONSTRAINT  "idx_col1" UNIQUE ("col1")
);

COMMENT ON TABLE "abandon_code" IS 'CURD生成模板';
COMMENT ON COLUMN "abandon_code"."idx1" IS 'The primary key of the table';

-- xxx_id_seq是pgsql自动为serial字段生成的序列名
ALTER SEQUENCE abandon_code_idx1_seq RESTART WITH 10;
