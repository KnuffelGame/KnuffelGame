#!/bin/sh
set -e

# Pass environment variables into the psql command
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create user 1 with password from environment variable
    CREATE USER $USER1_USER WITH PASSWORD '$USER1_PASSWORD';

    -- Create database 1 and set its owner
    CREATE DATABASE $USER1_DB OWNER $USER1_USER;

    -- Grant all privileges on database 1 to user 1
    GRANT ALL PRIVILEGES ON DATABASE $USER1_DB TO $USER1_USER;

    -- Create user 2 with password from environment variable
    CREATE USER $USER2_USER WITH PASSWORD '$USER2_PASSWORD';

    -- Create database 2 and set its owner
    CREATE DATABASE $USER2_DB OWNER $USER2_USER;

    -- Grant all privileges on database 2 to user 2
    GRANT ALL PRIVILEGES ON DATABASE $USER2_DB TO $USER2_USER;
EOSQL