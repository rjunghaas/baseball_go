package main

import (
  "fmt"
  "flag"
  "net"
  "log"

  "github.com/PuerkitoBio/goquery"
  "github.com/djimenez/iconv-go"
  "net/http"
  "strings"
  "strconv"

  context "golang.org/x/net/context"
  grpc "google.golang.org/grpc"

  pb "baseball/proto"
)

var (
  tls = flag.Bool ("tls", false, "Connection uses TLS if true, else plain TCP")
  port = flag.Int("port", 10000, "The server port")
)

type Player struct {
  name string
  id int
}

type Scraper struct {
  url string
  charset string
  doc *goquery.Document
}

type VorpServer struct {}

func NewScraper(url string, charset string) *Scraper {
  s := new(Scraper)
  s.url = url
  s.charset = charset
  s.doc = s.GetDocument()
  return s
}

func (s *Scraper) GetDocument() *goquery.Document {
  res := s.GetResponse()
  defer res.Body.Close()

  utfBody, err := iconv.NewReader(res.Body, s.charset, "utf-8")
  if err != nil {
    panic(err.Error())
  }

  doc, err := goquery.NewDocumentFromReader(utfBody)
  if err != nil{
    panic(err.Error())
  }
  return doc
}

func (s *Scraper) GetResponse() *http.Response {
  res,err := http.Get(s.url)
  if err != nil {
    panic(err.Error())
  }
  return res
}

func (s *Scraper) Find(selector string) *goquery.Selection {
  return s.doc.Find(selector)
}

func GetVorpValue(totalsRow [19]int ,summaryRow [4]float64) float64 {
  // CONSTANTS
  lgRunsPerOut := 0.1633
  replaceR := 0.8

  games := float64(totalsRow[0])
  ab := float64(totalsRow[2])
  hits := float64(totalsRow[4])
  dbl := float64(totalsRow[5])
  tpl := float64(totalsRow[6])
  hr := float64(totalsRow[7])
  bb := float64(totalsRow[9])
  hbp := float64(totalsRow[11])
  cs := float64(totalsRow[14])
  sh := float64(totalsRow[15])
  sf := float64(totalsRow[16])
  gdp := float64(totalsRow[17])

  totalOuts := ab - hits + cs + sh + sf + gdp
  totalSeasons := games / 150
  adjustedOuts := totalOuts / totalSeasons
  runsProduced := lgRunsPerOut * adjustedOuts

  replaceRunsProduced := replaceR * runsProduced

  totalBases := hits + dbl + (2 * tpl) + (3 * hr) + bb + hbp
  runsCreated := ((hits + bb) * totalBases) / (ab + bb)
  adjustedRunsCreated := runsCreated / totalSeasons

  return Round((adjustedRunsCreated - replaceRunsProduced), 2)
}

func (c *VorpServer) GetVorp(ctx context.Context, in *pb.PlayerId) (out *pb.PlayerVorp, err error) {
  // Parse inputs from request object
  id := in.Id
  startDate := in.StartDate
  endDate := in.EndDate

  // Call function to scrape and compute player's Vorp
  vorp := float32(CalcVorp(int(id), startDate, endDate))

  // Put result in object and return to GRPC client
  res := &pb.PlayerVorp {
    Vorp: vorp,
  }

  return res, nil
}

func encodeDateString(date string) string {
  var retArr string
  arr := strings.Split(date, "/")

  l := len(arr)
  for i:=0; i<l; i++ {
    retArr += arr[i]
    if(len(retArr)<8) {
      retArr += "%2F"
    }
  }
  return retArr
}

func CalcVorp(num int, startDate string, endDate string) float64 {
  // Declare constants
  rootUrl := "http://www.baseballmusings.com/cgi-bin/PlayerInfo.py?StartDate="
  var totalsRow [19]int
  var summaryRow [4]float64

  // Encode Date strings
  escStartDate := encodeDateString(startDate)
  escEndDate := encodeDateString(endDate)

  // Use player number returned from query to construct URL
  url := rootUrl + escStartDate + "&EndDate=" + escEndDate + "&GameType=all&PlayedFor=0&PlayedVs=0&Park=0&PlayerID=" + strconv.Itoa(num)
  // Construct Scraper struct and scrape data
  s := NewScraper(url, "utf-8")

  // Get length of Totals table
  len := 0
  s.Find(".dbd .toprow").Each(func(i int, e *goquery.Selection) {
    len += 1
  })

  // Get data from Totals in upper table
  s.Find(".dbd .toprow").Each(func(i int, e *goquery.Selection) {
    if (i == len - 2){
      e.Find("td").Each(func(j int, f *goquery.Selection){
        if (j == 0) {
          num := strings.Split(f.Text(), " ")
          games, err := strconv.Atoi(num[0])
          if err != nil {
            panic(err.Error())
          }
          totalsRow[0] = games
        }
        if (j > 0) {
          x, err := strconv.Atoi(f.Text())
          if err != nil {
            panic(err.Error())
          }
          totalsRow[j] = x
        }
      })
    }
  })

  // Get data from summary Table
  s.Find(".dbd").Each(func(k int, g *goquery.Selection) {
    if (k == 1) {
      g.Find("tr").Each(func(m int, h *goquery.Selection) {
        if (m == 1) {
          h.Find(".number").Each(func(n int, d *goquery.Selection) {
            if (n < 4) {
              y, err := strconv.ParseFloat(d.Text(), 64)
              if err != nil {
                panic(err.Error())
              }
              summaryRow[n] = y
            }
          })
        }
      })
    }
  })
  vorp := GetVorpValue(totalsRow, summaryRow)
  return vorp
}

func Round(v float64, decimals int) float64 {
     var pow float64 = 1
     for i:=0; i<decimals; i++ {
         pow *= 10
     }
     return float64(int((v * pow) + 0.5)) / pow
}

func main() {
  flag.Parse()
  lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
  if err != nil {
    log.Fatalf("failed to listen: %v", err)
  }

  // Create GRPC server and register
  grpcServer := grpc.NewServer()
  v := &VorpServer{}
  pb.RegisterVorpServer(grpcServer, v)
  grpcServer.Serve(lis)
}
