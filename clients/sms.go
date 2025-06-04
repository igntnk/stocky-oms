package clients

import (
	"context"
	"github.com/igntnk/stocky-2pc-controller/protobufs/sms_pb"
	"github.com/igntnk/stocky-oms/models"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SMSClient interface {
	// Product methods
	CreateProduct(ctx context.Context, storeCost float64) (string, error)
	DeleteProduct(ctx context.Context, uuid string) (string, error)
	SetProductCost(ctx context.Context, uuid string, cost float32) (string, error)
	SetProductAmount(ctx context.Context, uuid string, amount float32) (string, error)
	GetProductAmount(ctx context.Context, uuid string) (float32, error)
	RemoveCoupleProducts(ctx context.Context, req []models.ProductWriteOffRequest) ([]string, error)
	WriteOnCoupleProducts(ctx context.Context, req []models.ProductWriteOffRequest) ([]string, error)

	// Supply methods
	CreateSupply(
		ctx context.Context,
		supplyCost float32,
		desiredDate, comment, responsibleUser string,
		products []*SupplyProduct,
	) (string, error)
	DeleteSupply(ctx context.Context, uuid string) (string, error)
	UpdateSupplyInfo(
		ctx context.Context,
		uuid, comment, desiredDate, status, responsibleUser string,
		cost float32,
	) (string, error)
	GetActiveSupplies(ctx context.Context) ([]*sms_pb.SupplyModel, error)
	GetSupplyByID(ctx context.Context, uuid string) (*sms_pb.SupplyModel, error)
}

// SupplyProduct represents product in supply
type SupplyProduct struct {
	ProductUUID string
	Amount      float32
}

func NewSMSClient(conn *grpc.ClientConn) SMSClient {
	productClient := sms_pb.NewProductServiceClient(conn)
	supplyClient := sms_pb.NewSupplyServiceClient(conn)

	return &smsClient{
		productClient: productClient,
		supplyClient:  supplyClient,
	}
}

type smsClient struct {
	productClient sms_pb.ProductServiceClient
	supplyClient  sms_pb.SupplyServiceClient
}

func (c *smsClient) RemoveCoupleProducts(ctx context.Context, req []models.ProductWriteOffRequest) ([]string, error) {
	reqProducts := make([]*sms_pb.SetProductAmountRequest, len(req))

	for i, m := range req {
		reqProducts[i] = &sms_pb.SetProductAmountRequest{
			Uuid:        m.Uuid,
			StoreAmount: float32(m.Amount),
		}
	}

	request := &sms_pb.RemoveProductsRequest{
		Products: reqProducts,
	}

	res, err := c.productClient.RemoveCoupleProducts(ctx, request)
	if err != nil {
		return nil, err
	}

	return res.Uuids, nil
}

func (c *smsClient) WriteOnCoupleProducts(ctx context.Context, req []models.ProductWriteOffRequest) ([]string, error) {
	reqProducts := make([]*sms_pb.SetProductAmountRequest, len(req))

	for i, m := range req {
		reqProducts[i] = &sms_pb.SetProductAmountRequest{
			Uuid:        m.Uuid,
			StoreAmount: float32(m.Amount),
		}
	}

	request := &sms_pb.RemoveProductsRequest{
		Products: reqProducts,
	}

	res, err := c.productClient.WriteOnCoupleProducts(ctx, request)
	if err != nil {
		return nil, err
	}

	return res.Uuids, nil
}

// Product methods implementation
func (c *smsClient) CreateProduct(ctx context.Context, storeCost float64) (string, error) {
	req := &sms_pb.CreateProductMessage{
		StoreCost: float32(storeCost),
	}
	resp, err := c.productClient.CreateProduct(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) DeleteProduct(ctx context.Context, uuid string) (string, error) {
	req := &sms_pb.UuidRequest{
		Uuid: uuid,
	}
	resp, err := c.productClient.DeleteProduct(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) SetProductCost(ctx context.Context, uuid string, cost float32) (string, error) {
	req := &sms_pb.SetProductCostRequest{
		Uuid:      uuid,
		StoreCost: cost,
	}
	resp, err := c.productClient.SetStoreCost(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) SetProductAmount(ctx context.Context, uuid string, amount float32) (string, error) {
	req := &sms_pb.SetProductAmountRequest{
		Uuid:        uuid,
		StoreAmount: amount,
	}
	resp, err := c.productClient.SetStoreAmount(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) GetProductAmount(ctx context.Context, uuid string) (float32, error) {
	req := &sms_pb.UuidRequest{
		Uuid: uuid,
	}
	resp, err := c.productClient.GetStoreAmount(ctx, req)
	if err != nil {
		return 0, err
	}
	return resp.StoreAmount, nil
}

// Supply methods implementation
func (c *smsClient) CreateSupply(
	ctx context.Context,
	supplyCost float32,
	desiredDate, comment, responsibleUser string,
	products []*SupplyProduct,
) (string, error) {
	productModels := make([]*sms_pb.SupplyProductModel, len(products))
	for i, p := range products {
		productModels[i] = &sms_pb.SupplyProductModel{
			ProductUuid: p.ProductUUID,
			Amount:      p.Amount,
		}
	}

	req := &sms_pb.CreateSupplyRequest{
		SupplyCost:      supplyCost,
		DesiredDate:     desiredDate,
		Comment:         comment,
		ResponsibleUser: responsibleUser,
		Products:        productModels,
	}
	resp, err := c.supplyClient.CreateSupply(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) DeleteSupply(ctx context.Context, uuid string) (string, error) {
	req := &sms_pb.UuidRequest{
		Uuid: uuid,
	}
	resp, err := c.supplyClient.DeleteSupply(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) UpdateSupplyInfo(
	ctx context.Context,
	uuid, comment, desiredDate, status, responsibleUser string,
	cost float32,
) (string, error) {
	req := &sms_pb.UpdateSupplyInfoRequest{
		Uuid:            uuid,
		Comment:         comment,
		DesiredDate:     desiredDate,
		Status:          status,
		ResponsibleUser: responsibleUser,
		Cost:            cost,
	}
	resp, err := c.supplyClient.UpdateSupplyInfo(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Uuid, nil
}

func (c *smsClient) GetActiveSupplies(ctx context.Context) ([]*sms_pb.SupplyModel, error) {
	resp, err := c.supplyClient.GetActiveSupplies(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.Supplies, nil
}

func (c *smsClient) GetSupplyByID(ctx context.Context, uuid string) (*sms_pb.SupplyModel, error) {
	req := &sms_pb.UuidRequest{
		Uuid: uuid,
	}
	return c.supplyClient.GetSupplyById(ctx, req)
}
