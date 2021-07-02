package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	//_ "bitbucket.org/iamnd/identity-api/src/docs"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Printer struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	ActivePrint Print  `json:"activePrint"`
}

type Print struct {
	PrintId   int    `json:"printId"`
	UserId    string `json:"userId"`
	Active    bool   `json:"active"`
	StartTime string `json:"startTime"`
}

type PrintSubmission struct {
	UserId    string `json:"userId"`
	PrinterId int    `json:"printerId"`
	Duration  int    `json:"duration"`
}

//Global vars to simulate database conn
//These should probably have some locking garuntees, but that would be under the control
//of the database normally. no sense in pulling that complexity in

var PRINTERS = []Printer{}

//each printer gets 1 queue
var QUEUES = map[int][]Print{}
var LAST_PRINT = 0

func main() {

	//Initiate printers
	PRINTERS = append(PRINTERS, Printer{Id: 1, Name: "Ben’s Printer"})
	PRINTERS = append(PRINTERS, Printer{Id: 2, Name: "Jenn’s Printer"})
	PRINTERS = append(PRINTERS, Printer{Id: 3, Name: "Zach’s Printer 1"})
	PRINTERS = append(PRINTERS, Printer{Id: 4, Name: "Zach’s Printer 2"})

	r := chi.NewRouter()
	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/list_printers", ListPrinters)
	r.Post("/start_print", StartPrint)
	r.Get("/cancel_print/{printId}", CancelPrint)
	r.Get("/list_prints", ListPrints)
	r.Get("/debug", DebugPrint)

	err := http.ListenAndServe("localhost:8080", r)
	if err != nil {
		log.Fatal("Unable to start server:", err.Error())
	}
}

func DebugPrint(w http.ResponseWriter, r *http.Request) {
	spew.Dump(PRINTERS, QUEUES)
}
func ListPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	out, err := json.Marshal(PRINTERS)
	if err != nil {
		log.Println("Unable to marshal printer:", err)
		http.Error(w, "Failure listing printers", 500)
		return
	}

	fmt.Fprint(w, string(out))

}

func StartPrint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bodyBytes, _ := ioutil.ReadAll(r.Body)
	var body PrintSubmission
	err := json.Unmarshal(bodyBytes, &body)
	if err != nil {
		log.Println("Error unmarshalling input to start print: ", err.Error())
		http.Error(w, "Invalid input", 400)
		return
	}

	//check time
	if body.Duration < 1 {
		http.Error(w, "Duration must be at least 1 minute", 400)
		return
	}

	//Calculate duration
	duration := time.Duration(body.Duration) * time.Minute

	//pretend there is locking here. This would be done by the db normally
	LAST_PRINT = LAST_PRINT + 1
	printId := LAST_PRINT

	print := Print{
		PrintId: printId,
		UserId:  body.UserId,
		Active:  true,
		//Golang has a wierd way of formatting time https://golang.org/pkg/time/#Time.Format
		StartTime: time.Now().Add(duration).Format("2006-01-02T03:04:05.0000"),
	}

	//Ugh. I should be doing this in a in memory database I don't like this index mapping
	QUEUES[body.PrinterId-1] = append(QUEUES[body.PrinterId-1], print)

	log.Printf("Adding print %d to queue %d", printId, body.PrinterId-1)
	spew.Dump(QUEUES[body.PrinterId-1])

	PrintQueueHook()
	fmt.Fprint(w, "Print submitted successfully")
}

func CancelPrint(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	printId, err := strconv.Atoi(query.Get("printId"))
	if err != nil {
		http.Error(w, "printId must be an int", 400)
	}
	userId := query.Get("userId")
	if err != nil {
		http.Error(w, "userId must be an int", 400)
	}

	//Process queues
	PrintQueueHook()

	//Check the active prints and the queued prints
	for _, i := range PRINTERS {
		//Check that userid and printid match
		if printId == i.ActivePrint.PrintId && i.ActivePrint.UserId == userId {
			i.ActivePrint.Active = false
			fmt.Fprint(w, "Print cancelled")
			return
		}
	}
	//TODO
	//Check the queues
	//for printer_id, q := range QUEUES {
	//	for _, print := range q {
	//	}
	//}

	PrintQueueHook()
	http.Error(w, "Unable to find active print", 400)
}

//Run this to re-queue printers
func PrintQueueHook() {
	for i, printer := range PRINTERS {
		//check if any prints have finished
		if printer.ActivePrint.Active {
			print_time, err := time.Parse("2006-01-02T03:04:05.0000", printer.ActivePrint.StartTime)
			if err != nil {
				log.Println("Invalid timestamp in print ", err)
			}
			//If a print has finished clear it out
			if print_time.After(time.Now()) {
				//Clear out print

				p := PRINTERS[i]
				p.ActivePrint = Print{}
				log.Printf("print %d has finished\n", p.ActivePrint.PrintId)
				PRINTERS[i] = p
			}
		}

		//Grabe the next print from the queue
		if !printer.ActivePrint.Active {
			if len(QUEUES[i]) > 0 {
				//get the first element in the queue
				p := PRINTERS[i]
				p.ActivePrint = QUEUES[i][0]
				PRINTERS[i].ActivePrint = p.ActivePrint

				//remove the element from the queue
				QUEUES[i] = QUEUES[i][:1]
				log.Printf("moving print %d into printer %d", QUEUES[i][0].PrintId, printer.Id)
			}
		}
	}
}

func ListPrints(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	userId := query.Get("userId")

	active_prints := []Print{}
	for _, printer := range PRINTERS {
		if printer.ActivePrint.UserId == userId {
			active_prints = append(active_prints, printer.ActivePrint)
		}
	}

	out, err := json.Marshal(active_prints)
	if err != nil {
		log.Println("Unable to marshal printer:", err)
		http.Error(w, "Failure listing printers", 500)
		return
	}

	fmt.Fprint(w, string(out))
}
