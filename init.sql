CREATE TABLE items (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    topPriority BOOLEAN,
    completed BOOLEAN
);

CREATE TABLE users (
    id INT PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE logs (
    id INT PRIMARY KEY,
    severity TEXT,
    content TEXT
);