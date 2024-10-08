// Copyright 2023 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notification

import (
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/webpush"
)

func NewWebpushProvider(publicKey string, privateKey string, endpoint string) (*notify.Notify, error) {
	webpushSrv := webpush.New(publicKey, privateKey)

	subscription := webpush.Subscription{
		Endpoint: endpoint,
	}
	webpushSrv.AddReceivers(subscription)

	notifier := notify.NewWithServices(webpushSrv)

	return notifier, nil
}
