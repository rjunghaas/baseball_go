
 Version of Baseball App

Created a back-end service for calculating a player's VORP (Value over Replacement Player) Goal of app was to gain familiarity with Golang, use gRPC for communication between API layer and microservice, and to get experience with Docker. This version accomplishes the first two goals.

APIs: 
GET 127.0.0.1:3308/search/{searchString} - will take user input and match to closest name in database 
GET 127.0.0.1:3308/scrape/{startDate}/{endDate} - will take player from most recent search and calculate VORP over period from startDate to endDate
