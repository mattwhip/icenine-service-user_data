# icenine-service-user_data
User Data service for the IceNine system

This service is part of the IceNine project, a scalable cloud-based multiplayer server. The User Data service is built using Golang with the Buffalo web framework. It provides access to user-specific game data. Here are some important pieces in the repository code structure:
- *rpc*: RPC server code

Concepts/technologies used:
- Golang with Buffalo web framework for application creation
	- Object Relational Mapping for MySQL database access using Buffalo's Pop library
- gRPC for inter-service communication (e.g. login service retrieves user data details)
- Protobuf for all message (de)serialization
- Glicko2 rating system for player skill data
