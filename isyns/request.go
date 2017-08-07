package isyns

type Request interface {
	ID() string
	Complete(success bool) error
	requestSigil() request
}

type InstallRequest struct {
	request
}

type NodeQueryRequest struct {
	request
	NodeAddr string
}

type NodeStatusValuesRequest struct {
	request
	NodeAddr string
}

type AddAllNodesRequest struct {
	request
}

type AddNodeRequest struct {
	request
	NodeAddr    string
	NodeDefID   string
	PrimaryAddr string
	Name        string
}

type RemoveNodeRequest struct {
	request
	NodeAddr string
}

type RenameNodeRequest struct {
	request
	NodeAddr string
	Name     string
}

type EnableNodeRequest struct {
	request
	NodeAddr string
	Enabled  bool
}

type CommandRequest struct {
	request
	NodeAddr string
	Command  string
	Param    *CommandParam
	Params   map[string]CommandParam
}

type request struct {
	id     string
	server *Server
}

func (r request) ID() string {
	return r.id
}

func (r request) Complete(success bool) error {
	if r.id == "" {
		return nil
	}

	return r.server.client.ReportRequestStatus(r.id, success)
}

func (r request) requestSigil() request {
	return r
}
