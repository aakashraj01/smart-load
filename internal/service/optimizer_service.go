package service

import (
	"fmt"
	"log"
	"smart-load/internal/algorithm"
	"smart-load/internal/domain"
)

type OptimizerService struct {
	optimizer algorithm.Optimizer
}

func NewOptimizerService() *OptimizerService {
	return &OptimizerService{
		optimizer: algorithm.NewHybridOptimizer(),
	}
}

func NewOptimizerServiceWithAlgorithm(optimizer algorithm.Optimizer) *OptimizerService {
	return &OptimizerService{
		optimizer: optimizer,
	}
}

func (s *OptimizerService) OptimizeLoad(request domain.OptimizeRequest) (*domain.OptimizeResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	truck, orders, err := request.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	
	orders = s.preprocessOrders(*truck, orders)
	optimizer := s.selectOptimizer(request.OptimizationConfig, len(orders))
	
	var result algorithm.OptimizationResult
	if request.OptimizationConfig != nil && 
	   (request.OptimizationConfig.RevenueWeight != 1.0 || request.OptimizationConfig.UtilizationWeight != 0) {
		result = s.optimizeWithWeights(*truck, orders, 
			request.OptimizationConfig.RevenueWeight, 
			request.OptimizationConfig.UtilizationWeight)
	} else {
		log.Printf(" Optimizing %d orders for truck %s...", len(orders), truck.ID)
		result = optimizer.Optimize(*truck, orders)
	}
	
	log.Printf(" Found solution with %d orders, $%.2f payout in %dms",
		len(result.SelectedOrders),
		float64(result.TotalPayout)/100,
		result.ComputeTimeMs,
	)
	
	response := s.buildResponse(*truck, result)
	return response, nil
}

func (s *OptimizerService) selectOptimizer(config *domain.OptimizationConfig, numOrders int) algorithm.Optimizer {
	if config == nil || config.Algorithm == "auto" {
		return s.optimizer
	}
	
	switch config.Algorithm {
	case "dp":
		return algorithm.NewDPOptimizer()
	case "backtracking":
		return algorithm.NewBacktrackingOptimizer()
	case "greedy":
		return algorithm.NewGreedyOptimizer()
	default:
		return s.optimizer
	}
}

func (s *OptimizerService) preprocessOrders(truck domain.Truck, orders []domain.Order) []domain.Order {
	orders = domain.FilterFeasibleOrders(truck, orders)
	
	if len(orders) == 0 {
		return orders
	}
	
	hazmat, nonHazmat := domain.SeparateHazmatOrders(orders)
	
	if len(hazmat) > 0 && len(nonHazmat) > 0 {
		log.Printf("  Mixed hazmat/non-hazmat orders detected: %d hazmat, %d non-hazmat",
			len(hazmat), len(nonHazmat))
	}
	
	return orders
}

func (s *OptimizerService) buildResponse(truck domain.Truck, result algorithm.OptimizationResult) *domain.OptimizeResponse {
	orderIDs := make([]string, len(result.SelectedOrders))
	for i, order := range result.SelectedOrders {
		orderIDs[i] = order.ID
	}
	
	utilizationWeight := 0.0
	if truck.MaxWeightLbs > 0 {
		utilizationWeight = (float64(result.TotalWeight) / float64(truck.MaxWeightLbs)) * 100
	}
	
	utilizationVolume := 0.0
	if truck.MaxVolumeCuft > 0 {
		utilizationVolume = (float64(result.TotalVolume) / float64(truck.MaxVolumeCuft)) * 100
	}
	
	utilizationWeight = roundToTwoDecimals(utilizationWeight)
	utilizationVolume = roundToTwoDecimals(utilizationVolume)
	
	return &domain.OptimizeResponse{
		TruckID:                  truck.ID,
		SelectedOrderIDs:         orderIDs,
		TotalPayoutCents:         int64(result.TotalPayout),
		TotalWeightLbs:           result.TotalWeight,
		TotalVolumeCuft:          result.TotalVolume,
		UtilizationWeightPercent: utilizationWeight,
		UtilizationVolumePercent: utilizationVolume,
	}
}

func roundToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

