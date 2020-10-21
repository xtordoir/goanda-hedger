package hedge

import (
	"math"
	"sync"
)

// Tradeable interface defines a PositionSize function
// to compute the size of a trade to set expected position at a given price
type Tradeable interface {
	PositionSize(price float64) int64
	GetSize() int64
}

// StaticHedge definition
type StaticHedge struct {
	Size int64
}

// PositionSize computation (tradable interface)
func (hedge *StaticHedge) PositionSize(price float64) int64 {
	return hedge.Size
}

// GetSize Tradeable interface
func (hedge StaticHedge) GetSize() int64 {
	return hedge.Size
}

// DynamicHedge definition
type DynamicHedge struct {

	// static parameters of the Hedge
	Price0 float64
	Size0  int64
	Scale  float64

	// state of the Hedge
	LengthUp   int64
	LengthDown int64
	BoxUp      int64
	BoxDown    int64
	//I          int64
	//LastCrossI int64
	Size int64
}

func (hedge *DynamicHedge) nextI(price float64) int64 {
	// compute direction move size with sign
	z := -100.0 * (price - hedge.Price0) / hedge.Price0 / hedge.Scale
	// extract integer part
	y, _ := math.Modf(z)
	return int64(y)
}

func (hedge DynamicHedge) nextBox(price float64) int64 {
	z := 100.0 * (price - hedge.Price0) / hedge.Price0 / hedge.Scale
	return int64(math.Floor(z))
}

// PositionSize size computation (tradable interface)
func (hedge *DynamicHedge) PositionSize(price float64) int64 {
	// hard protection against Faulty Scale
	if hedge.Scale <= 0 {
		return 0
	}
	// init hack: set Price0 and box bounds
	if hedge.Price0 == 0.0 || (hedge.BoxUp == 0 && hedge.BoxDown == 0) {
		hedge.BoxUp = 1
		hedge.BoxDown = -2
		hedge.Price0 = price
	}

	//target := int64(float64(hedge.Size0) * (price/hedge.Price0 - 1.0) / hedge.Scale)
	i := hedge.nextI(price)
	bi := hedge.nextBox(price)

	// price went out of boundaries, we adapt position and box bounds
	if bi <= hedge.BoxDown {
		hedge.LengthDown += hedge.BoxDown - bi + 1
		hedge.BoxUp = bi + 2
		hedge.BoxDown = bi - 1
		hedge.Size = hedge.Size0 * i

	} else if bi >= hedge.BoxUp {
		hedge.LengthUp += bi + 1 - hedge.BoxUp
		hedge.BoxUp = bi + 1
		hedge.BoxDown = bi - 2
		hedge.Size = hedge.Size0 * i
	}

	return hedge.Size
}

// GetSize Tradeable interface
func (hedge DynamicHedge) GetSize() int64 {
	return hedge.Size
}

// Inventory is an Inventory of hedges
type Inventory map[*Tradeable]bool

// PositionSize  computation (tradable interface)
func (inv *Inventory) PositionSize(price float64) int64 {
	size := int64(0)
	for hedge := range *inv {
		size += (*hedge).PositionSize(price)
	}
	return size
}

// GetSize Tradeable interface
func (inv Inventory) GetSize() int64 {
	size := int64(0)
	for hedge := range inv {
		size += (*hedge).GetSize()
	}
	return size
}

// Manageable is the interface to add Hedges
type Manageable interface {
	AddHedge(hedge Tradeable)
	ListHedges() []Tradeable
}

// InventoryManager is a manager for the inventory Inventory with a RWMutex
type InventoryManager struct {
	sync.RWMutex
	Inventory *Inventory
}

// PositionSize of the inventory Lock protected to the inventory
func (manager *InventoryManager) PositionSize(price float64) int64 {
	manager.Lock()
	defer manager.Unlock()
	return manager.Inventory.PositionSize(price)
}

// GetSize of the inventory Lock protected to the inventory
func (manager *InventoryManager) GetSize() int64 {
	manager.RLock()
	defer manager.RUnlock()
	return manager.Inventory.GetSize()
}

// AddHedge to the inventory, Lock protected
func (manager *InventoryManager) AddHedge(hedge Tradeable) {
	manager.Lock()
	defer manager.Unlock()
	(*manager.Inventory)[&hedge] = true
}

// ListHedges in the inventory, Lock protected
func (manager *InventoryManager) ListHedges() []Tradeable {
	manager.RLock()
	defer manager.RUnlock()
	arr := make([]Tradeable, len(*manager.Inventory))
	i := 0
	for item := range *manager.Inventory {
		arr[i] = *item
		i++
	}
	return arr
}
