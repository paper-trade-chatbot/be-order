package service

import (
	"context"
	"fmt"

	"github.com/paper-trade-chatbot/be-common/logging"

	"github.com/paper-trade-chatbot/be-common/config"
	"github.com/paper-trade-chatbot/be-order/service/member"
	"github.com/paper-trade-chatbot/be-order/service/position"
	"github.com/paper-trade-chatbot/be-order/service/product"
	memberGrpc "github.com/paper-trade-chatbot/be-proto/member"
	positionGrpc "github.com/paper-trade-chatbot/be-proto/position"
	productGrpc "github.com/paper-trade-chatbot/be-proto/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var Impl ServiceImpl
var (
	MemberServiceHost    = config.GetString("MEMBER_GRPC_HOST")
	MemberServerGRpcPort = config.GetString("MEMBER_GRPC_PORT")
	memberServiceConn    *grpc.ClientConn

	ProductServiceHost    = config.GetString("PRODUCT_GRPC_HOST")
	ProductServerGRpcPort = config.GetString("PRODUCT_GRPC_PORT")
	productServiceConn    *grpc.ClientConn

	PositionServiceHost    = config.GetString("POSITION_GRPC_HOST")
	PositionServerGRpcPort = config.GetString("POSITION_GRPC_PORT")
	positionServiceConn    *grpc.ClientConn
)

type ServiceImpl struct {
	MemberIntf   member.MemberIntf
	ProductIntf  product.ProductIntf
	PositionIntf position.PositionIntf
}

func GrpcDial(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(20*1024*1024),
		grpc.MaxCallSendMsgSize(20*1024*1024)), grpc.WithUnaryInterceptor(clientInterceptor))
}

func Initialize(ctx context.Context) {

	var err error

	addr := MemberServiceHost + ":" + MemberServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	memberServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	memberConn := memberGrpc.NewMemberServiceClient(memberServiceConn)
	Impl.MemberIntf = member.New(memberConn)

	addr = ProductServiceHost + ":" + ProductServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	productServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	productConn := productGrpc.NewProductServiceClient(productServiceConn)
	Impl.ProductIntf = product.New(productConn)

	addr = PositionServiceHost + ":" + PositionServerGRpcPort
	fmt.Println("dial to order grpc server...", addr)
	positionServiceConn, err = GrpcDial(addr)
	if err != nil {
		fmt.Println("Can not connect to gRPC server:", err)
	}
	fmt.Println("dial done")
	positionConn := positionGrpc.NewPositionServiceClient(positionServiceConn)
	Impl.PositionIntf = position.New(positionConn)
}

func Finalize(ctx context.Context) {
	memberServiceConn.Close()
	productServiceConn.Close()
	positionServiceConn.Close()
}

func clientInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	requestId, _ := ctx.Value(logging.ContextKeyRequestId).(string)
	account, _ := ctx.Value(logging.ContextKeyAccount).(string)

	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		logging.ContextKeyRequestId: requestId,
		logging.ContextKeyAccount:   account,
	}))

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		fmt.Println("clientInterceptor err:", err.Error())
	}

	return err
}
