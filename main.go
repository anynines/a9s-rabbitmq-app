package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// struct for reading env
type ServiceInstance struct {
	Name        string      `json:"name"`
	Tags        []string    `json:"tags"`
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	Port     int    `json:"port"`
	URI      string `json:"uri"`
	Username string `json:"username"`
	Cacrt    string `json:"cacrt,omitempty"`
}

type AppServer struct {
	mqConn *amqp.Connection
	// the map which stores the messages
	messageStore sync.Map
	// template store
	templates        map[string]*template.Template
	publicDirHandler http.Handler
}

// Message struct to store in the Map
type Message struct {
	Message    string
	ReceivedAt time.Time
}

func NewAppServer() *AppServer {

	serviceInstance := getServiceInstance()
	conn, err := serviceInstance.Credentials.amqpDial()
	if err != nil {
		log.Fatal(err)
	}

	return &AppServer{
		mqConn:           conn,
		templates:        loadTemplates(),
		publicDirHandler: publicDirHandler(),
	}
}

func (a *AppServer) Close() {
	a.mqConn.Close()
}

func publicDirHandler() http.Handler {

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	return http.StripPrefix("/public/", http.FileServer(http.Dir(path.Join(dir, "public"))))
}

func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template, 0)
	templates["index"] = template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	templates["new"] = template.Must(template.ParseFiles("templates/new.html", "templates/base.html"))
	return templates
}

func getServiceInstance() ServiceInstance {
	vcapServices := os.Getenv("VCAP_SERVICES")
	if len(vcapServices) == 0 {
		log.Fatalln("VCAP_SERVICES env variable must not be empty")
	}

	var services map[string][]ServiceInstance

	if err := json.Unmarshal([]byte(vcapServices), &services); err != nil {
		log.Fatal(err)
	}

	var serviceInstance *ServiceInstance
	instanceName := os.Getenv("SERVICE_INSTANCE_NAME")
	if len(instanceName) == 0 {
		eachServiceInstance(services, func(si ServiceInstance) bool {
			if contains(si.Tags, "rabbitmq") {
				serviceInstance = &si
				return true
			}
			return false
		})
	} else {
		eachServiceInstance(services, func(si ServiceInstance) bool {
			if si.Name == instanceName {
				serviceInstance = &si
				return true
			}
			return false
		})
	}

	if serviceInstance == nil {
		log.Fatalln("no valid service instance was found; specify SERVICE_INSTANCE_NAME or ensure \"rabbitmq\" tag is present")
	}

	return *serviceInstance
}

func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

func eachServiceInstance(s map[string][]ServiceInstance, f func(si ServiceInstance) bool) {
	for _, serviceInstances := range s {
		for _, serviceInstance := range serviceInstances {
			if f(serviceInstance) {
				return
			}
		}
	}
}

func (a *AppServer) renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, _ := a.templates[name]
	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (c *Credentials) amqpDial() (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error
	if len(c.Cacrt) == 0 {
		conn, err = amqp.Dial(c.URI)
	} else {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(c.Cacrt)) {
			return nil, errors.New("CA certificate could not be parsed")
		}

		conn, err = amqp.DialTLS(c.URI, &tls.Config{RootCAs: certPool})
	}

	return conn, err
}

// the receiver for the RabbitMQ
func (a *AppServer) startReceiver() error {

	ch, err := a.mqConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("test-app", false, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	var id int
	for d := range msgs {
		message := Message{string(d.Body), time.Now()}

		id++
		k := strconv.Itoa(id)
		a.messageStore.Store(k, message)
	}

	return nil
}

// send message to a RabbitMQ queue
func (a *AppServer) sendMessage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	ch, err := a.mqConn.Channel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("test-app", false, false, false, false, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ch.Publish("", q.Name, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(r.PostFormValue("message")),
		})

	http.Redirect(w, r, "/", 302)
}

func (a *AppServer) newMessage(w http.ResponseWriter, r *http.Request) {
	a.renderTemplate(w, "new", "base", nil)
}

func (a *AppServer) getMessages(w http.ResponseWriter, r *http.Request) {
	storeView := make(map[string]Message, 0)
	a.messageStore.Range(func(key, value interface{}) bool {
		storeView[key.(string)] = value.(Message)
		return true
	})

	a.renderTemplate(w, "index", "base", storeView)
}

func (a *AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		a.getMessages(w, r)
	case r.URL.Path == "/messages/new":
		a.newMessage(w, r)
	case r.URL.Path == "/messages/send":
		a.sendMessage(w, r)
	case strings.HasPrefix(r.URL.Path, "/public/"):
		a.publicDirHandler.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	appServer := NewAppServer()
	defer appServer.Close()
	http.Handle("/", appServer)

	go appServer.startReceiver()

	log.Printf("Listening on port %v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
