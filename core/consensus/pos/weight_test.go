package pos

import (
	"fmt"
	"testing"
	"time"
)

type account struct {
	balance                      uint64
	lastUpdatedTimestamp         uint64
	weightAtLastUpdatedTimestamp uint64
}

var act *account

func init() {
	act = &account{}
	act.lastUpdatedTimestamp = uint64(time.Now().Unix())
	act.balance = uint64(600 * 1e4 * 1e4)
	act.weightAtLastUpdatedTimestamp = 0
}

func (a *account) decrease(v uint64) {

	if v == 0 {
		a.increase(v)
		return
	}
	balanceBeforeDecreament := a.balance
	a.balance -= v
	a.weightAtLastUpdatedTimestamp = a.weightAtLastUpdatedTimestamp *
		a.balance / balanceBeforeDecreament
	a.lastUpdatedTimestamp = uint64(time.Now().Unix())
	act.dump()
}

func (a *account) updateWeight() (uint64, uint64) {
	a.lastUpdatedTimestamp, a.weightAtLastUpdatedTimestamp = a.currentWeight()
	return a.lastUpdatedTimestamp, a.weightAtLastUpdatedTimestamp
}

func (a *account) increase(v uint64) {
	a.balance += v
	a.updateWeight()
	act.dump()
}

func (a *account) currentWeight() (uint64, uint64) {
	now := uint64(time.Now().Unix())
	return now, a.weightAtLastUpdatedTimestamp +
		uint64((now-a.lastUpdatedTimestamp)*a.balance)
}

func (a *account) dump() {

	fmt.Printf("weight: %d\n", a.weightAtLastUpdatedTimestamp)
	fmt.Printf("balance: %d\n", a.balance)
	fmt.Printf("lastUpdated: %d\n", a.lastUpdatedTimestamp)
	fmt.Printf("=================================\n")
}

func TestWeight(t *testing.T) {

	coin := uint64(3 * 10000 * 10000)
	day := uint64(2)

	act.increase(coin)
	time.Sleep(time.Duration(day) * time.Second)
	act.increase(coin)
	time.Sleep(time.Duration(day) * time.Second)
	act.increase(coin)
	time.Sleep(time.Duration(day) * time.Second)
	act.decrease(coin)
	time.Sleep(time.Duration(day) * time.Second)
	act.decrease(coin)

	time.Sleep(time.Duration(day) * time.Second)
	act.updateWeight()
	act.dump()

	time.Sleep(time.Duration(day) * time.Second)
	act.updateWeight()
	act.dump()

	for i := 0; i < 100; i++ {
		act.increase(2)
		act.decrease(2)
	}

	act.updateWeight()
	act.dump()

	time.Sleep(time.Duration(day) * time.Second)
	act.updateWeight()
	act.dump()
}

func TestTime(t *testing.T) {

	now := time.Now()

	Second := now.Unix()
	Nanosecond := now.Nanosecond()

	UnixNano := now.UnixNano()

	tm := time.Unix(10, 0)
	fmt.Println(tm.Format("2006-01-02 03:04:05 PM"))
	fmt.Println(tm.Format("02/01/2006 15:04:05 PM"))

	fmt.Printf("Second: %d\n", Second)
	fmt.Printf("Nanosecond: %d\n", Nanosecond)
	fmt.Printf("UnixNano: %d\n", UnixNano)
	fmt.Printf("=================================\n")

}
