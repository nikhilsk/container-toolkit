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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceCreate(t *testing.T) {
	tmpDir := t.TempDir()

	target := "target-file"
	linkPath := filepath.Join(tmpDir, "link")

	// Create initial symlink
	err := ForceCreate(target, linkPath)
	require.NoError(t, err)

	result, err := os.Readlink(linkPath)
	require.NoError(t, err)
	assert.Equal(t, target, result)

	// Force create with different target
	newTarget := "new-target"
	err = ForceCreate(newTarget, linkPath)
	require.NoError(t, err)

	result, err = os.Readlink(linkPath)
	require.NoError(t, err)
	assert.Equal(t, newTarget, result)

	// Force create over a regular file
	filePath := filepath.Join(tmpDir, "regular-file")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	err = ForceCreate(target, filePath)
	require.NoError(t, err)

	result, err = os.Readlink(filePath)
	require.NoError(t, err)
	assert.Equal(t, target, result)
}

func TestResolve(t *testing.T) {
	tmpDir := t.TempDir()

	target := "some-target"
	linkPath := filepath.Join(tmpDir, "link")

	// Non-existent link
	_, err := Resolve(linkPath)
	assert.ErrorIs(t, err, os.ErrNotExist)

	// Create and resolve
	require.NoError(t, os.Symlink(target, linkPath))
	result, err := Resolve(linkPath)
	require.NoError(t, err)
	assert.Equal(t, target, result)

	// Absolute target
	absTarget := "/absolute/path"
	absLink := filepath.Join(tmpDir, "abs-link")
	require.NoError(t, os.Symlink(absTarget, absLink))
	result, err = Resolve(absLink)
	require.NoError(t, err)
	assert.Equal(t, absTarget, result)
}

func TestResolveAbsolute(t *testing.T) {
	tmpDir := t.TempDir()

	// Relative target
	relTarget := "subdir/target"
	linkPath := filepath.Join(tmpDir, "link")
	require.NoError(t, os.Symlink(relTarget, linkPath))

	result, err := ResolveAbsolute(linkPath)
	require.NoError(t, err)
	expected := filepath.Join(tmpDir, relTarget)
	assert.Equal(t, expected, result)

	// Absolute target
	absTarget := "/absolute/target"
	absLink := filepath.Join(tmpDir, "abs-link")
	require.NoError(t, os.Symlink(absTarget, absLink))

	result, err = ResolveAbsolute(absLink)
	require.NoError(t, err)
	assert.Equal(t, absTarget, result)

	// Parent directory reference
	parentTarget := "../other/target"
	parentLink := filepath.Join(tmpDir, "parent-link")
	require.NoError(t, os.Symlink(parentTarget, parentLink))

	result, err = ResolveAbsolute(parentLink)
	require.NoError(t, err)
	expected = filepath.Clean(filepath.Join(tmpDir, parentTarget))
	assert.Equal(t, expected, result)
}
