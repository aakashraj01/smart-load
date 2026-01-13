package algorithm

import (
	"smart-load/internal/domain"
	"time"
)

type Optimizer interface {
	Optimize(truck domain.Truck, orders []domain.Order) OptimizationResult
}

type OptimizationResult struct {
	SelectedOrders []domain.Order
	TotalPayout    domain.Money
	TotalWeight    int
	TotalVolume    int
	ComputeTimeMs  int64
}

// DPOptimizer uses dynamic programming with bitmask for n <= 22
type DPOptimizer struct {
	checker domain.ConstraintChecker
	cache   map[int]domain.State
}

func NewDPOptimizer() *DPOptimizer {
	return &DPOptimizer{
		checker: domain.NewConstraintChecker(),
		cache:   make(map[int]domain.State),
	}
}

func (dp *DPOptimizer) Optimize(truck domain.Truck, orders []domain.Order) OptimizationResult {
	startTime := time.Now()
	
	if len(orders) == 0 {
		return OptimizationResult{
			SelectedOrders: []domain.Order{},
			TotalPayout:    0,
			TotalWeight:    0,
			TotalVolume:    0,
			ComputeTimeMs:  0,
		}
	}
	
	orders = domain.FilterFeasibleOrders(truck, orders)
	if len(orders) == 0 {
		return OptimizationResult{
			SelectedOrders: []domain.Order{},
			TotalPayout:    0,
			TotalWeight:    0,
			TotalVolume:    0,
			ComputeTimeMs:  time.Since(startTime).Milliseconds(),
		}
	}
	
	n := len(orders)
	
	// Precompute incompatibility bitmasks for O(1) compatibility checks
	incompatibleMask := make([]int, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j && !dp.checker.CanCombine(orders[i], orders[j]) {
				incompatibleMask[i] |= (1 << j)
			}
		}
	}
	
	maxStates := 1 << n
	
	dpPayout := make([]int64, maxStates)
	dpWeight := make([]int, maxStates)
	dpVolume := make([]int, maxStates)
	dpValid := make([]bool, maxStates)
	
	dpValid[0] = true
	
	for mask := 0; mask < maxStates; mask++ {
		if !dpValid[mask] {
			continue
		}
		
		currentWeight := dpWeight[mask]
		currentVolume := dpVolume[mask]
		
		for i := 0; i < n; i++ {
			if (mask & (1 << i)) != 0 {
				continue
			}
			
			if (mask & incompatibleMask[i]) != 0 {
				continue
			}
			
			order := orders[i]
			
			newWeight := currentWeight + order.WeightLbs
			newVolume := currentVolume + order.VolumeCuft
			if newWeight > truck.MaxWeightLbs || newVolume > truck.MaxVolumeCuft {
				continue
			}
			
			newMask := mask | (1 << i)
			newPayout := dpPayout[mask] + int64(order.Payout)
			
			if !dpValid[newMask] || newPayout > dpPayout[newMask] {
				dpPayout[newMask] = newPayout
				dpWeight[newMask] = newWeight
				dpVolume[newMask] = newVolume
				dpValid[newMask] = true
			}
		}
	}
	
	var bestPayout int64 = 0
	bestMask := 0
	
	for mask := 0; mask < maxStates; mask++ {
		if dpValid[mask] && dpPayout[mask] > bestPayout {
			bestPayout = dpPayout[mask]
			bestMask = mask
		}
	}
	
	selectedOrders := dp.extractOrders(bestMask, orders)
	
	computeTime := time.Since(startTime).Milliseconds()
	
	return OptimizationResult{
		SelectedOrders: selectedOrders,
		TotalPayout:    domain.Money(bestPayout),
		TotalWeight:    dpWeight[bestMask],
		TotalVolume:    dpVolume[bestMask],
		ComputeTimeMs:  computeTime,
	}
}

func (dp *DPOptimizer) isCompatibleWithState(mask int, orderIndex int, orders []domain.Order) bool {
	newOrder := orders[orderIndex]
	
	for j := 0; j < len(orders); j++ {
		if (mask & (1 << j)) != 0 {
			if !dp.checker.CanCombine(newOrder, orders[j]) {
				return false
			}
		}
	}
	
	return true
}

func (dp *DPOptimizer) extractOrders(mask int, orders []domain.Order) []domain.Order {
	selected := make([]domain.Order, 0)
	
	for i := 0; i < len(orders); i++ {
		if (mask & (1 << i)) != 0 {
			selected = append(selected, orders[i])
		}
	}
	
	return selected
}

func (dp *DPOptimizer) ClearCache() {
	dp.cache = make(map[int]domain.State)
}

// GreedyOptimizer fallback for n > 22
type GreedyOptimizer struct {
	checker domain.ConstraintChecker
}

func NewGreedyOptimizer() *GreedyOptimizer {
	return &GreedyOptimizer{
		checker: domain.NewConstraintChecker(),
	}
}

