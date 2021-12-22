package remote

import (
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"net"
	"time"

	"github.com/AsynkronIT/protoactor-go/extensions"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var extensionId = extensions.NextExtensionID()

type Remote struct {
	actorSystem  *actor.ActorSystem
	s            *grpc.Server
	edpReader    *endpointReader
	edpManager   *endpointManager
	config       *Config
	kinds        map[string]*actor.Props
	activatorPid *actor.PID
}

func (r *Remote) Connect(ctx context.Context, request *ConnectRequest) (*ConnectResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Remote) Receive(server Remoting_ReceiveServer) error {
	//TODO implement me
	panic("implement me")
}

func (r *Remote) ListProcesses(ctx context.Context, request *ListProcessesRequest) (*ListProcessesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Remote) GetProcessDiagnostics(ctx context.Context, request *GetProcessDiagnosticsRequest) (*GetProcessDiagnosticsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Remote) mustEmbedUnimplementedRemotingServer() {
	//TODO implement me
	panic("implement me")
}

func NewRemote(actorSystem *actor.ActorSystem, config Config) *Remote {
	r := &Remote{
		actorSystem: actorSystem,
		config:      &config,
		kinds:       make(map[string]*actor.Props),
	}
	for k, v := range config.Kinds {
		r.kinds[k] = v
	}

	actorSystem.Extensions.Register(r)

	return r
}

//goland:noinspection GoUnusedExportedFunction
func GetRemote(actorSystem *actor.ActorSystem) *Remote {
	r := actorSystem.Extensions.Get(extensionId)

	return r.(*Remote)
}

func (r *Remote) ExtensionID() extensions.ExtensionID {
	return extensionId
}

// Start the remote server
func (r *Remote) Start() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	lis, err := net.Listen("tcp", r.config.Address())
	if err != nil {
		panic(fmt.Errorf("failed to listen: %v", err))
	}

	var address string
	if r.config.AdvertisedHost != "" {
		address = r.config.AdvertisedHost
	} else {
		address = lis.Addr().String()
	}

	r.actorSystem.ProcessRegistry.RegisterAddressResolver(r.remoteHandler)
	r.actorSystem.ProcessRegistry.Address = address

	r.edpManager = newEndpointManager(r)
	r.edpManager.start()

	r.s = grpc.NewServer(r.config.ServerOptions...)
	r.edpReader = newEndpointReader(r)
	RegisterRemotingServer(r.s, r.edpReader)
	plog.Info("Starting Proto.Actor server", log.String("address", address))
	go r.s.Serve(lis)
}

func (r *Remote) Shutdown(graceful bool) {
	if graceful {
		// TODO: need more graceful
		r.edpReader.suspend(true)
		r.edpManager.stop()

		// For some reason GRPC doesn't want to stop
		// Setup timeout as workaround but need to figure out in the future.
		// TODO: grpc not stopping
		c := make(chan bool, 1)
		go func() {
			r.s.GracefulStop()
			c <- true
		}()

		select {
		case <-c:
			plog.Info("Stopped Proto.Actor server")
		case <-time.After(time.Second * 10):
			r.s.Stop()
			plog.Info("Stopped Proto.Actor server", log.String("err", "timeout"))
		}
	} else {
		r.s.Stop()
		plog.Info("Killed Proto.Actor server")
	}
}

func (r *Remote) SendMessage(pid *actor.PID, header actor.ReadonlyMessageHeader, message interface{}, sender *actor.PID, serializerID int32) {
	rd := &remoteDeliver{
		header:       header,
		message:      message,
		sender:       sender,
		target:       pid,
		serializerID: serializerID,
	}
	r.edpManager.remoteDeliver(rd)
}
