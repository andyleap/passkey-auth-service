#!/usr/bin/env node

const esbuild = require('esbuild');
const fs = require('fs');
const path = require('path');

// Build configuration
const buildConfig = {
  entryPoints: {
    // CSS entry point
    'design-system': './blue-design-system/design-system.css',
    // JS entry point
    'auth': './internal/ui/assets/oauth/auth.js'
  },
  bundle: true,
  minify: true,
  sourcemap: false,
  outdir: './internal/ui/assets/dist',
  loader: {
    '.css': 'css',
  },
  target: ['chrome90', 'firefox88', 'safari14'],
};

async function build() {
  try {
    console.log('🔨 Building assets with esbuild...');
    
    // Ensure output directory exists
    const outdir = path.resolve(buildConfig.outdir);
    if (!fs.existsSync(outdir)) {
      fs.mkdirSync(outdir, { recursive: true });
    }
    
    const result = await esbuild.build({
      ...buildConfig,
      metafile: true,
    });
    
    console.log('✅ Build completed successfully!');
    
    // Show build stats
    if (result.metafile) {
      const outputs = Object.keys(result.metafile.outputs);
      console.log('\n📦 Generated files:');
      outputs.forEach(output => {
        const size = result.metafile.outputs[output].bytes;
        const sizeKb = (size / 1024).toFixed(1);
        console.log(`  ${path.basename(output)} (${sizeKb} KB)`);
      });
    }
    
  } catch (error) {
    console.error('❌ Build failed:', error);
    process.exit(1);
  }
}

// Watch mode
const isWatch = process.argv.includes('--watch');

if (isWatch) {
  console.log('👀 Watching for changes...');
  esbuild.context(buildConfig).then(ctx => {
    ctx.watch();
  });
} else {
  build();
}