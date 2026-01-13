package domain

import (
	"fmt"
	"time"
)

type Money int64

func (m Money) ToDollars() string {
	dollars := float64(m) / 100.0
	return fmt.Sprintf("$%.2f", dollars)
}

func (m Money) Add(other Money) Money {
	return m + other
}

func (m Money) GreaterThan(other Money) bool {
	return m > other
}

type OptimizeRequest struct {
	Truck              TruckInput          `json:"truck"`
	Orders             []OrderInput        `json:"orders"`
	OptimizationConfig *OptimizationConfig `json:"optimization_config,omitempty"`
}

type OptimizationConfig struct {
	Objective         string  `json:"objective"`
	RevenueWeight     float64 `json:"revenue_weight"`
	UtilizationWeight float64 `json:"utilization_weight"`
	Algorithm         string  `json:"algorithm"`
}

type TruckInput struct {
	ID            string `json:"id"`
	MaxWeightLbs  int    `json:"max_weight_lbs"`
	MaxVolumeCuft int    `json:"max_volume_cuft"`
}

type OrderInput struct {
	ID           string `json:"id"`
	PayoutCents  int64  `json:"payout_cents"`
	WeightLbs    int    `json:"weight_lbs"`
	VolumeCuft   int    `json:"volume_cuft"`
	Origin       string `json:"origin"`
	Destination  string `json:"destination"`
	PickupDate   string `json:"pickup_date"`
	DeliveryDate string `json:"delivery_date"`
	IsHazmat     bool   `json:"is_hazmat"`
}

type Truck struct {
	ID            string
	MaxWeightLbs  int
	MaxVolumeCuft int
}

type Order struct {
	ID           string
	Payout       Money
	WeightLbs    int
	VolumeCuft   int
	Origin       string
	Destination  string
	PickupDate   time.Time
	DeliveryDate time.Time
	IsHazmat     bool
}

func (o Order) FitsIn(availableWeight, availableVolume int) bool {
	return o.WeightLbs <= availableWeight && o.VolumeCuft <= availableVolume
}

func (o Order) Route() string {
	return fmt.Sprintf("%s->%s", o.Origin, o.Destination)
}

