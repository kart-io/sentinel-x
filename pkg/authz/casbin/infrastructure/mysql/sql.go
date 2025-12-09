package mysql

// Schema returns the SQL schema for the casbin_rule table
const Schema = `
CREATE TABLE IF NOT EXISTS casbin_rule (
    id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
    p_type varchar(100),
    v0 varchar(100),
    v1 varchar(100),
    v2 varchar(100),
    v3 varchar(100),
    v4 varchar(100),
    v5 varchar(100),
    INDEX idx_casbin_rule_p_type (p_type),
    INDEX idx_casbin_rule_v0 (v0),
    INDEX idx_casbin_rule_v1 (v1),
    INDEX idx_casbin_rule_v2 (v2),
    INDEX idx_casbin_rule_v3 (v3),
    INDEX idx_casbin_rule_v4 (v4),
    INDEX idx_casbin_rule_v5 (v5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
