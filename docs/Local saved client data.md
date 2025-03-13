# Files
- Certificate and public key
- File with:
	- Joined groups + group password
	- Username
	- Saved chats:
		- Messages
		- Private symmetric key
		- Other chatter's public key
# Encryption at rest
Saved chats and other data should be encrypted to avoid other installed applications from snooping into this private information. The key used to encrypt the files at rest, however, will not be encrypted if one of the requirements is for the app to seamlessly start-up without a pin or password, as many of these chat applications do. 

Therefore the file itself is not directly safe from a real attack if, for example, the device is stolen, at which point the attacker may simply open the application rather than bother decrypting the files themselves using the available key.

For this reason, if the user wants to protect their chats securely locally, they may choose to use a pin (or password?) at start-up, making the key not saved in clear text but requiring the user to enter the pass to use the application every time.