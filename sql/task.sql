DROP TABLE IF EXISTS task;
CREATE TABLE task (
    id INT NOT NULL AUTO_INCREMENT,
    parent_id int NOT NULL, 
    prev_id int NOT NULL, 

    task varchar(100) NOT NULL,
    status int NOT NULL, 
    estimate int NOT NULL,
    start datetime DEFAULT CURRENT_TIMESTAMP,
    end datetime DEFAULT CURRENT_TIMESTAMP,

    `desc` varchar(5000) NOT NULL,
    metadata varchar(5000) NOT NULL, -- {k:v}

    create_time datetime DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) AUTO_INCREMENT=10;

