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
	"strings"
	"testing"

	"github.com/ROCm/container-toolkit/internal/logger"
	"github.com/ROCm/container-toolkit/internal/lookup/symlinks"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	logger.Init(true)
}

type dirOrLink struct {
	path   string
	target string // empty for directories
}

func makeFs(tmpdir string, items ...dirOrLink) error {
	if err := os.MkdirAll(tmpdir, 0755); err != nil {
		return err
	}
	for _, item := range items {
		fullPath := filepath.Join(tmpdir, item.path)
		if item.target == "" {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := os.Symlink(item.target, fullPath); err != nil && !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

func TestLinkExists(t *testing.T) {
	setup(t)
	tmpDir := t.TempDir()

	assert.NoError(t, makeFs(tmpDir,
		dirOrLink{path: "a/b/c", target: "d"},
		dirOrLink{path: "a/b/e", target: "/a/b/f"},
	))

	exists, err := linkExists("d", filepath.Join(tmpDir, "a/b/c"))
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = linkExists("/a/b/f", filepath.Join(tmpDir, "a/b/e"))
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = linkExists("different", filepath.Join(tmpDir, "a/b/c"))
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = linkExists("missing", filepath.Join(tmpDir, "nonexistent"))
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateLink(t *testing.T) {
	setup(t)

	tests := []struct {
		name          string
		containerFs   []dirOrLink
		target        string
		link          string
		wantErr       bool
		verifyTarget  string
		verifyMissing string
	}{
		{
			name:         "simple_relative_link",
			containerFs:  []dirOrLink{{path: "lib"}},
			target:       "libfoo.so.1",
			link:         "/lib/libfoo.so",
			verifyTarget: "libfoo.so.1",
		},
		{
			name:         "absolute_target",
			containerFs:  []dirOrLink{{path: "lib"}},
			target:       "/usr/lib/libfoo.so.1",
			link:         "/lib/libfoo.so",
			verifyTarget: "/usr/lib/libfoo.so.1",
		},
		{
			name: "link_through_symlink_dir",
			containerFs: []dirOrLink{
				{path: "lib/foo", target: "/"},
			},
			target:       "libbar.so.1",
			link:         "/lib/foo/libbar.so",
			verifyTarget: "libbar.so.1",
		},
		{
			name: "overwrite_existing_link",
			containerFs: []dirOrLink{
				{path: "lib/libfoo.so", target: "old-target"},
			},
			target:       "new-target",
			link:         "/lib/libfoo.so",
			verifyTarget: "new-target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerRoot := t.TempDir()
			assert.NoError(t, makeFs(containerRoot, tt.containerFs...))

			err := createLink(containerRoot, tt.target, tt.link)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if tt.verifyTarget != "" {
				linkPath := filepath.Join(containerRoot, tt.link)
				// Handle symlink dir resolution
				if strings.Contains(tt.link, "foo") {
					linkPath = filepath.Join(containerRoot, filepath.Base(tt.link))
				}
				target, err := symlinks.Resolve(linkPath)
				assert.NoError(t, err)
				assert.Equal(t, tt.verifyTarget, target)
			}

			if tt.verifyMissing != "" {
				_, err := os.Stat(filepath.Join(containerRoot, tt.verifyMissing))
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestCreateLinkCreatesParentDirs(t *testing.T) {
	setup(t)
	containerRoot := t.TempDir()

	err := createLink(containerRoot, "target", "/a/b/c/link")
	assert.NoError(t, err)

	target, err := symlinks.Resolve(filepath.Join(containerRoot, "a/b/c/link"))
	assert.NoError(t, err)
	assert.Equal(t, "target", target)
}
