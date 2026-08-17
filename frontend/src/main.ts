import "./style.css";
import {
  backend,
  onCommandEvent,
  onStatusEvent,
  showWindow,
  type DSHUpdateInfo,
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
  update: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12M7 10l5 5 5-5"/><path d="M5 20h14"/></svg>',
} as const;

const initialStatus: Status = {
  state: "starting",
  message: "正在连接桌面宿主…",
  version: "dev",
  dshVersion: "0.1.0-rc.7",
  runtimeMode: "auto",
};

let currentStatus = initialStatus;
let renderSequence = 0;
let dshUpdateInfo: DSHUpdateInfo | null = null;
let dshUpdateError = "";
let dshUpdateRequest: Promise<DSHUpdateInfo> | null = null;

const runtimeLabel = (status: Status): string => {
  if (status.runtimeMode === "offline") return "包内离线运行时";
  if (status.runtimeMode === "online") return "系统 Node / npx";
  return "正在检测运行时";
};

const syncUpdateIndicator = (): void => {
  const summary = dshUpdateInfo
    ? `latest ${dshUpdateInfo.latestVersion}${dshUpdateInfo.nextVersion ? ` · next ${dshUpdateInfo.nextVersion}` : " · next 未发布"}`
    : dshUpdateError
      ? "版本检查失败"
      : "正在检查版本…";
  root.querySelectorAll<HTMLElement>(".runtime-update-indicator").forEach((indicator) => {
    indicator.hidden = !dshUpdateInfo && !dshUpdateError;
    indicator.textContent = summary;
    indicator.classList.toggle("has-update", Boolean(dshUpdateInfo?.latestUpdateAvailable || dshUpdateInfo?.nextUpdateAvailable));
    indicator.classList.toggle("has-error", Boolean(dshUpdateError));
  });
  root.querySelectorAll<HTMLElement>(".dsh-update-menu-label").forEach((label) => {
    label.textContent = dshUpdateInfo ? `DSH 更新 · ${summary}` : dshUpdateError ? "DSH 更新 · 检查失败" : "检查 DSH 更新";
  });
};

const checkDSHUpdates = (force = false): Promise<DSHUpdateInfo> => {
  if (!force && dshUpdateInfo) return Promise.resolve(dshUpdateInfo);
  if (dshUpdateRequest) return dshUpdateRequest;
  dshUpdateError = "";
  syncUpdateIndicator();
  const request = backend().CheckDSHUpdate()
    .then((info) => {
      dshUpdateInfo = info;
      syncUpdateIndicator();
      return info;
    })
    .catch((error: unknown) => {
      dshUpdateError = error instanceof Error ? error.message : String(error);
      syncUpdateIndicator();
      throw error;
    })
    .finally(() => {
      dshUpdateRequest = null;
    });
  dshUpdateRequest = request;
  return request;
};

const renderDesktopMenu = (includeBrowser: boolean): string => `
  <details class="shell-menu">
    <summary>${icons.shell}<span>桌面工具</span></summary>
    <div class="shell-menu-popover">
      <button data-action="settings">${icons.settings}<span>代理与启动设置</span></button>
      <button data-action="dsh-update">${icons.update}<span class="dsh-update-menu-label">检查 DSH 更新</span></button>
      <button data-action="restart">${icons.restart}<span>重新启动 DSH</span></button>
      <button data-action="logs">${icons.logs}<span>打开日志目录</span></button>
      ${includeBrowser ? `<button data-action="browser">${icons.browser}<span>在浏览器中打开</span></button>` : ""}
      <button data-action="help">${icons.help}<span>使用帮助</span></button>
    </div>
  </details>
`;

const renderRuntimeBar = (status: Status, includeBrowser: boolean): string => `
  <div class="runtime-bar">
    <div class="runtime-state state-${escapeHTML(status.state)}">
      <span></span>
      DSH ${escapeHTML(status.dshVersion)} · Desktop ${escapeHTML(status.version)} · ${runtimeLabel(status)}
    </div>
    <div class="runtime-tools">
      <button type="button" class="runtime-update-indicator" data-action="dsh-update" hidden></button>
      ${renderDesktopMenu(includeBrowser)}
    </div>
  </div>
`;

