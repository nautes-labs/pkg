// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package context

import (
	rawctx "context"
	"fmt"

	nautescfg "github.com/nautes-labs/pkg/pkg/nautesconfigs"
)

type ContextKey string

func FromConfigContext(ctx rawctx.Context, key ContextKey) (*nautescfg.Config, error) {
	inter := ctx.Value(key)
	cfg, ok := inter.(nautescfg.Config)
	if !ok {
		return nil, fmt.Errorf("get nautes config failed")
	}

	return &cfg, nil
}
