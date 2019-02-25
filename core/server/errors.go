// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

import (
	"fmt"

	pb "github.com/ok-chain/okchain/protos"
)

// DuplicateHandlerError returned if attempt to register same chaincodeID while a stream already exists.
type DuplicateHandlerError struct {
	To pb.PeerEndpoint
}

func (d *DuplicateHandlerError) Error() string {
	return fmt.Sprintf("duplicate Handler error: %+v", d.To)
}
