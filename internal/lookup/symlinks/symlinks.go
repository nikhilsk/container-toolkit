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
	"fmt"
	"os"
	"path/filepath"
)

// ForceCreate creates a symlink at linkPath pointing to target.
// If a file or symlink already exists at linkPath, it is removed first.
// This ensures the symlink is created even if something exists at that location.
func ForceCreate(target, linkPath string) error {
	// Remove existing file/link if present
	// os.Remove works for both files and symlinks
	if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing file at %s: %w", linkPath, err)
	}

	// Create the new symlink
	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, target, err)
	}

	return nil
}

// Resolve reads the target of a symlink.
// Returns os.ErrNotExist if the link doesn't exist.
// Returns the target path (which may be relative or absolute) if successful.
func Resolve(linkPath string) (string, error) {
	target, err := os.Readlink(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf("failed to read symlink %s: %w", linkPath, err)
	}
	return target, nil
}

// ResolveAbsolute resolves a symlink and returns the absolute path of the target.
// This follows the symlink and converts relative targets to absolute paths.
func ResolveAbsolute(linkPath string) (string, error) {
	target, err := Resolve(linkPath)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(target) {
		return target, nil
	}

	// Relative target - resolve relative to link's directory
	linkDir := filepath.Dir(linkPath)
	absTarget := filepath.Join(linkDir, target)
	return filepath.Clean(absTarget), nil
}
