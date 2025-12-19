DROP TABLE IF EXISTS task;
CREATE TABLE task (
    id INT NOT NULL AUTO_INCREMENT,
    parent_id int NOT NULL, 
    prev_id int NOT NULL, 

    task varchar(100) NOT NULL,
    status int NOT NULL, 
    estimate int NOT NULL,
    start timestamp DEFAULT CURRENT_TIMESTAMP,
    end timestamp DEFAULT CURRENT_TIMESTAMP,

    `desc` varchar(5000) NOT NULL,
    metadata varchar(5000) NOT NULL, -- {k:v}

    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) AUTO_INCREMENT=10;

insert into task (parent_id,prev_id,task,status,estimate,`desc`,metadata)values(0,0,'test',0,0,'' ,'');
