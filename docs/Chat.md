# Message sharing and server behavior
The service must guarantee as much user anonymity within the network as possible. Total anonymity is not possible due to the server actually needing to know which users are connected at a moment, but data stored at rest should not give away information about user-to-user activity.

Server behavior will be different depending on each client's connectivity within a chat. 

In a direct message chat (only two clients, one on one), if both of them are connected at the same time when a message is sent, the server will simply acts as a relay and resend the message to the receiver.
On the other hand, if only one of the clients is active and sends a message which cannot be received immediately by the peer, the message must be saved locally in the server in the other client's inbox

To achieve the objective of maximizing user anonymity, metadata about every client's chat must also be kept private and only known to the participants of a chat. 

## Q: how to have an anonymous inbox that a user knows is his but the server can't know the sender or receiver?

### Possible solution

when first connecting (first message sent by one of the clients) the clients decide on a secret-identifier for the chat that the server doesn't know: `user1-user2-randombytes` &rarr; SHA512 &rarr; `inbox id`. This inbox id will be used for saving messages not yet received by the other client that is not connected at the moment of sending. When reconnecting, the receiver will know which inbox id(s) to access to fetch pending messages from their chats.

In case of a group chat, inbox id could be `groupname-randombytes`.

The chat initiator also decides on a shared symmetric key that the users will use to encrypt the messages between them.

## Problem: when to remove the message from the inbox/how to ensure user has the right to receive the message

### Possible solution
~~A token is shared with the server when sending a message to the inbox~~. When a message is sent to the server which has to be saved in the inbox, the server generates a random token which it saves (encrypted with the server startup key). When reconnecting, said token is sent encrypted with the user's (the one who is checking their inbox) public key, which they decrypt and send back to the server, who compares the token to the one it has and deletes the message if it matches. To allow this mechanism for group chat, share a counter with the server for how many user's are expected to access to said message before deletion. Given that the counter will be encrypted by the server, saving the number of clients in a group may also be a possibility.

The random token and message receive-counter must be encrypted with the server's startup key so that, if the server happens to shut down, the token and counter are still available if the right key is selected. [[Server startup key]]

### Problem: initial inbox id exchange only works if all parties are connected at the same time

Since they can't use the anonymous inbox system yet, they cannot exchange the id unless they are both connected.

### ~~Possible solution 1~~
~~Only allow to initiate new chats if all parties are connected.~~

### Possible solution 2
Have a public inbox that only contains the inbox id encrypted by the sender with the receiver's public key. When a user reconnects, they check their public inbox to get inbox-ids and then access the anonymous inbox they just got access to.

To again maximize anonymity of the database even if database is somehow leaked, the inbox id could be made into a random set of bytes + the username in SHA512 that would be shared upon user registration with a "password" encrypted by the server at rest which the user must provide when consulting the inbox.

## Another possible solution for inboxes
~~Use the user "public inbox" for all types of messages destined for the user, instead having a per conversation-inbox system. This eliminates the need for a shared secret inbox id which may not be as secret as it seems and may be inferred from activity. Messages are encrypted so the server still doesn't know who the user is talking to, but does know how active they are. Does it matter?~~

## Problem
This system really only works in case the database is leaked and compromises information about who chats with who. However, during live chatting between two or more clients, the server does directly know who is chatting with who, in-memory.

Probably unavoidable.

# Group chats
Group chats may be `public` or `private`.

Public group chats are accessible to anyone and may be entered by any authenticated user.

Private group chats require permission to enter, but can be viewed in the results for searching groups.

Just like direct chats, group chat also need an anonymous inbox for pending messages.

The group creator acts as the group admin. (Maybe think about having multiple admins. Complexity in adding, removing admins from a group).

## Join group cases
1. User tries to enter a public group where at least one member is active. Said active user is notified in order to exchange privately the inbox id of the group(using the new member's public key).
   
2. User tries to enter a public group where no users are currently active. 
   Option A, not allowed, wait for when group has active chatters (don't show groups with 0 active chatters in the group list). 
   Option B, some kind of public group inbox just like user public inboxes. Content is the name of the user. First member of the chat to connect reads from the public inbox and sends to the user (or their inbox if they are not connected) the group inbox id.
   
3. User tries to enter a private group.
   Option. Reuse option B from public group with no users case. Use a public user inbox for the group as a "Request to enter". Group admin may choose to accept, reject or ignore (save for later) a request for entry. 
   The user needs to wait to be accepted anyway, so it doesn't matter if there are active users or not in the group chat.
   Private groups will have a random password that needs to be provided and correct when connecting to the group. See [Messaging](##Messaging).

## Group admin
A group admin manages entry requests from non-members.

The original group creator automatically becomes the group admin and may add or remove other admins.

Admins that aren't the original creator of the group may only accept or reject entry requests. They do not have the power to add or remove admins.

TODO: how to manage without saving the user admin in the database. How to add/remove more admins

## Messaging
Group metadata is not saved in the database, meaning we do not know which users are part of a group. We do, however, need to know at runtime which users are connected to a group.

Users locally know which groups they are part of, so when connecting to the server, apart from the certificate and requesting inbox messages, it will also send a list of groups that it wants to receive messages from while connected. The server will have a map of [group]&rarr;[users], to keep to track of which users to broadcast group messages to (as well as [user]&rarr;[groups] to remove users from groups when they disconnect).

To prevent users from sending a list and connecting to groups they haven't officially joined, groups have passwords generated during the creation of the group that need to be provided at the time of connection to validate that the user is actually a part of the group.

The password will be saved with the group in the database hashed/PBKDF'd (?) using Argon2.

# Local save
[[Local saved client data]]