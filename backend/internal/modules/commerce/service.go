package commerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/modules/enrollment"
	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

type Service struct {
	pool       *pgxpool.Pool
	cfg        config.Config
	httpClient *http.Client
	enrollment *enrollment.Service
}

func NewService(pool *pgxpool.Pool, cfg config.Config, enr *enrollment.Service) *Service {
	return &Service{
		pool:       pool,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		enrollment: enr,
	}
}

type CoursePricing struct {
	ID           uuid.UUID `json:"id"`
	Type         string    `json:"type"` // free, one_time
	Amount       int64     `json:"amount"`
	Currency     string    `json:"currency"`
	TaxInclusive bool      `json:"taxInclusive"`
}

// EnsureProductForCourse guarantees a product row exists for a course.
func (s *Service) EnsureProductForCourse(ctx context.Context, schoolID, courseID uuid.UUID, courseTitle string) (store.Product, error) {
	var prod store.Product
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		prod, err = q.GetProductByRef(ctx, store.GetProductByRefParams{
			SchoolID: schoolID,
			Kind:     "course",
			RefID:    courseID,
		})
		if db.IsNotFound(err) {
			prod, err = q.CreateProduct(ctx, store.CreateProductParams{
				ID:       uuid.Must(uuid.NewV7()),
				SchoolID: schoolID,
				Kind:     "course",
				RefID:    courseID,
				Name:     courseTitle,
			})
		}
		return err
	})
	if err != nil {
		return store.Product{}, fmt.Errorf("commerce: ensure product: %w", err)
	}
	return prod, nil
}

// SetCoursePricing sets or updates the active pricing plan for a course.
func (s *Service) SetCoursePricing(ctx context.Context, schoolID, courseID uuid.UUID, pricingType string, amount int64) (store.PricingPlan, error) {
	if pricingType != "free" && pricingType != "one_time" {
		return store.PricingPlan{}, apperr.Validation("invalid_pricing_type", "type must be free or one_time")
	}
	if pricingType == "free" {
		amount = 0
	}

	prod, err := s.EnsureProductForCourse(ctx, schoolID, courseID, "Course")
	if err != nil {
		return store.PricingPlan{}, err
	}

	var plan store.PricingPlan
	err = db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		// Deactivate old pricing plans
		if err := q.DeactivatePricingPlans(ctx, store.DeactivatePricingPlansParams{
			SchoolID:  schoolID,
			ProductID: prod.ID,
		}); err != nil {
			return err
		}

		var err error
		plan, err = q.CreatePricingPlan(ctx, store.CreatePricingPlanParams{
			ID:           uuid.Must(uuid.NewV7()),
			SchoolID:     schoolID,
			ProductID:    prod.ID,
			Type:         pricingType,
			Amount:       amount,
			Currency:     "SAR",
			TaxInclusive: true,
			Active:       true,
		})
		return err
	})
	if err != nil {
		return store.PricingPlan{}, fmt.Errorf("commerce: set pricing: %w", err)
	}
	return plan, nil
}

// GetCoursePricing returns the active pricing plan for a course.
func (s *Service) GetCoursePricing(ctx context.Context, schoolID, courseID uuid.UUID) (*CoursePricing, error) {
	var prod store.Product
	var plans []store.PricingPlan

	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		prod, err = q.GetProductByRef(ctx, store.GetProductByRefParams{
			SchoolID: schoolID,
			Kind:     "course",
			RefID:    courseID,
		})
		if db.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}

		plans, err = q.ListPricingPlansByProduct(ctx, store.ListPricingPlansByProductParams{
			SchoolID:  schoolID,
			ProductID: prod.ID,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("commerce: get pricing: %w", err)
	}

	if len(plans) == 0 {
		return &CoursePricing{
			Type:         "free",
			Amount:       0,
			Currency:     "SAR",
			TaxInclusive: true,
		}, nil
	}

	p := plans[0]
	return &CoursePricing{
		ID:           p.ID,
		Type:         p.Type,
		Amount:       p.Amount,
		Currency:     p.Currency,
		TaxInclusive: p.TaxInclusive,
	}, nil
}

type CheckoutResult struct {
	OrderID        uuid.UUID `json:"orderId"`
	Status         string    `json:"status"` // pending, paid
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	Gateway        string    `json:"gateway"`
	PublishableKey string    `json:"publishableKey,omitempty"`
}

