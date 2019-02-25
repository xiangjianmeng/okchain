// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

type IConsensusBackupFactoryType func(server *PeerServer) IConsensusBackup
type IConsensusLeadFactoryType func(server *PeerServer) IConsensusLead

var ConsensusBackupFactory IConsensusBackupFactoryType
var ConsensusLeadFactory IConsensusLeadFactoryType

func (factory IConsensusBackupFactoryType) ProduceConsensusBackup(p *PeerServer) IConsensusBackup {
	return factory(p)
}

func (factory IConsensusLeadFactoryType) ProduceConsensusLead(p *PeerServer) IConsensusLead {
	return factory(p)
}
