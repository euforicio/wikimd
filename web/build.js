import { watch, existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const isWatch = process.argv.includes("--watch");
const isProduction = process.argv.includes("--production");

const buildConfig = {
  entrypoints: [
    resolve(__dirname, "src/entries/app.js"),
    resolve(__dirname, "src/entries/static-site.js"),
  ],
  outdir: resolve(__dirname, "../static/js"),
  target: "browser",
  format: "esm",
  sourcemap: "linked",
  minify: isProduction,
  splitting: true,
  naming: {
    entry: "[name].js",
    chunk: "chunks/[name]-[hash].js",
    asset: "chunks/[name]-[hash].[ext]",
  },
};

async function downloadVendors() {
  const vendorConfigPath = resolve(__dirname, "../vendor.json");

  if (!existsSync(vendorConfigPath)) {
    console.log("⚠️  vendor.json not found, skipping vendor downloads");
    return;
  }

  console.log("📦 Checking vendor files...");

  const vendorConfig = JSON.parse(readFileSync(vendorConfigPath, "utf-8"));
  const vendorDir = resolve(__dirname, "../static/vendor");

  // Ensure vendor directory exists
  mkdirSync(vendorDir, { recursive: true });

  let downloadCount = 0;
  let skipCount = 0;

  for (const [name, config] of Object.entries(vendorConfig)) {
    const outputPath = resolve(__dirname, "..", config.output);

    // Skip if file already exists and is not empty
    if (existsSync(outputPath)) {
      const stats = await Bun.file(outputPath).size;
      if (stats > 0) {
        skipCount++;
        continue;
      }
    }

    console.log(`⬇  Downloading ${name} (v${config.version})...`);

    try {
      const response = await fetch(config.url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const content = await response.text();
      writeFileSync(outputPath, content);
      downloadCount++;
      console.log(`✓ ${name} downloaded successfully`);
    } catch (error) {
      console.error(`✗ Failed to download ${name}: ${error.message}`);
      process.exit(1);
    }
  }

  if (downloadCount > 0) {
    console.log(`✅ Downloaded ${downloadCount} vendor file(s)`);
  }
  if (skipCount > 0) {
    console.log(`✓ Skipped ${skipCount} existing vendor file(s)`);
  }
}

async function build() {
  console.log(`Building${isWatch ? " (watch mode)" : ""}...`);
  try {
    const result = await Bun.build(buildConfig);

    if (!result.success) {
      console.error("Build failed:");
      for (const message of result.logs) {
        console.error(message);
      }
      process.exit(1);
    }

    console.log(`✓ Built ${result.outputs.length} files`);
    for (const output of result.outputs) {
      console.log(`  ${output.path}`);
    }
  } catch (error) {
    console.error("Build error:", error);
    process.exit(1);
  }
}

if (isWatch) {
  // Download vendors once at start
  await downloadVendors();

  // Initial build
  await build();

  // Watch for changes
  const srcDir = resolve(__dirname, "src");
  console.log(`Watching ${srcDir} for changes...`);

  watch(srcDir, { recursive: true }, async (event, filename) => {
    if (filename && /\.(js|ts|jsx|tsx)$/.test(filename)) {
      console.log(`\nFile changed: ${filename}`);
      await build();
    }
  });
} else {
  await downloadVendors();
  await build();
}
