package member

import (
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/member/infrastructure"
)

var ProviderSet = wire.NewSet(
	infrastructure.NewPostgresMemberRepository,
)
