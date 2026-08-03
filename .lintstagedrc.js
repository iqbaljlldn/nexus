const path = require('path');

module.exports = {
  'apps/web/**/*': (files) => {
    const cwd = path.resolve('apps/web');
    const relativeFiles = files.map(f => path.relative(cwd, f));
    return `bash -c "cd apps/web && pnpm exec biome check --write --no-errors-on-unmatched --files-ignore-unknown=true ${relativeFiles.join(' ')}"`;
  },
  '*.go': (files) => [
    `gofmt -w ${files.join(' ')}`,
    'bash -c "cd apps/api && golangci-lint run"',
    'bash -c "cd pkg && golangci-lint run"'
  ]
};
