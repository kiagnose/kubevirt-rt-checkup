/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package config

import (
	"errors"
	"strconv"
	"time"

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"
)

const (
	TargetNodeParamName                   = "targetNode"
	GuestImageSourcePVCNamespaceParamName = "guestImageSourcePVCNamespace"
	GuestImageSourcePVCNameParamName      = "guestImageSourcePVCName"
	OslatDurationParamName                = "oslatDuration"
	OslatLatencyThresholdParamName        = "oslatLatencyThresholdMicroSeconds"
)

const (
	OslatDefaultDuration         = 5 * time.Minute
	OslatDefaultLatencyThreshold = 40 * time.Microsecond
)

var (
	ErrInvalidOslatDuration         = errors.New("invalid oslat duration")
	ErrInvalidOslatLatencyThreshold = errors.New("invalid oslat latency threshold")
)

type Config struct {
	PodName                      string
	PodUID                       string
	TargetNode                   string
	GuestImageSourcePVCNamespace string
	GuestImageSourcePVCName      string
	OslatDuration                time.Duration
	OslatLatencyThreshold        time.Duration
}

func New(baseConfig kconfig.Config) (Config, error) {
	newConfig := Config{
		PodName:                      baseConfig.PodName,
		PodUID:                       baseConfig.PodUID,
		TargetNode:                   baseConfig.Params[TargetNodeParamName],
		GuestImageSourcePVCNamespace: baseConfig.Params[GuestImageSourcePVCNamespaceParamName],
		GuestImageSourcePVCName:      baseConfig.Params[GuestImageSourcePVCNameParamName],
		OslatDuration:                OslatDefaultDuration,
		OslatLatencyThreshold:        OslatDefaultLatencyThreshold,
	}

	if rawOslatDuration := baseConfig.Params[OslatDurationParamName]; rawOslatDuration != "" {
		oslatDuration, err := time.ParseDuration(rawOslatDuration)
		if err != nil {
			return Config{}, ErrInvalidOslatDuration
		}
		newConfig.OslatDuration = oslatDuration
	}

	if rawOslatLatencyThreshold := baseConfig.Params[OslatLatencyThresholdParamName]; rawOslatLatencyThreshold != "" {
		oslatLatencyThresholdMicroSeconds, err := strconv.Atoi(rawOslatLatencyThreshold)
		if err != nil {
			return Config{}, ErrInvalidOslatLatencyThreshold
		}
		newConfig.OslatLatencyThreshold = time.Duration(oslatLatencyThresholdMicroSeconds) * time.Microsecond
	}

	return newConfig, nil
}
