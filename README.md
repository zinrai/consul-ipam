# ipam-mvp-go

This project is a Minimum Viable Product (MVP) implementation of an IP Address Management (IPAM) system using Go and PostgreSQL.

## PostgreSQL Setup

1. Create database:
   ```sql
   CREATE DATABASE ipam;
   ```

2. Create user and grant privileges:
   ```sql
   CREATE USER ipam WITH PASSWORD 'ipampassword';
   GRANT ALL PRIVILEGES ON DATABASE ipam TO ipam;
   ```

3. Connect to the new database:
   ```sql
   \c ipam
   ```

4. Create tables:
   ```sql
   CREATE TABLE networks (
       id SERIAL PRIMARY KEY,
       cidr CIDR NOT NULL,
       gateway INET NOT NULL
   );

   CREATE TABLE ip_addresses (
       id SERIAL PRIMARY KEY,
       network_id INTEGER REFERENCES networks(id),
       address INET NOT NULL,
       hostname TEXT,
       status TEXT NOT NULL
   );
   ```

5. Grant privileges on the tables to ipam user:
   ```sql
   GRANT ALL PRIVILEGES ON TABLE networks TO ipam;
   GRANT ALL PRIVILEGES ON TABLE ip_addresses TO ipam;
   GRANT USAGE, SELECT ON SEQUENCE networks_id_seq TO ipam;
   GRANT USAGE, SELECT ON SEQUENCE ip_addresses_id_seq TO ipam;
   ```

6. Exit PostgreSQL:
   ```
   \q
   ```

Note: The password 'ipampassword' is used here as per your configuration. However, for production environments, it's strongly recommended to use a more secure password.

## Project Setup

1. Create a `config.yaml` file in the project root with the following content:
   ```yaml
   database:
     host: localhost
     port: 5432
     user: ipam
     password: ipampassword
     dbname: ipam
     sslmode: disable
   ```

2. Build the project:
   ```
   $ go build -o ipamserver cmd/ipamserver/main.go
   ```

3. Run the server:
   ```
   ./ipamserver
   ```

## API Usage

Here are some example curl commands to interact with the IPAM HTTP API:

Create a new network:

```
$ curl -X POST http://localhost:8080/network \
    -H "Content-Type: application/json" \
    -d '{"CIDR": "192.168.1.0/24", "Gateway": "192.168.1.1"}'
```

List all networks:

```
$ curl -X GET http://localhost:8080/network
```

Allocate an IP address:

```
$ curl -X POST http://localhost:8080/ip \
    -H "Content-Type: application/json" \
    -d '{"network_id": 1, "hostname": "example-host"}'
```

Allocate a specific IP address:

```
$ curl -X POST http://localhost:8080/ip \
    -H "Content-Type: application/json" \
    -d '{"network_id": 1, "requested_ip": "192.168.1.10", "hostname": "example-host"}'
```

List IP addresses for a network:

```
$ curl -X GET "http://localhost:8080/ip?network_id=1"
```

Release an IP address:

```
$ curl -X DELETE "http://localhost:8080/ip?ip_id=1"
```

Update IP address hostname:

```
$ curl -X PUT http://localhost:8080/ip \
    -H "Content-Type: application/json" \
    -d '{"ip_id": 1, "hostname": "new-hostname"}'
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
