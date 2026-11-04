-- Only the initialization account is a superuser. The runtime/migration account
-- owns the application database without cluster-wide administrative rights.
\getenv app_password APP_DATABASE_PASSWORD
CREATE ROLE schewise LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION PASSWORD :'app_password';
ALTER DATABASE schewise OWNER TO schewise;
