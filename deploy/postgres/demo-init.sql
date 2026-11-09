-- SPDX-License-Identifier: Apache-2.0
\getenv reader_password DEMO_DATABASE_PASSWORD
CREATE ROLE schewise_reader LOGIN PASSWORD :'reader_password' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
ALTER ROLE schewise_reader SET default_transaction_read_only = on;
GRANT CONNECT ON DATABASE schewise_demo TO schewise_reader;
\i /fixtures/demo.sql
