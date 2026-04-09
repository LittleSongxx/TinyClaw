package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/logger"
	botUtils "github.com/LittleSongxx/TinyClaw/utils"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	sqlite3TableSQLs = map[string]string{
		"users": `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id varchar(100) NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			token INTEGER NOT NULL DEFAULT '0',
			avail_token INTEGER NOT NULL DEFAULT 0,
			create_time INTEGER NOT NULL DEFAULT '0',
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			llm_config TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_users_user_id ON users(user_id);
	`,
		"records": `
		CREATE TABLE IF NOT EXISTS records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id varchar(100) NOT NULL DEFAULT '0',
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			content TEXT NOT NULL,
			create_time INTEGER NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			is_deleted INTEGER NOT NULL DEFAULT '0',
			token INTEGER NOT NULL DEFAULT 0,
			mode VARCHAR(100) NOT NULL DEFAULT '',
			record_type INTEGER NOT NULL DEFAULT 0, -- SQLite中用INTEGER代替tinyint
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_records_user_id ON records(user_id);
		CREATE INDEX IF NOT EXISTS idx_records_create_time ON records(create_time);
	`,
		"knowledge_files": `
		CREATE TABLE IF NOT EXISTS knowledge_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_name VARCHAR(255) NOT NULL DEFAULT '',
			file_md5 VARCHAR(255) NOT NULL DEFAULT '',
			vector_id TEXT NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			is_deleted INTEGER NOT NULL DEFAULT '0',
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
	`,
		"cron": `
		CREATE TABLE IF NOT EXISTS cron (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cron_name VARCHAR(255) NOT NULL DEFAULT '',
			cron VARCHAR(255) NOT NULL DEFAULT '',
			target_id TEXT NOT NULL,
			group_id TEXT NOT NULL,
			command VARCHAR(255) NOT NULL DEFAULT '',
			prompt TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1, -- 0:disable 1:enable
			cron_job_id INTEGER NOT NULL DEFAULT '0',
			create_time INTEGER NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			is_deleted INTEGER NOT NULL DEFAULT '0',
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
		    type VARCHAR(255) NOT NULL DEFAULT '',
		    create_by VARCHAR(255) NOT NULL DEFAULT ''
		);
	`,
		"agent_runs": `
		CREATE TABLE IF NOT EXISTS agent_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id VARCHAR(100) NOT NULL DEFAULT '',
			chat_id VARCHAR(255) NOT NULL DEFAULT '',
			msg_id VARCHAR(255) NOT NULL DEFAULT '',
			mode VARCHAR(100) NOT NULL DEFAULT '',
			input TEXT NOT NULL,
			final_output TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			token_total INTEGER NOT NULL DEFAULT 0,
			step_count INTEGER NOT NULL DEFAULT 0,
			replay_of INTEGER NOT NULL DEFAULT 0,
			skill_id VARCHAR(255) NOT NULL DEFAULT '',
			skill_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_version VARCHAR(100) NOT NULL DEFAULT '',
			selector_reason TEXT NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_user_id ON agent_runs(user_id);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON agent_runs(status);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_mode ON agent_runs(mode);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_create_time ON agent_runs(create_time);
	`,
		"agent_steps": `
		CREATE TABLE IF NOT EXISTS agent_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id INTEGER NOT NULL DEFAULT 0,
			step_index INTEGER NOT NULL DEFAULT 0,
			kind VARCHAR(50) NOT NULL DEFAULT '',
			name VARCHAR(255) NOT NULL DEFAULT '',
			tool_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_id VARCHAR(255) NOT NULL DEFAULT '',
			skill_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_version VARCHAR(100) NOT NULL DEFAULT '',
			input TEXT NOT NULL,
			raw_output TEXT NOT NULL DEFAULT '',
			observations TEXT NOT NULL DEFAULT '',
			allowed_tools TEXT NOT NULL DEFAULT '[]',
			step_context TEXT NOT NULL DEFAULT '',
			token INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			provider VARCHAR(100) NOT NULL DEFAULT '',
			model VARCHAR(255) NOT NULL DEFAULT '',
			create_time INTEGER NOT NULL DEFAULT '0',
			update_time INTEGER NOT NULL DEFAULT '0',
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_agent_steps_run_id ON agent_steps(run_id);
		CREATE INDEX IF NOT EXISTS idx_agent_steps_run_idx ON agent_steps(run_id, step_index);
	`,
		"sessions": `
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id VARCHAR(100) NOT NULL DEFAULT '',
			session_key TEXT NOT NULL DEFAULT '',
			channel VARCHAR(100) NOT NULL DEFAULT '',
			account_id VARCHAR(255) NOT NULL DEFAULT '',
			peer_id VARCHAR(255) NOT NULL DEFAULT '',
			group_id VARCHAR(255) NOT NULL DEFAULT '',
			thread_id VARCHAR(255) NOT NULL DEFAULT '',
			kind VARCHAR(50) NOT NULL DEFAULT '',
			transcript_path TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			message_count INTEGER NOT NULL DEFAULT 0,
			last_message_at INTEGER NOT NULL DEFAULT 0,
			create_time INTEGER NOT NULL DEFAULT 0,
			update_time INTEGER NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_channel ON sessions(channel);
		CREATE INDEX IF NOT EXISTS idx_sessions_update_time ON sessions(update_time);
	`,
	}

	mysqlInitializeSQLs = []string{
		// 1. users 表 (嵌入索引)
		`
       CREATE TABLE IF NOT EXISTS users (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          user_id varchar(100) NOT NULL DEFAULT '0',
          update_time INT(10) NOT NULL DEFAULT 0,
          token BIGINT NOT NULL DEFAULT 0,
          avail_token BIGINT NOT NULL DEFAULT 0,
           create_time INT(10) NOT NULL DEFAULT 0,
           from_bot VARCHAR(255) NOT NULL DEFAULT '',
           llm_config TEXT NOT NULL,
           
           -- 嵌入索引：idx_users_user_id
           INDEX idx_users_user_id (user_id)
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		// 2. records 表 (嵌入索引)
		`
       CREATE TABLE IF NOT EXISTS records (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          user_id varchar(100) NOT NULL DEFAULT '0',
          question MEDIUMTEXT NOT NULL,
          answer MEDIUMTEXT NOT NULL,
          content MEDIUMTEXT NOT NULL,
          create_time INT(10) NOT NULL DEFAULT 0,
           update_time INT(10) NOT NULL DEFAULT 0,
          is_deleted INT(10) NOT NULL DEFAULT 0,
          token INT(10) NOT NULL DEFAULT 0,
           mode VARCHAR(100) NOT NULL DEFAULT '',
           record_type tinyint(1) NOT NULL DEFAULT 0 COMMENT '0:text, 1:image 2:video 3: web',
           from_bot VARCHAR(255) NOT NULL DEFAULT '',
           
           -- 嵌入索引：idx_records_user_id 和 idx_records_create_time
           INDEX idx_records_user_id (user_id),
           INDEX idx_records_create_time (create_time)
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		// 3. knowledge_files table (no extra index beyond PRIMARY KEY)
		`CREATE TABLE IF NOT EXISTS knowledge_files (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          file_name VARCHAR(255) NOT NULL DEFAULT '',
          file_md5 VARCHAR(255) NOT NULL DEFAULT '',
          vector_id TEXT NOT NULL,
          create_time INT(10) NOT NULL DEFAULT 0,
          update_time INT(10) NOT NULL DEFAULT 0,
          is_deleted INT(10) NOT NULL DEFAULT 0,
          from_bot VARCHAR(255) NOT NULL DEFAULT ''
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		// 4. cron 表 (无额外索引，仅PRIMARY KEY)
		`CREATE TABLE IF NOT EXISTS cron (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          cron_name VARCHAR(255) NOT NULL DEFAULT '',
          cron VARCHAR(255) NOT NULL DEFAULT '',
          target_id TEXT NOT NULL,
          group_id TEXT NOT NULL,
          command VARCHAR(255) NOT NULL DEFAULT '',
          prompt TEXT NOT NULL,
          status tinyint(1) NOT NULL DEFAULT 1 COMMENT '0:disable 1:enable',
          cron_job_id INT(10) NOT NULL DEFAULT 0,
          create_time INT(10) NOT NULL DEFAULT 0,
          update_time INT(10) NOT NULL DEFAULT 0,
          is_deleted INT(10) NOT NULL DEFAULT 0,
          from_bot VARCHAR(255) NOT NULL DEFAULT '',
          type VARCHAR(255) NOT NULL DEFAULT '',
    	  create_by VARCHAR(255) NOT NULL DEFAULT ''
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		`CREATE TABLE IF NOT EXISTS agent_runs (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          user_id VARCHAR(100) NOT NULL DEFAULT '',
          chat_id VARCHAR(255) NOT NULL DEFAULT '',
          msg_id VARCHAR(255) NOT NULL DEFAULT '',
          mode VARCHAR(100) NOT NULL DEFAULT '',
          input MEDIUMTEXT NOT NULL,
          final_output MEDIUMTEXT NOT NULL,
          status VARCHAR(50) NOT NULL DEFAULT '',
          error MEDIUMTEXT NOT NULL,
          token_total INT NOT NULL DEFAULT 0,
          step_count INT NOT NULL DEFAULT 0,
          replay_of BIGINT NOT NULL DEFAULT 0,
          skill_id VARCHAR(255) NOT NULL DEFAULT '',
          skill_name VARCHAR(255) NOT NULL DEFAULT '',
          skill_version VARCHAR(100) NOT NULL DEFAULT '',
          selector_reason MEDIUMTEXT NOT NULL,
          create_time INT(10) NOT NULL DEFAULT 0,
          update_time INT(10) NOT NULL DEFAULT 0,
          from_bot VARCHAR(255) NOT NULL DEFAULT '',

          INDEX idx_agent_runs_user_id (user_id),
          INDEX idx_agent_runs_status (status),
          INDEX idx_agent_runs_mode (mode),
          INDEX idx_agent_runs_create_time (create_time)
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		`CREATE TABLE IF NOT EXISTS agent_steps (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          run_id INT NOT NULL DEFAULT 0,
          step_index INT NOT NULL DEFAULT 0,
          kind VARCHAR(50) NOT NULL DEFAULT '',
          name VARCHAR(255) NOT NULL DEFAULT '',
          tool_name VARCHAR(255) NOT NULL DEFAULT '',
          skill_id VARCHAR(255) NOT NULL DEFAULT '',
          skill_name VARCHAR(255) NOT NULL DEFAULT '',
          skill_version VARCHAR(100) NOT NULL DEFAULT '',
          input MEDIUMTEXT NOT NULL,
          raw_output MEDIUMTEXT NOT NULL,
          observations MEDIUMTEXT NOT NULL,
          allowed_tools MEDIUMTEXT NOT NULL,
          step_context MEDIUMTEXT NOT NULL,
          token INT NOT NULL DEFAULT 0,
          status VARCHAR(50) NOT NULL DEFAULT '',
          error MEDIUMTEXT NOT NULL,
          provider VARCHAR(100) NOT NULL DEFAULT '',
          model VARCHAR(255) NOT NULL DEFAULT '',
          create_time INT(10) NOT NULL DEFAULT 0,
          update_time INT(10) NOT NULL DEFAULT 0,
          from_bot VARCHAR(255) NOT NULL DEFAULT '',

          INDEX idx_agent_steps_run_id (run_id),
          INDEX idx_agent_steps_run_idx (run_id, step_index)
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
		`CREATE TABLE IF NOT EXISTS sessions (
          id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
          session_id VARCHAR(100) NOT NULL DEFAULT '',
          session_key MEDIUMTEXT NOT NULL,
          channel VARCHAR(100) NOT NULL DEFAULT '',
          account_id VARCHAR(255) NOT NULL DEFAULT '',
          peer_id VARCHAR(255) NOT NULL DEFAULT '',
          group_id VARCHAR(255) NOT NULL DEFAULT '',
          thread_id VARCHAR(255) NOT NULL DEFAULT '',
          kind VARCHAR(50) NOT NULL DEFAULT '',
          transcript_path MEDIUMTEXT NOT NULL,
          summary MEDIUMTEXT NOT NULL,
          message_count INT NOT NULL DEFAULT 0,
          last_message_at BIGINT NOT NULL DEFAULT 0,
          create_time INT(10) NOT NULL DEFAULT 0,
          update_time INT(10) NOT NULL DEFAULT 0,
          from_bot VARCHAR(255) NOT NULL DEFAULT '',

          UNIQUE KEY idx_sessions_session_id (session_id),
          INDEX idx_sessions_channel (channel),
          INDEX idx_sessions_update_time (update_time)
       ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`,
	}
)

var (
	DB *sql.DB
)

type DailyStat struct {
	Date     string `json:"date"`
	NewCount int    `json:"new_count"`
}

func InitTable() {
	var err error
	if _, err = os.Stat(botUtils.GetAbsPath("data")); os.IsNotExist(err) {
		// if dir don't exist, create it.
		err := os.MkdirAll(botUtils.GetAbsPath("data"), 0755)
		if err != nil {
			logger.Fatal("create direction fail:", "err", err)
			return
		}
		logger.Info("✅ create direction success")
	}

	DB, err = sql.Open(conf.BaseConfInfo.DBType, conf.BaseConfInfo.DBConf)
	if err != nil {
		logger.Fatal(err.Error())
	}

	// init table
	switch conf.BaseConfInfo.DBType {
	case "sqlite3":
		err = initializeSqlite3Table(DB)
		if err != nil {
			logger.Fatal("create sqlite table fail", "err", err)
		}
	case "mysql":
		err = initializeMySQLTables(DB)
		if err != nil {
			logger.Fatal("create mysql table fail", "err", err)
		}
	}

	err = migratePrimaryAgentTables(DB)
	if err != nil {
		logger.Fatal("migrate primary agent tables fail", "err", err)
	}

	InsertRecord(context.Background())

	err = InitFeatureDB()
	if err != nil {
		logger.Error("feature db initialize fail", "err", err)
	}

	logger.Info("db initialize successfully")
}

func initializeMySQLTables(db *sql.DB) error {
	for i, sqlStr := range mysqlInitializeSQLs {
		_, err := db.Exec(sqlStr)
		if err != nil {
			logger.Error("check table fail", "err", err)
			return fmt.Errorf("execute SQL batch %d fail: %v\nSQL: %s", i+1, err, sqlStr)
		}
	}

	return nil
}

// initializeSqlite3Table check table exist or not.
func initializeSqlite3Table(db *sql.DB) error {
	for tableName, createSQL := range sqlite3TableSQLs {
		_, err := db.Exec(createSQL)
		if err != nil {
			logger.Error("check table fail", "tableName", tableName, "err", err)
			return fmt.Errorf("create table %s fail: %v", tableName, err)
		}
	}

	return nil
}

func migratePrimaryAgentTables(db *sql.DB) error {
	if db == nil {
		return nil
	}

	switch conf.BaseConfInfo.DBType {
	case "sqlite3":
		if err := migrateSQLiteKnowledgeFileTable(db); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_runs", "skill_id", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_runs", "skill_name", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_runs", "skill_version", "VARCHAR(100) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_runs", "selector_reason", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_steps", "skill_id", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_steps", "skill_name", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_steps", "skill_version", "VARCHAR(100) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_steps", "allowed_tools", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(db, "agent_steps", "step_context", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	case "mysql":
		if err := migrateMySQLKnowledgeFileTable(db); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_runs", "skill_id", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_runs", "skill_name", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_runs", "skill_version", "VARCHAR(100) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_runs", "selector_reason", "MEDIUMTEXT NOT NULL"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_steps", "skill_id", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_steps", "skill_name", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_steps", "skill_version", "VARCHAR(100) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_steps", "allowed_tools", "MEDIUMTEXT NOT NULL"); err != nil {
			return err
		}
		if err := ensureMySQLColumn(db, "agent_steps", "step_context", "MEDIUMTEXT NOT NULL"); err != nil {
			return err
		}
	}

	return nil
}

func ensureSQLiteColumn(db *sql.DB, tableName, columnName, definition string) error {
	exists, err := sqliteColumnExists(db, tableName, columnName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition))
	return err
}

func sqliteColumnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err = rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}

	return false, rows.Err()
}