type ParetoSolution struct {
	OrderIDs                 []string `json:"order_ids"`
	TotalPayoutCents         int64    `json:"total_payout_cents"`
	TotalWeightLbs           int      `json:"total_weight_lbs"`
	TotalVolumeCuft          int      `json:"total_volume_cuft"`
	UtilizationWeightPercent float64  `json:"utilization_weight_percent"`
	UtilizationVolumePercent float64  `json:"utilization_volume_percent"`
	Score                    float64  `json:"score"`
}

func (s *OptimizerService) GetParetoOptimalSolutions(
	truck domain.Truck,
	orders []domain.Order,
	maxSolutions int,
) []ParetoSolution {
	solutions := make([]ParetoSolution, 0)
	seen := make(map[string]bool)
	
	weights := []struct{ revenue, utilization float64 }{
		{1.0, 0.0},
		{0.8, 0.2},
		{0.6, 0.4},
		{0.4, 0.6},
		{0.2, 0.8},
	}
	
	for _, w := range weights {
		result := s.optimizeWithWeights(truck, orders, w.revenue, w.utilization)
		
		key := ""
		for _, order := range result.SelectedOrders {
			key += order.ID + ","
		}
		
		if seen[key] {
			continue
		}
		seen[key] = true
		
		orderIDs := make([]string, len(result.SelectedOrders))
		for i, order := range result.SelectedOrders {
			orderIDs[i] = order.ID
		}
		
		weightUtil := 0.0
		volumeUtil := 0.0
		if truck.MaxWeightLbs > 0 {
			weightUtil = (float64(result.TotalWeight) / float64(truck.MaxWeightLbs)) * 100
		}
		if truck.MaxVolumeCuft > 0 {
			volumeUtil = (float64(result.TotalVolume) / float64(truck.MaxVolumeCuft)) * 100
		}
		
		score := w.revenue*float64(result.TotalPayout) + 
		         w.utilization*(weightUtil+volumeUtil)*1000
		
		solutions = append(solutions, ParetoSolution{
			OrderIDs:                 orderIDs,
			TotalPayoutCents:         int64(result.TotalPayout),
			TotalWeightLbs:           result.TotalWeight,
			TotalVolumeCuft:          result.TotalVolume,
			UtilizationWeightPercent: roundToTwoDecimals(weightUtil),
			UtilizationVolumePercent: roundToTwoDecimals(volumeUtil),
			Score:                    roundToTwoDecimals(score),
		})
		
		if len(solutions) >= maxSolutions {
			break
		}
	}
	
	return s.filterParetoOptimal(solutions)
}

func (s *OptimizerService) optimizeWithWeights(
	truck domain.Truck,
	orders []domain.Order,
	revenueWeight float64,
	utilizationWeight float64,
) algorithm.OptimizationResult {
	weighted := make([]domain.Order, len(orders))
	copy(weighted, orders)
	
	for i := range weighted {
		revenue := float64(weighted[i].Payout)
		utilization := (float64(weighted[i].WeightLbs)/float64(truck.MaxWeightLbs) + 
		               float64(weighted[i].VolumeCuft)/float64(truck.MaxVolumeCuft)) / 2
		
		score := revenueWeight*revenue + utilizationWeight*utilization*10000
		weighted[i].Payout = domain.Money(score)
	}
	
	return s.optimizer.Optimize(truck, weighted)
}

func (s *OptimizerService) filterParetoOptimal(solutions []ParetoSolution) []ParetoSolution {
	pareto := make([]ParetoSolution, 0)
	
	for i, sol1 := range solutions {
		dominated := false
		
		for j, sol2 := range solutions {
			if i == j {
				continue
			}
			
			if sol2.TotalPayoutCents >= sol1.TotalPayoutCents &&
			   sol2.UtilizationWeightPercent >= sol1.UtilizationWeightPercent &&
			   sol2.UtilizationVolumePercent >= sol1.UtilizationVolumePercent &&
			   (sol2.TotalPayoutCents > sol1.TotalPayoutCents ||
			    sol2.UtilizationWeightPercent > sol1.UtilizationWeightPercent ||
			    sol2.UtilizationVolumePercent > sol1.UtilizationVolumePercent) {
				dominated = true
				break
			}
		}
		
		if !dominated {
			pareto = append(pareto, sol1)
		}
	}
	
	return pareto
}

func (s *OptimizerService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":    "healthy",
		"service":   "optimizer",
		"algorithm": "hybrid-dp",
	}
}
