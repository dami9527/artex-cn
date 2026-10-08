package server

import (
	"os"
	"path/filepath"
	"testing"
)

// 回归防御测试：保证 task_archive_package.go 的归档包删除准备错误文案为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。
// 一旦有人把这个字面量改回非中文文案，本测试即失败。
//
// 调用图判定(参见 errArchiveStagedAndOriginalCoexist 常量注释)：这个错误由后台
// 归档 worker runOneTaskArchiveJob(task_archives.go:102) 经 FailTaskArchiveJob
// 存入 task_archives.error 列，tasks/page.tsx 的 TaskArchivesPanel 以 {archive.error}
// 直接显示在页面上，属于用户可见专用(actool·planner 回喂为 0)。同一传播路径上的
// 同类错误(archiveTask·validateArchivePath)也已是中文，本次翻译与之保持一致。
//
// stageTaskArchivePackageDelete 是不碰 DB 的纯文件函数，因此在无 DB 的本机
// 可以同时造出原始文件与删除临时文件(.deleting-N)，驱动真正的矛盾分支。
func TestTaskArchiveStagedAndOriginalCoexistErrorLocalized(t *testing.T) {
	dataDir := t.TempDir()
	archivePath := taskArchivePath(dataDir, 21, "42")
	if err := os.MkdirAll(filepath.Dir(archivePath), archiveDirMode); err != nil {
		t.Fatal(err)
	}
	// 制造原始归档包与删除临时文件同时存在的矛盾状态。
	if err := os.WriteFile(archivePath, []byte("original"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	staged := archivePath + ".deleting-21"
	if err := os.WriteFile(staged, []byte("archive"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	_, moved, err := stageTaskArchivePackageDelete(archivePath, 21)
	if err == nil {
		t.Fatalf("原始文件与删除临时文件共存时必须返回错误 (moved=%v)", moved)
	}
	if moved {
		t.Fatal("共存矛盾状态下 moved 不应为 true")
	}
	assertChineseError(t, "staged_and_original_coexist", err.Error())
}
