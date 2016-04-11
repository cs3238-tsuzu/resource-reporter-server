package main

import "net/http"
import "fmt"
import "encoding/json"

type ResourceInfo struct {
	MEM string `json:"mem"`
	CPU string `json:"cpu"`
	CPUName string `json:"cpu_name"`
	Host string `json:"host"`
	Conn string `json:"conn"`
}

type RIUpdate struct {
	resource *ResourceInfo
	id string
	res chan<- bool
}

type IdenManager struct {
	updateInfo chan RIUpdate
	addIden chan string
	exit chan bool
	req chan struct{c chan map[string]ResourceInfo;}
	
	info map[string]ResourceInfo
	
}

func (im *IdenManager) Run() {
	for {
		select {
			case ui := <-im.updateInfo:
				_, err := im.info[ui.id]
				
				if !err {
					ui.res<-false
				}else {
					im.info[ui.id] = *ui.resource
					
					ui.res<-true
				}
			case id := <-im.addIden:
				im.info[id] = ResourceInfo{}
				
			case req := <-im.req:
				req.c<-im.info
			
			case <-im.exit:
				return
		}
	}
}

func (im *IdenManager) AddIdentity(name string) {
	im.addIden<-name
}

func (im *IdenManager) Exit() {
	im.exit<-true
}

func (im *IdenManager) RequestAll() string {
	c := make(chan map[string]ResourceInfo)
	
	im.req <- struct{c chan map[string]ResourceInfo;}{c}
	
	ri := <-c
	
	j, _ := json.Marshal(ri)
	
	return string(j)
}

func (im *IdenManager) UpdateInfo(info ResourceInfo, id string) bool {
	c := make(chan bool)
	
	upd := RIUpdate{&info, id, c}
	
	im.updateInfo <- upd

	return <-c
}

func NewIdenManager() *IdenManager {
	im := IdenManager{}
	
	im.addIden = make(chan string)
	im.exit = make(chan bool)
	im.req = make(chan struct{c chan map[string]ResourceInfo;})
	im.updateInfo = make(chan RIUpdate)
	im.info = make(map[string]ResourceInfo)
	
	return &im
}

func main() {
	im := NewIdenManager()
	
	go im.Run()
	
	mux := http.NewServeMux()
	
	mux.HandleFunc(
		"/",
		func(rw http.ResponseWriter, req *http.Request){
			err := req.ParseForm()
			
			if err != nil || req.URL.Path != "/" {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("400 Bad Request"))
				
				return
			}

			if _, b := req.Header["Rr-Identity"]; !b {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("400 Bad Request"))
				
				return				
			}
			
			ri := ResourceInfo{req.Form["mem"][0], req.Form["cpu"][0], req.Form["cpu_name"][0], req.Form["host"][0], req.Form["conn"][0]}
			
			im.UpdateInfo(ri, req.Header["Rr-Identity"][0])
			
			
		})
		
	mux.HandleFunc(
		"/addIdentity",
		func(rw http.ResponseWriter, req *http.Request) {
			req.ParseForm()
			
			name, err := req.Form["name"]
			
			if !err {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("400 Bad Request"))

				return
			}
			
			im.AddIdentity(name[0])
			
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("200 OK"))
		})
		
	mux.HandleFunc(
		"/requestAll",
		func(rw http.ResponseWriter, req *http.Request) {
			j := im.RequestAll()
			
			rw.Header().Set("Content-Type", "application/json")
			rw.Header().Set("Content-Length", fmt.Sprint(len(j)))
			
			rw.WriteHeader(http.StatusOK)
			
			rw.Write([]byte(j))
		})
	
    server := &http.Server{
		Addr: ":34567",
		Handler: mux,
	}
    
    err := server.ListenAndServe()
    
    if err != nil {
		fmt.Println(err)
	}
}