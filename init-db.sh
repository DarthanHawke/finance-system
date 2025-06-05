#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE payment_db;
    CREATE DATABASE auth_db;
    GRANT ALL PRIVILEGES ON DATABASE payment_db TO postgres;
    GRANT ALL PRIVILEGES ON DATABASE auth_db TO postgres;
EOSQL