func ensureMySQLColumn(db *sql.DB, tableName, columnName, definition string) error {
	var count int
	query := `SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`
	if err := db.QueryRow(query, tableName, columnName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition))
	return err
}

func migrateSQLiteKnowledgeFileTable(db *sql.DB) error {
	oldExists, err := sqliteTableExists(db, "rag_files")
	if err != nil || !oldExists {
		return err
	}
	newExists, err := sqliteTableExists(db, "knowledge_files")
	if err != nil || !newExists {
		return err
	}
	_, err = db.Exec(`
		INSERT OR IGNORE INTO knowledge_files (id, file_name, file_md5, vector_id, create_time, update_time, is_deleted, from_bot)
		SELECT id, file_name, file_md5, vector_id, create_time, update_time, is_deleted, from_bot
		FROM rag_files`)
	return err
}

func sqliteTableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", tableName).Scan(&count)
	return count > 0, err
}

func migrateMySQLKnowledgeFileTable(db *sql.DB) error {
	oldExists, err := mysqlTableExists(db, "rag_files")
	if err != nil || !oldExists {
		return err
	}
	newExists, err := mysqlTableExists(db, "knowledge_files")
	if err != nil || !newExists {
		return err
	}
	_, err = db.Exec(`
		INSERT IGNORE INTO knowledge_files (id, file_name, file_md5, vector_id, create_time, update_time, is_deleted, from_bot)
		SELECT id, file_name, file_md5, vector_id, create_time, update_time, is_deleted, from_bot
		FROM rag_files`)
	return err
}

func mysqlTableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, tableName).Scan(&count)
	return count > 0, err
}
