package domain

type ConstraintChecker interface {
	CanCombine(order1, order2 Order) bool
	CanFit(truck Truck, currentWeight, currentVolume int, order Order) bool
	ValidateOrderSet(orders []Order) bool
}

type DefaultConstraintChecker struct{}

func NewConstraintChecker() ConstraintChecker {
	return &DefaultConstraintChecker{}
}

func (d *DefaultConstraintChecker) CanCombine(order1, order2 Order) bool {
	if order1.Route() != order2.Route() {
		return false
	}
	
	if order1.IsHazmat != order2.IsHazmat {
		return false
	}
	
	if !d.timeWindowsCompatible(order1, order2) {
		return false
	}
	
	return true
}

func (d *DefaultConstraintChecker) CanFit(truck Truck, currentWeight, currentVolume int, order Order) bool {
	newWeight := currentWeight + order.WeightLbs
	newVolume := currentVolume + order.VolumeCuft
	
	return newWeight <= truck.MaxWeightLbs && newVolume <= truck.MaxVolumeCuft
}

func (d *DefaultConstraintChecker) ValidateOrderSet(orders []Order) bool {
	if len(orders) == 0 {
		return true
	}
	
	for i := 0; i < len(orders); i++ {
		for j := i + 1; j < len(orders); j++ {
			if !d.CanCombine(orders[i], orders[j]) {
				return false
			}
		}
	}
	
	return true
}

func (d *DefaultConstraintChecker) timeWindowsCompatible(order1, order2 Order) bool {
	return true
}

func FilterFeasibleOrders(truck Truck, orders []Order) []Order {
	feasible := make([]Order, 0, len(orders))
	
	for _, order := range orders {
		if order.WeightLbs > truck.MaxWeightLbs || order.VolumeCuft > truck.MaxVolumeCuft {
			continue
		}
		feasible = append(feasible, order)
	}
	
	return feasible
}

func GroupOrdersByRoute(orders []Order) map[string][]Order {
	groups := make(map[string][]Order)
	
	for _, order := range orders {
		route := order.Route()
		groups[route] = append(groups[route], order)
	}
	
	return groups
}

func SeparateHazmatOrders(orders []Order) (hazmat []Order, nonHazmat []Order) {
	hazmat = make([]Order, 0)
	nonHazmat = make([]Order, 0)
	
	for _, order := range orders {
		if order.IsHazmat {
			hazmat = append(hazmat, order)
		} else {
			nonHazmat = append(nonHazmat, order)
		}
	}
	
	return
}

type StrictConstraintChecker struct {
	DefaultConstraintChecker
}

func (s *StrictConstraintChecker) CanCombine(order1, order2 Order) bool {
	if !s.DefaultConstraintChecker.CanCombine(order1, order2) {
		return false
	}
	
	daysDiff := order1.PickupDate.Sub(order2.PickupDate).Hours() / 24
	if daysDiff < 0 {
		daysDiff = -daysDiff
	}
	
	return daysDiff <= 1
}
