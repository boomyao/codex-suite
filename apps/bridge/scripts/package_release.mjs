#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";

function fail(message) {
  console.error(message);
  process.exit(1);
}

function parseArgs(argv) {
  const args = {};
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith("--")) {
      fail(`unexpected argument: ${token}`);
    }
    const key = token.slice(2);
    const next = argv[index + 1];
    if (!next || next.startsWith("--")) {
      fail(`missing value for --${key}`);
    }
    args[key] = next;
    index += 1;
  }
  return args;
}

function requireString(value, label) {
  if (typeof value !== "string" || value.trim() === "") {
    fail(`${label} must be a non-empty string`);
  }
  return value.trim();
}

function optionalString(value) {
  return typeof value === "string" && value.trim() !== "" ? value.trim() : "";
}

function ensureFile(filePath, label) {
  const resolved = path.resolve(filePath);
  let stat;
  try {
    stat = fs.statSync(resolved);
  } catch {
    fail(`${label} not found: ${resolved}`);
  }
  if (!stat.isFile()) {
    fail(`${label} is not a file: ${resolved}`);
  }
  return resolved;
}

function mkdirp(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function cpFile(source, destination, mode) {
  mkdirp(path.dirname(destination));
  fs.copyFileSync(source, destination);
  if (mode !== undefined) {
    fs.chmodSync(destination, mode);
  }
}

function goEnvForPlatform(platform) {
  const [goos, arch] = requireString(platform, "--platform").split("-");
  if (!goos || !arch) {
    fail(`invalid --platform: ${platform}`);
  }
  const goarch = arch === "x64" ? "amd64" : arch;
  return { GOOS: goos, GOARCH: goarch };
}

function releaseBaseName(version, platform) {
  const normalizedVersion = version.replace(/^v/, "");
  return `codex-bridge_${normalizedVersion}_${platform}`;
}

function targetBinaryName(platform) {
  return platform.includes("win32") || platform.includes("windows") ? "codex-bridge.exe" : "codex-bridge";
}

function runCommand(command, args, options = {}) {
  const result = spawnSync(command, args, {
    stdio: "inherit",
    ...options
  });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

const args = parseArgs(process.argv.slice(2));
const version = requireString(args.version, "--version");
const platform = requireString(args.platform, "--platform");
const outputDir = path.resolve(requireString(args["output-dir"], "--output-dir"));
const runtimePointerManifest = ensureFile(requireString(args["pointer-manifest"], "--pointer-manifest"), "--pointer-manifest");
const runtimeManifest = optionalString(args["runtime-manifest"]);
const runtimeArchive = optionalString(args["runtime-archive"]);
const binaryPath = optionalString(args["binary-path"]);
const repoRoot = path.resolve(optionalString(args["repo-root"]) || process.cwd());

if ((runtimeManifest && !runtimeArchive) || (!runtimeManifest && runtimeArchive)) {
  fail("--runtime-manifest and --runtime-archive must be provided together");
}

mkdirp(outputDir);

const baseName = releaseBaseName(version, platform);
const workRoot = path.join(outputDir, `${baseName}.work`);
const packageRoot = path.join(workRoot, baseName);
const binDir = path.join(packageRoot, "bin");
const runtimeDir = path.join(packageRoot, "runtime");

fs.rmSync(workRoot, { recursive: true, force: true });
mkdirp(binDir);
mkdirp(runtimeDir);

const targetBinaryPath = path.join(binDir, targetBinaryName(platform));
if (binaryPath) {
  cpFile(ensureFile(binaryPath, "--binary-path"), targetBinaryPath, 0o755);
} else {
  const env = {
    ...process.env,
    ...goEnvForPlatform(platform),
    CGO_ENABLED: "0"
  };
  runCommand("go", ["build", "-o", targetBinaryPath, "./cmd/codex-bridge"], {
    cwd: path.join(repoRoot, "apps/bridge"),
    env
  });
  fs.chmodSync(targetBinaryPath, 0o755);
}

cpFile(runtimePointerManifest, path.join(runtimeDir, path.basename(runtimePointerManifest)));
if (runtimeManifest && runtimeArchive) {
  cpFile(ensureFile(runtimeManifest, "--runtime-manifest"), path.join(runtimeDir, path.basename(runtimeManifest)));
  cpFile(ensureFile(runtimeArchive, "--runtime-archive"), path.join(runtimeDir, path.basename(runtimeArchive)));
}

const tarballPath = path.join(outputDir, `${baseName}.tar.gz`);
fs.rmSync(tarballPath, { force: true });
runCommand("tar", ["-czf", tarballPath, "-C", workRoot, baseName]);

console.log(JSON.stringify({
  packageRoot,
  tarball: tarballPath,
  bridgeBinary: targetBinaryPath,
  runtimeDir
}, null, 2));
