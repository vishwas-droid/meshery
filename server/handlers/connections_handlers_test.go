package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	"github.com/meshery/meshery/server/models"
	"github.com/meshery/meshery/server/models/connections"

	"github.com/meshery/meshkit/logger"
	"github.com/meshery/meshkit/models/events"

	"github.com/meshery/meshery/server/machines"
	"github.com/meshery/schemas/models/core"
)

func newConnectionsTestHandler(t *testing.T) *Handler {
	log, err := logger.New("test", logger.Options{})
	if err != nil {
		t.Fatal(err)
	}

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	return &Handler{
		log:      log,
		SystemID: &id,
	}
}

type connectionSpyProvider struct {
	*models.DefaultLocalProvider

	saveCalled   atomic.Bool
	deleteCalled atomic.Bool
	updateCalled atomic.Bool
}

func newConnectionSpyProvider() *connectionSpyProvider {
	base := &models.DefaultLocalProvider{}
	base.Initialize()

	return &connectionSpyProvider{
		DefaultLocalProvider: base,
	}
}

func (p *connectionSpyProvider) GetProviderToken(_ *http.Request) (string, error) {
	return "token", nil
}

func (p *connectionSpyProvider) PersistEvent(_ events.Event, _ string) error {
	return nil
}

func (p *connectionSpyProvider) SaveConnection(
	conn *connections.ConnectionPayload,
	token string,
	skip bool,
) (*connections.Connection, error) {

	p.saveCalled.Store(true)

	return &connections.Connection{
		Name: conn.Name,
	}, nil
}

func (p *connectionSpyProvider) DeleteConnection(
	_ *http.Request,
	_ core.Uuid,
) (*connections.Connection, error) {

	p.deleteCalled.Store(true)

	return &connections.Connection{
		Name: "deleted",
	}, nil
}

func (p *connectionSpyProvider) UpdateConnectionById(
	token string,
	conn *connections.ConnectionPayload,
	id string,
) (*connections.Connection, error) {

	p.updateCalled.Store(true)

	return &connections.Connection{
		Name: "updated",
	}, nil
}

func (p *connectionSpyProvider) GetK8sContext(
	token string,
	id string,
) (models.K8sContext, error) {

	return models.K8sContext{
		Name: "cluster",
	}, nil
}
func TestSaveConnection_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	provider := newConnectionSpyProvider()

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(`{"name":"demo"}`),
	)

	req = req.WithContext(
		context.WithValue(
			req.Context(),
			models.TokenCtxKey,
			"token",
		),
	)

	rr := httptest.NewRecorder()

	defer func() { _ = recover() }()

	h.SaveConnection(
		rr,
		req,
		nil,
		&models.User{
			ID: uuid.Must(uuid.NewV4()),
		},
		provider,
	)

	if !provider.saveCalled.Load() {
		t.Fatal("SaveConnection not called")
	}
}

func TestDeleteConnection_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	provider := newConnectionSpyProvider()

	req := httptest.NewRequest(
		http.MethodDelete,
		"/",
		nil,
	)

	req = mux.SetURLVars(req, map[string]string{
		"connectionId": uuid.Must(uuid.NewV4()).String(),
	})

	rr := httptest.NewRecorder()

	defer func() { _ = recover() }()

	h.DeleteConnection(
		rr,
		req,
		nil,
		&models.User{
			ID: uuid.Must(uuid.NewV4()),
		},
		provider,
	)

	if !provider.deleteCalled.Load() {
		t.Fatal("DeleteConnection not called")
	}
}

func TestUpdateConnectionById_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	provider := newConnectionSpyProvider()

	req := httptest.NewRequest(
		http.MethodPut,
		"/",
		strings.NewReader(`{}`),
	)

	req = mux.SetURLVars(req, map[string]string{
		"connectionId": uuid.Must(uuid.NewV4()).String(),
	})

	rr := httptest.NewRecorder()

	defer func() { _ = recover() }()

	h.UpdateConnectionById(
		rr,
		req,
		nil,
		&models.User{
			ID: uuid.Must(uuid.NewV4()),
		},
		provider,
	)

	if !provider.updateCalled.Load() {
		t.Fatal("UpdateConnectionById not called")
	}
}
func (p *connectionSpyProvider) GetConnections(
	req *http.Request,
	userID string,
	page, pageSize int,
	search, order, filter string,
	status []string,
	kind []string,
	connType []string,
	name string,
) (*connections.ConnectionPage, error) {

	return &connections.ConnectionPage{}, nil
}

