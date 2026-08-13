import "./style.css";
import {
  backend,
  onCommandEvent,
  onStatusEvent,
  type Settings,
  type Status,
} from "./wails";

const root = document.querySelector<HTMLElement>("#app");

if (!root) {
  throw new Error("找不到应用根节点");
}

const escapeHTML = (value: string): string =>
  value.replace(
    /[&<>'"]/g,
    (character) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#39;",
        '"': "&quot;",
      })[character] ?? character,
  );

const icons = {
  browser: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 3h6v6M10 14 21 3M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/></svg>',
  help: '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M9.6 9a2.6 2.6 0 1 1 4.3 2c-1.1.8-1.9 1.3-1.9 3M12 18h.01"/></svg>',
  logs: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7.5V6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1.5"/><path d="M3.4 10h17.2a1 1 0 0 1 1 1.2l-1.4 7A2 2 0 0 1 18.3 20H5.7a2 2 0 0 1-1.9-1.6l-1.4-7A1 1 0 0 1 3.4 10Z"/></svg>',
  restart: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"/></svg>',
  settings: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h10M18 6h2M4 12h2M10 12h10M4 18h7M15 18h5"/><circle cx="16" cy="6" r="2"/><circle cx="8" cy="12" r="2"/><circle cx="13" cy="18" r="2"/></svg>',
  shell: '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M8 9v11"/></svg>',
} as const;

const initialStatus: Status = {
  state: "starting",
  message: "正在连接桌面宿主…",
  version: "dev",
  dshVersion: "0.1.0-rc.6",
};

let currentStatus = initialStatus;

const render = (status: Status): void => {
  currentStatus = status;
  if (status.state === "ready" && status.url) {
    renderReady(status);
    return;
  }

  const isBusy = status.state === "starting" || status.state === "idle";
  const eyebrow = isBusy ? "LOCAL RUNTIME" : "STARTUP DIAGNOSTIC";
  const description = isBusy
    ? "桌面宿主正在准备官方 DSH Web UI。首次运行需要下载较大的依赖闭包，可能耗时数分钟。"
    : "桌面宿主没有修改 DSH 内核。你可以调整代理、查看原始日志，修复问题后重试。";

  root.innerHTML = `
    <section class="splash">
      <div class="ambient ambient-one"></div>
      <div class="ambient ambient-two"></div>
      <div class="panel">
        <div class="mark" aria-hidden="true"><span>D</span></div>
        <p class="eyebrow">${eyebrow}</p>
        <h1>${escapeHTML(status.message)}</h1>
        <p class="description">${description}</p>
        ${
          status.detail
            ? `<pre class="detail">${escapeHTML(status.detail)}</pre>`
            : `<div class="progress" role="progressbar" aria-label="正在启动"><span></span></div>`
        }
        <div class="actions">
          ${isBusy ? "" : `<button class="button primary" data-action="retry">${icons.restart}<span>重新启动</span></button>`}
          <button class="button secondary" data-action="settings">${icons.settings}<span>代理设置</span></button>
          <button class="button secondary" data-action="logs">${icons.logs}<span>打开日志</span></button>
          <button class="button secondary" data-action="help">${icons.help}<span>帮助</span></button>
        </div>
        <footer>
          <span>Desktop ${escapeHTML(status.version)}</span>
          <span class="dot"></span>
          <span>DSH ${escapeHTML(status.dshVersion)}</span>
        </footer>
      </div>
    </section>
  `;
  bindCommonActions();
};

const renderReady = (status: Status): void => {
  const source = status.url ?? "";
  root.innerHTML = `
    <section class="workspace">
      <div class="runtime-bar">
        <div class="runtime-state"><span></span> DSH ${escapeHTML(status.dshVersion)}</div>
        <details class="shell-menu">
          <summary>${icons.shell}<span>桌面工具</span></summary>
          <div class="shell-menu-popover">
            <button data-action="settings">${icons.settings}<span>代理与启动设置</span></button>
            <button data-action="restart">${icons.restart}<span>重新启动 DSH</span></button>
            <button data-action="logs">${icons.logs}<span>打开日志目录</span></button>
            <button data-action="browser">${icons.browser}<span>在浏览器中打开</span></button>
            <button data-action="help">${icons.help}<span>使用帮助</span></button>
          </div>
        </details>
      </div>
      <iframe
        class="dsh-frame"
        src="${escapeHTML(source)}"
        title="DeepSeek Harness"
        allow="clipboard-read; clipboard-write"
      ></iframe>
    </section>
  `;
  bindCommonActions();
  root.querySelector<HTMLButtonElement>("[data-action='browser']")?.addEventListener("click", () => {
    void backend().OpenInBrowser();
  });
  root.querySelector<HTMLButtonElement>("[data-action='restart']")?.addEventListener("click", () => {
    render({ ...status, state: "starting", message: "正在重新启动 DeepSeek Harness…", detail: undefined });
    void backend().Retry();
  });
};