const renderSplash = (status: Status): string => {
  const isBusy = status.state === "starting" || status.state === "idle";
  const runtimeText = runtimeLabel(status);
  const eyebrow = isBusy ? "LOCAL RUNTIME" : "STARTUP DIAGNOSTIC";
  const description = isBusy
    ? status.runtimeMode === "offline"
      ? "桌面宿主正在使用包内固定版本的 Node 与 DSH，不会访问 npm registry。"
      : "桌面宿主正在检查包内离线运行时；普通包会通过系统 Node 与 npx 准备官方 DSH。"
    : "桌面宿主没有修改 DSH 内核。你可以调整代理、查看原始日志，修复问题后重试。";
  const progress = Math.min(100, Math.max(0, Math.round(status.progress ?? 0)));
  const progressStage = status.stage ?? "等待桌面宿主报告启动阶段…";

  return `
    <section class="splash">
      <div class="panel">
        <div class="mark" aria-hidden="true"><span>D</span></div>
        <p class="eyebrow">${eyebrow}</p>
        <h1>${escapeHTML(status.message)}</h1>
        <p class="description">${description}</p>
        ${
          status.detail
            ? `<pre class="detail">${escapeHTML(status.detail)}</pre>`
            : `<div class="progress" role="progressbar" aria-label="DSH 启动阶段进度" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${progress}"><span style="width: ${progress}%"></span></div><div class="progress-meta"><strong>${progress}%</strong><span>${escapeHTML(progressStage)}</span></div>`
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
          <span class="dot"></span>
          <span>${runtimeText}</span>
        </footer>
      </div>
    </section>
  `;
};

const render = (status: Status): void => {
  currentStatus = status;
  const sequence = ++renderSequence;
  if (status.state === "ready" && status.url) {
    renderReady(status, sequence);
    return;
  }

  const isBusy = status.state === "starting" || status.state === "idle";
  if (isBusy) {
    root.innerHTML = renderSplash(status);
    bindCommonActions();
    return;
  }

  const description = isBusy
    ? status.runtimeMode === "offline"
      ? "桌面宿主正在使用包内固定版本的 Node 与 DSH，不会访问 npm registry。"
      : "桌面宿主正在检查包内离线运行时；普通包会通过系统 Node 与 npx 准备官方 DSH。"
    : "桌面宿主没有修改 DSH 内核。你可以调整代理、查看原始日志，修复问题后重试。";
  const bannerTone = isBusy ? "is-busy" : "is-error";
  const bannerRole = isBusy ? "status" : "alert";

  root.innerHTML = `
    <section class="workspace workspace-status">
      ${renderRuntimeBar(status, false)}
      <div class="status-banner ${bannerTone}" role="${bannerRole}">
        <div class="status-summary">
          <span class="status-indicator" aria-hidden="true"></span>
          <div>
            <strong>${escapeHTML(status.message)}</strong>
            <small>${escapeHTML(status.detail ?? description)}</small>
          </div>
        </div>
        <div class="status-actions">
          ${status.detail ? `<button class="status-action" data-action="error-details">查看详情</button>` : ""}
          ${isBusy ? "" : `<button class="status-action primary" data-action="retry">${icons.restart}<span>重试</span></button>`}
          <button class="status-action" data-action="settings">${icons.settings}<span>代理</span></button>
          <button class="status-action" data-action="logs">${icons.logs}<span>日志</span></button>
        </div>
      </div>
      <div class="startup-stage" aria-busy="${isBusy}">
        <div class="startup-mark" aria-hidden="true"><span>ST</span><i></i></div>
        <p>${isBusy ? "正在后台准备官方 DSH 页面" : "DSH 页面暂时不可用"}</p>
        <small>${isBusy ? "检查完成后会自动载入，无需操作。" : "请根据上方提示修复后重试。"}</small>
      </div>
    </section>
  `;
  bindCommonActions();
  if (!isBusy) showWindow();
};

const renderReady = (status: Status, sequence: number): void => {
  const source = status.url ?? "";
  const startupStatus: Status = {
    ...status,
    state: "starting",
    message: "正在载入 DeepSeek Harness…",
    detail: undefined,
    progress: 99,
    stage: "DSH 已就绪，正在载入桌面 WebView…",
  };
  root.innerHTML = `
    <section class="workspace workspace-ready">
      ${renderRuntimeBar(status, true)}
      <div class="frame-stage">
        <iframe
          class="dsh-frame"
          title="DeepSeek Harness"
          allow="clipboard-read; clipboard-write"
        ></iframe>
        <div class="startup-overlay">${renderSplash(startupStatus)}</div>
      </div>
    </section>
  `;
  bindCommonActions();
  const frame = root.querySelector<HTMLIFrameElement>(".dsh-frame");
  const overlay = root.querySelector<HTMLElement>(".startup-overlay");
  if (!frame) {
    showWindow();
    return;
  }
  frame.addEventListener("load", () => {
    if (sequence !== renderSequence || !frame.isConnected) return;
    showWindow();
    requestAnimationFrame(() => {
      frame.classList.add("is-loaded");
      overlay?.classList.add("is-hidden");
    });
    window.setTimeout(() => overlay?.remove(), 360);
  }, { once: true });
  frame.src = source;
};

const restart = (): void => {
  render({
    ...currentStatus,
    state: "starting",
    message: "正在重新启动 DeepSeek Harness…",
    detail: undefined,
  });
  void backend().Retry();
};

const bindCommonActions = (): void => {
  root.querySelectorAll<HTMLButtonElement>(".shell-menu-popover button").forEach((button) => {
    button.addEventListener("click", () => {
      const menu = button.closest<HTMLDetailsElement>(".shell-menu");
      if (menu) menu.open = false;
    });
  });
  root.querySelector<HTMLButtonElement>("[data-action='retry']")?.addEventListener("click", () => {
    restart();
  });
  root.querySelector<HTMLButtonElement>("[data-action='restart']")?.addEventListener("click", restart);
  root.querySelector<HTMLButtonElement>("[data-action='browser']")?.addEventListener("click", () => {
    void backend().OpenInBrowser();
  });
  root.querySelector<HTMLButtonElement>("[data-action='error-details']")?.addEventListener("click", () => {
    showErrorDialog(currentStatus.message, currentStatus.detail ?? "没有更多错误信息。");
  });
  root.querySelector<HTMLButtonElement>("[data-action='logs']")?.addEventListener("click", () => {
    void backend().OpenLogs();
  });
  root.querySelector<HTMLButtonElement>("[data-action='settings']")?.addEventListener("click", () => {
    void openSettings();
  });
  root.querySelectorAll<HTMLButtonElement>("[data-action='dsh-update']").forEach((button) => button.addEventListener("click", () => {
    void openSettings(true);
  }));
  root.querySelector<HTMLButtonElement>("[data-action='help']")?.addEventListener("click", openHelp);
  syncUpdateIndicator();
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

const openSettings = async (checkUpdate = false): Promise<void> => {
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
    <p class="dialog-intro">代理只用于 DSH/npm 子进程和官方版本检查；外壳访问本地 DSH 时始终绕过代理。保存代理后会立即重启 DSH。</p>
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
      <section class="runtime-update-card">
        <div>
          <strong>官方 DSH 更新通道</strong>
          <small>启动后会通过当前代理设置自动检查 latest 与 next，但不会静默切换运行版本。当前 ${escapeHTML(currentStatus.dshVersion)}；Desktop ${escapeHTML(currentStatus.version)} 默认兼容版本保持不变。</small>
        </div>
        <button type="button" class="button secondary" data-action="dsh-update-check">${icons.update}<span>刷新 npm latest / next</span></button>
        <div class="update-result" role="status" aria-live="polite"></div>
      </section>
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
  const updateOutput = backdrop.querySelector<HTMLElement>(".update-result");
  const checkUpdateButton = backdrop.querySelector<HTMLButtonElement>("[data-action='dsh-update-check']");
  const setUpdateMessage = (message: string): void => {
    const output = updateOutput?.querySelector<HTMLParagraphElement>("p");
    if (output) output.textContent = message;
  };
  const updateProxyField = (): void => {
    const mode = form?.querySelector<HTMLInputElement>("input[name='proxy-mode']:checked")?.value;
    proxyField?.classList.toggle("is-disabled", mode !== "custom");
    if (proxyInput) proxyInput.disabled = mode !== "custom";
  };
  form?.querySelectorAll<HTMLInputElement>("input[name='proxy-mode']").forEach((input) => {
    input.addEventListener("change", updateProxyField);
  });
  updateProxyField();
  const renderUpdateInfo = (info: DSHUpdateInfo): void => {
    if (!updateOutput) return;
    const badges = [
      `<span>当前 ${escapeHTML(info.currentVersion)}</span>`,
      `<span>latest ${escapeHTML(info.latestVersion)}</span>`,
      `<span>next ${escapeHTML(info.nextVersion ?? "未发布")}</span>`,
    ].join("");
    const latestButton = info.latestUpdateAvailable && info.canApply
      ? `<button type="button" class="button primary" data-action="dsh-update-apply" data-channel="latest" data-version="${escapeHTML(info.latestVersion)}">应用 latest ${escapeHTML(info.latestVersion)}</button>`
      : "";
    const nextButton = info.nextUpdateAvailable && info.canApply && info.nextVersion && info.nextVersion !== info.latestVersion
      ? `<button type="button" class="button secondary" data-action="dsh-update-apply" data-channel="next" data-version="${escapeHTML(info.nextVersion)}">试用 next ${escapeHTML(info.nextVersion)}</button>`
      : "";
    const resetButton = info.canReset
      ? `<button type="button" class="button secondary" data-action="dsh-update-reset">恢复默认 ${escapeHTML(info.defaultVersion)}</button>`
      : "";
    updateOutput.innerHTML = `<div class="update-badges">${badges}</div><p>${escapeHTML(info.message)}</p><div class="update-actions">${latestButton}${nextButton}${resetButton}</div>`;
    updateOutput.querySelectorAll<HTMLButtonElement>("[data-action='dsh-update-apply']").forEach((applyButton) => applyButton.addEventListener("click", () => {
      const channel = applyButton.dataset.channel as "latest" | "next";
      const targetVersion = applyButton.dataset.version ?? "";
      const warning = channel === "next"
        ? `next 是预览通道，版本 ${targetVersion} 尚未随当前 Desktop Release 完成六平台验证。仍要从 ${info.currentVersion} 切换并重启吗？`
        : `将 DSH 从 ${info.currentVersion} 更新到 npm latest ${targetVersion} 并重启。是否继续？`;
      if (!window.confirm(warning)) return;
      updateOutput?.querySelectorAll<HTMLButtonElement>("button").forEach((actionButton) => {
        actionButton.disabled = true;
      });
      updateOutput?.insertAdjacentHTML("beforeend", `<div class="update-operation-progress"><div class="progress" role="progressbar" aria-label="更新准备进度" aria-valuemin="0" aria-valuemax="100" aria-valuenow="10"><span style="width: 10%"></span></div><div class="progress-meta"><strong>10%</strong><span>正在重新核对 npm 官方 ${channel} 通道…</span></div></div>`);
      void backend().ApplyDSHUpdate(channel)
        .then((status) => {
          dshUpdateInfo = null;
          closeDialog();
          render(status);
          void checkDSHUpdates(true).catch(() => undefined);
        })
        .catch((error: unknown) => {
          renderUpdateInfo(info);
          setUpdateMessage(error instanceof Error ? error.message : String(error));
        });
    }));
    updateOutput.querySelector<HTMLButtonElement>("[data-action='dsh-update-reset']")?.addEventListener("click", (event) => {
      if (!window.confirm(`恢复 Desktop 内置兼容版本 ${info.defaultVersion} 并重启 DSH？`)) return;
      const button = event.currentTarget as HTMLButtonElement;
      button.disabled = true;
      void backend().ResetDSHVersion()
        .then((status) => {
          closeDialog();
          render(status);
        })
        .catch((error: unknown) => {
          button.disabled = false;
          setUpdateMessage(error instanceof Error ? error.message : String(error));
        });
    });
  };
  const runUpdateCheck = (): void => {
    if (checkUpdateButton) checkUpdateButton.disabled = true;
    if (updateOutput) updateOutput.textContent = "正在通过当前代理设置查询 npm 官方 latest / next…";
    void checkDSHUpdates(true)
      .then(renderUpdateInfo)
      .catch((error: unknown) => {
        if (updateOutput) updateOutput.textContent = error instanceof Error ? error.message : String(error);
      })
      .finally(() => {
        if (checkUpdateButton) checkUpdateButton.disabled = false;
      });
  };
  checkUpdateButton?.addEventListener("click", runUpdateCheck);
  if (dshUpdateInfo && !checkUpdate) renderUpdateInfo(dshUpdateInfo);
  else runUpdateCheck();
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
      <section><h3>首次启动</h3><p>普通包需要系统 Node.js 22.19+ 或 24+，并会通过 npx 下载固定版本的 DSH。<code>offline-full</code> 已包含 Node 与 DSH，不访问 npm registry；第一次选择工作区仍是 DSH 自身的正常初始化。</p></section>
      <section><h3>代理怎么填</h3><p>如果代理软件监听本机端口，选择“自定义代理”，填写 <code>http://127.0.0.1:端口</code>。例如端口 10808 就填 <code>http://127.0.0.1:10808</code>。</p></section>
      <section><h3>出错排查</h3><p>先打开日志检查 Node、npm 下载或网络错误；改完代理后保存，外壳会自动重启 DSH。模型服务的密钥仍在官方 DSH 页面中配置。</p></section>
      <section><h3>PowerShell 与权限</h3><p>权限模式由官方 DSH 页面选择。<code>danger-full-access</code> 可以按当前 Windows 用户权限执行更广泛的命令和文件操作，但会失去工作区沙箱保护；宿主不会因命令失败自动切换。该模式也不等于“以管理员身份运行”，不会自动获得 UAC 管理员令牌。</p></section>
      <section><h3>普通包与离线包</h3><p>Setup.exe 和普通 ZIP 体积较小，需要系统 Node/npm。<code>offline-full</code> 是较大的便携包，内含发布时固定并经原生门禁的 DSH 依赖，但模型服务、远程 MCP 和联网工具仍可能需要网络。</p></section>
      <section><h3>安装与覆盖升级</h3><p>Windows Setup 使用同一应用身份；关闭正在运行的应用后，新版会复用已记录的安装目录并覆盖程序文件，用户配置和 DSH 工作区不在安装目录中。<code>offline-full</code> 当前是便携压缩包，不属于安装器；请解压到新目录验证后再删除旧目录，不要覆盖正在运行的文件。</p></section>
      <section><h3>DSH 更新</h3><p>应用启动后会使用“代理与启动设置”自动查询 npm 官方 <code>latest</code> 与 <code>next</code>，顶栏只提示版本，不会静默切换。在线包可确认应用稳定或预览通道并随时恢复 Desktop 内置兼容版本；切换时只回收本应用持有的 DSH 子进程树。离线包需要下载内置新 DSH 且重新通过六平台原生门禁的 <code>offline-full</code>。</p></section>
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
for (const eventName of ["dsh:progress", "dsh:ready", "dsh:failed", "dsh:stopped"] as const) {
  onStatusEvent(eventName, render);
}
onCommandEvent("shell:open-settings", () => void openSettings());
onCommandEvent("shell:open-help", openHelp);
void backend()
  .GetStatus()
  .then((status) => {
    render(status);
    void checkDSHUpdates().catch(() => undefined);
  })
  .catch((error: unknown) => {
    render({
      ...initialStatus,
      state: "failed",
      message: "无法连接桌面宿主",
      detail: error instanceof Error ? error.message : String(error),
    });
  });