func (p *connectionSpyProvider) GetConnectionByID(
	token string,
	id core.Uuid,
) (*connections.Connection, int, error) {

	return &connections.Connection{
		Name: "demo",
		Kind: "kubernetes",
	}, http.StatusOK, nil
}

func TestGetConnections_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/connections?page=1&pagesize=all",
		nil,
	)

	rr := httptest.NewRecorder()

	h.GetConnections(
		rr,
		req,
		nil,
		&models.User{ID: uuid.Must(uuid.NewV4())},
		newConnectionSpyProvider(),
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestGetConnectionsByKind_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/connections/kubernetes",
		nil,
	)

	req = mux.SetURLVars(req, map[string]string{
		"connectionKind": "kubernetes",
	})

	rr := httptest.NewRecorder()

	h.GetConnectionsByKind(
		rr,
		req,
		nil,
		&models.User{ID: uuid.Must(uuid.NewV4())},
		newConnectionSpyProvider(),
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestGetConnectionByID_Success(t *testing.T) {
	h := newConnectionsTestHandler(t)

	id := uuid.Must(uuid.NewV4())

	req := httptest.NewRequest(
		http.MethodGet,
		"/connections/"+id.String(),
		nil,
	)

	req = mux.SetURLVars(req, map[string]string{
		"connectionId": id.String(),
	})

	rr := httptest.NewRecorder()

	h.GetConnectionByID(
		rr,
		req,
		nil,
		&models.User{ID: uuid.Must(uuid.NewV4())},
		newConnectionSpyProvider(),
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleProcessTermination_ValidJSON(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/",
		strings.NewReader(`{"id":"`+uuid.Must(uuid.NewV4()).String()+`"}`),
	)

	h.ConnectionToStateMachineInstanceTracker =
		&machines.ConnectionToStateMachineInstanceTracker{}

	rr := httptest.NewRecorder()

	h.handleProcessTermination(rr, req)
}

func TestProcessConnectionRegistration_DeleteMethod(t *testing.T) {
	h := newConnectionsTestHandler(t)

	h.ConnectionToStateMachineInstanceTracker =
		&machines.ConnectionToStateMachineInstanceTracker{}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/",
		strings.NewReader(`{"id":"`+uuid.Must(uuid.NewV4()).String()+`"}`),
	)

	rr := httptest.NewRecorder()

	h.ProcessConnectionRegistration(
		rr,
		req,
		nil,
		&models.User{ID: uuid.Must(uuid.NewV4())},
		newConnectionSpyProvider(),
	)
}
func TestProcessConnectionRegistration_BadJSON(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("{bad"),
	)

	rr := httptest.NewRecorder()

	defer func() { _ = recover() }()

	var provider models.Provider

	h.ProcessConnectionRegistration(
		rr,
		req,
		nil,
		&models.User{
			ID: uuid.Must(uuid.NewV4()),
		},
		provider,
	)
}
func TestNotifySmOfConnectionStatusChange_EmptyStatus(t *testing.T) {
	h := newConnectionsTestHandler(t)

	conn := &connections.ConnectionPayload{}

	_, err := h.NotifySmOfConnectionStatusChange(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		newConnectionSpyProvider(),
		"",
		conn,
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestNotifySmOfConnectionStatusChange_NilConnection(t *testing.T) {
	h := newConnectionsTestHandler(t)

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	h.NotifySmOfConnectionStatusChange(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		newConnectionSpyProvider(),
		"",
		nil,
	)
}

func TestProcessConnectionRegistration_BadJSON_2(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("{bad json"),
	)

	rr := httptest.NewRecorder()

	h.ProcessConnectionRegistration(
		rr,
		req,
		nil,
		&models.User{ID: uuid.Must(uuid.NewV4())},
		newConnectionSpyProvider(),
	)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleProcessTermination_BadJSON(t *testing.T) {
	h := newConnectionsTestHandler(t)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/",
		strings.NewReader("{bad json"),
	)

	rr := httptest.NewRecorder()

	h.handleProcessTermination(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleMeshSyncDeploymentModeChange_NilConnection(t *testing.T) {
	h := newConnectionsTestHandler(t)

	_, _, _, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		nil,
		"",
		uuid.Must(uuid.NewV4()),
		newConnectionSpyProvider(),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleMeshSyncDeploymentModeChange_NilSystemID(t *testing.T) {
	h := newConnectionsTestHandler(t)
	h.SystemID = nil

	_, _, _, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		&connections.ConnectionPayload{},
		"",
		uuid.Must(uuid.NewV4()),
		newConnectionSpyProvider(),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeProvider struct {
	*models.DefaultLocalProvider

	conn *connections.Connection
}

func newFakeProvider() *fakeProvider {
	p := &fakeProvider{
		DefaultLocalProvider: &models.DefaultLocalProvider{},
	}

	p.Initialize()

	return p
}

func (p *fakeProvider) GetProviderToken(*http.Request) (string, error) {
	return "token", nil
}

func (p *fakeProvider) PersistEvent(events.Event, string) error {
	return nil
}

func (p *fakeProvider) GetK8sContext(
	string,
	string,
) (models.K8sContext, error) {

	return models.K8sContext{
		ID:           "ctx-1",
		Name:         "cluster",
		ConnectionID: uuid.Must(uuid.NewV4()).String(),
	}, nil
}

func (p *fakeProvider) GetConnectionByID(
	string,
	core.Uuid,
) (*connections.Connection, int, error) {

	if p.conn == nil {
		p.conn = &connections.Connection{
			ID:   uuid.Must(uuid.NewV4()),
			Kind: "kubernetes",
		}
	}

	return p.conn, http.StatusOK, nil
}

func (p *fakeProvider) UpdateConnectionById(
	string,
	*connections.ConnectionPayload,
	string,
) (*connections.Connection, error) {

	return &connections.Connection{
		Name: "updated",
	}, nil
}
func newHeavyHandler(t *testing.T) *Handler {
	h := newConnectionsTestHandler(t)

	h.config = &models.HandlerConfig{
		EventBroadcaster: models.NewBroadcaster("test"),
		OperatorTracker:  models.NewOperatorTracker(true),
	}

	h.ConnectionToStateMachineInstanceTracker =
		&machines.ConnectionToStateMachineInstanceTracker{
			ConnectToInstanceMap: map[core.Uuid]*machines.StateMachine{},
		}

	return h
}
func TestHandleMeshSyncDeploymentModeChange_NilConnection_2(t *testing.T) {
	h := newHeavyHandler(t)

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	_, _, changed, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		id,
		nil,
		"",
		id,
		newFakeProvider(),
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if changed {
		t.Fatal("expected changed=false")
	}
}

func TestHandleMeshSyncDeploymentModeChange_SystemIDNil(t *testing.T) {
	h := newHeavyHandler(t)
	h.SystemID = nil

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	conn := &connections.ConnectionPayload{
		MetaData: map[string]interface{}{},
	}

	_, _, _, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		id,
		conn,
		"",
		id,
		newFakeProvider(),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}
func TestHandleMeshSyncDeploymentModeChange_NotKubernetes(t *testing.T) {
	h := newHeavyHandler(t)

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	p := newFakeProvider()
	p.conn = &connections.Connection{
		ID:   id,
		Kind: "prometheus",
	}

	conn := &connections.ConnectionPayload{
		MetaData: map[string]interface{}{},
	}

	_, _, changed, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		id,
		conn,
		"",
		id,
		p,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if changed {
		t.Fatal("expected changed=false")
	}
}
func TestHandleMeshSyncDeploymentModeChange_NilTracker(t *testing.T) {
	h := newHeavyHandler(t)

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	h.ConnectionToStateMachineInstanceTracker = nil

	p := newFakeProvider()
	p.conn = &connections.Connection{
		ID:       id,
		Kind:     "kubernetes",
		Metadata: map[string]interface{}{},
	}

	conn := &connections.ConnectionPayload{
		MetaData: map[string]interface{}{},
	}

	_, _, changed, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		id,
		conn,
		"",
		id,
		p,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if changed {
		t.Fatal("expected changed=false")
	}
}

func TestHandleMeshSyncDeploymentModeChange_MachineNotFound(t *testing.T) {
	h := newHeavyHandler(t)

	id := core.Uuid(uuid.Must(uuid.NewV4()))

	p := newFakeProvider()
	p.conn = &connections.Connection{
		ID:   id,
		Kind: "kubernetes",
		Metadata: map[string]interface{}{
			"meshsyncDeploymentMode": "operator",
		},
	}

	conn := &connections.ConnectionPayload{
		MetaData: map[string]interface{}{
			"meshsyncDeploymentMode": "sidecar",
		},
	}

	_, _, changed, err := h.handleMeshSyncDeploymentModeChange(
		context.Background(),
		id,
		conn,
		"",
		id,
		p,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if changed {
		t.Fatal("expected changed=false")
	}
}
