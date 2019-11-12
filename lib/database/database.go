/*
 * Copyright 2019 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package database

import (
	"context"
	"github.com/SENERGY-Platform/incident-worker/lib/database/mongo"
	"github.com/SENERGY-Platform/incident-worker/lib/interfaces"
	"github.com/SENERGY-Platform/incident-worker/lib/util"
)

type FactoryType struct{}

var Factory = &FactoryType{}

func (this *FactoryType) Get(ctx context.Context, config util.Config) (interfaces.Database, error) {
	return mongo.New(ctx, config)
}
