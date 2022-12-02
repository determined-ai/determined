#!/usr/bin/env node
/* eslint-disable no-console, @typescript-eslint/no-var-requires */

/**
 * Whenever we upgrade the Ant Design package version,
 * run this script to update the various theme CSS files to be brought in
 * appropriately so we can support dynamic dark mode / light mode toggling.
 */

const fs = require('fs');

const rimraf = require('rimraf');

const ANTD_CSS_PATH = 'node_modules/antd/dist';
const ANTD_CSS_FILES = [
  'antd.min.css',
  'antd.min.css.map',
  'antd.dark.min.css',
  'antd.dark.min.css.map',
];
const PUBLIC_PATH = 'public/themes';
const COLOR_UPDATES = [
  {
    match: /dark/,
    updates: [
      { new: '#57a3fa', old: '#177ddc' },
      { new: '#8dc0fb', old: '#165996' },
    ],
  },
];

if (!fs.existsSync(ANTD_CSS_PATH)) {
  throw new Error('Ant Design CSS path not found!');
}

// Remove existing PUBLIC folder and create a new one.
if (fs.existsSync(PUBLIC_PATH)) {
  rimraf.sync(PUBLIC_PATH);
}
fs.mkdirSync(PUBLIC_PATH, '0755');

// These Ant Design theme files are needed in the PUBLIC folder to support dynamic dark/light modes.
ANTD_CSS_FILES.forEach((file) => {
  const srcPath = `${ANTD_CSS_PATH}/${file}`;
  const dstPath = `${PUBLIC_PATH}/${file}`;

  if (!fs.existsSync(srcPath)) {
    throw new Error(`Ant Design CSS file "${srcPath}" not found!`);
  }

  console.log(`Copying ${srcPath} => ${dstPath}`);

  fs.copyFileSync(srcPath, dstPath);

  // Lighten main active color for dark theme CSS files.
  COLOR_UPDATES.forEach((changes) => {
    if (changes.match.test(file)) {
      try {
        console.log(`Reading content of ${dstPath}.`);
        let content = fs.readFileSync(dstPath, 'utf8');

        changes.updates.forEach((change) => {
          const regex = new RegExp(change.old, 'ig');
          content = content.replace(regex, change.new);
          console.log(`  Changing ${change.old} to ${change.new}.`);
        });

        console.log(`  Writing changes to ${dstPath}.`);
        fs.writeFileSync(dstPath, content, 'utf8');
      } catch (e) {
        console.error(e);
        throw new Error(`Unable to read "${dstPath}"!`);
      }
    }
  });
});
