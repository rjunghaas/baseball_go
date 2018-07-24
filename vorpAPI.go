package main

import (
  "log"
  "flag"
  "fmt"

  "database/sql"
  "net/http"
  _ "github.com/go-sql-driver/mysql"
  "encoding/json"

  grpc "google.golang.org/grpc"
  context "golang.org/x/net/context"
  pb "baseball/proto"
)

type Player struct {
  name string
  id int
}

var currPlayer Player
var serverAddr = flag.String("server_addr", "vorp_service:10000", "The server address in the format host:port")

func GetPlayerId(name string) int {
  var res Player
  queryStringHead := "SELECT num FROM baseball.player where name="

  // Open database
  db, err := sql.Open("mysql", "remote:mypass@tcp(baseball_db:3306)/baseball")
  if err != nil {
    panic(err.Error())
  }
  defer db.Close()

  // Construct SQL query with player's name
  queryString := queryStringHead + "\"" + name + "\""
  err = db.QueryRow(queryString).Scan(&res.id)
  if err != nil {
    panic(err.Error())
  }

  // Set Current Player's ID
  currPlayer.id = res.id
  return res.id
}

func GetPlayerName(name string) string {
  var res Player
  queryStringHead := "SELECT name FROM baseball.player where name LIKE "

  // Open database
  db, err := sql.Open("mysql", "remote:mypass@tcp(baseball_db:3306)/baseball")
  if err != nil {
  	fmt.Println(err)
  }
  db.SetMaxIdleConns(0)
  if err != nil {
    panic(err.Error())
  }
  defer db.Close()

  // Construct SQL query with player's name
  queryString := queryStringHead + "\"%" + name + "%\""
  err = db.QueryRow(queryString).Scan(&res.name)

  // Set default player if none found in database
  if err != nil {
  fmt.Println(err)
    res.name = "Yoenis Cespedes"
  }

  // Set currPlayer's name
  currPlayer.name = res.name
  return res.name
}

func RespondWithSearchJSON(w http.ResponseWriter, code int, payload string){
  // Create JSON object that we will use for response
  type searchRes struct{
    Name string
  }

  // Package data in JSON object and write response
  res, _ := json.Marshal(searchRes{payload})
  w.Header().Set("Content-Type", "application/json")
  w.Header().Set("charset", "UTF-8")
  w.WriteHeader(code)
  w.Write(res)
}

func RespondWithScrapeJSON(w http.ResponseWriter, code int, vorp float32){
  // Create JSON object that we will use for response
  type scrapeRes struct{
    Vorp float32
  }

  // Package data in JSON object and write response
  res, _ := json.Marshal(scrapeRes{vorp})
  w.Header().Set("Content-Type", "application/json")
  w.Header().Set("charset", "UTF-8")
  w.WriteHeader(code)
  w.Write(res)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
  // Parse user input from URL and search for match in database
  nameString := r.URL.Path[8:]
  name := GetPlayerName(nameString)
  currPlayer.name = name

  RespondWithSearchJSON(w, http.StatusOK, name)
}

func CalcVorpHandlerWrapper(w http.ResponseWriter, r *http.Request, client pb.VorpClient) {
  // Parse start and end dates from URL
  startDate := r.URL.Path[8:18]
  endDate := r.URL.Path[19:]

  // Get currPlayer's name and id
  name := currPlayer.name
  id := int32(GetPlayerId(name))

  // Create request object for GRPC server
  playerData := &pb.PlayerId {
    Id: id,
    StartDate: startDate,
    EndDate: endDate,
  }

  // Use gRPC stub to call GetVorp
  vorp, _ := client.GetVorp(context.Background(), playerData)

  // Write result to JSON and send back to client
  RespondWithScrapeJSON(w, http.StatusOK, float32(vorp.Vorp))
}

func main() {
  // When no TLS is used, must specify an insecure GRPC connection
  opts := []grpc.DialOption{
    grpc.WithInsecure(),
  }

  // Create GRPC connection
  conn, err := grpc.Dial(*serverAddr, opts...)
  if err != nil {
    fmt.Println(err)
    log.Fatalf("fail to dial: %v", err)
  }
  defer conn.Close()

  // Create GRPC stub
  client := pb.NewVorpClient(conn)

  // Create routes for /search and /scrape endpoints
  http.HandleFunc("/search/", SearchHandler)
  // Create closure to send GRPC client to handler function
  http.HandleFunc("/scrape/", func(w http.ResponseWriter, r *http.Request) {
    CalcVorpHandlerWrapper(w, r, client)
  })
  log.Fatal(http.ListenAndServe(":3308", nil))
}
