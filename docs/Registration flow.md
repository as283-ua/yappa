1. Client sends registration info (username for now, 28-02-2025)
	- Account metadata registered as **unconfirmed** on the server.
	- Server returns one time registration token for user, stored temporarily.
2. Certificate request by user, CSR (Certificate Signing Request)
	- User generates asymmetric key pairs. (ECDH)
	- User send CSR to a separate CA server that will sign the user's certificate, for use in Yappa, as well as the token.
3. CA server signing of certificate
	- CA server queries message server for the one time token and verifies.
	- If it doesn't match, deny and end this flow.
	- Otherwise, sign the certificate, generate a one time token and respond to user with both of these.
4. Confirm registration
	- Generate another key pair for client key exchange (ECDH)
	- User confirms their registration as complete by sending the messaging server their certificate and the public key of their ECDH.
	- Server verifies that the CA is valid and that the token matches that of the CA server. 
	- If correct, delete one time token and adds user cert and public ECDH key to database.
	- Notifies CA server to delete its one time token as well.

Users will log in automatically to the server using their certificate. No passwords are required for this process.