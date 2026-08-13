package tables

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var createTablePattern = regexp.MustCompile(`(?m)^CREATE TABLE (t_[a-z0-9_]+) \(`)

// TestBaselineMatchesActiveTableConstants 验证可执行基线与当前真实仓储表名契约完全一致。
func TestBaselineMatchesActiveTableConstants(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob(filepath.Join("..", "baseline", "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	actual := make(map[string]struct{})
	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		for _, match := range createTablePattern.FindAllSubmatch(content, -1) {
			actual[string(match[1])] = struct{}{}
		}
	}
	expected := map[string]struct{}{
		TOutbox: {}, TPlayerAccount: {}, TAuthAuditLog: {},
		TPlayerProfile: {}, TPlayerBuilding: {}, TTrainingTask: {}, TUnitRoster: {}, TUnitGrantOperation: {},
		TResourceAccountBalance: {}, TResourceOperation: {},
		TConfigVersion: {}, TConfigAuditLog: {},
		TSchemaMigration: {}, TSnapshotTask: {}, TArchiveTask: {}, TBackupTask: {},
	}
	assert.Equal(t, expected, actual)
}
