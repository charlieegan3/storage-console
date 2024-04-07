SET search_path TO curry_club;

ALTER TABLE sessions ADD COLUMN authenticated boolean NOT NULL DEFAULT false;
