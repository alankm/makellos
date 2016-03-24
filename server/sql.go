package server

const (
	tblFiles = `CREATE TABLE IF NOT EXISTS files(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type VARCHAR(16) NOT NULL,
		name VARCHAR(255),
		path VARCHAR(4096),
		own VARCHAR(32) NOT NULL,
		grp VARCHAR(32) NOT NULL,
		mod SMALLINT NOT NULL,
		UNIQUE (name,path)
		)`

	tblJournal = `CREATE TABLE IF NOT EXISTS journal(
		id INTEGER PRIMARY KEY,
		time UNSIGNED BIG INT NOT NULL,
		sev TINYINT NOT NULL,
		msg VARCHAR(255),
		code VARCHAR(64) NOT NULL,
		FOREIGN KEY(id) REFERENCES files(id) ON DELETE CASCADE
		)`

	tblJData = `CREATE TABLE IF NOT EXISTS jdata(
		id INTEGER PRIMARY KEY,
		key VARCHAR(128) NOT NULL,
		val VARCHAR(128) NOT NULL,
		FOREIGN KEY(id) REFERENCES journal(id) ON DELETE CASCADE
		)`

	tblImages = `CREATE TABLE IF NOT EXISTS images(
		id INTEGER PRIMARY KEY,
		time UNSIGNED BIG INT,
		auth VARCHAR(128),
		desc TEXT,
		FOREIGN KEY(id) REFERENCES files(id) ON DELETE CASCADE
		)`
)