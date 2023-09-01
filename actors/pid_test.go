package actors

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/log"
	testpb "github.com/tochemey/goakt/test/data/pb/v1"
	"google.golang.org/protobuf/proto"
)

const (
	receivingDelay   = 1 * time.Second
	receivingTimeout = 100 * time.Millisecond
	passivateAfter   = 200 * time.Millisecond
)

func TestActorReceive(t *testing.T) {
	ctx := context.TODO()

	// create the actor path
	actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))

	// create the actor ref
	pid := newPID(
		ctx,
		actorPath,
		NewTester(),
		withInitMaxRetries(1),
		withCustomLogger(log.DefaultLogger),
		withSendReplyTimeout(receivingTimeout))

	assert.NotNil(t, pid)
	// let us send 10 public to the actor
	count := 10
	for i := 0; i < count; i++ {
		recvContext := &receiveContext{
			ctx:            ctx,
			message:        new(testpb.TestSend),
			sender:         NoSender,
			recipient:      pid,
			mu:             sync.Mutex{},
			isAsyncMessage: true,
		}

		pid.doReceive(recvContext)
	}
	assert.EqualValues(t, count, pid.ReceivedCount(ctx))
	// stop the actor
	err := pid.Shutdown(ctx)
	assert.NoError(t, err)
}
func TestActorWithPassivation(t *testing.T) {
	ctx := context.TODO()
	// create a Ping actor
	opts := []pidOption{
		withInitMaxRetries(1),
		withPassivationAfter(passivateAfter),
		withSendReplyTimeout(receivingTimeout),
	}

	// create the actor path
	actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))
	pid := newPID(ctx, actorPath, NewTester(), opts...)
	assert.NotNil(t, pid)

	// let us sleep for some time to make the actor idle
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		time.Sleep(receivingDelay)
		wg.Done()
	}()
	// block until timer is up
	wg.Wait()
	// let us send a message to the actor
	err := Tell(ctx, pid, new(testpb.TestSend))
	assert.Error(t, err)
	assert.EqualError(t, err, ErrNotReady.Error())
}
func TestActorWithReply(t *testing.T) {
	t.Run("with happy path", func(t *testing.T) {
		ctx := context.TODO()
		// create a Ping actor
		opts := []pidOption{
			withInitMaxRetries(1),
			withCustomLogger(log.DiscardLogger),
		}

		// create the actor path
		actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))
		pid := newPID(ctx, actorPath, NewTester(), opts...)
		assert.NotNil(t, pid)

		actual, err := Ask(ctx, pid, new(testpb.TestReply), receivingTimeout)
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		expected := &testpb.Reply{Content: "received message"}
		assert.True(t, proto.Equal(expected, actual))
		// stop the actor
		err = pid.Shutdown(ctx)
		assert.NoError(t, err)
	})
	t.Run("with timeout", func(t *testing.T) {
		ctx := context.TODO()
		// create a Ping actor
		opts := []pidOption{
			withInitMaxRetries(1),
			withCustomLogger(log.DiscardLogger),
		}

		// create the actor path
		actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))
		pid := newPID(ctx, actorPath, NewTester(), opts...)
		assert.NotNil(t, pid)

		actual, err := Ask(ctx, pid, new(testpb.TestSend), receivingTimeout)
		assert.Error(t, err)
		assert.EqualError(t, err, ErrRequestTimeout.Error())
		assert.Nil(t, actual)
		// stop the actor
		err = pid.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
