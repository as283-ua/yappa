# Message sharing and server behavior
The service must guarantee total user anonymity within the network.

Server behavior will be different depending on each client's connectivity within a chat. 

In a direct message chat (only two clients, one on one), if both of them are connected at the same time when a message is sent, the server will simply acts as a relay and resend the message to the receiver.
On the other hand, if only one of the clients is active and sends a message which cannot be received immediately by the peer, the message must be saved locally in the server in the other client's inbox

To achieve the objective of total user anonymity, metadata about every client's chat must also be kept private and only known to the participants of a chat. 

## Q: how to have an anonymous inbox that a user knows is his but the server can't know the sender or receiver?

### Possible solution

when first connecting (first message sent by one of the clients) the clients decide on a secret-identifier for the chat that the server doesn't know: `user1-user2-randombytes` &rarr; SHA512 &rarr; `inbox id`. This inbox id will be used for saving messages not yet received by the other client that is not connected at the moment of sending. When reconnecting, the receiver will know which inbox id(s) to access to fetch pending messages from their chats.

In case of a group chat, inbox id could be `groupname-randombytes`.

## Problem: when to remove the message from the inbox.

### Possible solution
A token is shared with the server when sending a message to the inbox. When reconnecting, said token is sent encrypted with the user's (the one who is checking their inbox) public key, which they decrypt and send back to the server, who compares the token to the one it has and deletes the message if it matches. To allow this mechanism for group chat, share a counter with the server for how many user's are expected to access to said message before deletion.

### Problem: initial inbox id exchange only works if all parties are connected at the same time

Since they can't use the anonymous inbox system yet, they cannot exchange the id unless they are both connected.

### Possible solution 1
Only allow to initiate new chats if all parties are connected.

### Possible solution 2
Have a public inbox that only contains the inbox id encrypted by the sender with the receiver's public key. When a user reconnects, they check their public inbox to get inbox-ids and then access the anonymous inbox they just got access to.

## Problem
This system really only works in case the database is leaked and compromises information about who chats with who. However, during live chatting between two or more clients, the server does directly know who is chatting with who, in-memory.

Is this a problem?

# Group chats
Group chats may be `public` or `private`.

Public group chats are accessible to anyone and may be entered by any authenticated user.

Private group chats require permission to enter, but can be viewed in the results for searching groups.

Just like direct chats, group chat also need an anonymous inbox for pending messages.

The group creator acts as the group admin. (Maybe think about having multiple admins. Complexity in adding, removing admins from a group). 
## Cases
1. User tries to enter a public group where at least one member is active. Said active user is notified in order to exchange privately the inbox id of the group (using the new member's public key).
   
2. User tries to enter a public group where no users are currently active. 
   Option A, not allowed, wait for when group has active chatters (don't show groups with 0 active chatters in the list). 
   Option B, some kind of public group inbox just like user public inboxes. Content is the name of the user. First member of the chat to connect reads from the public inbox and sends to the user (or their inbox if they are not connected) the inbox id.
   
3. User tries to enter a private group.
   Option. Reuse option B from public group with no users case. Use a public user inbox for the group as a "Request to enter". Group admin may choose to accept, reject or ignore (save for later) a request for entry. 
   The user needs to wait to be accepted anyway, so it doesn't matter if there are active users or not in the group chat.

## Group admin
A group admin manages entry requests from non-members.

The original group creator automatically becomes the group admin and may add or remove other admins.

Admins that aren't the original creator of the group may only accept or reject entry requests. They do not have the power to add or remove admins.

TODO: how to manage without saving the user admin in the database. How to add/remove more admins