// CreateCheckout creates an order and initializes payment or immediately enrolls if free.
func (s *Service) CreateCheckout(
	ctx context.Context,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	studentName, studentEmail string,
	courseID uuid.UUID,
	pricingPlanID *uuid.UUID,
	gateway string,
	idempotencyKey *string,
) (*CheckoutResult, error) {
	pricing, err := s.GetCoursePricing(ctx, schoolID, courseID)
	if err != nil {
		return nil, err
	}

	prod, err := s.EnsureProductForCourse(ctx, schoolID, courseID, "Course")
	if err != nil {
		return nil, err
	}

	planID := pricing.ID
	if planID == uuid.Nil {
		planID = uuid.Must(uuid.NewV7())
	}

	orderID := uuid.Must(uuid.NewV7())
	isFree := pricing.Type == "free" || pricing.Amount == 0

	orderStatus := "pending"
	selectedGateway := gateway
	if isFree {
		orderStatus = "paid"
		selectedGateway = "free"
	}

	err = db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		// 1. Create Order
		_, err := q.CreateOrder(ctx, store.CreateOrderParams{
			ID:             orderID,
			SchoolID:       schoolID,
			StudentID:      studentID,
			Status:         orderStatus,
			Subtotal:       pricing.Amount,
			Discount:       0,
			Tax:            0,
			Total:          pricing.Amount,
			Currency:       pricing.Currency,
			Gateway:        selectedGateway,
			BuyerName:      studentName,
			BuyerEmail:     studentEmail,
			IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			return err
		}

		// 2. Create Order Item
		_, err = q.CreateOrderItem(ctx, store.CreateOrderItemParams{
			ID:            uuid.Must(uuid.NewV7()),
			SchoolID:      schoolID,
			OrderID:       orderID,
			ProductID:     prod.ID,
			PricingPlanID: planID,
			Amount:        pricing.Amount,
			Tax:           0,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("commerce: create order: %w", err)
	}

	if isFree {
		// Immediately grant course enrollment
		_, err = s.enrollment.Enroll(ctx, schoolID, studentID, courseID, "order", &orderID)
		if err != nil {
			return nil, fmt.Errorf("commerce: free enrollment: %w", err)
		}
	}

	pubKey := ""
	if selectedGateway == "moyasar" {
		pubKey = s.cfg.MoyasarPublishableKey
	}

	return &CheckoutResult{
		OrderID:        orderID,
		Status:         orderStatus,
		Amount:         pricing.Amount,
		Currency:       pricing.Currency,
		Gateway:        selectedGateway,
		PublishableKey: pubKey,
	}, nil
}

// VerifyAndFulfillMoyasar verifies a Moyasar payment callback and fulfills the order.
func (s *Service) VerifyAndFulfillMoyasar(ctx context.Context, schoolID, orderID uuid.UUID, paymentID string) error {
	if s.cfg.MoyasarSecretKey == "" {
		// In development or test mode without Moyasar keys, mark order as paid directly
		return s.fulfillOrder(ctx, schoolID, orderID, "moyasar", paymentID)
	}

	// Verify payment with Moyasar API
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.moyasar.com/v1/payments/"+paymentID, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.cfg.MoyasarSecretKey, "")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("moyasar api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("moyasar verification failed with status: %d", resp.StatusCode)
	}

	var moyasarResp struct {
		ID     string `json:"id"`
		Status string `json:"status"` // paid, failed
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&moyasarResp); err != nil {
		return fmt.Errorf("failed to decode moyasar response: %w", err)
	}

	if moyasarResp.Status != "paid" {
		return apperr.Validation("payment_not_paid", "payment status is "+moyasarResp.Status)
	}

	return s.fulfillOrder(ctx, schoolID, orderID, "moyasar", paymentID)
}

func (s *Service) fulfillOrder(ctx context.Context, schoolID, orderID uuid.UUID, gateway, paymentRef string) error {
	var studentID uuid.UUID
	var courseID uuid.UUID

	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, tx pgx.Tx) error {
		// Update Order
		order, err := q.UpdateOrderStatus(ctx, store.UpdateOrderStatusParams{
			SchoolID:        schoolID,
			ID:              orderID,
			Status:          "paid",
			GatewayOrderRef: &paymentRef,
		})
		if err != nil {
			return err
		}
		studentID = order.StudentID

		// Find Course ID from Order Items -> Products
		err = tx.QueryRow(ctx, `
			SELECT p.ref_id
			FROM order_items oi
			JOIN products p ON p.id = oi.product_id AND p.school_id = oi.school_id
			WHERE oi.school_id = $1 AND oi.order_id = $2
			LIMIT 1
		`, schoolID, orderID).Scan(&courseID)
		if err != nil {
			return err
		}

		// Record Payment
		_, err = q.CreatePayment(ctx, store.CreatePaymentParams{
			ID:               uuid.Must(uuid.NewV7()),
			SchoolID:         schoolID,
			OrderID:          orderID,
			Gateway:          gateway,
			GatewayPaymentID: &paymentRef,
			Amount:           order.Total,
			Currency:         order.Currency,
			Status:           "succeeded",
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("commerce: fulfill order: %w", err)
	}

	// Grant enrollment
	_, err = s.enrollment.Enroll(ctx, schoolID, studentID, courseID, "order", &orderID)
	if err != nil {
		return fmt.Errorf("commerce: fulfill enrollment: %w", err)
	}

	return nil
}

// ListOrders returns a paginated list of orders for a school.
func (s *Service) ListOrders(ctx context.Context, schoolID uuid.UUID, limit, offset int32) ([]store.Order, error) {
	var orders []store.Order
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		orders, err = q.ListOrders(ctx, store.ListOrdersParams{
			SchoolID: schoolID,
			Limit:    limit,
			Offset:   offset,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("commerce: list orders: %w", err)
	}
	return orders, nil
}
