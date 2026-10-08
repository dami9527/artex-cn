package server

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// TestSkillZipErrorsLocalized 是 F3b skill_zip.go 这一束的守卫：
// 每一条用户可见的 skill 上传错误响应(fsUploadSkill 经
// writeErr(400, err.Error()) 暴露，见 server_mgmt.go)都必须是中文，
// 且任何压缩方法显示名都不得残留旧文案。任何一条改回旧文案都会让本测试失败。
func TestSkillZipErrorsLocalized(t *testing.T) {
	// 用样例参数渲染的消息常量。
	assertChineseError(t, "errSkillZipParse",
		fmt.Errorf(errSkillZipParse, fmt.Errorf("boom")).Error())
	assertChineseError(t, "errSkillZipEncrypted",
		fmt.Sprintf(errSkillZipEncrypted, "demo/SKILL.md"))
	assertChineseError(t, "errSkillZipUnsupported",
		fmt.Sprintf(errSkillZipUnsupported, "LZMA", 14, "demo/SKILL.md"))

	// 非 zip 上传必须以中文提示到达用户，而不是原始的 stdlib 错误。
	if _, err := newSkillZipReader([]byte("this is not a zip archive")); err == nil {
		t.Fatal("非 zip 输入必须返回错误")
	} else {
		assertChineseError(t, "newSkillZipReader", err.Error())
		if !strings.Contains(err.Error(), "压缩包") {
			t.Fatalf("parse error = %q, want '压缩包' 提示", err.Error())
		}
	}

	// 任何压缩方法显示名都不得残留谚文；
	// 两个已本地化的(AES、未知回落)必须是中文。
	for m, name := range zipMethodNames {
		for _, r := range name {
			if unicode.Is(unicode.Hangul, r) {
				t.Fatalf("zipMethodNames[%d] = %q 仍残留谚文", m, name)
			}
		}
	}
	assertChineseError(t, "zipMethodName(AES)", zipMethodName(zipMethodAES))
	assertChineseError(t, "zipMethodName(unknown)", zipMethodName(0xffff))
}
