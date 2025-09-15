# Details

## shc-ca.pem
Downloaded from https://github.com/BoschSmartHome/bosch-shc-api-docs/blob/master/best_practice/Smart%20Home%20Controller%20Issuing%20CA.pem

Then the CA certificates have been concatenated.

## Generating client key + cert
openssl req -x509 -nodes -days 9999 -newkey rsa:2048 -keyout secrets/shc-client-key.pem -out secrets/shc-client-cert.pem


