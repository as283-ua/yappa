Connection to the server to connect with other users as well as registration and other "round-trip" requests will be established through HTTP3 (HTTP over QUIC).

Three main end-points will exist on the main chat server:
- `POST /register`. First time users of the client will have to register by sending their username and email. The server will respond with with a single use token that the client will use to get a certificate from the CA server.
  A possibility would be to disable the need for email to filter bot registers, but leave it on by default.
- `POST /register/confirm`. After using the token from the previous request to get successfully get a certificate, the client will access this end-point providing the certificate, which will be saved in the database to identify them, and another single use token generated by the CA.
- `CONNECT /connect`. This end-point will serve not only as the primary source of data exchange for this service, allowing clients to chat, but also for authentication, in which the server will identify the connecting user by mTLS. This makes the use of stateless tokens like JWT unnecessary, given that the ability to connect correctly is proof enough of the user's identity. The connection is long-lived and uses a QUIC stream to exchange data through a single connection.
  Data will be either immediately resent to message receiver's or stored securely in the database for when they connect the next time. [[Chat]]
- `POST /register/refresh`. Renew a user's certificate in case it's close to expiration (<30 days).
- `GET /users?q={query}&page={page}&size{size}`. Fetch a list of users filtering by name (contains) with pagination.
- `GET /groups?q={query}&page={page}&size{size}`. Fetch a list of groups filtering by name (contains) with pagination.
- `POST /groups/{name}`. Create a group chat. In the server, a group is simply an entity with a name and id. It doesn't have a direct persistent relation with the users in the database. It may only keep the number of members in the group.
- `POST /groups/{name}/join`. Join a group chat. Acts as subscribing to the message inbox for said group. Increment the member count of the group.
- `POST /groups/{name}/leave`. Join a group chat. Stop receiving messages from a group chat. Decrement the number of members in the group.

The CA server acts as a separate service, whose only purpose is to sign, revoke and renew certificates for users. It has these end-points available:
- `POST /allow/{username}`. End-point only accessible by the chat server using mTLS. Saves the single use token on the CA server, allowing the client to then provide said token to verify their identity.
- `POST /sign/{username}`. The client provides the single use token generated by the chat server, their username and their public key. The server responds with a certificate or an error response, depending on if the token/username pair is correct or not.
- `GET /certificates`. Admin console only. Get a list of certificates and their owners (clients).
- `POST /revoke/{username}`. Admin console only. Marks a certificate as revoked in the CA's database.
- `POST /reinstate/{username}`. Back-up in case a revocation is done accidentally.