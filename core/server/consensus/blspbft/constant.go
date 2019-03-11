// Copyright The go-okchain Authors 2018,  All rights reserved.

/*
Copyright 2018 The Okchain Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
* DESCRIPTION

 */

package blspbft

type ConsensusState_Type int32

const (
	ToleranceFraction                     = 0.667
	INITIAL           ConsensusState_Type = 0

	// for lead
	ANNOUNCE_DONE           ConsensusState_Type = 1
	CHALLENGE_DONE          ConsensusState_Type = 3
	COLLECTIVESIG_DONE      ConsensusState_Type = 5
	FINALCHALLENGE_DONE     ConsensusState_Type = 7
	FINALCOLLECTIVESIG_DONE ConsensusState_Type = 9
	BROADCASTBLOCK_DONE		ConsensusState_Type = 11

	// for backup
	COMMIT_DONE        ConsensusState_Type = 2
	RESPONSE_DONE      ConsensusState_Type = 4
	FINALCOMMIT_DONE   ConsensusState_Type = 6
	FINALRESPONSE_DONE ConsensusState_Type = 8
)
