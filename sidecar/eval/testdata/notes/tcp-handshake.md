# TCP three-way handshake

The client sends SYN with its initial sequence number. The server answers SYN-ACK, acknowledging that number and sending
its own. The client replies ACK and the connection is established. Closing takes four segments: FIN, ACK, FIN, ACK.
A SYN flood fills the half-open queue of the server; SYN cookies avoid keeping state until the final ACK arrives.
