// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"log"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
)

const thresholdFetchInterval = 10 * time.Second

var resourceTypes = []prompt.Suggest{
	{Text: "clusters"}, // valid only for federation apiservers
	// aliases
	{Text: "cs"},
}

func init() {
	lastFetchedAt = new(sync.Map)
	serviceList = new(sync.Map)
}

/* LastFetchedAt */

var (
	lastFetchedAt *sync.Map
)

func shouldFetch(key string) bool {
	v, ok := lastFetchedAt.Load(key)
	if !ok {
		log.Printf("[WARN] Not found %s in lastFetchedAt", key)
		return true
	}
	t, ok := v.(time.Time)
	if !ok {
		return true
	}
	return time.Since(t) > thresholdFetchInterval
}

func updateLastFetchedAt(key string) {
	lastFetchedAt.Store(key, time.Now())
}

/* Service */

var (
	serviceList *sync.Map
)

func fetchServiceList(namespace string) {
	key := "service_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	// l, _ := getClient().CoreV1().Services(namespace).List(metav1.ListOptions{})
	// serviceList.Store(namespace, l)
	// return
}

func getServiceSuggestions() []prompt.Suggest {
	// namespace := metav1.NamespaceAll
	namespace := "test"
	go fetchServiceList(namespace)
	x, ok := serviceList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	_ = x
	// l, ok := x.(*corev1.ServiceAccountList)
	// if !ok || len(l.Items) == 0 {
	// 	return []prompt.Suggest{}
	// }
	// s := make([]prompt.Suggest, len(l.Items))
	// for i := range l.Items {
	// 	s[i] = prompt.Suggest{
	// 		Text: l.Items[i].Name,
	// 	}
	// }
	// return s
	return nil
}
