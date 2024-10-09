// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: internal/cluster.proto

package internalpbconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	internalpb "github.com/zyreio/goakt/v2/internal/internalpb"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// ClusterServiceName is the fully-qualified name of the ClusterService service.
	ClusterServiceName = "internalpb.ClusterService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ClusterServiceGetNodeMetricProcedure is the fully-qualified name of the ClusterService's
	// GetNodeMetric RPC.
	ClusterServiceGetNodeMetricProcedure = "/internalpb.ClusterService/GetNodeMetric"
	// ClusterServiceGetKindsProcedure is the fully-qualified name of the ClusterService's GetKinds RPC.
	ClusterServiceGetKindsProcedure = "/internalpb.ClusterService/GetKinds"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	clusterServiceServiceDescriptor             = internalpb.File_internal_cluster_proto.Services().ByName("ClusterService")
	clusterServiceGetNodeMetricMethodDescriptor = clusterServiceServiceDescriptor.Methods().ByName("GetNodeMetric")
	clusterServiceGetKindsMethodDescriptor      = clusterServiceServiceDescriptor.Methods().ByName("GetKinds")
)

// ClusterServiceClient is a client for the internalpb.ClusterService service.
type ClusterServiceClient interface {
	// GetNodeMetric returns the node metric
	GetNodeMetric(context.Context, *connect.Request[internalpb.GetNodeMetricRequest]) (*connect.Response[internalpb.GetNodeMetricResponse], error)
	// GetKinds returns the list of cluster kinds
	GetKinds(context.Context, *connect.Request[internalpb.GetKindsRequest]) (*connect.Response[internalpb.GetKindsResponse], error)
}

// NewClusterServiceClient constructs a client for the internalpb.ClusterService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewClusterServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ClusterServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &clusterServiceClient{
		getNodeMetric: connect.NewClient[internalpb.GetNodeMetricRequest, internalpb.GetNodeMetricResponse](
			httpClient,
			baseURL+ClusterServiceGetNodeMetricProcedure,
			connect.WithSchema(clusterServiceGetNodeMetricMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getKinds: connect.NewClient[internalpb.GetKindsRequest, internalpb.GetKindsResponse](
			httpClient,
			baseURL+ClusterServiceGetKindsProcedure,
			connect.WithSchema(clusterServiceGetKindsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// clusterServiceClient implements ClusterServiceClient.
type clusterServiceClient struct {
	getNodeMetric *connect.Client[internalpb.GetNodeMetricRequest, internalpb.GetNodeMetricResponse]
	getKinds      *connect.Client[internalpb.GetKindsRequest, internalpb.GetKindsResponse]
}

// GetNodeMetric calls internalpb.ClusterService.GetNodeMetric.
func (c *clusterServiceClient) GetNodeMetric(ctx context.Context, req *connect.Request[internalpb.GetNodeMetricRequest]) (*connect.Response[internalpb.GetNodeMetricResponse], error) {
	return c.getNodeMetric.CallUnary(ctx, req)
}

// GetKinds calls internalpb.ClusterService.GetKinds.
func (c *clusterServiceClient) GetKinds(ctx context.Context, req *connect.Request[internalpb.GetKindsRequest]) (*connect.Response[internalpb.GetKindsResponse], error) {
	return c.getKinds.CallUnary(ctx, req)
}

// ClusterServiceHandler is an implementation of the internalpb.ClusterService service.
type ClusterServiceHandler interface {
	// GetNodeMetric returns the node metric
	GetNodeMetric(context.Context, *connect.Request[internalpb.GetNodeMetricRequest]) (*connect.Response[internalpb.GetNodeMetricResponse], error)
	// GetKinds returns the list of cluster kinds
	GetKinds(context.Context, *connect.Request[internalpb.GetKindsRequest]) (*connect.Response[internalpb.GetKindsResponse], error)
}

// NewClusterServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewClusterServiceHandler(svc ClusterServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	clusterServiceGetNodeMetricHandler := connect.NewUnaryHandler(
		ClusterServiceGetNodeMetricProcedure,
		svc.GetNodeMetric,
		connect.WithSchema(clusterServiceGetNodeMetricMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	clusterServiceGetKindsHandler := connect.NewUnaryHandler(
		ClusterServiceGetKindsProcedure,
		svc.GetKinds,
		connect.WithSchema(clusterServiceGetKindsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/internalpb.ClusterService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ClusterServiceGetNodeMetricProcedure:
			clusterServiceGetNodeMetricHandler.ServeHTTP(w, r)
		case ClusterServiceGetKindsProcedure:
			clusterServiceGetKindsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedClusterServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedClusterServiceHandler struct{}

func (UnimplementedClusterServiceHandler) GetNodeMetric(context.Context, *connect.Request[internalpb.GetNodeMetricRequest]) (*connect.Response[internalpb.GetNodeMetricResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("internalpb.ClusterService.GetNodeMetric is not implemented"))
}

func (UnimplementedClusterServiceHandler) GetKinds(context.Context, *connect.Request[internalpb.GetKindsRequest]) (*connect.Response[internalpb.GetKindsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("internalpb.ClusterService.GetKinds is not implemented"))
}
