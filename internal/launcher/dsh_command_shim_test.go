package launcher

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDSHCommandShimDefaultsOnlyPluginProfile(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("测试需要 PATH 中存在 Node.js")
	}
	captureScript := filepath.Join(t.TempDir(), "capture-args.mjs")
	if err := os.WriteFile(captureScript, []byte("process.stdout.write(JSON.stringify(process.argv.slice(2)))\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err := prepareDSHCommandShim()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	environment, err := withDSHCommandShim(os.Environ(), directory, dshCommandSpec{
		nodePath:    nodePath,
		commandPath: nodePath,
		prefix:      []string{captureScript},
	})
	if err != nil {
		t.Fatal(err)
	}
	pathValue := environmentValue(environment, "PATH")
	if !strings.HasPrefix(pathValue, directory+string(os.PathListSeparator)) {
		t.Fatalf("PATH 没有优先使用兼容目录：%s", pathValue)
	}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"plugin 默认 web", []string{"plugin", "--help"}, []string{"plugin", "--profile", "web", "--help"}},
		{"plugin 保留显式 profile", []string{"plugin", "--profile", "tui", "--help"}, []string{"plugin", "--profile", "tui", "--help"}},
		{"plugin 保留等号 profile", []string{"plugin", "--profile=tui", "--help"}, []string{"plugin", "--profile=tui", "--help"}},
		{"其他命令不改写", []string{"--version"}, []string{"--version"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := runDSHShimTestCommand(t, environment, test.args)
			var actual []string
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &actual); err != nil {
				t.Fatalf("无法解析兼容入口输出 %q：%v", output, err)
			}
			if strings.Join(actual, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("转发参数 = %#v，期望 %#v", actual, test.want)
			}
		})
	}
}

func runDSHShimTestCommand(t *testing.T, environment, args []string) string {
	t.Helper()
	commandText := "dsh " + strings.Join(args, " ")
	var commands []*exec.Cmd
	if runtime.GOOS == "windows" {
		powerShell, err := exec.LookPath("pwsh.exe")
		if err != nil {
			t.Fatal("Windows 兼容入口测试需要 pwsh.exe")
		}
		commands = []*exec.Cmd{
			exec.Command(powerShell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", commandText),
			exec.Command("cmd.exe", "/d", "/c", commandText),
		}
	} else {
		commands = []*exec.Cmd{exec.Command("/bin/sh", "-c", commandText)}
	}
	outputs := make([]string, 0, len(commands))
	for _, command := range commands {
		command.Env = environment
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("执行 %q 失败：%v（%s）", commandText, err, output)
		}
		outputs = append(outputs, strings.TrimSpace(string(output)))
	}
	for _, output := range outputs[1:] {
		if output != outputs[0] {
			t.Fatalf("不同 Shell 的转发结果不一致：%q 与 %q", outputs[0], output)
		}
	}
	return outputs[0]
}

func environmentValue(environment []string, key string) string {
	for _, entry := range environment {
		entryKey, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(entryKey, key) {
			return value
		}
	}
	return ""
}
