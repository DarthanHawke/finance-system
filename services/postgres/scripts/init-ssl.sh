#!/usr/bin/env bash
set -e

SSL_DIR="/var/lib/postgresql/certs"
mkdir -p $SSL_DIR
chown -R postgres:postgres $SSL_DIR
chmod -R 750 $SSL_DIR

openssl req -new -x509 -days 3650 -nodes -text \
  -out $SSL_DIR/ca.crt \
  -keyout $SSL_DIR/ca.key \
  -subj "/CN=PostgreSQL CA"
chmod 600 $SSL_DIR/ca.key

openssl req -new -nodes -text -out $SSL_DIR/server.csr \
  -keyout $SSL_DIR/server.key \
  -subj "/CN=postgres"
chmod 600 $SSL_DIR/server.key

openssl x509 -req -in $SSL_DIR/server.csr -text -days 365 \
  -CA $SSL_DIR/ca.crt \
  -CAkey $SSL_DIR/ca.key \
  -CAcreateserial \
  -out $SSL_DIR/server.crt \
  -extfile <(echo -e "subjectAltName=DNS:postgres,DNS:localhost,IP:127.0.0.1")

chown postgres:postgres $SSL_DIR/*

openssl req -new -nodes -text -out $SSL_DIR/client.csr \
  -keyout $SSL_DIR/client.key \
  -subj "/CN=pg_client"
chmod 600 $SSL_DIR/client.key

openssl x509 -req -in $SSL_DIR/client.csr -text -days 365 \
  -CA $SSL_DIR/ca.crt \
  -CAkey $SSL_DIR/ca.key \
  -CAcreateserial \
  -out $SSL_DIR/client.crt