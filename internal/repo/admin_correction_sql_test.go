package repo

import (
	"strings"
	"testing"

	"csu-star-backend/internal/model"
)

// TestCorrectionCurrentValueSQLCoversAllFields 守卫 model.CorrectionFieldsByTargetType 与
// correctionCurrentValueSQL 之间的漂移。
//
// 为什么需要这个测试：service 的 apply switch 新增一个可纠错字段时，如果忘了同步这里的 CASE，
// SQL 会走到 ELSE ” 分支静默返回空字符串——审核员看到「当前值：（空）」，
// 会以为该字段本来就没填，从而误采纳一条错误的纠错。没有任何报错能提示这一点。
func TestCorrectionCurrentValueSQLCoversAllFields(t *testing.T) {
	for targetType, fields := range model.CorrectionFieldsByTargetType {
		if !strings.Contains(correctionCurrentValueSQL, "corrections.target_type = '"+string(targetType)+"'") {
			t.Errorf("correctionCurrentValueSQL 缺少 target_type = %q 的分支", targetType)
			continue
		}
		for _, field := range fields {
			if !strings.Contains(correctionCurrentValueSQL, "WHEN '"+field+"'") {
				t.Errorf("correctionCurrentValueSQL 缺少 target_type=%s 的字段 %q 的 WHEN 分支", targetType, field)
			}
		}
	}
}

// TestCorrectionCurrentValueSQLIsAppendable 守卫拼接契约。
//
// correctionCurrentValueSQL 是被 append 到 Select 字符串末尾的，它自带前导逗号。
// 如果哪天有人给前面那段 Select 补上尾逗号（看起来完全合理），就会拼出 "...processor_role,,CASE"，
// 而这是一个只在真正查库时才会炸的语法错误——单元测试和 go build 都发现不了。
func TestCorrectionCurrentValueSQLIsAppendable(t *testing.T) {
	if !strings.HasPrefix(correctionCurrentValueSQL, ",") {
		t.Fatal("correctionCurrentValueSQL 必须以逗号开头，否则拼接后会与前一个 select 项粘连")
	}
	if strings.Contains(correctionCurrentValueSQL, ",,") {
		t.Error("correctionCurrentValueSQL 出现连续逗号")
	}
	for _, alias := range []string{"current_value", "current_value_display", "suggested_value_display", "target_missing"} {
		if !strings.Contains(correctionCurrentValueSQL, "AS "+alias) {
			t.Errorf("correctionCurrentValueSQL 缺少别名 %q，对应的 DTO 字段会静默保持零值", alias)
		}
	}
}

// TestCorrectionCurrentValueSQLHasNoExtraFields 反向守卫：SQL 里不该出现白名单之外的字段，
// 否则说明白名单漏登记了某个字段，前端的字段名字典也会跟着漏。
func TestCorrectionCurrentValueSQLHasNoExtraFields(t *testing.T) {
	known := map[string]bool{}
	for _, fields := range model.CorrectionFieldsByTargetType {
		for _, field := range fields {
			known[field] = true
		}
	}
	for _, line := range strings.Split(correctionCurrentValueSQL, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "WHEN '") {
			continue
		}
		field := trimmed[len("WHEN '"):]
		field = field[:strings.Index(field, "'")]
		if !known[field] {
			t.Errorf("correctionCurrentValueSQL 出现了未登记在 model.CorrectionFieldsByTargetType 的字段 %q", field)
		}
	}
}
