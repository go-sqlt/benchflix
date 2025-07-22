CREATE TABLE IF NOT EXISTS movies (
    id INTEGER PRIMARY KEY
    , title TEXT NOT NULL
    , added_at DATE NOT NULL
    , rating NUMERIC NOT NULL
);

CREATE TABLE IF NOT EXISTS people (
    id SERIAL PRIMARY KEY
    , name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS movie_directors (
    movie_id INTEGER REFERENCES movies (id) ON DELETE CASCADE
    , person_id INTEGER REFERENCES people (id) ON DELETE CASCADE
    , PRIMARY KEY (movie_id, person_id)
);
