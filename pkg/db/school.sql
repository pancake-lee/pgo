
DROP TABLE IF EXISTS "course_swap_request";
CREATE TABLE course_swap_request (
    "id" SERIAL NOT NULL,

    "src_teacher" varchar(100) NOT NULL, -- 需要换课的老师
    "src_date" time NOT NULL, -- 需要换课的日期 
    "src_course_num" int NOT NULL, -- 需要换课的序号（第几节）
    "src_course" varchar(100)  NOT NULL, -- 需要换课的课名
    "src_class" varchar(100)  NOT NULL, -- 需要换课的班级

    "dst_teacher" varchar(100) NOT NULL, -- 被换课的老师
    "dst_date" time NOT NULL, -- 被换课的日期
    "dst_course_num" int NOT NULL, -- 被换课的序号（第几节）
    "dst_course" varchar(100)  NOT NULL, -- 被换课的课名
    "dst_class" varchar(100)  NOT NULL, -- 被换课的班级

    "create_time" time DEFAULT CURRENT_TIMESTAMP,
    "status" int NOT NULL,
    PRIMARY KEY ("id")
);