const bindCommonActions = (): void => {
	root.querySelectorAll<HTMLButtonElement>(".shell-menu-popover button").forEach((button) => {
		button.addEventListener("click", () => {
			const menu = button.closest<HTMLDetailsElement>(".shell-menu");
			if (menu) menu.open = false;
		});
	});
  root.querySelector<HTMLButtonElement>("[data-action='retry']")?.addEventListener("click", () => {
    render({ ...currentStatus, state: "starting", message: "正在重新启动 DeepSeek Harness…", detail: undefined });
    void backend().Retry();
  });
  root.querySelector<HTMLButtonElement>("[data-action='logs']")?.addEventListener("click", () => {
    void backend().OpenLogs();
  });
  root.querySelector<HTMLButtonElement>("[data-action='settings']")?.addEventListener("click", () => {
    void openSettings();
  });
  root.querySelector<HTMLButtonElement>("[data-action='help']")?.addEventListener("click", openHelp);
};

const closeDialog = (): void => {
  root.querySelector(".dialog-backdrop")?.remove();
};

const showDialog = (content: string): HTMLElement => {
  closeDialog();
  root.insertAdjacentHTML("beforeend", `<div class="dialog-backdrop"><section class="dialog" role="dialog" aria-modal="true">${content}</section></div>`);
  const backdrop = root.querySelector<HTMLElement>(".dialog-backdrop");
  if (!backdrop) {
    throw new Error("无法创建对话框");
  }
  backdrop.querySelectorAll<HTMLElement>("[data-dialog-close]").forEach((button) => {
    button.addEventListener("click", closeDialog);
  });
  backdrop.addEventListener("click", (event) => {
    if (event.target === backdrop) closeDialog();
  });
  return backdrop;
};

const openSettings = async (): Promise<void> => {
  let settings: Settings;
  try {
    settings = await backend().GetSettings();
  } catch (error: unknown) {
    showErrorDialog("无法读取代理设置", error);
    return;
  }
  const backdrop = showDialog(`
    <header class="dialog-header">
      <div><p class="eyebrow">STARTUP SETTINGS</p><h2>代理与启动设置</h2></div>
      <button class="icon-button" data-dialog-close aria-label="关闭">×</button>
    </header>
    <p class="dialog-intro">代理只传给 DSH/npm 子进程；外壳访问本地 DSH 时始终绕过代理。保存后会立即重启 DSH。</p>
    <form class="settings-form">
      <label class="mode-option">
        <input type="radio" name="proxy-mode" value="inherit" ${settings.proxyMode === "inherit" ? "checked" : ""}>
        <span><strong>继承系统环境</strong><small>使用启动应用时已有的 HTTP_PROXY / HTTPS_PROXY。</small></span>
      </label>
      <label class="mode-option">
        <input type="radio" name="proxy-mode" value="custom" ${settings.proxyMode === "custom" ? "checked" : ""}>
        <span><strong>自定义代理</strong><small>适合本机 VPN/代理软件只开放监听端口的情况。</small></span>
      </label>
      <div class="proxy-field">
        <label for="proxy-url">代理地址</label>
        <input id="proxy-url" name="proxy-url" type="text" value="${escapeHTML(settings.proxyUrl ?? "")}" placeholder="http://127.0.0.1:10808" spellcheck="false">
        <small>支持 HTTP/HTTPS；也可直接填写 127.0.0.1:10808。</small>
      </div>
      <label class="mode-option">
        <input type="radio" name="proxy-mode" value="disabled" ${settings.proxyMode === "disabled" ? "checked" : ""}>
        <span><strong>禁用代理</strong><small>移除继承的代理变量，所有外部请求直接连接。</small></span>
      </label>
      <p class="form-error" role="alert"></p>
      <div class="dialog-actions">
        <button type="button" class="button secondary" data-dialog-close>取消</button>
        <button type="submit" class="button primary">保存并重启 DSH</button>
      </div>
    </form>
  `);
  const form = backdrop.querySelector<HTMLFormElement>(".settings-form");
  const proxyField = backdrop.querySelector<HTMLElement>(".proxy-field");
  const proxyInput = backdrop.querySelector<HTMLInputElement>("#proxy-url");
  const errorOutput = backdrop.querySelector<HTMLElement>(".form-error");
  const updateProxyField = (): void => {
    const mode = form?.querySelector<HTMLInputElement>("input[name='proxy-mode']:checked")?.value;
    proxyField?.classList.toggle("is-disabled", mode !== "custom");
    if (proxyInput) proxyInput.disabled = mode !== "custom";
  };
  form?.querySelectorAll<HTMLInputElement>("input[name='proxy-mode']").forEach((input) => {
    input.addEventListener("change", updateProxyField);
  });
  updateProxyField();
  form?.addEventListener("submit", (event) => {
    event.preventDefault();
    const mode = form.querySelector<HTMLInputElement>("input[name='proxy-mode']:checked")?.value as Settings["proxyMode"];
    const submit = form.querySelector<HTMLButtonElement>("button[type='submit']");
    if (submit) submit.disabled = true;
    if (errorOutput) errorOutput.textContent = "";
    void backend()
      .SaveSettings({ proxyMode: mode, proxyUrl: proxyInput?.value ?? "" })
      .then((status) => {
        closeDialog();
        render(status);
      })
      .catch((error: unknown) => {
        if (errorOutput) errorOutput.textContent = error instanceof Error ? error.message : String(error);
        if (submit) submit.disabled = false;
      });
  });
};

