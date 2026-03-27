#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

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
      args[key] = "true";
      continue;
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

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(path.resolve(filePath), "utf8"));
}

async function fetchJSON(url) {
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
  const headers = {
    "Accept": "application/vnd.github+json",
    "User-Agent": "codex-bridge"
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  const response = await fetch(url, {
    headers
  });
  if (!response.ok) {
    const body = await response.text();
    fail(`request failed: ${response.status} ${response.statusText} ${body.slice(0, 512)}`);
  }
  return response.json();
}

function releaseVersion(tag) {
  return tag.replace(/^rust-v/, "").replace(/^v/, "");
}

function updateLock(lock, latestRelease) {
  const version = releaseVersion(latestRelease.tag_name);
  lock.runtimeVersion = `${version}-rt1`;
  for (const platform of Object.keys(lock.platforms ?? {})) {
    const entry = lock.platforms[platform];
    entry.upstreamRelease.tag = latestRelease.tag_name;
    entry.desktopWebview.version = version;
    entry.appServer.version = version;
  }
  return lock;
}

function summarize(lock, latestRelease) {
  const results = [];
  const latestVersion = releaseVersion(latestRelease.tag_name);
  const assets = new Set((latestRelease.assets ?? []).map((asset) => asset.name));

  for (const [platform, entry] of Object.entries(lock.platforms ?? {})) {
    const currentTag = entry.upstreamRelease?.tag ?? "";
    const currentVersion = entry.desktopWebview?.version ?? "";
    const requiredAssets = [entry.desktopWebview?.asset, entry.appServer?.asset].filter(Boolean);
    const missingAssets = requiredAssets.filter((asset) => !assets.has(asset));
    const outdated = currentTag !== latestRelease.tag_name || currentVersion !== latestVersion;
    results.push({
      platform,
      currentTag,
      latestTag: latestRelease.tag_name,
      currentVersion,
      latestVersion,
      requiredAssets,
      missingAssets,
      upToDate: !outdated && missingAssets.length === 0
    });
  }

  return {
    repo: lock.platforms?.[Object.keys(lock.platforms ?? {})[0]]?.upstreamRelease?.repo ?? "",
    latestTag: latestRelease.tag_name,
    latestVersion,
    allUpToDate: results.every((item) => item.upToDate),
    platforms: results
  };
}

const args = parseArgs(process.argv.slice(2));
const lockPath = requireString(args.lock, "--lock");
const writeLock = args["write-lock"] === "true";

const lock = readJSON(lockPath);
const firstPlatform = Object.keys(lock.platforms ?? {})[0];
if (!firstPlatform) {
  fail("runtime lock does not define any platforms");
}

const repo = requireString(lock.platforms[firstPlatform].upstreamRelease?.repo, "upstream release repo");
const latestRelease = await fetchJSON(`https://api.github.com/repos/${repo}/releases/latest`);
const summary = summarize(lock, latestRelease);

if (writeLock && !summary.allUpToDate) {
  const nextLock = updateLock(structuredClone(lock), latestRelease);
  fs.writeFileSync(path.resolve(lockPath), `${JSON.stringify(nextLock, null, 2)}\n`);
}

console.log(JSON.stringify(summary, null, 2));
if (!summary.allUpToDate) {
  process.exitCode = 2;
}
