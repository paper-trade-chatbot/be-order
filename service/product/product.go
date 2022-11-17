package product

import (
	"context"

	"github.com/paper-trade-chatbot/be-order/config"
	"github.com/paper-trade-chatbot/be-proto/product"
)

type ProductIntf interface {
	GetExchange(ctx context.Context, in *product.GetExchangeReq) (*product.GetExchangeRes, error)
	GetExchanges(ctx context.Context, in *product.GetExchangesReq) (*product.GetExchangesRes, error)
	CreateProduct(ctx context.Context, in *product.CreateProductReq) (*product.CreateProductRes, error)
	GetProduct(ctx context.Context, in *product.GetProductReq) (*product.GetProductRes, error)
	GetProducts(ctx context.Context, in *product.GetProductsReq) (*product.GetProductsRes, error)
	ModifyProduct(ctx context.Context, in *product.ModifyProductReq) (*product.ModifyProductRes, error)
	DeleteProduct(ctx context.Context, in *product.DeleteProductReq) (*product.DeleteProductRes, error)
}

type ProductImpl struct {
	ProductClient product.ProductServiceClient
}

var (
	ProductServiceHost    = config.GetString("PRODUCT_GRPC_HOST")
	ProductServerGRpcPort = config.GetString("PRODUCT_GRPC_PORT")
)

func New(productClient product.ProductServiceClient) ProductIntf {
	return &ProductImpl{
		ProductClient: productClient,
	}
}

func (impl *ProductImpl) GetExchange(ctx context.Context, in *product.GetExchangeReq) (*product.GetExchangeRes, error) {
	return impl.ProductClient.GetExchange(ctx, in)
}

func (impl *ProductImpl) GetExchanges(ctx context.Context, in *product.GetExchangesReq) (*product.GetExchangesRes, error) {
	return impl.ProductClient.GetExchanges(ctx, in)
}

func (impl *ProductImpl) CreateProduct(ctx context.Context, in *product.CreateProductReq) (*product.CreateProductRes, error) {
	return impl.ProductClient.CreateProduct(ctx, in)
}

func (impl *ProductImpl) GetProduct(ctx context.Context, in *product.GetProductReq) (*product.GetProductRes, error) {
	return impl.ProductClient.GetProduct(ctx, in)
}

func (impl *ProductImpl) GetProducts(ctx context.Context, in *product.GetProductsReq) (*product.GetProductsRes, error) {
	return impl.ProductClient.GetProducts(ctx, in)
}

func (impl *ProductImpl) ModifyProduct(ctx context.Context, in *product.ModifyProductReq) (*product.ModifyProductRes, error) {
	return impl.ProductClient.ModifyProduct(ctx, in)
}

func (impl *ProductImpl) DeleteProduct(ctx context.Context, in *product.DeleteProductReq) (*product.DeleteProductRes, error) {
	return impl.ProductClient.DeleteProduct(ctx, in)
}
