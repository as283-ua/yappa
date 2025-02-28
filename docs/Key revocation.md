A key revocation system must be implemented in order to invalidate certificates signed by the CA server if the key becomes compromised, is misused, etc.

The revocation options shall be available via a client (will be referred to as ca-client) that is available on the same machine that runs the CA server, and the features must only be accessible from 127.0.0.1 to ensure that only admin users are allowed to revoke certificates from users. 

The ca-client, at least initially, shall only show a list of clients and their certificate from which to choose which ones to revoke.
