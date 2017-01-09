//blockparty.go handles the webserver functions for the blockparty demo
package main

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/satori/go.uuid"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"reflect"
)

var (
	pool    *redis.Pool
	mainURL string
	store   *sessions.CookieStore = sessions.NewCookieStore([]byte("BlockParty"))
)

type cfServices struct {
	Services []cfService `json:"services"`
}

type cfService struct {
	Platform string `json:"platform"`
	Name     string `json:"service"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
}

//House that will be listed
type House struct {
	ID             string  `json:"id" redis:"id"`
	Name           string  `json:"name" redis:"name"`
	Owner          string  `json:"owner" redis:"owner"`
	Address        string  `json:"address" redis:"address"`
	Price          float64 `json:"price" redis:"price"`
	Image          string  `json:"image" redis:"image"`
	Contract       string  `json:"contract" redis:"contract"`
	Description    string  `json:"description" redis:"description"`
	Bedrooms       float64 `json:"bedrooms" redis:"bedrooms"`
	Bathrooms      float64 `json:"bathrooms" redis:"bathrooms"`
	Status         string  `json:"status" redis:"status"`
	Quality        int     `json:"quality" redis:"quality"`
	InspectionDate string  `json:"inspectionDate" redis:"inspectionDate"`
}

func newHouse() House {
	var h House
	h.ID = getNewHouseID()
	return h
}

// A Bid on a house
type Bid struct {
	User    string  `json:"user" redis:"user"`
	Amount  float64 `json:"amount" redis:"amount"`
	HouseID string  `json:"houseID" redis:"houseID"`
	Status  string  `json:"status" redis:"status"`
}
// A Mortgage on a house
type Mortgage struct {
	User    string  `json:"user" redis:"user"`
	Amount  float64 `json:"amount" redis:"amount"`
	HouseID string  `json:"houseID" redis:"houseID"`
	Lender  string  `json:"lender" redis:"lender"`
	Status  string  `json:"status" redis:"status"`
}

func (m Mortgage) getKey() string {
	return "mortgage:" + m.HouseID + ":" + m.User
}

func (b Bid) getKey() string {
	return "bid:" + b.HouseID + ":" + b.User
}

//Payload is a generic Container to hold data for templates
type Payload struct {
	Houses    []House    `json:"data" redis:"data"`
	Bids      []Bid      `json:"bids" redis:"bids"`
	Mortgages []Mortgage `json:"mortgages" redis:"mortgages"`
	User      string     `json:"user" redis:"user"`
	URL       string     `json:"URL" redis:"URL"`
	Parameters   []byte     `json:"parameters" redis:"parameters"`
}

// User is a user from ethereum
type User struct {
	Name    string `json:"name" redis:"name"`
	Address string `json:"address" redis:"address"`
}

// UserList is a list of ethereum users
type UserList struct {
	Data []User `json:"data" redis:"data"`
}

type SystemUsers struct {
	Seller string	`json:"seller" redis:"seller"`
	Lender string	`json:"lender" redis:"lender"`
	Inspector string `json:"inspector" redis:"inspector"`
}

func getSystemUsers() SystemUsers {
	var u SystemUsers

	c := pool.Get()
	defer c.Close()

	r,err := c.Do("HGETALL", "users")
	err = redis.ScanStruct(r.([]interface{}),&u)
	check("ScanStruct", err)


	f:=reflect.TypeOf(u)
	v:=reflect.ValueOf(u)
	for  i:=0; i < f.NumField(); i++ {
		log.Print(f.Field(i).Name + " " + v.Field(i).Interface().(string))
	}

	return u
}

func newPool(addr string, port string, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr+":"+port)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
}

func check(function string, e error) {
	if e != nil {
		log.Fatal(function, e)
	}
}

func newPayload() *Payload {
	return &Payload{URL: mainURL}
}

func getHouses() []House {
	var h House
	houses:= make([]House, 0)

	c := pool.Get()
	defer c.Close()

	n, err := redis.Strings(c.Do("KEYS", "house:*"))
	check("Keys", err)

	for _, v := range n {
		r, err := redis.Values(c.Do("HGETALL", v))
		err = redis.ScanStruct(r, &h)
		check("ScanStruct", err)
		houses = append(houses, House{ID: h.ID, Name: h.Name, Address: h.Address, Price: h.Price, Image: h.Image, Contract: h.Contract,
			Description: h.Description, Bedrooms: h.Bedrooms, Bathrooms: h.Bathrooms, Status: h.Status, Quality: h.Quality})
	}
	return houses
}

func setDefaultHouses() {
	c := pool.Get()
	defer c.Close()

	n, err := redis.Strings(c.Do("KEYS", "house:*"))
	check("KEYS", err)
	if len(n) == 0 {
		fmt.Println("Creating default houses.")
		file, err := ioutil.ReadFile("./houses.json")
		check("Read houses JSON", err)

		var listings Payload
		err = json.Unmarshal(file, &listings)
		check("Unmarshal", err)

		for _, v := range listings.Houses {
			h := newHouse()
			_, err = c.Do("HMSET", "house:"+h.ID, "id", h.ID, "name", v.Name, "address", v.Address, "price",
				v.Price, "image", v.Image, "contract", v.Contract, "description", v.Description, "bedrooms", v.Bedrooms, "bathrooms", v.Bathrooms, "status", v.Status, "quality", v.Quality, "inspectionDate", "")
			check("HMSET", err)
		}
	} else {
		fmt.Println("Default houses already exist. Skipping house creation.")
	}
}

func setDefaultUsers() {
	c := pool.Get()
	defer c.Close()

	var users UserList
	n, err := redis.Strings(c.Do("KEYS", "users"))
	check("KEYS", err)
	if len(n) == 0 {
		fmt.Println("Creating default users.")
		file, err := ioutil.ReadFile("./users.json")
		check("Read houses JSON", err)

		err = json.Unmarshal(file, &users)
		check("Unmarshal", err)
	}

	for _, v := range users.Data {
		_, err = c.Do("HSET", "users", v.Name, v.Address)
		check("HSET", err)
	}
}

func changeHouseStatus(i string, status string) error {
	c := pool.Get()
	defer c.Close()

	if status == "sold" {
		err := changeHousePrice(i, 0)
		check("ChangeHousePrice", err)
	}

	key := "house:" + i
	_, err := c.Do("HSET", key, "status", status)
	return err
}

func changeHousePrice(i string, price float64) error {
	c := pool.Get()
	defer c.Close()

	key := "house:" + i
	_, err := c.Do("HSET", key, "price", price)
	return err
}
func changeHouseQuality(i string, q int) error {
	c := pool.Get()
	defer c.Close()

	key := "house:" + i
	_, err := c.Do("HSET", key, "quality", q)
	return err
}
func changeBidStatus(i string, u string, status string) error {
	c := pool.Get()
	defer c.Close()

	key := "bid:" + i + ":" + u
	_, err := c.Do("HSET", key, "status", status)
	return err
}

func rejectOtherBids(i string, u string) error {
	var err error
	c := pool.Get()
	defer c.Close()

	bids := getBids(i + ":*")
	for _, v := range bids {
		if v.User != u {
			key := "bid:" + i + ":" + v.User
			_, err := c.Do("HSET", key, "status", "Rejected")
			check("HSET", err)
		}
	}
	return err

}

func changeMortgageStatus(i string, u string, status string) error {
	c := pool.Get()
	defer c.Close()

	key := "mortgage:" + i + ":" + u
	_, err := c.Do("HSET", key, "status", status)
	return err
}

func setInspectionDate(i string, d string) error {
	c := pool.Get()
	defer c.Close()

	key := "house:" + i
	_, err := c.Do("HSET", key, "inspectionDate", d)
	return err
}
func initialize() {
	var cfServices cfServices
	fmt.Println("Starting")
	file, err := ioutil.ReadFile("./services.json")
	check("Read services JSON", err)

	err = json.Unmarshal(file, &cfServices)
	check("Unmarshal", err)

	env, _ := cfenv.Current()
	mainURL = "http://" + env.ApplicationURIs[0]
	services := env.Services

	var credentials map[string]interface{}
	var host string
	var password string
	var port string

	for _, service := range cfServices.Services {
		if _, ok := services[service.Name]; ok {
			credentials = services[service.Name][0].Credentials
			if _, ok := credentials[service.Host]; ok {
				host = credentials[service.Host].(string)
			} else {
				log.Fatal("Unable to identify Redis host from config. Platform attempted:" + service.Platform)
			}
			if _, ok := credentials[service.Password]; ok {
				password = credentials[service.Password].(string)
			} else {
				log.Fatal("Unable to identify Redis password from config. Platform attempted:" + service.Platform)
			}
			if _, ok := credentials[service.Port]; ok {
				switch credentials[service.Port].(type) {
				case string:
					port = credentials[service.Port].(string)
				case float64:
					port = strconv.FormatFloat(credentials[service.Port].(float64), 'f', -1, 64)
				default:
					log.Fatal("Redis port value is of unexpected type.")
				}
			} else {
				log.Fatal("Unable to identify Redis port from config. Platform attempted:" + service.Platform)
			}
			break
		}
	}

	pool = newPool(host, port, password)
	setDefaultHouses()
	setDefaultUsers()
}

func getNewHouseID() string {
	return uuid.NewV4().String()
}

func getUserID() string {
	return uuid.NewV4().String()
}

func getHouse(i string) House {
	var h House
	c := pool.Get()
	defer c.Close()

	n, err := redis.Values(c.Do("HGETALL", "house:"+i))
	check("HGETALL", err)
	err = redis.ScanStruct(n, &h)
	check("ScanStruct", err)
	return h
}

func getBid(ID string) Bid {
	var bid Bid
	c := pool.Get()
	defer c.Close()

	n, err := redis.Values(c.Do("HGETALL", ID))
	check("HGETALL", err)
	err = redis.ScanStruct(n, &bid)
	check("ScanStruct", err)
	return bid
}

func getBids(filter string) []Bid {
	bids:= make([]Bid, 0)

	c := pool.Get()
	defer c.Close()

	bidKeys, err := redis.Strings(c.Do("KEYS", "bid:"+filter))
	check("Strings", err)

	for _, v := range bidKeys {
		bids = append(bids, getBid(v))
	}

	return bids
}

func getHouseBids(i string) []Bid {
	return getBids(i + ":*")
}

func getMyBids(u string) []Bid {
	return getBids("*:" + u)
}

func getMyBid(i string, u string) Bid {
	return getBid("bid:" + i + ":" + u)

}

func getMortgage(ID string) Mortgage {
	var mortgage Mortgage
	c := pool.Get()
	defer c.Close()

	n, err := redis.Values(c.Do("HGETALL", ID))
	check("HGETALL", err)
	err = redis.ScanStruct(n, &mortgage)
	check("ScanStruct", err)
	return mortgage
}

func getMortgages(filter string) []Mortgage {
	mortgages:= make([]Mortgage, 0)

	c := pool.Get()
	defer c.Close()

	mortgageKeys, err := redis.Strings(c.Do("KEYS", "mortgage:"+filter))
	check("Strings", err)

	for _, v := range mortgageKeys {
		mortgages = append(mortgages, getMortgage(v))
	}

	return mortgages
}

func getMyMortgages(u string) []Mortgage {
	return getMortgages("*:" + u)
}

func listingsHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["user"]; ok {
		u = session.Values["user"].(string)
	} else {
		u = getUserID()
		session.Values["user"] = u
	}

	session.Save(r, w)
	t, err := template.ParseFiles("templates/listings.tmpl", "templates/housePanel.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	listings := newPayload()
	listings.Houses = getHouses()
	listings.User = u
	check("Marshal",err)
	t.Execute(w, listings)
}

func detailsHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	c := pool.Get()
	defer c.Close()

	var h House
	h = getHouse(i)

	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t, err := template.ParseFiles("templates/details.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)
}

func listHouseHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	c := pool.Get()
	defer c.Close()

	var h House
	h = getHouse(i)

	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t, err := template.ParseFiles("templates/listHouse.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)
}

func bidsHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	vars := mux.Vars(r)
	i := vars["houseID"]

	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	payload := newPayload()
	payload.Bids = getHouseBids(i)
	payload.User = u
	t, err := template.ParseFiles("templates/bids.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)

}

func enterBidHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]
	a := r.PostFormValue("bidAmount")
	a = strings.Replace(a, ",", "", -1)
	a = strings.Replace(a, ".", "", -1)

	var h House
	h = getHouse(i)

	c := pool.Get()
	defer c.Close()

	key := "bid:" + i + ":" + u

	_, err = c.Do("HMSET", key, "user", u, "amount", a, "houseID", i, "status", "Submitted")
	check("HMSET", err)

	var payload = newPayload()
	payload.User = u
	payload.Bids = append(payload.Bids,getBid(key))
	payload.Houses = append(payload.Houses, h)

	t, err := template.ParseFiles("templates/myBid.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)
}

func enterListingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	a := r.PostFormValue("askingPrice")
	a = strings.Replace(a, ",", "", -1)
	a = strings.Replace(a, ".", "", -1)

	ap, err := strconv.ParseFloat(a, 64)
	err = changeHousePrice(i, ap)
	check("changeHousePrice", err)
	err = changeHouseStatus(i, "listed")
	check("changeHouseStatus", err)

	http.Redirect(w, r, mainURL+"/seller", http.StatusFound)
}

func enterMortgageHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]
	a := r.PostFormValue("amount")
	a = strings.Replace(a, ",", "", -1)
	a = strings.Replace(a, ".", "", -1)
	l := r.PostFormValue("lender")

	c := pool.Get()
	defer c.Close()

	key := "mortgage:" + i + ":" + u

	_, err = c.Do("HMSET", key, "user", u, "amount", a, "houseID", i, "lender", l, "status", "Submitted")
	check("HMSET", err)
	http.Redirect(w, r, mainURL+"/myMortgages", http.StatusFound)
}

func applyHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	var h House
	h = getHouse(i)
	t, err := template.ParseFiles("templates/apply.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t.Execute(w, payload)
}

func mortgageHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	var h House
	h = getHouse(i)
	t, err := template.ParseFiles("templates/mortgage.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t.Execute(w, payload)
}

func myBidHandler(w http.ResponseWriter, r *http.Request) {
	var u string

	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]
	myBid := getMyBid(i, u)
	house := getHouse(i)

	t, err := template.ParseFiles("templates/myBid.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = append(payload.Houses, house)
	payload.User = u
	payload.Bids = append(payload.Bids, myBid)
	t.Execute(w, payload)
}

func inspectHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	var h House
	h = getHouse(i)
	t, err := template.ParseFiles("templates/inspect.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t.Execute(w, payload)
}

func lenderHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	t, err := template.ParseFiles("templates/lender.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Mortgages = getMortgages("*")
	payload.User = u
	t.Execute(w, payload)
}

func scheduleInspectionHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	vars := mux.Vars(r)
	i := vars["houseID"]

	var h House
	h = getHouse(i)
	err = changeHouseQuality(i, 0)
	check("changeHouseQuality", err)
	t, err := template.ParseFiles("templates/scheduleInspection.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = append(payload.Houses, h)
	payload.User = u
	t.Execute(w, payload)
}

func enterInspectionAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	d := r.PostFormValue("date")

	err := setInspectionDate(i, d)
	check("setInspectionDate", err)
	http.Redirect(w, r, mainURL+"/myMortgages", http.StatusFound)
}

func changeHouseStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	s := vars["status"]

	err := changeHouseStatus(i, s)
	check("changeHouseStatus", err)
	http.Redirect(w, r, mainURL+"/seller", http.StatusFound)
}

func changeHouseQualityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	q, err := strconv.Atoi(vars["quality"])

	err = changeHouseQuality(i, q)
	check("changeHouseStatus", err)
	http.Redirect(w, r, mainURL+"/inspector", http.StatusFound)
}

func changeBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	u := vars["user"]
	s := vars["status"]

	err := changeBidStatus(i, u, s)
	check("changeBidStatus", err)

	if s == "Accepted" {
		err := rejectOtherBids(i, u)
		check("rejectOtherBids", err)
		err = changeHouseStatus(i, "sold")
		check("changeHouseStatus", err)
	}
	http.Redirect(w, r, mainURL+"/seller", http.StatusFound)
}

func changeMortgageStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	u := vars["user"]
	s := vars["status"]

	err := changeMortgageStatus(i, u, s)
	check("changeMortgageStatus", err)

	http.Redirect(w, r, mainURL+"/lender", http.StatusFound)
}

func checkBidStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	u := vars["user"]

	key := "bid:" + i + ":" + u
	bid := getBid(key)
	response, err := json.Marshal(bid)
	check("Marshal", err)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func checkMortgageStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := vars["houseID"]
	u := vars["user"]

	key := "mortgage" + i + ":" + u
	mortgage := getMortgage(key)
	response, err := json.Marshal(mortgage)
	check("Marshal", err)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func inspectorHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	t, err := template.ParseFiles("templates/inspector.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	for _, h := range getHouses() {
		if h.Quality == 0 {
			payload.Houses = append(payload.Houses, h)
		}
	}
	payload.User = u
	t.Execute(w, payload)
}
func sellerHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	t, err := template.ParseFiles("templates/seller.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	var payload = newPayload()
	payload.Houses = getHouses()
	payload.User = u
	t.Execute(w, payload)
}

func myBidsHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	myBids := getMyBids(u)
	var payload = newPayload()
	payload.Bids = myBids
	payload.User = u

	t, err := template.ParseFiles("templates/myBids.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)
}

func myMortgagesHandler(w http.ResponseWriter, r *http.Request) {
	var u string
	session, err := store.Get(r, "BlockPartySession")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u = session.Values["user"].(string)

	myMortgages := getMyMortgages(u)
	var payload = newPayload()
	payload.Mortgages = myMortgages
	payload.User = u

	for _, m := range myMortgages {
		payload.Houses = append(payload.Houses, getHouse(m.HouseID))
	}
	t, err := template.ParseFiles("templates/myMortgages.tmpl", "templates/head.tmpl", "templates/navbar.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)
}

func missingRequirementsHandler(w http.ResponseWriter, r *http.Request) {
	payload := newPayload()
	t, err := template.ParseFiles("templates/missingRequirements.tmpl")
	check("Parse template", err)
	t.Execute(w, payload)

}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	keys:= make([]string, 0)
	keys = append(keys, "house:", "bid:", "mortgage:", "users")

	c := pool.Get()
	defer c.Close()

	for _, k := range keys {
		n, err := redis.Strings(c.Do("KEYS", k+"*"))
		check("Keys", err)
		for _, v := range n {
			_, err := c.Do("DEL", v)
			check("DEL", err)
			log.Print("Deleted key: " + v)
		}
	}

	setDefaultHouses()
	setDefaultUsers()
	http.Redirect(w, r, mainURL, http.StatusFound)
}

func main() {
	initialize()
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", listingsHandler)
	router.HandleFunc("/seller", sellerHandler)
	router.HandleFunc("/house/{houseID}", detailsHandler)
	router.HandleFunc("/house/{houseID}/listHouse", listHouseHandler)
	router.HandleFunc("/house/{houseID}/enterListing", enterListingHandler)
	router.HandleFunc("/house/{houseID}/enterBid", enterBidHandler)
	router.HandleFunc("/house/{houseID}/myBid", myBidHandler)
	router.HandleFunc("/myBids", myBidsHandler)
	router.HandleFunc("/house/{houseID}/bids", bidsHandler)
	router.HandleFunc("/lender", lenderHandler)
	router.HandleFunc("/house/{houseID}/applyForMortgage", applyHandler)
	router.HandleFunc("/house/{houseID}/enterMortgage", enterMortgageHandler)
	router.HandleFunc("/house/{houseID}/mortgage", mortgageHandler)
	router.HandleFunc("/house/{houseID}/mortgage/{user}/changeStatus/{status}", changeMortgageStatusHandler)
	router.HandleFunc("/house/{houseID}/mortgage/{user}/checkStatus", checkMortgageStatusHandler)
	router.HandleFunc("/house/{houseID}/inspect", inspectHandler)
	router.HandleFunc("/myMortgages", myMortgagesHandler)
	router.HandleFunc("/inspector", inspectorHandler)
	router.HandleFunc("/house/{houseID}/scheduleInspection", scheduleInspectionHandler)
	router.HandleFunc("/house/{houseID}/enterInspectionAppointment", enterInspectionAppointmentHandler)
	router.HandleFunc("/house/{houseID}/changeStatus/{status}", changeHouseStatusHandler)
	router.HandleFunc("/house/{houseID}/changeQuality/{quality}", changeHouseQualityHandler)
	router.HandleFunc("/house/{houseID}/bid/{user}/changeStatus/{status}", changeBidStatusHandler)
	router.HandleFunc("/house/{houseID}/bid/{user}/checkStatus", checkBidStatusHandler)
	router.HandleFunc("/missingRequirements", missingRequirementsHandler)
	router.HandleFunc("/reset", resetHandler)

	http.Handle("/images/", http.FileServer(http.Dir("/app")))
	http.Handle("/css/", http.FileServer(http.Dir("/app")))
	http.Handle("/fonts/", http.FileServer(http.Dir("/app")))
	http.Handle("/js/", http.FileServer(http.Dir("/app")))
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
