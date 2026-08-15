package dshversion

import "testing"

func TestComparePrereleaseVersions(t *testing.T) {
	comparison, err := Compare("0.1.0-rc.7", "0.1.0-rc.6")
	if err != nil {
		t.Fatalf("比较 DSH 版本失败：%v", err)
	}
	if comparison <= 0 {
		t.Fatalf("预期 rc.7 新于 rc.6，得到：%d", comparison)
	}
}

func TestCompareStableVersionAfterPrerelease(t *testing.T) {
	comparison, err := Compare("0.1.0", "0.1.0-rc.6")
	if err != nil {
		t.Fatalf("比较 DSH 版本失败：%v", err)
	}
	if comparison <= 0 {
		t.Fatalf("预期正式版新于预发布版，得到：%d", comparison)
	}
}

func TestNormalizeRejectsInvalidVersion(t *testing.T) {
	if _, err := Normalize("latest"); err == nil {
		t.Fatal("不应接受未固定的 latest 版本")
	}
}
