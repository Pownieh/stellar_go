package actions

import "github.com/pownieh/stellar_go/services/horizon/internal/corestate"

type CoreStateGetter interface {
	GetCoreState() corestate.State
}
