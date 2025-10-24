//go:build scenario

package scenario

import (
	"context"
	"sync"
	"testing"
	"time"

	"mcmocknald-order-kiosk/internal/domain"
	"mcmocknald-order-kiosk/internal/infrastructure/memory"
	"mcmocknald-order-kiosk/internal/logger"
	"mcmocknald-order-kiosk/internal/service"
	"mcmocknald-order-kiosk/pkg/queue"
	"mcmocknald-order-kiosk/test/helpers"

	"github.com/stretchr/testify/assert"
)

// TestLargeLoad tests the system with 10,000 Regular, 5,000 VIP customers, 1,250 cook bots
// This represents a high-volume stress test scenario
// 1 order per customer per second for 3 minutes (15,000 orders/second)
// Records completion rate every 20 seconds
func TestLargeLoad(t *testing.T) {
	const (
		numRegularCustomers = 10000
		numVIPCustomers     = 5000
		numCooks            = 1250
		testDuration        = 3 * time.Minute
		reportInterval      = 20 * time.Second
		servingDuration     = 10 * time.Second
	)

	ctx := context.Background()
	log := logger.NewNoOpLogger()

	// Initialize repositories
	userRepo := memory.NewUserRepository()
	foodRepo := memory.NewFoodRepository()
	orderRepo := memory.NewOrderRepository(userRepo, foodRepo)

	// Create customers using helper functions
	regularCustomers := helpers.CreateCustomers(ctx, t, userRepo, numRegularCustomers, domain.RoleRegularCustomer)
	vipCustomers := helpers.CreateCustomers(ctx, t, userRepo, numVIPCustomers, domain.RoleVIPCustomer)

	// Create cook bots
	cooks := helpers.CreateCooks(ctx, t, userRepo, numCooks)

	// Create sample foods
	foods := helpers.CreateSampleFoods(ctx, t, foodRepo)

	// Initialize services
	orderQueue := queue.NewPriorityQueue()
	orderService := service.NewOrderService(orderRepo, userRepo, orderQueue, log, servingDuration)
	cookService := service.NewCookService(userRepo, orderRepo, orderQueue, log, servingDuration)

	// Start cook workers
	for _, cook := range cooks {
		go helpers.StartCookWorker(ctx, cookService, cook.ID, testDuration)
	}

	// Statistics tracking
	var statsLock sync.Mutex
	stats := make(map[time.Duration]helpers.OrderStats)

	// Start reporting goroutine
	stopReporting := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		startTime := time.Now()
		for {
			select {
			case <-stopReporting:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				completed, incomplete, _ := orderService.GetOrderStats(ctx)
				statsLock.Lock()
				stats[elapsed] = helpers.OrderStats{
					Completed:  completed,
					Incomplete: incomplete,
					QueueSize:  orderService.GetQueueSize(),
				}
				statsLock.Unlock()
				t.Logf("[%v] Completed: %d, Incomplete: %d, Queue: %d",
					elapsed.Round(time.Second), completed, incomplete, orderService.GetQueueSize())
			}
		}
	}()

	// Generate orders (1 per customer per second)
	stopOrders := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()

		startTime := time.Now()
		allCustomers := append(regularCustomers, vipCustomers...)

		// Create initial batch of orders at t=0 (before ticker starts)
		var orderWg sync.WaitGroup
		for _, customer := range allCustomers {
			orderWg.Add(1)
			go func(custID int) {
				defer orderWg.Done()
				foodIDs := []int{foods[0].ID}
				_, err := orderService.CreateOrder(ctx, custID, foodIDs)
				if err != nil {
					t.Logf("Failed to create order for customer %d: %v", custID, err)
				}
			}(customer.ID)
		}
		orderWg.Wait()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopOrders:
				return
			case <-ticker.C:
				if time.Since(startTime) >= testDuration {
					return
				}

				// Create orders for all customers concurrently
				var orderWg sync.WaitGroup
				for _, customer := range allCustomers {
					orderWg.Add(1)
					go func(custID int) {
						defer orderWg.Done()
						foodIDs := []int{foods[0].ID}
						_, err := orderService.CreateOrder(ctx, custID, foodIDs)
						if err != nil {
							t.Logf("Failed to create order for customer %d: %v", custID, err)
						}
					}(customer.ID)
				}
				orderWg.Wait()
			}
		}
	}()

	// Wait for test duration
	time.Sleep(testDuration)
	close(stopOrders)
	close(stopReporting)
	wg.Wait()

	// Wait a bit for final orders to complete
	time.Sleep(servingDuration + 2*time.Second)

	// Final statistics
	completed, incomplete, _ := orderService.GetOrderStats(ctx)
	t.Logf("\n=== Large Load Test Results ===")
	t.Logf("Regular Customers: %d", numRegularCustomers)
	t.Logf("VIP Customers: %d", numVIPCustomers)
	t.Logf("Cook Bots: %d", numCooks)
	t.Logf("Test Duration: %v", testDuration)
	t.Logf("Final Completed: %d", completed)
	t.Logf("Final Incomplete: %d", incomplete)
	t.Logf("Completion Rate: %.2f%%", float64(completed)/float64(completed+incomplete)*100)

	assert.Greater(t, completed, 0, "Should have completed some orders")
}