func TestActorRestart(t *testing.T) {
	t.Run("restart a stopped actor", func(t *testing.T) {
		ctx := context.TODO()

		// create a Ping actor
		actor := NewTester()
		assert.NotNil(t, actor)

		// create the actor path
		actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))
		// create the actor ref
		pid := newPID(ctx, actorPath, actor,
			withInitMaxRetries(1),
			withPassivationAfter(10*time.Second),
			withCustomLogger(log.DiscardLogger),
			withSendReplyTimeout(receivingTimeout))
		assert.NotNil(t, pid)

		// stop the actor
		err := pid.Shutdown(ctx)
		assert.NoError(t, err)

		time.Sleep(time.Second)

		// let us send a message to the actor
		err = Tell(ctx, pid, new(testpb.TestSend))
		assert.Error(t, err)
		assert.EqualError(t, err, ErrNotReady.Error())

		// restart the actor
		err = pid.Restart(ctx)
		assert.NoError(t, err)
		assert.True(t, pid.IsRunning())
		// let us send 10 public to the actor
		count := 10
		for i := 0; i < count; i++ {
			err = Tell(ctx, pid, new(testpb.TestSend))
			assert.NoError(t, err)
		}
		assert.EqualValues(t, count, pid.ReceivedCount(ctx))
		// stop the actor
		err = pid.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("restart an actor", func(t *testing.T) {
		ctx := context.TODO()

		// create a Ping actor
		actor := NewTester()
		assert.NotNil(t, actor)
		// create the actor path
		actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))

		// create the actor ref
		pid := newPID(ctx, actorPath, actor,
			withInitMaxRetries(1),
			withPassivationAfter(passivateAfter),
			withCustomLogger(log.DiscardLogger),
			withSendReplyTimeout(receivingTimeout))
		assert.NotNil(t, pid)
		// let us send 10 public to the actor
		count := 10
		for i := 0; i < count; i++ {
			err := Tell(ctx, pid, new(testpb.TestSend))
			assert.NoError(t, err)
		}
		assert.EqualValues(t, count, pid.ReceivedCount(ctx))

		// restart the actor
		err := pid.Restart(ctx)
		assert.NoError(t, err)
		assert.True(t, pid.IsRunning())
		// let us send 10 public to the actor
		for i := 0; i < count; i++ {
			err = Tell(ctx, pid, new(testpb.TestSend))
			assert.NoError(t, err)
		}
		assert.EqualValues(t, count, pid.ReceivedCount(ctx))
		// stop the actor
		err = pid.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
func TestChildActor(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// create a test context
		ctx := context.TODO()
		// create the actor path
		actorPath := NewPath("Parent", NewAddress(protocol, "sys", "host", 1))

		// create the parent actor
		parent := newPID(ctx, actorPath,
			NewMonitor(),
			withInitMaxRetries(1),
			withCustomLogger(log.DiscardLogger),
			withSendReplyTimeout(receivingTimeout))
		assert.NotNil(t, parent)

		// create the child actor
		child, err := parent.SpawnChild(ctx, "Child", NewMonitored())
		assert.NoError(t, err)
		assert.NotNil(t, child)

		assert.Len(t, parent.Children(ctx), 1)
		// let us send 10 public to the actors
		count := 10
		for i := 0; i < count; i++ {
			assert.NoError(t, Tell(ctx, parent, new(testpb.TestSend)))
			assert.NoError(t, Tell(ctx, child, new(testpb.TestSend)))
		}
		assert.EqualValues(t, count, parent.ReceivedCount(ctx))
		assert.EqualValues(t, count, child.ReceivedCount(ctx))
		//stop the actor
		err = parent.Shutdown(ctx)
		assert.NoError(t, err)
	})
	t.Run("test child panic with stop as default strategy", func(t *testing.T) {
		// create a test context
		ctx := context.TODO()
		// create the actor path
		actorPath := NewPath("Parent", NewAddress(protocol, "sys", "host", 1))

		// create the parent actor
		parent := newPID(ctx,
			actorPath,
			NewMonitor(),
			withInitMaxRetries(1),
			withCustomLogger(log.DiscardLogger),
			withPassivationDisabled(),
			withSendReplyTimeout(receivingTimeout))
		assert.NotNil(t, parent)

		// create the child actor
		child, err := parent.SpawnChild(ctx, "Child", NewMonitored())
		assert.NoError(t, err)
		assert.NotNil(t, child)

		assert.Len(t, parent.Children(ctx), 1)
		// send a test panic message to the actor
		assert.NoError(t, Tell(ctx, child, new(testpb.TestPanic)))

		// wait for the child to properly shutdown
		time.Sleep(time.Second)

		// assert the actor state
		assert.False(t, child.IsRunning())
		assert.Len(t, parent.Children(ctx), 0)

		//stop the actor
		err = parent.Shutdown(ctx)
		assert.NoError(t, err)
	})
	t.Run("test child panic with restart as default strategy", func(t *testing.T) {
		// create a test context
		ctx := context.TODO()

		logger := log.New(log.DebugLevel, os.Stdout)
		// create the actor path
		actorPath := NewPath("Parent", NewAddress(protocol, "sys", "host", 1))
		// create the parent actor
		parent := newPID(ctx,
			actorPath,
			NewMonitor(),
			withInitMaxRetries(1),
			withCustomLogger(logger),
			withPassivationDisabled(),
			withSupervisorStrategy(RestartDirective),
			withSendReplyTimeout(receivingTimeout))
		assert.NotNil(t, parent)

		// create the child actor
		child, err := parent.SpawnChild(ctx, "Child", NewMonitored())
		assert.NoError(t, err)
		assert.NotNil(t, child)

		assert.Len(t, parent.Children(ctx), 1)
		// send a test panic message to the actor
		assert.NoError(t, Tell(ctx, child, new(testpb.TestPanic)))

		// wait for the child to properly shutdown
		time.Sleep(time.Second)

		// assert the actor state
		assert.True(t, child.IsRunning())
		require.Len(t, parent.Children(ctx), 1)

		//stop the actor
		err = parent.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
func TestActorBehavior(t *testing.T) {
	ctx := context.TODO()
	// create a Ping actor
	opts := []pidOption{
		withInitMaxRetries(1),
		withCustomLogger(log.DiscardLogger),
	}

	// create the actor path
	actor := &UserActor{}
	actorPath := NewPath("Test", NewAddress(protocol, "sys", "host", 1))
	pid := newPID(ctx, actorPath, actor, opts...)
	require.NotNil(t, pid)

	// send Login
	var expected proto.Message
	success, err := Ask(ctx, pid, new(testpb.TestLogin), receivingTimeout)
	require.NoError(t, err)
	require.NotNil(t, success)
	expected = &testpb.TestLoginSuccess{}
	require.True(t, proto.Equal(expected, success))

	// send a reply message
	ready, err := Ask(ctx, pid, new(testpb.TestReadiness), receivingTimeout)
	require.NoError(t, err)
	require.NotNil(t, ready)
	expected = &testpb.TestReady{}
	require.True(t, proto.Equal(expected, ready))

	// send a ready message
	reply, err := Ask(ctx, pid, new(testpb.TestReply), receivingTimeout)
	require.NoError(t, err)
	require.NotNil(t, reply)

	expected = &testpb.Reply{}
	require.True(t, proto.Equal(expected, reply))

	// stop the actor
	err = pid.Shutdown(ctx)
	assert.NoError(t, err)
}
