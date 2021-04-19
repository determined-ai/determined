#!/usr/bin/env node
/* eslint-disable no-console */

const express = require('express');
const morgan = require('morgan');

const PORT = process.argv[2] || 8180;
const buildDir = process.cwd() + '/build';

const app = express();
app.use(morgan('dev'));

app.use(express.static(buildDir));

app.use('*', (req, res) => {
  res.sendFile(buildDir + '/index.html');
});

app.listen(PORT);
console.log(`Listening on http://localhost:${PORT}`);
