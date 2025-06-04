package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/igntnk/stocky-oms/models"
	"github.com/igntnk/stocky-oms/requests"
	"github.com/igntnk/stocky-oms/service"
	"net/http"
)

type orderController struct {
	orders service.OrderService
}

func NewOrderController(orders service.OrderService) Controller {
	return &orderController{
		orders: orders,
	}
}

func (o *orderController) Register(r *gin.Engine) {
	group := r.Group("/api/SAGA/order")
	group.POST("/create", o.Create)
}

func (o *orderController) Create(context *gin.Context) {
	var err error

	receivedOrder := requests.CreateOrder{}
	err = context.ShouldBindBodyWithJSON(&receivedOrder)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": errors.Join(err, errors.New("failed to parse body")).Error()})
		return
	}

	products := []models.OrderProductInput{}
	for _, product := range receivedOrder.Products {
		prUuid, err := uuid.Parse(product.Uuid)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		products = append(products, models.OrderProductInput{
			ProductID: prUuid,
			Amount:    int(product.Amount),
		})
	}

	order, err := o.orders.CreateSagaOrder(context, models.OrderCreateRequest{
		UserID:   "000000000000000000000000",
		StaffID:  "000000000000000000000000",
		Comment:  receivedOrder.Comment,
		Products: products,
	})
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, gin.H{"order": order})
}
