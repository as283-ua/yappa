`Yappa` is a private messaging service designed on top of http/3 as its underlying communication protocol and protobuf as the serialization method. The service should guarantee safe, private connectivity between clients, keeping a zero-knowledge design structure server-side.

The following list includes the flags \[M\]ust, \[S\]hould or \[C\]ould, depending on how necessary it is to complete said the corresponding objective. 

The primary objectives of this project are:
- \[M\] basic client to client real time communication through HTTP3. 
- \[M\] E2EE and client managed keys. The server must not store private user data (messages) or their symmetric or asymmetric encryption keys.
- \[M\] self hosted certificate authority system for asymmetric user keys. Used in user registration to validate user authenticity. See [[Registration flow]]
- \[M\] key revocation.
- \[M\] public key pair sharing across devices.
- \[M\] group chats.
- \[M\] detailed logging.
- \[M\] secure database for user metadata.
- \[S\] Report option for spam/illegal activity. 
- \[S\] pin login client-side. Server login is done with the certificate, this pin would be an option for the client to protect itself E.g. if the device is stolen.
- \[C\] user backups.