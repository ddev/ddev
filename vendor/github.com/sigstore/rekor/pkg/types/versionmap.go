//
// Copyright 2021 The Sigstore Authors.
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

package types

import (
	"fmt"
	"sync"

	"github.com/sigstore/rekor/pkg/internal/log"
)

// VersionEntryFactoryMap defines a map-like interface to find the correct implementation for a version string
// This could be a simple map[string][EntryFactory], or something more elegant
type VersionEntryFactoryMap interface {
	GetEntryFactory(string) (EntryFactory, error) // return the entry factory for the specified version
	SetEntryFactory(string, EntryFactory) error   // set the entry factory for the specified version
	Count() int                                   // return the count of entry factories currently in the map
	SupportedVersions() []string                  // return a list of versions currently stored in the map
}

// EntryFactoryMap implements a thread-safe map of version strings to EntryFactory functions
type EntryFactoryMap struct {
	factoryMap map[string]EntryFactory

	sync.RWMutex
}

func NewEntryFactoryMap() VersionEntryFactoryMap {
	s := EntryFactoryMap{}
	s.factoryMap = make(map[string]EntryFactory)
	return &s
}

func (s *EntryFactoryMap) Count() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.factoryMap)
}

func (s *EntryFactoryMap) GetEntryFactory(version string) (EntryFactory, error) {
	s.RLock()
	defer s.RUnlock()

	if ef, ok := s.factoryMap[version]; ok {
		return ef, nil
	}

	return nil, fmt.Errorf("unable to locate entry for version %s", version)
}

func (s *EntryFactoryMap) SetEntryFactory(version string, ef EntryFactory) error {
	s.Lock()
	defer s.Unlock()

	if version == "" {
		err := fmt.Errorf("empty version string")
		log.Logger.Error(err)
		return err
	}

	s.factoryMap[version] = ef
	return nil
}

func (s *EntryFactoryMap) SupportedVersions() []string {
	s.RLock()
	defer s.RUnlock()
	var versions []string
	for k := range s.factoryMap {
		versions = append(versions, k)
	}
	return versions
}
