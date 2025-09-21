#!/bin/sh
openssl req -x509 -config openssl-client-cert.cnf -noenc -days 9999 -newkey rsa:2048 -keyout shc-client-key.pem -out shc-client-cert.pem
