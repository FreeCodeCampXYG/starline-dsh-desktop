import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, extname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const markdownFiles = [];

const collectMarkdown = (path) => {
  for (const entry of readdirSync(path, { withFileTypes: true })) {
    const target = resolve(path, entry.name);
    if (entry.isDirectory()) {
      collectMarkdown(target);
    } else if (extname(entry.name).toLowerCase() === ".md") {
      markdownFiles.push(target);
    }
  }
};

collectMarkdown(resolve(repositoryRoot, "docs"));
for (const name of ["README.md", "CONTRIBUTING.md", "SECURITY.md", "AUTHORS.md", "NOTICE.md", "CHANGELOG.md"]) {
  markdownFiles.push(resolve(repositoryRoot, name));
}

const failures = [];
const linkPattern = /!?\[[^\]]*\]\(([^)]+)\)/g;
for (const file of markdownFiles) {
  const content = readFileSync(file, "utf8");
  for (const match of content.matchAll(linkPattern)) {
    let target = match[1].trim();
    if (target.startsWith("<") && target.endsWith(">")) {
      target = target.slice(1, -1);
    }
    if (!target || target.startsWith("#") || /^[a-z][a-z0-9+.-]*:/i.test(target)) {
      continue;
    }
    target = target.split("#", 1)[0];
    const absolute = resolve(dirname(file), decodeURIComponent(target));
    if (!existsSync(absolute) || (!statSync(absolute).isFile() && !statSync(absolute).isDirectory())) {
      failures.push(`${file.slice(repositoryRoot.length + 1)} -> ${match[1]}`);
    }
  }
}

if (failures.length > 0) {
  console.error("发现失效的本地文档链接：");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log(`文档链接检查通过：${markdownFiles.length} 个 Markdown 文件。`);