const openHelp = (): void => {
  showDialog(`
    <header class="dialog-header">
      <div><p class="eyebrow">QUICK HELP</p><h2>Starline DSH Desktop 使用帮助</h2></div>
      <button class="icon-button" data-dialog-close aria-label="关闭">×</button>
    </header>
    <div class="help-content">
      <section><h3>首次启动</h3><p>需要 Node.js 22.19+ 或 24+。应用会通过 npx 下载固定版本的 DSH，首次运行可能需要数分钟；第一次选择工作区是 DSH 自身的正常初始化。</p></section>
      <section><h3>代理怎么填</h3><p>如果代理软件监听本机端口，选择“自定义代理”，填写 <code>http://127.0.0.1:端口</code>。例如端口 10808 就填 <code>http://127.0.0.1:10808</code>。</p></section>
      <section><h3>出错排查</h3><p>先打开日志检查 Node、npm 下载或网络错误；改完代理后保存，外壳会自动重启 DSH。模型服务的密钥仍在官方 DSH 页面中配置。</p></section>
      <section><h3>安装版与便携版</h3><p>Setup.exe 安装到当前用户目录并创建卸载项；ZIP 解压即用。两种版本都不会内嵌或改写 DSH 内核。</p></section>
    </div>
    <div class="dialog-actions">
      <button class="button secondary" data-action="help-logs">${icons.logs}<span>打开日志目录</span></button>
      <button class="button primary" data-dialog-close>知道了</button>
    </div>
  `).querySelector<HTMLButtonElement>("[data-action='help-logs']")?.addEventListener("click", () => {
    void backend().OpenLogs();
  });
};

const showErrorDialog = (title: string, error: unknown): void => {
  showDialog(`
    <header class="dialog-header"><h2>${escapeHTML(title)}</h2><button class="icon-button" data-dialog-close aria-label="关闭">×</button></header>
    <pre class="detail">${escapeHTML(error instanceof Error ? error.message : String(error))}</pre>
    <div class="dialog-actions"><button class="button primary" data-dialog-close>关闭</button></div>
  `);
};

render(initialStatus);
for (const eventName of ["dsh:ready", "dsh:failed", "dsh:stopped"] as const) {
  onStatusEvent(eventName, render);
}
onCommandEvent("shell:open-settings", () => void openSettings());
onCommandEvent("shell:open-help", openHelp);
void backend()
  .GetStatus()
  .then(render)
  .catch((error: unknown) => {
    render({
      ...initialStatus,
      state: "failed",
      message: "无法连接桌面宿主",
      detail: error instanceof Error ? error.message : String(error),
    });
  });
