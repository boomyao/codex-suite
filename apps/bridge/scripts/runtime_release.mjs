#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

function fail(message) {
  console.error(message);
  process.exit(1);
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  if (!command) {
    fail("usage: runtime_release.mjs <print-env|write-bridge-manifest> [--key value]");
  }

  const args = {};
  for (let index = 0; index < rest.length; index += 1) {
    const token = rest[index];
    if (!token.startsWith("--")) {
      fail(`unexpected argument: ${token}`);
    }
    const key = token.slice(2);
    const value = rest[index + 1];
    if (!value || value.startsWith("--")) {
      fail(`missing value for --${key}`);
    }
    args[key] = value;
    index += 1;
  }

  return { command, args };
}

function readJSON(filePath) {
  const resolved = path.resolve(filePath);
  return JSON.parse(fs.readFileSync(resolved, "utf8"));
}

function requireString(value, label) {
  if (typeof value !== "string" || value.trim() === "") {
    fail(`${label} must be a non-empty string`);
  }
  return value.trim();
}

function requireInteger(value, label) {
  if (!Number.isInteger(value) || value <= 0) {
    fail(`${label} must be a positive integer`);
  }
  return value;
}

function slug(value) {
  return value.trim().replace(/[^a-zA-Z0-9._-]+/g, "-");
}

function loadRuntimeLock(lockPath, platform) {
  const lock = readJSON(lockPath);
  requireInteger(lock.schemaVersion, "runtime lock schemaVersion");
  const runtimeVersion = requireString(lock.runtimeVersion, "runtime lock runtimeVersion");
  const patchSchemaVersion = requireInteger(lock.patchSchemaVersion, "runtime lock patchSchemaVersion");
  const platformEntry = lock.platforms?.[platform];
  if (!platformEntry || typeof platformEntry !== "object") {
    fail(`runtime lock does not define platform ${platform}`);
  }

  const upstreamRelease = platformEntry.upstreamRelease ?? {};
  const desktopWebview = platformEntry.desktopWebview ?? {};
  const appServer = platformEntry.appServer ?? {};

  return {
    schemaVersion: lock.schemaVersion,
    runtimeVersion,
    patchSchemaVersion,
    platform: requireString(platform, "platform"),
    upstreamRelease: {
      repo: requireString(upstreamRelease.repo, "upstreamRelease.repo"),
      tag: requireString(upstreamRelease.tag, "upstreamRelease.tag")
    },
    desktopWebview: {
      source: requireString(desktopWebview.source, "desktopWebview.source"),
      version: requireString(desktopWebview.version, "desktopWebview.version"),
      asset: requireString(desktopWebview.asset, "desktopWebview.asset")
    },
    appServer: {
      source: requireString(appServer.source, "appServer.source"),
      version: requireString(appServer.version, "appServer.version"),
      asset: requireString(appServer.asset, "appServer.asset")
    }
  };
}

function runtimeID(lock) {
  return [
    slug(lock.platform),
    `release-${slug(lock.upstreamRelease.tag)}`,
    `desktop-${slug(lock.desktopWebview.version)}`,
    `cli-${slug(lock.appServer.version)}`,
    `patch-${lock.patchSchemaVersion}`
  ].join("_");
}

function printEnv(lock) {
  const id = runtimeID(lock);
  const runtimeAsset = `runtime-${lock.platform}.tar.gz`;
  const runtimeManifestAsset = `runtime-manifest-${lock.platform}.json`;
  process.stdout.write(`PLATFORM=${lock.platform}\n`);
  process.stdout.write(`RUNTIME_VERSION=${lock.runtimeVersion}\n`);
  process.stdout.write(`PATCH_SCHEMA_VERSION=${lock.patchSchemaVersion}\n`);
  process.stdout.write(`UPSTREAM_REPO=${lock.upstreamRelease.repo}\n`);
  process.stdout.write(`UPSTREAM_TAG=${lock.upstreamRelease.tag}\n`);
  process.stdout.write(`DESKTOP_SOURCE=${lock.desktopWebview.source}\n`);
  process.stdout.write(`DESKTOP_VERSION=${lock.desktopWebview.version}\n`);
  process.stdout.write(`DESKTOP_ASSET=${lock.desktopWebview.asset}\n`);
  process.stdout.write(`CODEX_SOURCE=${lock.appServer.source}\n`);
  process.stdout.write(`CODEX_VERSION=${lock.appServer.version}\n`);
  process.stdout.write(`CODEX_ASSET=${lock.appServer.asset}\n`);
  process.stdout.write(`RUNTIME_ASSET=${runtimeAsset}\n`);
  process.stdout.write(`RUNTIME_MANIFEST_ASSET=${runtimeManifestAsset}\n`);
  process.stdout.write(`RUNTIME_ID=${id}\n`);
  process.stdout.write(`RUNTIME_TAG=runtime-${id}\n`);
}

function stripLeadingV(version) {
  return version.replace(/^v/, "");
}

function writeBridgeManifest(lock, bridgeVersion, runtimeRepo, outputPath) {
  const runtimeAsset = `runtime-${lock.platform}.tar.gz`;
  const runtimeManifestAsset = `runtime-manifest-${lock.platform}.json`;
  const manifest = {
    schemaVersion: 1,
    bridgeVersion,
    platform: lock.platform,
    runtimeRepo,
    runtimeId: runtimeID(lock),
    runtimeTag: `runtime-${runtimeID(lock)}`,
    runtimeVersion: lock.runtimeVersion,
    runtimeAsset,
    runtimeManifestAsset,
    patchSchemaVersion: lock.patchSchemaVersion,
    upstreamRelease: {
      repo: lock.upstreamRelease.repo,
      tag: lock.upstreamRelease.tag
    },
    desktopWebview: {
      source: lock.desktopWebview.source,
      version: lock.desktopWebview.version,
      asset: lock.desktopWebview.asset
    },
    appServer: {
      source: lock.appServer.source,
      version: lock.appServer.version,
      asset: lock.appServer.asset
    },
    releaseCompatibilityKey: [
      slug(lock.platform),
      `bridge-${slug(stripLeadingV(bridgeVersion))}`,
      `runtime-${slug(lock.runtimeVersion)}`
    ].join("_")
  };

  fs.mkdirSync(path.dirname(path.resolve(outputPath)), { recursive: true });
  fs.writeFileSync(path.resolve(outputPath), `${JSON.stringify(manifest, null, 2)}\n`);
}

const { command, args } = parseArgs(process.argv.slice(2));
const lockPath = requireString(args.lock, "--lock");
const platform = requireString(args.platform, "--platform");
const lock = loadRuntimeLock(lockPath, platform);

switch (command) {
  case "print-env":
    printEnv(lock);
    break;
  case "write-bridge-manifest":
    writeBridgeManifest(
      lock,
      requireString(args["bridge-version"], "--bridge-version"),
      requireString(args["runtime-repo"], "--runtime-repo"),
      requireString(args.output, "--output")
    );
    break;
  default:
    fail(`unsupported command: ${command}`);
}
