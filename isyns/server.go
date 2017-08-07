package isyns

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/apparentlymart/go-isy/isy"
	"github.com/gorilla/mux"
)

// Server is the main type in this package, representing a single node server.
//
// After creating a Server using NewServer, call either ListenAndServe or Serve
// in a separate goroutine and then read the channel Requests until it is
// closed, indicating a shutdown.
//
//    s, err := isyns.NewServer(config, profileNum, isyConfig)
//    // (handle possible error in "err")
//
//    serverErr := make(chan error)
//    go func() {
//        serverErr <- s.ListenAndServe()
//        close(serverErr)
//    }()
//
//    Events:
//    for {
//        select {
//        case err := <-serverErr:
//            log.Printf("error: %s", err)
//            break Events
//        case req, ok := <-s.Requests:
//            // handle "req" e.g. with a type switch
//            if !ok {
//                break Events
//            }
//
//        // (also handle events for whichever external system the node server is representing)
//
//        }
//    }

var router *mux.Router

type Server struct {
	Requests       <-chan Request
	rawReqs        chan Request
	client         nsClient
	httpServer     *http.Server
	username       string
	passwordSHA256 []byte
}

type Config struct {
	ListenAddr string
	TLSConfig  *tls.Config
	ErrorLog   *log.Logger

	// Credentials used for the ISY to authenticate to the node server
	Username string
	Password string
}

func NewServer(config *Config, profileNum int, isyConfig *isy.ClientConfig) (*Server, error) {
	relPath := path.Join("rest", "ns", strconv.Itoa(profileNum))
	relURL, err := url.Parse(relPath)
	if err != nil {
		// should never happen
		panic("failed to parse self-generated service relative path")
	}

	baseURL, err := url.Parse(isyConfig.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ISY base URL: %s", err)
	}

	hs := &http.Server{
		Addr:      config.ListenAddr,
		TLSConfig: config.TLSConfig,
		ErrorLog:  config.ErrorLog,
	}

	s := &Server{}
	s.rawReqs = make(chan Request)
	s.Requests = s.rawReqs // read-only version for public consumption
	s.httpServer = hs
	s.username = config.Username
	passwordSHA256 := sha256.Sum256([]byte(config.Password))
	s.passwordSHA256 = passwordSHA256[:]

	hs.Handler = http.HandlerFunc(s.handler)

	s.client = nsClient{
		BaseURL:  baseURL.ResolveReference(relURL),
		Username: isyConfig.Username,
		Password: isyConfig.Password,
	}

	return s, nil
}

func (s *Server) Serve(l net.Listener) error {
	return s.httpServer.Serve(l)
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	username, password, authed := r.BasicAuth()
	if !authed {
		http.Error(w, "Unauthorized", 401)
		return
	}
	if username != s.username {
		http.Error(w, "Unauthorized", 401)
		return
	}
	passwordSHA256 := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(passwordSHA256[:], s.passwordSHA256) != 1 {
		http.Error(w, "Unauthorized", 401)
		return
	}

	match := mux.RouteMatch{}
	matched := router.Match(r, &match)
	if !matched {
		http.Error(w, "Not Found", 404)
		return
	}

	var req Request
	switch match.Route.GetName() {
	case "install":
		req = &InstallRequest{
			request: s.makeCommonReq(r),
		}
	case "nodeQuery":
		req = &NodeQueryRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
		}
	case "nodeStatus":
		req = &NodeStatusValuesRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
		}
	case "addAllNodes":
		req = &AddAllNodesRequest{
			request: s.makeCommonReq(r),
		}
	case "addNode":
		req = &AddNodeRequest{
			request:     s.makeCommonReq(r),
			NodeAddr:    match.Vars["nodeAddr"],
			NodeDefID:   match.Vars["nodeDefId"],
			PrimaryAddr: r.URL.Query().Get("primary"),
			Name:        r.URL.Query().Get("name"),
		}
	case "removeNode":
		req = &RemoveNodeRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
		}
	case "renameNode":
		req = &RenameNodeRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Name:     r.URL.Query().Get("name"),
		}
	case "enableNode":
		req = &EnableNodeRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Enabled:  true,
		}
	case "disableNode":
		req = &EnableNodeRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Enabled:  false,
		}
	case "nodeCommand":
		req = &CommandRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Command:  match.Vars["command"],
			Params:   s.makeCommandParams(r),
		}
	case "nodeCommandValue":
		req = &CommandRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Command:  match.Vars["command"],
			Param: &CommandParam{
				Value: match.Vars["value"],
			},
			Params: s.makeCommandParams(r),
		}
	case "nodeCommandValueUnit":
		unit, err := strconv.Atoi(match.Vars["unit"])
		if err != nil {
			// ignore invalid request
			break
		}
		req = &CommandRequest{
			request:  s.makeCommonReq(r),
			NodeAddr: match.Vars["nodeAddr"],
			Command:  match.Vars["command"],
			Param: &CommandParam{
				Value: match.Vars["value"],
				UOM:   isy.UOM(unit),
			},
			Params: s.makeCommandParams(r),
		}
	}

	if req == nil {
		http.Error(w, "Not Found", 404)
		return
	}

	// The ISY protocol calls for us to return immediately if we recognize
	// the request, and then deal with the request contents asynchronously.
	w.WriteHeader(http.StatusNoContent)

	s.rawReqs <- req
}

