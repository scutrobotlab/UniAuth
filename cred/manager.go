// Copyright 2021 The Casdoor Authors. All Rights Reserved.
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

package cred

type CredManager interface {
	GetHashedPassword(password string, userSalt string, organizationSalt string) string
	IsPasswordCorrect(password string, passwordHash string, userSalt string, organizationSalt string) bool
}

func GetCredManager(passwordType string) CredManager {
	switch passwordType {
	case "plain":
		return NewPlainCredManager()
	case "salt":
		return NewSha256SaltCredManager()
	case "sha512-salt":
		return NewSha512SaltCredManager()
	case "md5-salt":
		return NewMd5UserSaltCredManager()
	case "bcrypt":
		return NewBcryptCredManager()
	case "pbkdf2-salt":
		return NewPbkdf2SaltCredManager()
	case "argon2id":
		return NewArgon2idCredManager()
	default:
		return nil
	}
}
