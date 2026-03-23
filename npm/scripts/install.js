#!/usr/bin/env node

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const os = require("os");

const REPO = "nishchaysinha/grotto";
const BIN_DIR = path.join(__dirname, "..", "bin");

function getPlatform() {
  const platform = os.platform();
  switch (platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }
}

function getArch() {
  const arch = os.arch();
  switch (arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }
}

function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: "api.github.com",
      path: `/repos/${REPO}/releases/latest`,
      headers: {
        "User-Agent": "grotto-npm-installer",
      },
    };

    https
      .get(options, (res) => {
        let data = "";
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () => {
          try {
            const json = JSON.parse(data);
            resolve(json.tag_name);
          } catch (e) {
            reject(new Error("Failed to parse release info"));
          }
        });
      })
      .on("error", reject);
  });
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);

    const request = (url) => {
      https
        .get(url, (res) => {
          if (res.statusCode === 302 || res.statusCode === 301) {
            request(res.headers.location);
            return;
          }
          res.pipe(file);
          file.on("finish", () => {
            file.close(resolve);
          });
        })
        .on("error", (err) => {
          fs.unlink(dest, () => {});
          reject(err);
        });
    };

    request(url);
  });
}

async function main() {
  try {
    const platform = getPlatform();
    const arch = getArch();
    const version = await getLatestVersion();
    const versionNum = version.replace(/^v/, "");

    console.log(`Installing grotto ${version} for ${platform}-${arch}...`);

    const ext = platform === "windows" ? "zip" : "tar.gz";
    const archiveName = `grotto-${versionNum}-${platform}-${arch}.${ext}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${archiveName}`;

    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "grotto-"));
    const archivePath = path.join(tmpDir, archiveName);

    console.log(`Downloading from ${downloadUrl}...`);
    await downloadFile(downloadUrl, archivePath);

    console.log("Extracting...");
    if (!fs.existsSync(BIN_DIR)) {
      fs.mkdirSync(BIN_DIR, { recursive: true });
    }

    if (platform === "windows") {
      execSync(`tar -xf "${archivePath}" -C "${tmpDir}"`, { stdio: "inherit" });
    } else {
      execSync(`tar -xzf "${archivePath}" -C "${tmpDir}"`, { stdio: "inherit" });
    }

    const binaryName = platform === "windows" ? "grotto.exe" : "grotto";
    const srcBinary = path.join(tmpDir, binaryName);
    const destBinary = path.join(BIN_DIR, binaryName);

    fs.copyFileSync(srcBinary, destBinary);
    fs.chmodSync(destBinary, 0o755);

    // Clean up
    fs.rmSync(tmpDir, { recursive: true, force: true });

    console.log(`Successfully installed grotto ${version}`);
  } catch (err) {
    console.error("Installation failed:", err.message);
    process.exit(1);
  }
}

main();