func (g *GreedyOptimizer) Optimize(truck domain.Truck, orders []domain.Order) OptimizationResult {
	startTime := time.Now()
	
	orders = domain.FilterFeasibleOrders(truck, orders)
	
	sortedOrders := g.sortByValueDensity(orders)
	
	selected := make([]domain.Order, 0)
	totalWeight := 0
	totalVolume := 0
	totalPayout := domain.Money(0)
	
	for _, order := range sortedOrders {
		if !g.checker.CanFit(truck, totalWeight, totalVolume, order) {
			continue
		}
		
		compatible := true
		for _, selectedOrder := range selected {
			if !g.checker.CanCombine(order, selectedOrder) {
				compatible = false
				break
			}
		}
		
		if compatible {
			selected = append(selected, order)
			totalWeight += order.WeightLbs
			totalVolume += order.VolumeCuft
			totalPayout = totalPayout.Add(order.Payout)
		}
	}
	
	return OptimizationResult{
		SelectedOrders: selected,
		TotalPayout:    totalPayout,
		TotalWeight:    totalWeight,
		TotalVolume:    totalVolume,
		ComputeTimeMs:  time.Since(startTime).Milliseconds(),
	}
}

func (g *GreedyOptimizer) sortByValueDensity(orders []domain.Order) []domain.Order {
	sorted := make([]domain.Order, len(orders))
	copy(sorted, orders)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			density_i := float64(sorted[i].Payout) / float64(sorted[i].WeightLbs)
			density_j := float64(sorted[j].Payout) / float64(sorted[j].WeightLbs)
			
			if density_j > density_i {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// BacktrackingOptimizer uses recursive backtracking with pruning
type BacktrackingOptimizer struct {
	checker    domain.ConstraintChecker
	bestPayout domain.Money
	bestOrders []domain.Order
	bestWeight int
	bestVolume int
}

func NewBacktrackingOptimizer() *BacktrackingOptimizer {
	return &BacktrackingOptimizer{
		checker: domain.NewConstraintChecker(),
	}
}

func (b *BacktrackingOptimizer) Optimize(truck domain.Truck, orders []domain.Order) OptimizationResult {
	startTime := time.Now()
	
	orders = domain.FilterFeasibleOrders(truck, orders)
	
	b.bestPayout = 0
	b.bestOrders = []domain.Order{}
	b.bestWeight = 0
	b.bestVolume = 0
	
	currentOrders := []domain.Order{}
	b.backtrack(truck, orders, currentOrders, 0, 0, 0, 0)
	
	return OptimizationResult{
		SelectedOrders: b.bestOrders,
		TotalPayout:    b.bestPayout,
		TotalWeight:    b.bestWeight,
		TotalVolume:    b.bestVolume,
		ComputeTimeMs:  time.Since(startTime).Milliseconds(),
	}
}

func (b *BacktrackingOptimizer) backtrack(
	truck domain.Truck,
	orders []domain.Order,
	currentOrders []domain.Order,
	index int,
	currentPayout domain.Money,
	currentWeight int,
	currentVolume int,
) {
	// Update best solution if current is better
	if currentPayout > b.bestPayout {
		b.bestPayout = currentPayout
		b.bestOrders = make([]domain.Order, len(currentOrders))
		copy(b.bestOrders, currentOrders)
		b.bestWeight = currentWeight
		b.bestVolume = currentVolume
	}
	
	// Base case: tried all orders
	if index >= len(orders) {
		return
	}
	
	// Pruning: calculate upper bound for remaining orders
	remainingPayout := domain.Money(0)
	for i := index; i < len(orders); i++ {
		remainingPayout += orders[i].Payout
	}
	if currentPayout+remainingPayout <= b.bestPayout {
		return // Prune this branch
	}
	
	// Try including current order
	order := orders[index]
	newWeight := currentWeight + order.WeightLbs
	newVolume := currentVolume + order.VolumeCuft
	
	canFit := newWeight <= truck.MaxWeightLbs && newVolume <= truck.MaxVolumeCuft
	compatible := true
	
	if canFit {
		for _, selected := range currentOrders {
			if !b.checker.CanCombine(order, selected) {
				compatible = false
				break
			}
		}
		
		if compatible {
			// Include this order
			b.backtrack(
				truck,
				orders,
				append(currentOrders, order),
				index+1,
				currentPayout+order.Payout,
				newWeight,
				newVolume,
			)
		}
	}
	
	// Try excluding current order
	b.backtrack(truck, orders, currentOrders, index+1, currentPayout, currentWeight, currentVolume)
}

type HybridOptimizer struct {
	dpOptimizer     *DPOptimizer
	greedyOptimizer *GreedyOptimizer
	maxDPSize       int
}

func NewHybridOptimizer() *HybridOptimizer {
	return &HybridOptimizer{
		dpOptimizer:     NewDPOptimizer(),
		greedyOptimizer: NewGreedyOptimizer(),
		maxDPSize:       22,
	}
}

func (h *HybridOptimizer) Optimize(truck domain.Truck, orders []domain.Order) OptimizationResult {
	if len(orders) <= h.maxDPSize {
		return h.dpOptimizer.Optimize(truck, orders)
	}
	return h.greedyOptimizer.Optimize(truck, orders)
}
