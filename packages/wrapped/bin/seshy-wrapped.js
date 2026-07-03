#!/usr/bin/env node
import('../src/cli.js').then((m) => m.main()).catch((e) => {
  process.stderr.write(`seshy-wrapped failed: ${e && e.message ? e.message : e}\n`);
  process.exit(1);
});
