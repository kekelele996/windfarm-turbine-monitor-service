package workorder

import "fmt"

// Part is a consumable spare part.
type Part struct {
	SKU      string
	Name     string
	OnHand   int
	Reserved int
}

// Inventory tracks spare parts and reservations.
type Inventory struct {
	parts map[string]*Part
}

func NewInventory(parts []Part) *Inventory {
	inv := &Inventory{parts: map[string]*Part{}}
	for _, p := range parts {
		cp := p
		inv.parts[p.SKU] = &cp
	}
	return inv
}

func (i *Inventory) Available(sku string) int {
	if p, ok := i.parts[sku]; ok {
		return p.OnHand - p.Reserved
	}
	return 0
}

// Reserve decrements available inventory for an order.
func (i *Inventory) Reserve(sku string, qty int) error {
	p, ok := i.parts[sku]
	if !ok {
		return fmt.Errorf("part %s unknown", sku)
	}
	if p.OnHand-p.Reserved < qty {
		return fmt.Errorf("part %s insufficient: have %d need %d", sku, p.OnHand-p.Reserved, qty)
	}
	p.Reserved += qty
	return nil
}

// Release returns reserved parts to availability.
func (i *Inventory) Release(sku string, qty int) {
	if p, ok := i.parts[sku]; ok {
		p.Reserved -= qty
		if p.Reserved < 0 {
			p.Reserved = 0
		}
	}
}
