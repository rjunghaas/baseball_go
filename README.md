Version of Baseball App

Created a back-end service for calculating a player's VORP (Value Over Replacement Player).  Goal of app was to gain familiarity wiht Golang, use gRPC for communication between API layer and microservice, and to get experience with Docker.

APIs: 

GET {host}:3308/search/{searchString} - will take user input and match to closest name in database.

GET {host}:3308/scrape/{startDate}/{endDate} - will take player from most recent search and calculate VORP over period from startDate to endDate.

Have added Docker to this version with Dockerfiles for API, Microservice, and Databases.  Use Docker Compose to standup and run containers.
