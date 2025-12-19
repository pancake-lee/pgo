DROP TABLE IF EXISTS course_swap_request;
CREATE TABLE course_swap_request (
    id INT NOT NULL AUTO_INCREMENT,

    src_teacher varchar(100) NOT NULL,
    src_date date NOT NULL,
    src_course_num int NOT NULL,
    src_course varchar(100)  NOT NULL,
    src_class varchar(100)  NOT NULL,

    dst_teacher varchar(100) NOT NULL,
    dst_date date NOT NULL,
    dst_course_num int NOT NULL,
    dst_course varchar(100)  NOT NULL,
    dst_class varchar(100)  NOT NULL,

    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    status int NOT NULL,
    PRIMARY KEY (id)
) AUTO_INCREMENT=10;
