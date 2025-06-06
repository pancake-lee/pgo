
DROP TABLE IF EXISTS "course_swap_request";
CREATE TABLE course_swap_request (
    "id" SERIAL NOT NULL,

    "src_teacher" varchar(100) NOT NULL,
    "src_date" time NOT NULL,
    "src_course_num" int NOT NULL,
    "src_course" varchar(100)  NOT NULL,
    "src_class" varchar(100)  NOT NULL,

    "dst_teacher" varchar(100) NOT NULL,
    "dst_date" time NOT NULL,
    "dst_course_num" int NOT NULL,
    "dst_course" varchar(100)  NOT NULL,
    "dst_class" varchar(100)  NOT NULL,

    "create_time" time DEFAULT CURRENT_TIMESTAMP,
    "status" int NOT NULL,
    PRIMARY KEY ("id")
);
ALTER SEQUENCE course_swap_request_id_seq RESTART WITH 10;
