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

// CI-specific test configuration constants
// These values are optimized for CI/CD environments where execution time is critical
const (
	ciSmallTestDuration    = 1 * time.Minute  // CI: 1 minute (vs 3 minutes in original)
	ciSmallReportInterval  = 10 * time.Second // CI: 10 seconds (vs 20 seconds in original)
	ciSmallServingDuration = 10 * time.Second // Keep same as original
)

// TestSmallLoadCI tests the system with 100 Regular, 50 VIP customers, 25 cook bots
// This is the CI/CD-optimized version with shorter duration for faster feedback
// 1 order per customer per second for 1 minute (configurable via constants)
// Records completion rate every 10 seconds
func TestSmallLoadCI(t *testing.T) {
	const (
		numRegularCustomers = 100
		numVIPCustomers     = 50
		numCooks            = 25
	)

	ctx := context.Background()

	// Use file logger to capture detailed cook bot activity
	log, err := logger.NewFileLogger("./logs")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	log.Info("=== Starting Small Load CI Test ===")
	log.Info("Parameters: %d Regular, %d VIP customers, %d cooks", numRegularCustomers, numVIPCustomers, numCooks)
	log.Info("CI Configuration: Duration=%v, ReportInterval=%v", ciSmallTestDuration, ciSmallReportInterval)

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
	orderService := service.NewOrderService(orderRepo, userRepo, foodRepo, orderQueue, log, ciSmallServingDuration)
	cookService := service.NewCookService(userRepo, orderRepo, orderQueue, log, ciSmallServingDuration)

	// Start cook workers
	for _, cook := range cooks {
		go helpers.StartCookWorker(ctx, cookService, cook.ID, ciSmallTestDuration)
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
		ticker := time.NewTicker(ciSmallReportInterval)
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
		for _, customer := range allCustomers {
			foodIDs := []int{foods[0].ID} // Simple: just one food item
			_, err := orderService.CreateOrder(ctx, customer.ID, foodIDs)
			if err != nil {
				t.Logf("Failed to create order for customer %d: %v", customer.ID, err)
			}
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopOrders:
				return
			case <-ticker.C:
				if time.Since(startTime) >= ciSmallTestDuration {
					return
				}

				// Create orders for all customers
				for _, customer := range allCustomers {
					foodIDs := []int{foods[0].ID} // Simple: just one food item
					_, err := orderService.CreateOrder(ctx, customer.ID, foodIDs)
					if err != nil {
						t.Logf("Failed to create order for customer %d: %v", customer.ID, err)
					}
				}
			}
		}
	}()

	// Wait for test duration
	time.Sleep(ciSmallTestDuration)
	close(stopOrders)
	close(stopReporting)
	wg.Wait()

	// Wait a bit for final orders to complete
	time.Sleep(ciSmallServingDuration + 2*time.Second)

	// Final statistics
	completed, incomplete, _ := orderService.GetOrderStats(ctx)

	log.Info("=== Small Load CI Test Results ===")
	log.Info("Regular Customers: %d", numRegularCustomers)
	log.Info("VIP Customers: %d", numVIPCustomers)
	log.Info("Cook Bots: %d", numCooks)
	log.Info("Test Duration: %v", ciSmallTestDuration)
	log.Info("Report Interval: %v", ciSmallReportInterval)
	log.Info("Final Completed: %d", completed)
	log.Info("Final Incomplete: %d", incomplete)
	log.Info("Completion Rate: %.2f%%", float64(completed)/float64(completed+incomplete)*100)

	t.Logf("\n=== Small Load CI Test Results ===")
	t.Logf("Regular Customers: %d", numRegularCustomers)
	t.Logf("VIP Customers: %d", numVIPCustomers)
	t.Logf("Cook Bots: %d", numCooks)
	t.Logf("Test Duration: %v", ciSmallTestDuration)
	t.Logf("Report Interval: %v", ciSmallReportInterval)
	t.Logf("Final Completed: %d", completed)
	t.Logf("Final Incomplete: %d", incomplete)
	t.Logf("Completion Rate: %.2f%%", float64(completed)/float64(completed+incomplete)*100)

	assert.Greater(t, completed, 0, "Should have completed some orders")
}
