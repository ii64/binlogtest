DROP DATABASE IF EXISTS testing_db;
CREATE DATABASE testing_db;

USE testing_db;

CREATE TABLE IF NOT EXISTS ent_assets (
    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(24)
);

INSERT IGNORE ent_assets (`name`) VALUES
    ('PLC1'), 
    ('PLC2'),
    ('PLC3')
;