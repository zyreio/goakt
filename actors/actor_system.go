/*
 * MIT License
 *
 * Copyright (c) 2022-2024  Arsene Tochemey Gandote
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package actors

import (
	"context"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/reugn/go-quartz/logger"
	"go.uber.org/atomic"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/tochemey/goakt/v2/address"
	"github.com/tochemey/goakt/v2/discovery"
	"github.com/tochemey/goakt/v2/goaktpb"
	"github.com/tochemey/goakt/v2/hash"
	"github.com/tochemey/goakt/v2/internal/cluster"
	"github.com/tochemey/goakt/v2/internal/errorschain"
	"github.com/tochemey/goakt/v2/internal/eventstream"
	"github.com/tochemey/goakt/v2/internal/internalpb"
	"github.com/tochemey/goakt/v2/internal/internalpb/internalpbconnect"
	"github.com/tochemey/goakt/v2/internal/tcp"
	"github.com/tochemey/goakt/v2/internal/types"
	"github.com/tochemey/goakt/v2/log"
)

// ActorSystem defines the contract of an actor system
type ActorSystem interface {
	// Name returns the actor system name
	Name() string
	// Actors returns the list of Actors that are alive in the actor system
	Actors() []*PID
	// Start starts the actor system
	Start(ctx context.Context) error
	// Stop stops the actor system
	Stop(ctx context.Context) error
	// Spawn creates an actor in the system and starts it
	Spawn(ctx context.Context, name string, actor Actor, opts ...SpawnOption) (*PID, error)
	// SpawnFromFunc creates an actor with the given receive function. One can set the PreStart and PostStop lifecycle hooks
	// in the given optional options
	SpawnFromFunc(ctx context.Context, receiveFunc ReceiveFunc, opts ...FuncOption) (*PID, error)
	// SpawnNamedFromFunc creates an actor with the given receive function and provided name. One can set the PreStart and PostStop lifecycle hooks
	// in the given optional options
	SpawnNamedFromFunc(ctx context.Context, name string, receiveFunc ReceiveFunc, opts ...FuncOption) (*PID, error)
	// SpawnRouter creates a new router. One can additionally set the router options.
	// A router is a special type of actor that helps distribute messages of the same type over a set of actors, so that messages can be processed in parallel.
	// A single actor will only process one message at a time.
	SpawnRouter(ctx context.Context, poolSize int, routeesKind Actor, opts ...RouterOption) (*PID, error)
	// Kill stops a given actor in the system
	Kill(ctx context.Context, name string) error
	// ReSpawn recreates a given actor in the system
	// During restart all messages that are in the mailbox and not yet processed will be ignored.
	// Only the direct alive children of the given actor will be shudown and respawned with their initial state.
	// Bear in mind that restarting an actor will reinitialize the actor to initial state.
	// In case any of the direct child restart fails the given actor will not be started at all.
	ReSpawn(ctx context.Context, name string) (*PID, error)
	// NumActors returns the total number of active actors in the system
	NumActors() uint64
	// LocalActor returns the reference of a local actor.
	// A local actor is an actor that reside on the same node where the given actor system has started
	LocalActor(actorName string) (*PID, error)
	// RemoteActor returns the address of a remote actor when cluster is enabled
	// When the cluster mode is not enabled an actor not found error will be returned
	// One can always check whether cluster is enabled before calling this method or just use the ActorOf method.
	RemoteActor(ctx context.Context, actorName string) (addr *address.Address, err error)
	// ActorOf returns an existing actor in the local system or in the cluster when clustering is enabled
	// When cluster mode is activated, the PID will be nil.
	// When remoting is enabled this method will return and error
	// An actor not found error is return when the actor is not found.
	ActorOf(ctx context.Context, actorName string) (addr *address.Address, pid *PID, err error)
	// InCluster states whether the actor system has started within a cluster of nodes
	InCluster() bool
	// GetPartition returns the partition where a given actor is located
	GetPartition(actorName string) int
	// Subscribe creates an event subscriber to consume events from the actor system.
	// Remember to use the Unsubscribe method to avoid resource leakage.
	Subscribe() (eventstream.Subscriber, error)
	// Unsubscribe unsubscribes a subscriber.
	Unsubscribe(subscriber eventstream.Subscriber) error
	// ScheduleOnce schedules a message that will be delivered to the receiver actor
	// This will send the given message to the actor after the given interval specified.
	// The message will be sent once
	ScheduleOnce(ctx context.Context, message proto.Message, pid *PID, interval time.Duration) error
	// Schedule schedules a message that will be delivered to the receiver actor
	// This will send the given message to the actor after the given interval specified.
	Schedule(ctx context.Context, message proto.Message, pid *PID, interval time.Duration) error
	// RemoteScheduleOnce schedules a message to be sent to a remote actor in the future.
	// This requires remoting to be enabled on the actor system.
	// This will send the given message to the actor after the given interval specified
	// The message will be sent once
	RemoteScheduleOnce(ctx context.Context, message proto.Message, address *address.Address, interval time.Duration) error
	// RemoteSchedule schedules a message to be sent to a remote actor in the future.
	// This requires remoting to be enabled on the actor system.
	// This will send the given message to the actor after the given interval specified
	RemoteSchedule(ctx context.Context, message proto.Message, address *address.Address, interval time.Duration) error
	// ScheduleWithCron schedules a message to be sent to an actor in the future using a cron expression.
	ScheduleWithCron(ctx context.Context, message proto.Message, pid *PID, cronExpression string) error
	// RemoteScheduleWithCron schedules a message to be sent to an actor in the future using a cron expression.
	RemoteScheduleWithCron(ctx context.Context, message proto.Message, address *address.Address, cronExpression string) error
	// PeerAddress returns the actor system address known in the cluster. That address is used by other nodes to communicate with the actor system.
	// This address is empty when cluster mode is not activated
	PeerAddress() string
	// Register registers an actor for future use. This is necessary when creating an actor remotely
	Register(ctx context.Context, actor Actor) error
	// Deregister removes a registered actor from the registry
	Deregister(ctx context.Context, actor Actor) error
	// Logger returns the logger sets when creating the actor system
	Logger() log.Logger
	// Host returns the actor system node host address
	// This is the bind address for remote communication
	Host() string
	// Port returns the actor system node port.
	// This is the bind port for remote communication
	Port() int32
	// Uptime returns the number of seconds since the actor system started
	Uptime() int64
	// handleRemoteAsk handles a synchronous message to another actor and expect a response.
	// This block until a response is received or timed out.
	handleRemoteAsk(ctx context.Context, to *PID, message proto.Message, timeout time.Duration) (response proto.Message, err error)
	// handleRemoteTell handles an asynchronous message to an actor
	handleRemoteTell(ctx context.Context, to *PID, message proto.Message) error
	// setActor sets actor in the actor system actors registry
	broadcastActor(actor *PID)
	// getPeerStateFromCache returns the peer state from the cache
	getPeerStateFromCache(address string) (*internalpb.PeerState, error)
	// reservedName returns reserved actor's name
	reservedName(nameType nameType) string
	// getCluster returns the cluster engine
	getCluster() cluster.Interface
	// tree returns the actors tree
	tree() *pidTree

	completeRebalancing()
	getRootGuardian() *PID
	getSystemGuardian() *PID
	getUserGuardian() *PID
	getJanitor() *PID
	getDeadletters() *PID
}

// ActorSystem represent a collection of actors on a given node
// Only a single instance of the ActorSystem can be created on a given node
type actorSystem struct {
	// hold the actors tree in the system
	actors *pidTree

	// states whether the actor system has started or not
	started atomic.Bool

	// Specifies the actor system name
	name string
	// Specifies the logger to use in the system
	logger log.Logger
	// Specifies at what point in time to passivate the actor.
	// when the actor is passivated it is stopped which means it does not consume
	// any further resources like memory and cpu. The default value is 5s
	expireActorAfter time.Duration
	// Specifies how long the sender of a message should wait to receive a reply
	// when using SendReply. The default value is 5s
	askTimeout time.Duration
	// Specifies the shutdown timeout. The default value is 30s
	shutdownTimeout time.Duration
	// Specifies the maximum of retries to attempt when the actor
	// initialization fails. The default value is 5
	actorInitMaxRetries int
	// Specifies the actors initialization timeout
	// The default value is 1s
	actorInitTimeout time.Duration

	// Specifies whether remoting is enabled.
	// This allows to handle remote messaging
	remotingEnabled atomic.Bool
	remoting        *Remoting
	// Specifies the remoting port
	port int32
	// Specifies the remoting host
	host string
	// Specifies the remoting server
	server   *nethttp.Server
	listener net.Listener

	// cluster settings
	clusterEnabled atomic.Bool
	// cluster mode
	cluster            cluster.Interface
	actorsChan         chan *internalpb.ActorRef
	clusterEventsChan  <-chan *cluster.Event
	clusterSyncStopSig chan types.Unit
	partitionHasher    hash.Hasher
	clusterNode        *discovery.Node

	// help protect some the fields to set
	locker sync.Mutex
	// specifies the stash capacity
	stashEnabled bool

	stopGC          chan types.Unit
	janitorInterval time.Duration

	// specifies the events stream
	eventsStream *eventstream.EventsStream

	// specifies the message scheduler
	scheduler *scheduler

	registry   types.Registry
	reflection *reflection

	peersStateLoopInterval time.Duration
	peersCache             *sync.Map
	clusterConfig          *ClusterConfig
	rebalancingChan        chan *cluster.Event

	rebalancer     *PID
	rootGuardian   *PID
	userGuardian   *PID
	systemGuardian *PID
	janitor        *PID
	deadletters    *PID
	startedAt      *atomic.Int64
	rebalancing    *atomic.Bool
}

var (
	// enforce compilation error when all methods of the ActorSystem interface are not implemented
	// by the struct actorSystem
	_ ActorSystem                              = (*actorSystem)(nil)
	_ internalpbconnect.RemotingServiceHandler = (*actorSystem)(nil)
	_ internalpbconnect.ClusterServiceHandler  = (*actorSystem)(nil)
)

// NewActorSystem creates an instance of ActorSystem
func NewActorSystem(name string, opts ...Option) (ActorSystem, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if match, _ := regexp.MatchString("^[a-zA-Z0-9][a-zA-Z0-9-_]*$", name); !match {
		return nil, ErrInvalidActorSystemName
	}

	system := &actorSystem{
		actorsChan:             make(chan *internalpb.ActorRef, 10),
		name:                   name,
		logger:                 log.New(log.ErrorLevel, os.Stderr),
		expireActorAfter:       DefaultPassivationTimeout,
		actorInitMaxRetries:    DefaultInitMaxRetries,
		locker:                 sync.Mutex{},
		shutdownTimeout:        DefaultShutdownTimeout,
		stashEnabled:           false,
		stopGC:                 make(chan types.Unit, 1),
		janitorInterval:        DefaultJanitorInterval,
		eventsStream:           eventstream.New(),
		partitionHasher:        hash.DefaultHasher(),
		actorInitTimeout:       DefaultInitTimeout,
		clusterEventsChan:      make(chan *cluster.Event, 1),
		registry:               types.NewRegistry(),
		clusterSyncStopSig:     make(chan types.Unit, 1),
		peersCache:             &sync.Map{},
		peersStateLoopInterval: DefaultPeerStateLoopInterval,
		port:                   0,
		host:                   "127.0.0.1",
		actors:                 newTree(),
		startedAt:              atomic.NewInt64(0),
		rebalancing:            atomic.NewBool(false),
	}

	system.started.Store(false)
	system.remotingEnabled.Store(false)
	system.clusterEnabled.Store(false)

	system.reflection = newReflection(system.registry)

	// apply the various options
	for _, opt := range opts {
		opt.Apply(system)
	}

	// set the host and port
	if err := system.setHostPort(); err != nil {
		return nil, err
	}

	// we need to make sure the cluster kinds are defined
	if system.clusterEnabled.Load() {
		if err := system.clusterConfig.Validate(); err != nil {
			return nil, err
		}
	}

	system.scheduler = newScheduler(system.logger,
		system.shutdownTimeout,
		withSchedulerCluster(system.cluster),
		withSchedulerRemoting(NewRemoting()))

	return system, nil
}

// Uptime returns the number of seconds since the actor system started
func (x *actorSystem) Uptime() int64 {
	if x.started.Load() {
		return time.Now().Unix() - x.startedAt.Load()
	}
	return 0
}

// Host returns the actor system node host address
// This is the bind address for remote communication
func (x *actorSystem) Host() string {
	x.locker.Lock()
	host := x.host
	x.locker.Unlock()
	return host
}

// Port returns the actor system node port.
// This is the bind port for remote communication
func (x *actorSystem) Port() int32 {
	x.locker.Lock()
	port := x.port
	x.locker.Unlock()
	return port
}

// Logger returns the logger sets when creating the actor system
func (x *actorSystem) Logger() log.Logger {
	x.locker.Lock()
	logger := x.logger
	x.locker.Unlock()
	return logger
}

// Deregister removes a registered actor from the registry
func (x *actorSystem) Deregister(_ context.Context, actor Actor) error {
	x.locker.Lock()
	defer x.locker.Unlock()

	if !x.started.Load() {
		return ErrActorSystemNotStarted
	}

	x.registry.Deregister(actor)
	return nil
}

// Register registers an actor for future use. This is necessary when creating an actor remotely
func (x *actorSystem) Register(_ context.Context, actor Actor) error {
	x.locker.Lock()
	defer x.locker.Unlock()

	if !x.started.Load() {
		return ErrActorSystemNotStarted
	}

	x.registry.Register(actor)
	return nil
}

// Schedule schedules a message that will be delivered to the receiver actor
// This will send the given message to the actor at the given interval specified.
func (x *actorSystem) Schedule(ctx context.Context, message proto.Message, pid *PID, interval time.Duration) error {
	return x.scheduler.Schedule(ctx, message, pid, interval)
}

// RemoteSchedule schedules a message to be sent to a remote actor in the future.
// This requires remoting to be enabled on the actor system.
// This will send the given message to the actor at the given interval specified
func (x *actorSystem) RemoteSchedule(ctx context.Context, message proto.Message, address *address.Address, interval time.Duration) error {
	return x.scheduler.RemoteSchedule(ctx, message, address, interval)
}

// ScheduleOnce schedules a message that will be delivered to the receiver actor
// This will send the given message to the actor after the given interval specified.
// The message will be sent once
func (x *actorSystem) ScheduleOnce(ctx context.Context, message proto.Message, pid *PID, interval time.Duration) error {
	return x.scheduler.ScheduleOnce(ctx, message, pid, interval)
}

// RemoteScheduleOnce schedules a message to be sent to a remote actor in the future.
// This requires remoting to be enabled on the actor system.
// This will send the given message to the actor after the given interval specified
// The message will be sent once
func (x *actorSystem) RemoteScheduleOnce(ctx context.Context, message proto.Message, address *address.Address, interval time.Duration) error {
	return x.scheduler.RemoteScheduleOnce(ctx, message, address, interval)
}

// ScheduleWithCron schedules a message to be sent to an actor in the future using a cron expression.
func (x *actorSystem) ScheduleWithCron(ctx context.Context, message proto.Message, pid *PID, cronExpression string) error {
	return x.scheduler.ScheduleWithCron(ctx, message, pid, cronExpression)
}

// RemoteScheduleWithCron schedules a message to be sent to an actor in the future using a cron expression.
func (x *actorSystem) RemoteScheduleWithCron(ctx context.Context, message proto.Message, address *address.Address, cronExpression string) error {
	return x.scheduler.RemoteScheduleWithCron(ctx, message, address, cronExpression)
}

// Subscribe creates an event subscriber to consume events from the actor system.
// Remember to use the Unsubscribe method to avoid resource leakage.
func (x *actorSystem) Subscribe() (eventstream.Subscriber, error) {
	if !x.started.Load() {
		return nil, ErrActorSystemNotStarted
	}
	subscriber := x.eventsStream.AddSubscriber()
	x.eventsStream.Subscribe(subscriber, eventsTopic)
	return subscriber, nil
}

// Unsubscribe unsubscribes a subscriber.
func (x *actorSystem) Unsubscribe(subscriber eventstream.Subscriber) error {
	if !x.started.Load() {
		return ErrActorSystemNotStarted
	}
	x.eventsStream.Unsubscribe(subscriber, eventsTopic)
	x.eventsStream.RemoveSubscriber(subscriber)
	return nil
}

// GetPartition returns the partition where a given actor is located
func (x *actorSystem) GetPartition(actorName string) int {
	if !x.InCluster() {
		// TODO: maybe add a partitioner function
		return 0
	}
	return x.cluster.GetPartition(actorName)
}

// InCluster states whether the actor system is started within a cluster of nodes
func (x *actorSystem) InCluster() bool {
	return x.clusterEnabled.Load() && x.cluster != nil
}

// NumActors returns the total number of active actors in the system
func (x *actorSystem) NumActors() uint64 {
	return uint64(len(x.Actors()))
}

// Spawn creates or returns the instance of a given actor in the system
func (x *actorSystem) Spawn(ctx context.Context, name string, actor Actor, opts ...SpawnOption) (*PID, error) {
	if !x.started.Load() {
		return nil, ErrActorSystemNotStarted
	}

	// check some preconditions
	if err := x.checkSpawnPreconditions(ctx, name, actor); err != nil {
		return nil, err
	}

	actorAddress := x.actorAddress(name)
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if exist {
		pid := pidNode.GetValue()
		if pid.IsRunning() {
			return pid, nil
		}
	}

	pid, err := x.configPID(ctx, name, actor, opts...)
	if err != nil {
		return nil, err
	}

	// add the given actor to the tree and supervise it
	_ = x.actors.AddNode(x.userGuardian, pid)
	x.actors.AddWatcher(pid, x.janitor)
	x.broadcastActor(pid)
	return pid, nil
}

// SpawnNamedFromFunc creates an actor with the given receive function and provided name. One can set the PreStart and PostStop lifecycle hooks
// in the given optional options
func (x *actorSystem) SpawnNamedFromFunc(ctx context.Context, name string, receiveFunc ReceiveFunc, opts ...FuncOption) (*PID, error) {
	if !x.started.Load() {
		return nil, ErrActorSystemNotStarted
	}

	config := newFuncConfig(opts...)
	actor := newFuncActor(name, receiveFunc, config)

	// check some preconditions
	if err := x.checkSpawnPreconditions(ctx, name, actor); err != nil {
		return nil, err
	}

	actorAddress := x.actorAddress(name)
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if exist {
		pid := pidNode.GetValue()
		if pid.IsRunning() {
			return pid, nil
		}
	}

	pid, err := x.configPID(ctx, name, actor, WithMailbox(config.mailbox))
	if err != nil {
		return nil, err
	}

	_ = x.actors.AddNode(x.userGuardian, pid)
	x.actors.AddWatcher(pid, x.janitor)
	x.broadcastActor(pid)
	return pid, nil
}

// SpawnFromFunc creates an actor with the given receive function.
func (x *actorSystem) SpawnFromFunc(ctx context.Context, receiveFunc ReceiveFunc, opts ...FuncOption) (*PID, error) {
	return x.SpawnNamedFromFunc(ctx, uuid.NewString(), receiveFunc, opts...)
}

// SpawnRouter creates a new router. One can additionally set the router options.
// A router is a special type of actor that helps distribute messages of the same type over a set of actors, so that messages can be processed in parallel.
// A single actor will only process one message at a time.
func (x *actorSystem) SpawnRouter(ctx context.Context, poolSize int, routeesKind Actor, opts ...RouterOption) (*PID, error) {
	router := newRouter(poolSize, routeesKind, x.logger, opts...)
	routerName := x.reservedName(routerType)
	return x.Spawn(ctx, routerName, router)
}

// Kill stops a given actor in the system
func (x *actorSystem) Kill(ctx context.Context, name string) error {
	if !x.started.Load() {
		return ErrActorSystemNotStarted
	}

	actorAddress := x.actorAddress(name)
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if exist {
		pid := pidNode.GetValue()
		// stop the given actor. No need to record error in the span context
		// because the shutdown method is taking care of that
		return pid.Shutdown(ctx)
	}

	return ErrActorNotFound(actorAddress.String())
}

// ReSpawn recreates a given actor in the system
// During restart all messages that are in the mailbox and not yet processed will be ignored.
// Only the direct alive children of the given actor will be shudown and respawned with their initial state.
// Bear in mind that restarting an actor will reinitialize the actor to initial state.
// In case any of the direct child restart fails the given actor will not be started at all.
func (x *actorSystem) ReSpawn(ctx context.Context, name string) (*PID, error) {
	if !x.started.Load() {
		return nil, ErrActorSystemNotStarted
	}

	actorAddress := x.actorAddress(name)
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if exist {
		pid := pidNode.GetValue()

		parent := NoSender
		if parentNode, ok := x.actors.Parent(pid); ok {
			parent = parentNode.GetValue()
		}

		if err := pid.Restart(ctx); err != nil {
			return nil, fmt.Errorf("failed to restart actor=%s: %w", actorAddress.String(), err)
		}

		// no need to handle the error here because the only time this method
		// returns an error if when the parent does not exist which was taken care of in the
		// lines above
		_ = x.actors.AddNode(parent, pid)
		x.actors.AddWatcher(pid, x.janitor)
		return pid, nil
	}

	return nil, ErrActorNotFound(actorAddress.String())
}

// Name returns the actor system name
func (x *actorSystem) Name() string {
	x.locker.Lock()
	name := x.name
	x.locker.Unlock()
	return name
}

// Actors returns the list of Actors that are alive in the actor system
func (x *actorSystem) Actors() []*PID {
	x.locker.Lock()
	pidNodes := x.actors.Nodes()
	x.locker.Unlock()
	actors := make([]*PID, 0, len(pidNodes))
	for _, pidNode := range pidNodes {
		pid := pidNode.GetValue()
		if !isReservedName(pid.Name()) {
			actors = append(actors, pid)
		}
	}
	return actors
}

// PeerAddress returns the actor system address known in the cluster. That address is used by other nodes to communicate with the actor system.
// This address is empty when cluster mode is not activated
func (x *actorSystem) PeerAddress() string {
	x.locker.Lock()
	defer x.locker.Unlock()
	if x.clusterEnabled.Load() {
		return x.clusterNode.PeersAddress()
	}
	return ""
}

// ActorOf returns an existing actor in the local system or in the cluster when clustering is enabled
// When cluster mode is activated, the PID will be nil.
// When remoting is enabled this method will return and error
// An actor not found error is return when the actor is not found.
func (x *actorSystem) ActorOf(ctx context.Context, actorName string) (addr *address.Address, pid *PID, err error) {
	x.locker.Lock()

	if !x.started.Load() {
		x.locker.Unlock()
		return nil, nil, ErrActorSystemNotStarted
	}

	// first check whether the actor exist locally
	actorAddress := x.actorAddress(actorName)
	if lpidNode, ok := x.actors.GetNode(actorAddress.String()); ok {
		x.locker.Unlock()
		lpid := lpidNode.GetValue()
		return lpid.Address(), lpid, nil
	}

	// check in the cluster
	if x.clusterEnabled.Load() {
		actor, err := x.cluster.GetActor(ctx, actorName)
		if err != nil {
			if errors.Is(err, cluster.ErrActorNotFound) {
				x.logger.Infof("actor=%s not found", actorName)
				x.locker.Unlock()
				return nil, nil, ErrActorNotFound(actorName)
			}

			x.locker.Unlock()
			return nil, nil, fmt.Errorf("failed to fetch remote actor=%s: %w", actorName, err)
		}

		x.locker.Unlock()
		return address.From(actor.GetActorAddress()), nil, nil
	}

	if x.remotingEnabled.Load() {
		x.locker.Unlock()
		return nil, nil, ErrMethodCallNotAllowed
	}

	x.logger.Infof("actor=%s not found", actorName)
	x.locker.Unlock()
	return nil, nil, ErrActorNotFound(actorName)
}

// LocalActor returns the reference of a local actor.
// A local actor is an actor that reside on the same node where the given actor system has started
func (x *actorSystem) LocalActor(actorName string) (*PID, error) {
	x.locker.Lock()

	if !x.started.Load() {
		x.locker.Unlock()
		return nil, ErrActorSystemNotStarted
	}

	actorAddress := x.actorAddress(actorName)
	if lpidNode, ok := x.actors.GetNode(actorAddress.String()); ok {
		x.locker.Unlock()
		lpid := lpidNode.GetValue()
		return lpid, nil
	}

	x.logger.Infof("actor=%s not found", actorName)
	x.locker.Unlock()
	return nil, ErrActorNotFound(actorName)
}

// RemoteActor returns the address of a remote actor when cluster is enabled
// When the cluster mode is not enabled an actor not found error will be returned
// One can always check whether cluster is enabled before calling this method or just use the ActorOf method.
func (x *actorSystem) RemoteActor(ctx context.Context, actorName string) (addr *address.Address, err error) {
	x.locker.Lock()

	if !x.started.Load() {
		e := ErrActorSystemNotStarted
		x.locker.Unlock()
		return nil, e
	}

	if x.cluster == nil {
		x.locker.Unlock()
		return nil, ErrClusterDisabled
	}

	actor, err := x.cluster.GetActor(ctx, actorName)
	if err != nil {
		if errors.Is(err, cluster.ErrActorNotFound) {
			x.logger.Infof("actor=%s not found", actorName)
			x.locker.Unlock()
			return nil, ErrActorNotFound(actorName)
		}

		x.locker.Unlock()
		return nil, fmt.Errorf("failed to fetch remote actor=%s: %w", actorName, err)
	}

	x.locker.Unlock()
	return address.From(actor.GetActorAddress()), nil
}

// Start starts the actor system
func (x *actorSystem) Start(ctx context.Context) error {
	x.logger.Infof("%s actor system starting..", x.name)
	x.started.Store(true)
	if err := errorschain.
		New(errorschain.ReturnFirst()).
		AddError(x.spawnRootGuardian(ctx)).
		AddError(x.spawnSystemGuardian(ctx)).
		AddError(x.spawnUserGuardian(ctx)).
		AddError(x.spawnRebalancer(ctx)).
		AddError(x.spawnJanitor(ctx)).
		AddError(x.spawnDeadletters(ctx)).
		AddError(x.enableRemoting(ctx)).
		AddError(x.enableClustering(ctx)).
		Error(); err != nil {
		// reset the start
		x.started.Store(false)
		return err
	}

	x.scheduler.Start(ctx)
	x.startedAt.Store(time.Now().Unix())
	x.logger.Infof("%s actor system successfully started..:)", x.name)
	return nil
}

// Stop stops the actor system
func (x *actorSystem) Stop(ctx context.Context) error {
	x.logger.Infof("%s shutting down...", x.name)

	// make sure the actor system has started
	if !x.started.Load() {
		return ErrActorSystemNotStarted
	}

	x.stopGC <- types.Unit{}
	x.logger.Infof("%s is shutting down..:)", x.name)

	x.started.Store(false)
	x.scheduler.Stop(ctx)

	ctx, cancel := context.WithTimeout(ctx, x.shutdownTimeout)
	defer cancel()

	var actorNames []string
	for _, actor := range x.Actors() {
		actorNames = append(actorNames, actor.Name())
	}

	if err := x.getRootGuardian().Shutdown(ctx); err != nil {
		x.reset()
		x.logger.Errorf("%s failed to shutdown cleanly: %w", x.name, err)
		return err
	}
	x.actors.DeleteNode(x.getRootGuardian())

	if x.eventsStream != nil {
		x.eventsStream.Shutdown()
	}

	if x.clusterEnabled.Load() {
		if err := errorschain.
			New(errorschain.ReturnFirst()).
			AddError(x.cleanupCluster(ctx, actorNames)).
			AddError(x.cluster.Stop(ctx)).
			Error(); err != nil {
			x.reset()
			x.logger.Errorf("%s failed to shutdown cleanly: %w", x.name, err)
			return err
		}

		close(x.actorsChan)
		x.clusterSyncStopSig <- types.Unit{}
		x.clusterEnabled.Store(false)
		close(x.rebalancingChan)
	}

	if x.remotingEnabled.Load() {
		x.remoting.Close()
		if err := x.shutdownHTTPServer(ctx); err != nil {
			x.reset()
			x.logger.Errorf("%s failed to shutdown: %w", x.name, err)
			return err
		}

		x.remotingEnabled.Store(false)
		x.server = nil
		x.listener = nil
	}

	x.reset()
	x.logger.Infof("%s shuts down successfully", x.name)
	return nil
}

// RemoteLookup for an actor on a remote host.
func (x *actorSystem) RemoteLookup(ctx context.Context, request *connect.Request[internalpb.RemoteLookupRequest]) (*connect.Response[internalpb.RemoteLookupResponse], error) {
	logger := x.logger
	msg := request.Msg

	if !x.remotingEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
	if remoteAddr != net.JoinHostPort(msg.GetHost(), strconv.Itoa(int(msg.GetPort()))) {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	if x.clusterEnabled.Load() {
		actorName := msg.GetName()
		actor, err := x.cluster.GetActor(ctx, actorName)
		if err != nil {
			if errors.Is(err, cluster.ErrActorNotFound) {
				logger.Error(ErrAddressNotFound(actorName).Error())
				return nil, ErrAddressNotFound(actorName)
			}

			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&internalpb.RemoteLookupResponse{Address: actor.GetActorAddress()}), nil
	}

	addr := address.New(msg.GetName(), x.Name(), msg.GetHost(), int(msg.GetPort()))
	pidNode, exist := x.actors.GetNode(addr.String())
	if !exist {
		logger.Error(ErrAddressNotFound(addr.String()).Error())
		return nil, ErrAddressNotFound(addr.String())
	}

	pid := pidNode.GetValue()
	return connect.NewResponse(&internalpb.RemoteLookupResponse{Address: pid.Address().Address}), nil
}

// RemoteAsk is used to send a message to an actor remotely and expect a response
// immediately. With this type of message the receiver cannot communicate back to Sender
// except reply the message with a response. This one-way communication
func (x *actorSystem) RemoteAsk(ctx context.Context, stream *connect.BidiStream[internalpb.RemoteAskRequest, internalpb.RemoteAskResponse]) error {
	logger := x.logger

	if !x.remotingEnabled.Load() {
		return connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	for {
		switch err := ctx.Err(); {
		case errors.Is(err, context.Canceled):
			return connect.NewError(connect.CodeCanceled, err)
		case errors.Is(err, context.DeadlineExceeded):
			return connect.NewError(connect.CodeDeadlineExceeded, err)
		}

		request, err := stream.Receive()
		if eof(err) {
			return nil
		}

		if err != nil {
			logger.Error(err)
			return connect.NewError(connect.CodeUnknown, err)
		}

		message := request.GetRemoteMessage()
		receiver := message.GetReceiver()
		name := receiver.GetName()

		remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
		if remoteAddr != net.JoinHostPort(receiver.GetHost(), strconv.Itoa(int(receiver.GetPort()))) {
			return connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
		}

		addr := x.actorAddress(name)
		pidNode, exist := x.actors.GetNode(addr.String())
		if !exist {
			logger.Error(ErrAddressNotFound(addr.String()).Error())
			return ErrAddressNotFound(addr.String())
		}

		timeout := x.askTimeout
		if request.GetTimeout() != nil {
			timeout = request.GetTimeout().AsDuration()
		}

		pid := pidNode.GetValue()
		reply, err := x.handleRemoteAsk(ctx, pid, message, timeout)
		if err != nil {
			logger.Error(ErrRemoteSendFailure(err).Error())
			return ErrRemoteSendFailure(err)
		}

		marshaled, _ := anypb.New(reply)
		response := &internalpb.RemoteAskResponse{Message: marshaled}
		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeUnknown, err)
		}
	}
}

// RemoteTell is used to send a message to an actor remotely by another actor
func (x *actorSystem) RemoteTell(ctx context.Context, stream *connect.ClientStream[internalpb.RemoteTellRequest]) (*connect.Response[internalpb.RemoteTellResponse], error) {
	logger := x.logger

	if !x.remotingEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	requestc := make(chan *internalpb.RemoteTellRequest, 1)
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(2)

	eg.Go(
		func() error {
			defer close(requestc)
			for stream.Receive() {
				select {
				case requestc <- stream.Msg():
				case <-ctx.Done():
					logger.Error(ctx.Err())
					return connect.NewError(connect.CodeCanceled, ctx.Err())
				}
			}

			if err := stream.Err(); err != nil {
				logger.Error(err)
				return connect.NewError(connect.CodeUnknown, err)
			}

			return nil
		},
	)

	eg.Go(
		func() error {
			for request := range requestc {
				receiver := request.GetRemoteMessage().GetReceiver()
				addr := address.New(receiver.GetName(), x.Name(), receiver.GetHost(), int(receiver.GetPort()))
				pidNode, exist := x.actors.GetNode(addr.String())
				if !exist {
					logger.Error(ErrAddressNotFound(addr.String()).Error())
					return ErrAddressNotFound(addr.String())
				}

				pid := pidNode.GetValue()
				if err := x.handleRemoteTell(ctx, pid, request.GetRemoteMessage()); err != nil {
					logger.Error(ErrRemoteSendFailure(err))
					return ErrRemoteSendFailure(err)
				}
			}
			return nil
		},
	)

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return connect.NewResponse(new(internalpb.RemoteTellResponse)), nil
}

// RemoteReSpawn is used the handle the re-creation of an actor from a remote host or from an api call
func (x *actorSystem) RemoteReSpawn(ctx context.Context, request *connect.Request[internalpb.RemoteReSpawnRequest]) (*connect.Response[internalpb.RemoteReSpawnResponse], error) {
	logger := x.logger

	msg := request.Msg

	if !x.remotingEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
	if remoteAddr != net.JoinHostPort(msg.GetHost(), strconv.Itoa(int(msg.GetPort()))) {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	actorAddress := address.New(msg.GetName(), x.Name(), msg.GetHost(), int(msg.GetPort()))
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if !exist {
		logger.Error(ErrAddressNotFound(actorAddress.String()).Error())
		return nil, ErrAddressNotFound(actorAddress.String())
	}

	pid := pidNode.GetValue()
	parent := NoSender
	if parentNode, ok := x.actors.Parent(pid); ok {
		parent = parentNode.GetValue()
	}

	if err := pid.Restart(ctx); err != nil {
		return nil, fmt.Errorf("failed to restart actor=%s: %w", actorAddress.String(), err)
	}

	if err := x.actors.AddNode(parent, pid); err != nil {
		return nil, err
	}

	return connect.NewResponse(new(internalpb.RemoteReSpawnResponse)), nil
}

// RemoteStop stops an actor on a remote machine
func (x *actorSystem) RemoteStop(ctx context.Context, request *connect.Request[internalpb.RemoteStopRequest]) (*connect.Response[internalpb.RemoteStopResponse], error) {
	logger := x.logger

	msg := request.Msg

	if !x.remotingEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
	if remoteAddr != net.JoinHostPort(msg.GetHost(), strconv.Itoa(int(msg.GetPort()))) {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	actorAddress := address.New(msg.GetName(), x.Name(), msg.GetHost(), int(msg.GetPort()))
	pidNode, exist := x.actors.GetNode(actorAddress.String())
	if !exist {
		logger.Error(ErrAddressNotFound(actorAddress.String()).Error())
		return nil, ErrAddressNotFound(actorAddress.String())
	}

	pid := pidNode.GetValue()
	if err := pid.Shutdown(ctx); err != nil {
		return nil, fmt.Errorf("failed to stop actor=%s: %w", actorAddress.String(), err)
	}

	return connect.NewResponse(new(internalpb.RemoteStopResponse)), nil
}

// RemoteSpawn handles the remoteSpawn call
func (x *actorSystem) RemoteSpawn(ctx context.Context, request *connect.Request[internalpb.RemoteSpawnRequest]) (*connect.Response[internalpb.RemoteSpawnResponse], error) {
	logger := x.logger

	msg := request.Msg
	if !x.remotingEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrRemotingDisabled)
	}

	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
	if remoteAddr != net.JoinHostPort(msg.GetHost(), strconv.Itoa(int(msg.GetPort()))) {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	actor, err := x.reflection.ActorFrom(msg.GetActorType())
	if err != nil {
		logger.Errorf(
			"failed to create actor=[(%s) of type (%s)] on [host=%s, port=%d]: reason: (%v)",
			msg.GetActorName(), msg.GetActorType(), msg.GetHost(), msg.GetPort(), err,
		)

		if errors.Is(err, ErrTypeNotRegistered) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, ErrTypeNotRegistered)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if _, err = x.Spawn(ctx, msg.GetActorName(), actor); err != nil {
		logger.Errorf("failed to create actor=(%s) on [host=%s, port=%d]: reason: (%v)", msg.GetActorName(), msg.GetHost(), msg.GetPort(), err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logger.Infof("actor=(%s) successfully created on [host=%s, port=%d]", msg.GetActorName(), msg.GetHost(), msg.GetPort())
	return connect.NewResponse(new(internalpb.RemoteSpawnResponse)), nil
}

// GetNodeMetric handles the GetNodeMetric request send the given node
func (x *actorSystem) GetNodeMetric(_ context.Context, request *connect.Request[internalpb.GetNodeMetricRequest]) (*connect.Response[internalpb.GetNodeMetricResponse], error) {
	if !x.clusterEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrClusterDisabled)
	}

	req := request.Msg

	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)
	if remoteAddr != req.GetNodeAddress() {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	actorCount := x.actors.Size()
	return connect.NewResponse(
		&internalpb.GetNodeMetricResponse{
			NodeRemoteAddress: remoteAddr,
			ActorsCount:       uint64(actorCount),
		},
	), nil
}

// GetKinds returns the cluster kinds
func (x *actorSystem) GetKinds(_ context.Context, request *connect.Request[internalpb.GetKindsRequest]) (*connect.Response[internalpb.GetKindsResponse], error) {
	if !x.clusterEnabled.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrClusterDisabled)
	}

	req := request.Msg
	remoteAddr := fmt.Sprintf("%s:%d", x.host, x.port)

	// routine check
	if remoteAddr != req.GetNodeAddress() {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidHost)
	}

	kinds := make([]string, len(x.clusterConfig.Kinds()))
	for i, kind := range x.clusterConfig.Kinds() {
		kinds[i] = types.TypeName(kind)
	}

	return connect.NewResponse(&internalpb.GetKindsResponse{Kinds: kinds}), nil
}

// handleRemoteAsk handles a synchronous message to another actor and expect a response.
// This block until a response is received or timed out.
func (x *actorSystem) handleRemoteAsk(ctx context.Context, to *PID, message proto.Message, timeout time.Duration) (response proto.Message, err error) {
	return Ask(ctx, to, message, timeout)
}

// handleRemoteTell handles an asynchronous message to an actor
func (x *actorSystem) handleRemoteTell(ctx context.Context, to *PID, message proto.Message) error {
	return Tell(ctx, to, message)
}

// getRootGuardian returns the system root guardian
func (x *actorSystem) getRootGuardian() *PID {
	x.locker.Lock()
	rootGuardian := x.rootGuardian
	x.locker.Unlock()
	return rootGuardian
}

// getUserGuardian returns the user guardian
func (x *actorSystem) getUserGuardian() *PID {
	x.locker.Lock()
	userGuardian := x.userGuardian
	x.locker.Unlock()
	return userGuardian
}

// getSystemGuardian returns the system guardian
func (x *actorSystem) getSystemGuardian() *PID {
	x.locker.Lock()
	systemGuardian := x.systemGuardian
	x.locker.Unlock()
	return systemGuardian
}

// getJanitor returns the system janitor
func (x *actorSystem) getJanitor() *PID {
	x.locker.Lock()
	janitor := x.janitor
	x.locker.Unlock()
	return janitor
}

// getDeadletters returns the system deadletters actor
func (x *actorSystem) getDeadletters() *PID {
	x.locker.Lock()
	deadletters := x.deadletters
	x.locker.Unlock()
	return deadletters
}

func (x *actorSystem) completeRebalancing() {
	x.rebalancing.Store(false)
}

// getPeerStateFromCache returns the peer state from the cache
func (x *actorSystem) getPeerStateFromCache(address string) (*internalpb.PeerState, error) {
	x.locker.Lock()
	value, ok := x.peersCache.Load(address)
	x.locker.Unlock()
	if !ok {
		return nil, ErrPeerNotFound
	}

	peerState := new(internalpb.PeerState)
	if err := proto.Unmarshal(value.([]byte), peerState); err != nil {
		return nil, err
	}
	return peerState, nil
}

// broadcastActor broadcast the newly (re)spawned actor into the cluster
func (x *actorSystem) broadcastActor(actor *PID) {
	if x.clusterEnabled.Load() {
		x.actorsChan <- &internalpb.ActorRef{
			ActorAddress: actor.Address().Address,
			ActorType:    types.TypeName(actor.Actor()),
		}
	}
}

// enableClustering enables clustering. When clustering is enabled remoting is also enabled to facilitate remote
// communication
func (x *actorSystem) enableClustering(ctx context.Context) error {
	if !x.clusterEnabled.Load() {
		return nil
	}

	x.logger.Info("enabling clustering...")

	if !x.remotingEnabled.Load() {
		x.logger.Error("clustering needs remoting to be enabled")
		return errors.New("clustering needs remoting to be enabled")
	}

	x.clusterNode = &discovery.Node{
		Name:          x.Name(),
		Host:          x.host,
		DiscoveryPort: x.clusterConfig.DiscoveryPort(),
		PeersPort:     x.clusterConfig.PeersPort(),
		RemotingPort:  int(x.port),
	}

	clusterEngine, err := cluster.NewEngine(
		x.Name(),
		x.clusterConfig.Discovery(),
		x.clusterNode,
		cluster.WithLogger(x.logger),
		cluster.WithPartitionsCount(x.clusterConfig.PartitionCount()),
		cluster.WithHasher(x.partitionHasher),
		cluster.WithMinimumPeersQuorum(x.clusterConfig.MinimumPeersQuorum()),
		cluster.WithReplicaCount(x.clusterConfig.ReplicaCount()),
	)
	if err != nil {
		x.logger.Errorf("failed to initialize cluster engine: %v", err)
		return err
	}

	x.logger.Info("starting cluster engine...")
	if err := clusterEngine.Start(ctx); err != nil {
		x.logger.Errorf("failed to start cluster engine: %v", err)
		return err
	}

	bootstrapChan := make(chan struct{}, 1)
	timer := time.AfterFunc(
		time.Second, func() {
			bootstrapChan <- struct{}{}
		},
	)
	<-bootstrapChan
	timer.Stop()

	x.logger.Info("cluster engine successfully started...")

	x.locker.Lock()
	x.cluster = clusterEngine
	x.clusterEventsChan = clusterEngine.Events()
	x.rebalancingChan = make(chan *cluster.Event, 1)
	for _, kind := range x.clusterConfig.Kinds() {
		x.registry.Register(kind)
		x.logger.Infof("cluster kind=(%s) registered", types.TypeName(kind))
	}
	x.locker.Unlock()

	go x.clusterEventsLoop()
	go x.replicationLoop()
	go x.peersStateLoop()
	go x.rebalancingLoop()

	x.logger.Info("clustering enabled...:)")
	return nil
}

// enableRemoting enables the remoting service to handle remote messaging
func (x *actorSystem) enableRemoting(ctx context.Context) error {
	if !x.remotingEnabled.Load() {
		return nil
	}
	x.logger.Info("enabling remoting...")
	remotingServicePath, remotingServiceHandler := internalpbconnect.NewRemotingServiceHandler(x)
	clusterServicePath, clusterServiceHandler := internalpbconnect.NewClusterServiceHandler(x)

	mux := nethttp.NewServeMux()
	mux.Handle(remotingServicePath, remotingServiceHandler)
	mux.Handle(clusterServicePath, clusterServiceHandler)

	x.locker.Lock()
	defer x.locker.Unlock()

	// configure the appropriate server
	if err := x.configureServer(ctx, mux); err != nil {
		x.locker.Unlock()
		x.logger.Error(fmt.Errorf("failed enable remoting: %w", err))
		return err
	}

	go func() {
		if err := x.startHTTPServer(); err != nil {
			if !errors.Is(err, nethttp.ErrServerClosed) {
				x.logger.Panic(fmt.Errorf("failed to start remoting service: %w", err))
			}
		}
	}()

	x.remoting = NewRemoting()
	x.logger.Info("remoting enabled...:)")
	return nil
}

// reset the actor system
func (x *actorSystem) reset() {
	x.actors.Reset()
}

// replicationLoop publishes newly created actor into the cluster when cluster is enabled
func (x *actorSystem) replicationLoop() {
	for actor := range x.actorsChan {
		// never replicate system actors because there are specific to the
		// started node
		if isReservedName(actor.GetActorAddress().GetName()) {
			continue
		}
		if x.InCluster() {
			ctx := context.Background()
			if err := x.cluster.PutActor(ctx, actor); err != nil {
				x.logger.Panic(err.Error())
			}
		}
	}
}

// clusterEventsLoop listens to cluster events and send them to the event streams
func (x *actorSystem) clusterEventsLoop() {
	for event := range x.clusterEventsChan {
		if x.InCluster() {
			if event != nil && event.Payload != nil {
				// push the event to the event stream
				message, _ := event.Payload.UnmarshalNew()
				if x.eventsStream != nil {
					x.logger.Debugf("node=(%s) publishing cluster event=(%s)....", x.name, event.Type)
					x.eventsStream.Publish(eventsTopic, message)
					x.logger.Debugf("cluster event=(%s) successfully published by node=(%s)", event.Type, x.name)
				}
				x.rebalancingChan <- event
			}
		}
	}
}

// peersStateLoop fetches the cluster peers' PeerState and update the node peersCache
func (x *actorSystem) peersStateLoop() {
	x.logger.Info("peers state synchronization has started...")
	ticker := time.NewTicker(x.peersStateLoopInterval)
	tickerStopSig := make(chan types.Unit, 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				eg, ctx := errgroup.WithContext(context.Background())

				peersChan := make(chan *cluster.Peer)

				eg.Go(
					func() error {
						defer close(peersChan)
						peers, err := x.cluster.Peers(ctx)
						if err != nil {
							return err
						}

						for _, peer := range peers {
							select {
							case peersChan <- peer:
							case <-ctx.Done():
								return ctx.Err()
							}
						}
						return nil
					},
				)

				eg.Go(
					func() error {
						for peer := range peersChan {
							if err := x.processPeerState(ctx, peer); err != nil {
								return err
							}
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								// pass
							}
						}
						return nil
					},
				)

				if err := eg.Wait(); err != nil {
					x.logger.Error(err)
					// TODO: stop or panic
				}

			case <-x.clusterSyncStopSig:
				tickerStopSig <- types.Unit{}
				return
			}
		}
	}()

	<-tickerStopSig
	ticker.Stop()
	x.logger.Info("peers state synchronization has stopped...")
}

// rebalancingLoop help perform cluster rebalancing
func (x *actorSystem) rebalancingLoop() {
	for event := range x.rebalancingChan {
		if x.InCluster() {
			// get peer state
			peerState, err := x.nodeLeftStateFromEvent(event)
			if err != nil {
				x.logger.Error(err)
				continue
			}

			ctx := context.Background()
			if !x.shouldRebalance(ctx, peerState) {
				continue
			}

			if x.rebalancing.Load() {
				x.logger.Debugf("%s on %s rebalancing already ongoing...", x.Name(), x.clusterNode.PeersAddress())
				continue
			}
			x.logger.Infof("%s on %s starting rebalancing...", x.Name(), x.clusterNode.PeersAddress())
			x.rebalancing.Store(true)
			message := &internalpb.Rebalance{PeerState: peerState}
			if err := x.systemGuardian.Tell(ctx, x.rebalancer, message); err != nil {
				x.logger.Error(err)
			}
		}
	}
}

// shouldRebalance returns true when the current can perform the cluster rebalancing
func (x *actorSystem) shouldRebalance(ctx context.Context, peerState *internalpb.PeerState) bool {
	return !(peerState == nil ||
		proto.Equal(peerState, new(internalpb.PeerState)) ||
		len(peerState.GetActors()) == 0 ||
		!x.cluster.IsLeader(ctx))
}

// processPeerState processes a given peer synchronization record.
func (x *actorSystem) processPeerState(ctx context.Context, peer *cluster.Peer) error {
	peerAddress := net.JoinHostPort(peer.Host, strconv.Itoa(peer.Port))

	x.logger.Infof("processing peer sync:(%s)", peerAddress)
	peerState, err := x.cluster.GetState(ctx, peerAddress)
	if err != nil {
		if errors.Is(err, cluster.ErrPeerSyncNotFound) {
			return nil
		}
		x.logger.Error(err)
		return err
	}

	x.logger.Debugf("peer (%s) actors count (%d)", peerAddress, len(peerState.GetActors()))

	// no need to handle the error
	bytea, err := proto.Marshal(peerState)
	if err != nil {
		x.logger.Error(err)
		return err
	}

	x.peersCache.Store(peerAddress, bytea)
	x.logger.Infof("peer sync(%s) successfully processed", peerAddress)
	return nil
}

// configPID constructs a PID provided the actor name and the actor kind
// this is a utility function used when spawning actors
func (x *actorSystem) configPID(ctx context.Context, name string, actor Actor, opts ...SpawnOption) (*PID, error) {
	addr := x.actorAddress(name)
	if err := addr.Validate(); err != nil {
		return nil, err
	}

	// define the pid options
	// pid inherit the actor system settings defined during instantiation
	pidOpts := []pidOption{
		withInitMaxRetries(x.actorInitMaxRetries),
		withCustomLogger(x.logger),
		withActorSystem(x),
		withEventsStream(x.eventsStream),
		withInitTimeout(x.actorInitTimeout),
	}

	spawnConfig := newSpawnConfig(opts...)
	// set the mailbox option
	if spawnConfig.mailbox != nil {
		pidOpts = append(pidOpts, withMailbox(spawnConfig.mailbox))
	}

	// set the supervisor strategies when defined
	if len(spawnConfig.supervisorStrategies) != 0 {
		pidOpts = append(pidOpts, withSupervisorStrategies(spawnConfig.supervisorStrategies...))
	}

	// enable stash
	if x.stashEnabled {
		pidOpts = append(pidOpts, withStash())
	}

	// disable passivation for system actor
	if isReservedName(name) {
		pidOpts = append(pidOpts, withPassivationDisabled())
	} else {
		pidOpts = append(pidOpts, withPassivationAfter(x.expireActorAfter))
	}

	pid, err := newPID(
		ctx,
		addr,
		actor,
		pidOpts...,
	)

	if err != nil {
		return nil, err
	}
	return pid, nil
}

// tree returns the actors tree
func (x *actorSystem) tree() *pidTree {
	return x.actors
}

// getCluster returns the cluster engine
func (x *actorSystem) getCluster() cluster.Interface {
	return x.cluster
}

// reservedName returns reserved actor's name
func (x *actorSystem) reservedName(nameType nameType) string {
	return systemNames[nameType]
}

// actorAddress returns the actor path provided the actor name
func (x *actorSystem) actorAddress(name string) *address.Address {
	return address.New(name, x.name, x.host, int(x.port))
}

// startHTTPServer starts the appropriate http server
func (x *actorSystem) startHTTPServer() error {
	return x.server.Serve(x.listener)
}

// shutdownHTTPServer stops the appropriate http server
func (x *actorSystem) shutdownHTTPServer(ctx context.Context) error {
	return x.server.Shutdown(ctx)
}

// configureServer configure the various http server and listeners based upon the various settings
func (x *actorSystem) configureServer(ctx context.Context, mux *nethttp.ServeMux) error {
	hostPort := net.JoinHostPort(x.host, strconv.Itoa(int(x.port)))
	httpServer := getServer(ctx, hostPort)
	// create a tcp listener
	lnr, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}

	// set the http server
	x.server = httpServer
	// For gRPC clients, it's convenient to support HTTP/2 without TLS.
	x.server.Handler = h2c.NewHandler(
		mux, &http2.Server{
			IdleTimeout: 1200 * time.Second,
		},
	)
	// set the non-secure http server
	x.listener = lnr
	return nil
}

// nodeLeftStateFromEvent returns the node left state from the cluster event
func (x *actorSystem) nodeLeftStateFromEvent(event *cluster.Event) (*internalpb.PeerState, error) {
	if event.Type != cluster.NodeLeft {
		return nil, nil
	}
	nodeLeft := new(goaktpb.NodeLeft)
	if err := event.Payload.UnmarshalTo(nodeLeft); err != nil {
		return nil, err
	}

	return x.getPeerStateFromCache(nodeLeft.GetAddress())
}

// setHostPort sets the host and port
func (x *actorSystem) setHostPort() error {
	var err error
	// combine host and port into an hostPort string
	hostPort := net.JoinHostPort(x.host, strconv.Itoa(int(x.port)))
	x.host, err = tcp.GetBindIP(hostPort)
	if err != nil {
		return err
	}
	return nil
}

// spawnRootGuardian creates the root guardian
func (x *actorSystem) spawnRootGuardian(ctx context.Context) error {
	var err error
	actorName := x.reservedName(rootGuardianType)
	x.rootGuardian, err = x.configPID(ctx, actorName, newRootGuardian())
	if err != nil {
		return fmt.Errorf("actor=%s failed to start root guardian: %w", actorName, err)
	}

	// rootGuardian is the root node of the actors tree
	_ = x.actors.AddNode(NoSender, x.rootGuardian)
	return nil
}

// spawnSystemGuardian creates the system guardian
func (x *actorSystem) spawnSystemGuardian(ctx context.Context) error {
	var err error
	actorName := x.reservedName(systemGuardianType)
	x.systemGuardian, err = x.configPID(ctx,
		actorName,
		newSystemGuardian(),
		WithSupervisorStrategies(
			NewSupervisorStrategy(PanicError{}, NewStopDirective()),
		),
	)
	if err != nil {
		return fmt.Errorf("actor=%s failed to start system guardian: %w", actorName, err)
	}

	// systemGuardian is a child actor of the rootGuardian actor
	_ = x.actors.AddNode(x.rootGuardian, x.systemGuardian)
	return nil
}

// spawnUserGuardian creates the user guardian
func (x *actorSystem) spawnUserGuardian(ctx context.Context) error {
	var err error
	actorName := x.reservedName(userGuardianType)
	x.userGuardian, err = x.configPID(ctx,
		actorName,
		newUserGuardian(),
		WithSupervisorStrategies(
			NewSupervisorStrategy(PanicError{}, NewStopDirective()),
		))
	if err != nil {
		return fmt.Errorf("actor=%s failed to start user guardian: %w", actorName, err)
	}

	// userGuardian is a child actor of the rootGuardian actor
	_ = x.actors.AddNode(x.rootGuardian, x.userGuardian)
	return nil
}

// spawnRebalancer creates the cluster rebalancer
func (x *actorSystem) spawnJanitor(ctx context.Context) error {
	var err error
	actorName := x.reservedName(janitorType)
	x.janitor, err = x.configPID(ctx,
		actorName,
		newJanitor(),
		WithSupervisorStrategies(
			NewSupervisorStrategy(PanicError{}, NewRestartDirective()),
		),
	)
	if err != nil {
		return fmt.Errorf("actor=%s failed to start the janitor: %w", actorName, err)
	}

	// the janitor is a child actor of the system guardian
	_ = x.actors.AddNode(x.systemGuardian, x.janitor)
	return nil
}

// spawnRebalancer creates the cluster rebalancer
func (x *actorSystem) spawnRebalancer(ctx context.Context) error {
	var err error
	actorName := x.reservedName(rebalancerType)
	x.rebalancer, err = x.configPID(ctx,
		actorName,
		newRebalancer(x.reflection),
		WithSupervisorStrategies(
			NewSupervisorStrategy(PanicError{}, NewRestartDirective()),
		),
	)
	if err != nil {
		return fmt.Errorf("actor=%s failed to start cluster rebalancer: %w", actorName, err)
	}

	// the rebalancer is a child actor of the system guardian
	_ = x.actors.AddNode(x.systemGuardian, x.rebalancer)
	return nil
}

// spawnDeadletters creates the deadletters synthetic actor
func (x *actorSystem) spawnDeadletters(ctx context.Context) error {
	var err error
	actorName := x.reservedName(deadletters)
	x.deadletters, err = x.configPID(ctx,
		actorName,
		newDeadLetters(),
		WithSupervisorStrategies(
			NewSupervisorStrategy(PanicError{}, NewResumeDirective()),
			NewSupervisorStrategy(&runtime.PanicNilError{}, NewResumeDirective()),
		),
	)
	if err != nil {
		return fmt.Errorf("actor=%s failed to start deadletters: %w", actorName, err)
	}

	// the deadletters is a child actor of the system guardian
	_ = x.actors.AddNode(x.systemGuardian, x.deadletters)
	return nil
}

// checkSpawnPreconditions make sure before an actor is created some pre-conditions are checks
func (x *actorSystem) checkSpawnPreconditions(ctx context.Context, actorName string, kind Actor) error {
	// check the existence of the actor given the kind prior to creating it
	if x.clusterEnabled.Load() {
		existed, err := x.cluster.GetActor(ctx, actorName)
		if err != nil {
			if errors.Is(err, cluster.ErrActorNotFound) {
				return nil
			}
			return err
		}

		if existed.GetActorType() == types.TypeName(kind) {
			return ErrActorAlreadyExists(actorName)
		}
	}

	return nil
}

// cleanupCluster cleans up the cluster
func (x *actorSystem) cleanupCluster(ctx context.Context, actorNames []string) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, actorName := range actorNames {
		actorName := actorName
		eg.Go(func() error {
			if err := x.cluster.RemoveActor(ctx, actorName); err != nil {
				logger.Errorf("failed to remove [actor=%s] from cluster: %v", actorName, err)
				return err
			}
			logger.Infof("[actor=%s] removed from cluster", actorName)
			return nil
		})
	}
	return eg.Wait()
}

func isReservedName(name string) bool {
	return strings.HasPrefix(name, systemNamePrefix)
}

// getServer creates an instance of http server
func getServer(ctx context.Context, address string) *nethttp.Server {
	return &nethttp.Server{
		Addr: address,
		// The maximum duration for reading the entire request, including the body.
		// It’s implemented in net/http by calling SetReadDeadline immediately after Accept
		// ReadTimeout := handler_timeout + ReadHeaderTimeout + wiggle_room
		ReadTimeout: 3 * time.Second,
		// ReadHeaderTimeout is the amount of time allowed to read request headers
		ReadHeaderTimeout: time.Second,
		// WriteTimeout is the maximum duration before timing out writes of the response.
		// It is reset whenever a new request’s header is read.
		// This effectively covers the lifetime of the ServeHTTP handler stack
		WriteTimeout: time.Second,
		// IdleTimeout is the maximum amount of time to wait for the next request when keep-alive are enabled.
		// If IdleTimeout is zero, the value of ReadTimeout is used. Not relevant to request timeouts
		IdleTimeout: 1200 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}
