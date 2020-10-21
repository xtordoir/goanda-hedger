package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "hedge/hedge"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/xtordoir/goanda/api"
	"github.com/xtordoir/goanda/models"
)

type inventoryApp struct {
	Manageable
}

func (app *inventoryApp) addStaticHedge(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var staticHedge StaticHedge
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&staticHedge)
		// deal with error here
		if err != nil {
			http.Error(w, "Cannot parse a StaticHedge", 500)
			return
		}
		// Add new Hedge to Inventory
		app.Manageable.AddHedge(&staticHedge)
	}
}

func (app *inventoryApp) addDynamicHedge(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var dynamicHedge DynamicHedge
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&dynamicHedge)
		// deal with error here
		if err != nil {
			http.Error(w, "Cannot parse a DynamicHedge", 500)
			return
		}
		// Add new Hedge to Inventory
		app.Manageable.AddHedge(&dynamicHedge)
	}
}

func (app *inventoryApp) getHedges(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		hedges := app.Manageable.ListHedges()
		b, errm := json.Marshal(hedges)
		if errm != nil {
			log.Printf("%s\n", errm)
		}
		fmt.Fprintln(w, string(b))
		//fmt.Fprint(w, manager.CumProfit())
	}
}

// Type used to parse the statefile to recover the Hedges Inventory from
type hedgeState struct {
	// static parameters of the Hedge
	Price0 *float64
	Size0  *int64
	Scale  *float64

	// state of the Hedge
	LengthUp   *int64
	LengthDown *int64
	BoxUp      *int64
	BoxDown    *int64
	//I          int64
	//LastCrossI int64
	Size *int64
}

func parseState(s io.Reader) []Tradeable {

	ret := make([]Tradeable, 0)

	hedgeState := make([]hedgeState, 0)
	json.NewDecoder(s).Decode(&hedgeState)
	for _, hedge := range hedgeState {
		// if StaticHedge
		if hedge.Price0 == nil {
			ret = append(ret, &StaticHedge{Size: *hedge.Size})
		} else {
			// DynamicHedge
			ret = append(ret, &DynamicHedge{
				Price0:     *hedge.Price0,
				Size0:      *hedge.Size0,
				Scale:      *hedge.Scale,
				LengthUp:   *hedge.LengthUp,
				LengthDown: *hedge.LengthDown,
				BoxUp:      *hedge.BoxUp,
				BoxDown:    *hedge.BoxDown,
				Size:       *hedge.Size})
		}
	}

	return ret
}

type instrumentPosition struct {
	Instrument string
	Position   int64
}

// sink for Hearbeats does nothing
func runHeartbeats(hChan chan models.PricingHeartbeat) {
	for {
		hb := <-hChan
		fmt.Printf("%+v\n", hb)
	}
}

// async function to comnpute positions
func runPositions(manager Tradeable,
	priceChan chan models.Tick, positionChan chan instrumentPosition) {
	for {
		price := <-priceChan
		position := instrumentPosition{price.Instrument, manager.PositionSize(price.Price())}
		positionChan <- position
	}
}

// async function to execute trades to correct position
func runOrders(api *api.API, positionChan chan instrumentPosition) {
	for {
		nextExposure := <-positionChan
		positions, err := api.GetPosition(nextExposure.Instrument)
		if err != nil {
			continue
		}
		var current int64
		current = 0
		current += positions.Position.Long.Units
		current += positions.Position.Short.Units

		units := nextExposure.Position - current
		if units != 0 {
			fmt.Printf("Trading %d Units\n", units)
			api.PostMarketOrder(nextExposure.Instrument, units)
		}
	}
}

func main() {

	var statefile string
	flag.StringVar(&statefile, "statefile", "", "state of hedges to initiate portfolio")

	flag.Parse()

	_inventory := make(Inventory, 0)
	inv := &InventoryManager{Inventory: &_inventory}
	var manager Manageable
	manager = inv

	if statefile != "" {
		s, err := os.Open(statefile)
		hedges := parseState(s)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		s.Close()
		for _, h := range hedges {
			manager.AddHedge(h)
		}
	}

	// Active the broker Client to Run orders as required

	// the oanda API
	ctx := api.Context{
		Token:        os.Getenv("OANDA_API_KEY"),
		Account:      os.Getenv("OANDA_ACCOUNT"),
		ApiURL:       os.Getenv("OANDA_API_URL"),
		StreamApiURL: os.Getenv("OANDA_STREAM_URL"),
		Application:  "Goapp",
	}

	// create the oanda api and channel for orders
	oanda := ctx.CreateAPI()
	stream := ctx.CreateStreamAPI()

	positionChan := make(chan instrumentPosition)
	tickChan := make(chan models.Tick)
	hChan := make(chan models.PricingHeartbeat)

	go runHeartbeats(hChan)

	go stream.TickStream([]string{"EUR_USD"}, tickChan, hChan)
	// start the orders loop
	go runOrders(&oanda, positionChan)

	var trader Tradeable
	trader = inv

	go runPositions(trader, tickChan, positionChan)

	app := inventoryApp{manager}

	mux := http.NewServeMux()

	mux.HandleFunc("/hedge/dynamic", app.addDynamicHedge)
	mux.HandleFunc("/hedge/static", app.addStaticHedge)
	mux.HandleFunc("/hedge", app.getHedges)

	fmt.Println("Server starting...")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}

}
