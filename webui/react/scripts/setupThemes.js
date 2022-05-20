#!/usr/bin/env node
/* eslint-disable no-console, @typescript-eslint/no-var-requires */

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

if (!fs.existsSync(ANTD_CSS_PATH)) {
  throw new Error('Ant Design CSS path not found!');
}

// Remove existing PUBLIC folder and create a new one.
if (fs.existsSync(PUBLIC_PATH)) {
  rimraf.sync(PUBLIC_PATH);
}
fs.mkdirSync(PUBLIC_PATH, '0755');

// These Ant Design theme files are needed in the PUBLIC folder to support dynamic dark/light modes.
ANTD_CSS_FILES.forEach(file => {
  const srcPath = `${ANTD_CSS_PATH}/${file}`;
  const dstPath = `${PUBLIC_PATH}/${file}`;

  if (!fs.existsSync(srcPath)) {
    throw new Error(`Ant Design CSS file "${srcPath}" not found!`);
  }

  console.log(`Copying ${srcPath} => ${dstPath}`);

  fs.copyFileSync(srcPath, dstPath);
});