type OptimizeResponse struct {
	TruckID                  string   `json:"truck_id"`
	SelectedOrderIDs         []string `json:"selected_order_ids"`
	TotalPayoutCents         int64    `json:"total_payout_cents"`
	TotalWeightLbs           int      `json:"total_weight_lbs"`
	TotalVolumeCuft          int      `json:"total_volume_cuft"`
	UtilizationWeightPercent float64  `json:"utilization_weight_percent"`
	UtilizationVolumePercent float64  `json:"utilization_volume_percent"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type State struct {
	Mask       int
	Payout     Money
	WeightLbs  int
	VolumeCuft int
	Valid      bool
}

func NewState() State {
	return State{
		Mask:       0,
		Payout:     0,
		WeightLbs:  0,
		VolumeCuft: 0,
		Valid:      true,
	}
}

func (s State) AddOrder(order Order, orderIndex int) State {
	return State{
		Mask:       s.Mask | (1 << orderIndex),
		Payout:     s.Payout.Add(order.Payout),
		WeightLbs:  s.WeightLbs + order.WeightLbs,
		VolumeCuft: s.VolumeCuft + order.VolumeCuft,
		Valid:      true,
	}
}

func (s State) IsBetterThan(other State) bool {
	return s.Payout.GreaterThan(other.Payout)
}

func (s State) HasOrder(orderIndex int) bool {
	return (s.Mask & (1 << orderIndex)) != 0
}

func (r *OptimizeRequest) Validate() error {
	if r.Truck.ID == "" {
		return fmt.Errorf("truck id is required")
	}
	if r.Truck.MaxWeightLbs <= 0 {
		return fmt.Errorf("truck max_weight_lbs must be positive")
	}
	if r.Truck.MaxWeightLbs > 1000000 {
		return fmt.Errorf("truck max_weight_lbs exceeds maximum allowed value")
	}
	if r.Truck.MaxVolumeCuft <= 0 {
		return fmt.Errorf("truck max_volume_cuft must be positive")
	}
	if r.Truck.MaxVolumeCuft > 100000 {
		return fmt.Errorf("truck max_volume_cuft exceeds maximum allowed value")
	}
	if len(r.Orders) == 0 {
		return fmt.Errorf("orders list cannot be empty")
	}
	if len(r.Orders) > 22 {
		return fmt.Errorf("orders list cannot exceed 22 items for optimal solution (got %d)", len(r.Orders))
	}
	
	seenIDs := make(map[string]bool)
	for i, order := range r.Orders {
		if seenIDs[order.ID] {
			return fmt.Errorf("duplicate order id: %s", order.ID)
		}
		seenIDs[order.ID] = true
		
		if err := order.Validate(); err != nil {
			return fmt.Errorf("order[%d]: %w", i, err)
		}
	}
	
	if r.OptimizationConfig != nil {
		if err := r.OptimizationConfig.Validate(); err != nil {
			return fmt.Errorf("optimization_config: %w", err)
		}
	}
	
	return nil
}

func (c *OptimizationConfig) Validate() error {
	if c.Objective == "" {
		c.Objective = "revenue"
	}
	
	validObjectives := map[string]bool{
		"revenue":     true,
		"utilization": true,
		"balanced":    true,
	}
	if !validObjectives[c.Objective] {
		return fmt.Errorf("invalid objective: %s (must be revenue, utilization, or balanced)", c.Objective)
	}
	
	if c.RevenueWeight < 0 || c.RevenueWeight > 1 {
		return fmt.Errorf("revenue_weight must be between 0 and 1")
	}
	if c.UtilizationWeight < 0 || c.UtilizationWeight > 1 {
		return fmt.Errorf("utilization_weight must be between 0 and 1")
	}
	
	if c.RevenueWeight == 0 && c.UtilizationWeight == 0 {
		switch c.Objective {
		case "revenue":
			c.RevenueWeight = 1.0
			c.UtilizationWeight = 0.0
		case "utilization":
			c.RevenueWeight = 0.0
			c.UtilizationWeight = 1.0
		case "balanced":
			c.RevenueWeight = 0.5
			c.UtilizationWeight = 0.5
		}
	}
	
	if c.Algorithm == "" {
		c.Algorithm = "auto"
	}
	validAlgorithms := map[string]bool{
		"dp":           true,
		"backtracking": true,
		"greedy":       true,
		"auto":         true,
	}
	if !validAlgorithms[c.Algorithm] {
		return fmt.Errorf("invalid algorithm: %s (must be dp, backtracking, greedy, or auto)", c.Algorithm)
	}
	
	return nil
}

func (o *OrderInput) Validate() error {
	if o.ID == "" {
		return fmt.Errorf("order id is required")
	}
	if o.PayoutCents <= 0 {
		return fmt.Errorf("payout_cents must be positive")
	}
	if o.PayoutCents > 100000000000 {
		return fmt.Errorf("payout_cents exceeds maximum allowed value")
	}
	if o.WeightLbs <= 0 {
		return fmt.Errorf("weight_lbs must be positive")
	}
	if o.WeightLbs > 1000000 {
		return fmt.Errorf("weight_lbs exceeds maximum allowed value")
	}
	if o.VolumeCuft <= 0 {
		return fmt.Errorf("volume_cuft must be positive")
	}
	if o.VolumeCuft > 100000 {
		return fmt.Errorf("volume_cuft exceeds maximum allowed value")
	}
	if o.Origin == "" || o.Destination == "" {
		return fmt.Errorf("origin and destination are required")
	}
	if len(o.Origin) > 200 || len(o.Destination) > 200 {
		return fmt.Errorf("origin and destination must be less than 200 characters")
	}
	if len(o.ID) > 100 {
		return fmt.Errorf("order id must be less than 100 characters")
	}
	
	pickup, err := time.Parse("2006-01-02", o.PickupDate)
	if err != nil {
		return fmt.Errorf("invalid pickup_date format (expected YYYY-MM-DD)")
	}
	delivery, err := time.Parse("2006-01-02", o.DeliveryDate)
	if err != nil {
		return fmt.Errorf("invalid delivery_date format (expected YYYY-MM-DD)")
	}
	if delivery.Before(pickup) {
		return fmt.Errorf("delivery_date cannot be before pickup_date")
	}
	
	return nil
}

func (r *OptimizeRequest) ToDomain() (*Truck, []Order, error) {
	truck := &Truck{
		ID:            r.Truck.ID,
		MaxWeightLbs:  r.Truck.MaxWeightLbs,
		MaxVolumeCuft: r.Truck.MaxVolumeCuft,
	}
	
	orders := make([]Order, 0, len(r.Orders))
	for _, orderInput := range r.Orders {
		order, err := orderInput.ToDomain()
		if err != nil {
			return nil, nil, err
		}
		orders = append(orders, order)
	}
	
	return truck, orders, nil
}

func (o *OrderInput) ToDomain() (Order, error) {
	pickup, _ := time.Parse("2006-01-02", o.PickupDate)
	delivery, _ := time.Parse("2006-01-02", o.DeliveryDate)
	
	return Order{
		ID:           o.ID,
		Payout:       Money(o.PayoutCents),
		WeightLbs:    o.WeightLbs,
		VolumeCuft:   o.VolumeCuft,
		Origin:       o.Origin,
		Destination:  o.Destination,
		PickupDate:   pickup,
		DeliveryDate: delivery,
		IsHazmat:     o.IsHazmat,
	}, nil
}
