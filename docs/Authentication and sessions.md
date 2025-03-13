The use of certificates may cause some difficulties regarding lost messages due to expiration date passed without the user refreshing their certificate and other users sending messages to them.

This problem may be solved by checking on the client side if the peer's certificate is not expired (to not saturate the server by having it do this), and if it is, reject the new message and indicate that the user that is trying to be contacted is not active and has an expired session (key).

