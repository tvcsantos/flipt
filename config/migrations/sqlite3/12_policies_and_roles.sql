CREATE TABLE IF NOT EXISTS roles (
  id VARCHAR(255) PRIMARY KEY UNIQUE NOT NULL,
  `key` VARCHAR(255) UNIQUE NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `description` TEXT,
  `default` BOOLEAN DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS namespace_permissions (
  id VARCHAR(255) PRIMARY KEY UNIQUE NOT NULL,
  namespace_key VARCHAR(255) NOT NULL REFERENCES namespaces ON DELETE CASCADE,
  role_id VARCHAR(255) NOT NULL REFERENCES roles ON DELETE CASCADE,
  `action` INTEGER DEFAULT 0 NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
  
  UNIQUE (namespace_key, role_id, action),
);



