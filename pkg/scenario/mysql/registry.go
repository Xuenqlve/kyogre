package mysql

import base "github.com/xuenqlve/kyogre/pkg/scenario/base"

func init() {
	base.RegisterBuilder(BuilderType, &Builder{}, false)
}
