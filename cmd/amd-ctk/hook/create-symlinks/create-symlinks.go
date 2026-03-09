/**
# Copyright (c) Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package symlinks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/sys/symlink"
	"github.com/urfave/cli/v2"

	"github.com/ROCm/container-toolkit/internal/logger"
	"github.com/ROCm/container-toolkit/internal/lookup/symlinks"
	"github.com/ROCm/container-toolkit/internal/oci"
)

type config struct {
	links         cli.StringSlice
	containerSpec string
}

// AddNewCommand creates the create-symlinks command
func AddNewCommand() *cli.Command {
	cfg := config{}

	return &cli.Command{
		Name:      "create-symlinks",
		Usage:     "Create symlinks in the container filesystem",
		UsageText: "amd-ctk hook create-symlinks --link target::link [--link ...] [--container-spec path]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "link",
				Usage:       "Symlink specification in format target::link (can be repeated)",
				Destination: &cfg.links,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "container-spec",
				Usage:       "Path to OCI container spec (empty or '-' reads from stdin)",
				Destination: &cfg.containerSpec,
				Value:       "",
			},
		},
		Action: func(c *cli.Context) error {
			return run(&cfg)
		},
	}
}

func run(cfg *config) error {
	// Load container state
	state, err := oci.LoadContainerState(cfg.containerSpec)
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	containerRoot, err := state.GetContainerRoot()
	if err != nil {
		return fmt.Errorf("failed to determine container root: %w", err)
	}

	logger.Log.Printf("Container root: %s", containerRoot)

	// Process each link specification
	created := make(map[string]bool)
	for _, linkSpec := range cfg.links.Value() {
		if created[linkSpec] {
			logger.Log.Printf("Link %s already processed", linkSpec)
			continue
		}

		parts := strings.Split(linkSpec, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid symlink specification %s (expected target::link)", linkSpec)
		}

		target, link := parts[0], parts[1]
		if err := createLink(containerRoot, target, link); err != nil {
			return fmt.Errorf("failed to create link %s -> %s: %w", link, target, err)
		}
		created[linkSpec] = true
	}

	return nil
}

// createLink creates a symbolic link in the container root.
// This is equivalent to: chroot containerRoot ln -sf target link
func createLink(containerRoot, target, link string) error {
	linkPath := filepath.Join(containerRoot, link)

	// Check if link already exists and points to correct target
	exists, err := linkExists(target, linkPath)
	if err != nil {
		return fmt.Errorf("failed to check link existence: %w", err)
	}
	if exists {
		logger.Log.Printf("Link %s already exists with correct target", linkPath)
		return nil
	}

	// Resolve link parent within container root to handle symlinks in path
	// We don't resolve the full link path itself, as that would follow an
	// existing symlink at the location we want to create/overwrite
	resolvedParent, err := symlink.FollowSymlinkInScope(filepath.Dir(linkPath), containerRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve link parent %s in %s: %w",
			filepath.Dir(link), containerRoot, err)
	}
	resolvedLinkPath := filepath.Join(resolvedParent, filepath.Base(linkPath))

	logger.Log.Printf("Creating symlink %s -> %s", resolvedLinkPath, target)

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(resolvedLinkPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create or overwrite the symlink
	if err := symlinks.ForceCreate(target, resolvedLinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// linkExists checks if the link exists and points to the target
func linkExists(target, link string) (bool, error) {
	currentTarget, err := symlinks.Resolve(link)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to resolve symlink: %w", err)
	}
	return currentTarget == target, nil
}