func (s *Server) makeCommonReq(r *http.Request) request {
	rid := r.URL.Query().Get("requestId")
	return request{
		id:     rid,
		server: s,
	}
}

func (s *Server) makeCommandParams(r *http.Request) map[string]CommandParam {
	var ret map[string]CommandParam
	for k, vs := range r.URL.Query() {
		if k == "requestId" {
			continue
		}
		if len(vs) == 0 {
			continue
		}

		var name string
		var uom isy.UOM
		if splitPos := strings.Index(k, ".uom"); splitPos != -1 {
			namePrefix := k[0:splitPos]
			uomStr := k[splitPos+4:]
			unit, err := strconv.Atoi(uomStr)
			if err != nil {
				uom = isy.UOMUnknown
				name = k
			} else {
				uom = isy.UOM(unit)
				name = namePrefix
			}
		} else {
			uom = isy.UOMUnknown
			name = k
		}

		if ret == nil {
			ret = make(map[string]CommandParam)
		}
		ret[name] = CommandParam{
			Value: vs[0],
			UOM:   uom,
		}
	}
	return ret
}

type nsClient struct {
	BaseURL  *url.URL
	Username string
	Password string
}

func (c *nsClient) Request(url *url.URL) error {
	url = c.BaseURL.ResolveReference(url)
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}
	return nil
}

func (c *nsClient) MakeURL(parts ...string) *url.URL {
	for i, raw := range parts {
		parts[i] = url.PathEscape(raw)
	}

	rel := path.Join(parts...)
	relURL, _ := url.Parse(rel)
	return c.BaseURL.ResolveReference(relURL)
}

func (c *nsClient) ReportRequestStatus(id string, success bool) error {
	var url *url.URL
	if success {
		url = c.MakeURL("report", "status", id, "success")
	} else {
		url = c.MakeURL("report", "status", id, "fail")
	}
	return c.Request(url)
}

func init() {
	router = mux.NewRouter()
	router.Path("/install/{profileNum}").Name("install")
	router.Path("/nodes/{nodeAddr}/query").Name("nodeQuery")
	router.Path("/nodes/{nodeAddr}/status").Name("nodeStatus")
	router.Path("/add/nodes").Name("addAllNodes")
	router.Path("/nodes/{nodeAddr}/report/add/{nodeDefId}").Name("addNode")
	router.Path("/nodes/{nodeAddr}/report/remove").Name("removeNode")
	router.Path("/nodes/{nodeAddr}/report/rename").Name("renameNode")
	router.Path("/nodes/{nodeAddr}/report/enable").Name("enableNode")
	router.Path("/nodes/{nodeAddr}/report/disable").Name("disableNode")
	router.Path("/nodes/{nodeAddr}/cmd/{command}").Name("nodeCommand")
	router.Path("/nodes/{nodeAddr}/cmd/{command}/{value}").Name("nodeCommandValue")
	router.Path("/nodes/{nodeAddr}/cmd/{command}/{value}/{unit}").Name("nodeCommandValueUnit")
}
