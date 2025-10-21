#!/usr/bin/env bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE transaction_db;
    CREATE DATABASE sso_db;
    CREATE DATABASE billing_db;
    GRANT ALL PRIVILEGES ON DATABASE transaction_db TO postgres;
    GRANT ALL PRIVILEGES ON DATABASE sso_db TO postgres;
    GRANT ALL PRIVILEGES ON DATABASE billing_db TO postgres;
EOSQL
